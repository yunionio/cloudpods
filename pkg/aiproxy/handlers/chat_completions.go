// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const headerAiVirtualKey = "X-Ai-Virtual-Key"

func extractVirtualKey(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get(headerAiVirtualKey)); v != "" {
		return v
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authz, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func upstreamErrorStatusCode(uerr *upstream.Error) int {
	if uerr == nil || uerr.StatusCode <= 0 {
		return 0
	}
	return uerr.StatusCode
}

func writeUpstreamError(ctx context.Context, w http.ResponseWriter, uerr *upstream.Error) {
	if ctx.Err() != nil {
		return
	}
	status := http.StatusBadGateway
	if uerr != nil && uerr.StatusCode > 0 {
		status = uerr.StatusCode
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if uerr != nil && len(uerr.Body) > 0 {
		_, _ = w.Write(uerr.Body)
		return
	}
	msg := "upstream request failed"
	if uerr != nil {
		msg = uerr.Error()
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": msg,
		},
	})
}

func flushIf(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// streamChunksWithCancel forwards chunks until ch closes, then releases reqCtx.
func streamChunksWithCancel(ch <-chan upstream.StreamChunk, cancel context.CancelFunc) <-chan upstream.StreamChunk {
	out := make(chan upstream.StreamChunk, 16)
	go func() {
		defer cancel()
		defer close(out)
		for chunk := range ch {
			out <- chunk
		}
	}()
	return out
}

// chatCompletionsHandler implements OpenAI-compatible POST /openai/v1/chat/completions.
// Auth is the ai_virtual_key only (Authorization: Bearer <vk> or X-Ai-Virtual-Key).
// Upstream is resolved: ai_virtual_key -> project ai_routing -> ai_routing_model -> ai_key (by catalog model_key).
func chatCompletionsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &chatlog.Record{
		RequestID: appctx.AppContextRequestId(ctx),
		Timestamp: start,
		Path:      r.URL.Path,
		Client:    r.RemoteAddr,
	}
	defer func() {
		if rec.StatusCode == 0 {
			rec.StatusCode = http.StatusInternalServerError
		}
		rec.LatencyMs = time.Since(start).Milliseconds()
		chatlog.Write(rec)
	}()
	fail := func(status int, code string, err error) {
		rec.StatusCode = status
		rec.Success = false
		rec.ErrorCode = code
		if err != nil {
			rec.ErrorMessage = err.Error()
		}
	}

	if r.Method != http.MethodPost {
		fail(http.StatusBadRequest, "invalid_method", nil)
		httperrors.InvalidInputError(ctx, w, "only POST is supported")
		return
	}

	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		fail(http.StatusBadRequest, "read_body", err)
		httperrors.InvalidInputError(ctx, w, "read body: %v", err)
		return
	}

	body, err := jsonutils.Parse(raw)
	if err != nil {
		fail(http.StatusBadRequest, "invalid_json", err)
		httperrors.InvalidInputError(ctx, w, "invalid JSON body: %v", err)
		return
	}
	dict, ok := body.(*jsonutils.JSONDict)
	if !ok {
		fail(http.StatusBadRequest, "invalid_body", nil)
		httperrors.InvalidInputError(ctx, w, "body must be a JSON object")
		return
	}
	rec.ModelRequested, _ = dict.GetString("model")
	rec.Stream, _ = dict.Bool("stream")
	rec.ToolCallEnabled = dict.Contains("tools") || dict.Contains("tool_choice")
	if metadata, err := dict.Get("metadata"); err == nil {
		rec.Metadata = json.RawMessage([]byte(metadata.String()))
	}

	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	up, err := models.ResolveChatUpstream(ctx, userCred, vk, dict)
	if err != nil {
		fail(http.StatusInternalServerError, "resolve_upstream", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	rec.VirtualKey = up.VirtualKeyId
	rec.ProjectID = up.ProjectId
	rec.DomainID = up.DomainId
	rec.AiKey = up.AiKeyId
	rec.ModelFinal = up.UpstreamModel
	rec.Provider = up.ProviderKey
	if up.RoutingLog != nil {
		rec.RoutingEnabled = up.RoutingLog.Enabled
		rec.RoutingCandidates = up.RoutingLog.Candidates
		rec.RoutingSelectedModel = up.RoutingLog.SelectedModel
		rec.RoutingMethod = up.RoutingLog.Method
		rec.RoutingScores = up.RoutingLog.Scores
		rec.RoutingConfidence = up.RoutingLog.Confidence
		rec.RoutingReason = up.RoutingLog.Reason
		rec.RoutingLatencyMs = up.RoutingLog.LatencyMs
		rec.RoutingError = up.RoutingLog.Error
	}

	if err := models.TakeVirtualKeyRequestsPerMinute(up.VirtualKeyId, up.RequestsPerMinute); err != nil {
		fail(http.StatusInternalServerError, "rate_limit", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	var vkLim *api.SAiVirtualKeyLimits
	if up.MaxTokensPerRequest > 0 {
		vkLim = &api.SAiVirtualKeyLimits{
			MaxTokensPerRequest: up.MaxTokensPerRequest,
		}
	}
	if err := models.EnforceVirtualKeyMaxTokens(dict, vkLim); err != nil {
		fail(http.StatusInternalServerError, "max_tokens", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	isStream := rec.Stream
	prov := providers.Get(up.ProviderKey)
	if _, err := prov.BuildUpstreamRequest(&providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}, dict, isStream); err != nil {
		fail(http.StatusBadRequest, "provider_request", err)
		httperrors.InvalidInputError(ctx, w, "provider request: %v", err)
		return
	}
	timeout := 120 * time.Second
	if isStream {
		timeout = 2 * time.Hour
	}

	if !isStream {
		resp, uerr := chatCompletionWithKeyFailover(ctx, up, dict, isStream, timeout)
		if uerr != nil {
			recordUpstreamError(rec, uerr)
			writeUpstreamError(ctx, w, uerr)
			return
		}
		body := resp.Body
		if norm, nerr := prov.NormalizeResponse(body); nerr == nil && len(norm) > 0 {
			body = norm
		}
		chatlog.FillUsageFromJSON(rec, body)
		chatlog.FillToolCallsFromJSON(rec, body)
		rec.Success = true
		rec.StatusCode = http.StatusOK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}

	ch, uerr := chatCompletionStreamWithKeyFailover(ctx, up, dict, isStream, prov, timeout)
	if uerr != nil {
		recordUpstreamError(rec, uerr)
		writeUpstreamError(ctx, w, uerr)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	streamOK := true
	sawUsage := false
	for chunk := range ch {
		if chunk.Done {
			break
		}
		if len(chunk.Data) == 0 {
			continue
		}
		if bytes.Contains(chunk.Data, []byte(`"usage"`)) && chatlog.FillUsageFromJSON(rec, chunk.Data) {
			sawUsage = true
		}
		chatlog.FillToolCallsFromJSON(rec, chunk.Data)
		if isUpstreamErrorChunk(chunk.Data) {
			streamOK = false
			rec.Success = false
			rec.ErrorCode, rec.ErrorMessage = parseUpstreamErrorInfo(chunk.Data)
			models.RecordAiKeyFailure(up.AiKeyId, parseUpstreamErrorStatus(chunk.Data))
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk.Data)
			flushIf(w)
			break
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk.Data)
		flushIf(w)
	}
	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	flushIf(w)
	rec.StatusCode = http.StatusOK
	if !sawUsage {
		rec.UsageMissing = true
	}
	if streamOK {
		rec.Success = true
		models.RecordAiKeySuccess(up.AiKeyId)
	}
}

func recordUpstreamError(rec *chatlog.Record, uerr *upstream.Error) {
	if rec == nil {
		return
	}
	rec.StatusCode = http.StatusBadGateway
	if uerr != nil && uerr.StatusCode > 0 {
		rec.StatusCode = uerr.StatusCode
	}
	rec.Success = false
	rec.ErrorCode = "upstream_error"
	if uerr != nil {
		rec.ErrorMessage = uerr.Error()
		if len(uerr.Body) > 0 {
			if code, msg := parseUpstreamErrorInfo(uerr.Body); code != "" || msg != "" {
				rec.ErrorCode = code
				if msg != "" {
					rec.ErrorMessage = msg
				}
			}
		}
	}
}

func parseUpstreamErrorInfo(data []byte) (string, string) {
	var wrap struct {
		Error struct {
			Code    interface{} `json:"code"`
			Message string      `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &wrap) != nil {
		return "upstream_error", strings.TrimSpace(string(data))
	}
	code := "upstream_error"
	if wrap.Error.Code != nil {
		code = fmt.Sprint(wrap.Error.Code)
	}
	return code, wrap.Error.Message
}

func isUpstreamErrorChunk(data []byte) bool {
	var wrap struct {
		Error interface{} `json:"error"`
	}
	return json.Unmarshal(data, &wrap) == nil && wrap.Error != nil
}

func parseUpstreamErrorStatus(data []byte) int {
	var wrap struct {
		Error struct {
			Code interface{} `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &wrap) != nil {
		return 0
	}
	switch c := wrap.Error.Code.(type) {
	case float64:
		return int(c)
	case int:
		return c
	default:
		return 0
	}
}

func chatCompletionWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	dict *jsonutils.JSONDict,
	stream bool,
	timeout time.Duration,
) (*upstream.Response, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := buildProviderUpstream(up, dict, stream)
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, uerr := upstream.ChatCompletion(reqCtx, upReq)
		cancel()
		if uerr == nil {
			models.RecordAiKeySuccess(up.AiKeyId)
			return resp, nil
		}
		last = uerr
		status := upstreamErrorStatusCode(uerr)
		models.RecordAiKeyFailure(up.AiKeyId, status)
		if up.AiKeyId == "" || !models.IsRetryableAiKeyUpstreamStatus(status) || attempt+1 >= models.MaxAiKeyFailoverAttempts {
			break
		}
		if err := models.RepickUpstreamAPIKey(up, tried); err != nil {
			break
		}
		if up.AiKeyId != "" {
			tried[up.AiKeyId] = true
		}
	}
	return nil, last
}

func chatCompletionStreamWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	dict *jsonutils.JSONDict,
	stream bool,
	prov providers.Provider,
	timeout time.Duration,
) (<-chan upstream.StreamChunk, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := buildProviderUpstream(up, dict, stream)
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		ch, uerr := providerStreamChunks(reqCtx, up, upReq, prov)
		if uerr != nil {
			cancel()
		} else {
			ch = streamChunksWithCancel(ch, cancel)
		}
		if uerr == nil {
			return ch, nil
		}
		last = uerr
		status := upstreamErrorStatusCode(uerr)
		models.RecordAiKeyFailure(up.AiKeyId, status)
		if up.AiKeyId == "" || !models.IsRetryableAiKeyUpstreamStatus(status) || attempt+1 >= models.MaxAiKeyFailoverAttempts {
			break
		}
		if err := models.RepickUpstreamAPIKey(up, tried); err != nil {
			break
		}
		if up.AiKeyId != "" {
			tried[up.AiKeyId] = true
		}
	}
	return nil, last
}

func buildProviderUpstream(up *models.ChatUpstream, dict *jsonutils.JSONDict, isStream bool) (*upstream.Request, error) {
	prov := providers.Get(up.ProviderKey)
	httpReq, err := prov.BuildUpstreamRequest(&providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}, dict, isStream)
	if err != nil {
		return nil, err
	}
	return providers.ToUpstreamRequest(httpReq, up.APIKey), nil
}

func providerStreamChunks(
	ctx context.Context,
	up *models.ChatUpstream,
	upReq *upstream.Request,
	prov providers.Provider,
) (<-chan upstream.StreamChunk, *upstream.Error) {
	chatCtx := &providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}
	if providers.OpenAIStreamPassthrough(prov, chatCtx) {
		return upstream.ChatCompletionStream(ctx, upReq)
	}
	rawCh, uerr := upstream.ChatCompletionStreamRaw(ctx, upReq)
	if uerr != nil {
		return nil, uerr
	}
	out := make(chan upstream.StreamChunk, 16)
	go func() {
		defer close(out)
		state := &providers.StreamState{Model: up.UpstreamModel}
		for evt := range rawCh {
			chunks, err := prov.ConvertStreamEvent(evt.Event, evt.Data, state)
			if err != nil {
				msg, _ := json.Marshal(map[string]interface{}{
					"error": map[string]interface{}{"message": err.Error()},
				})
				out <- upstream.StreamChunk{Data: msg}
				return
			}
			for _, c := range chunks {
				if len(c.Data) > 0 {
					out <- upstream.StreamChunk{Data: c.Data}
				}
				if c.Done {
					out <- upstream.StreamChunk{Done: true}
					return
				}
			}
		}
	}()
	return out, nil
}
