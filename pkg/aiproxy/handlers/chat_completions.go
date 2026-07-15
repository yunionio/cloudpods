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
	rec := newAPILogRecord(ctx, r, start)
	defer finishAPILogRecord(rec, start)

	if r.Method != http.MethodPost {
		failAPILogRecord(rec, http.StatusBadRequest, "invalid_method", nil)
		httperrors.InvalidInputError(ctx, w, "only POST is supported")
		return
	}

	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		failAPILogRecord(rec, http.StatusBadRequest, "read_body", err)
		httperrors.InvalidInputError(ctx, w, "read body: %v", err)
		return
	}

	body, err := jsonutils.Parse(raw)
	if err != nil {
		failAPILogRecord(rec, http.StatusBadRequest, "invalid_json", err)
		httperrors.InvalidInputError(ctx, w, "invalid JSON body: %v", err)
		return
	}
	dict, ok := body.(*jsonutils.JSONDict)
	if !ok {
		failAPILogRecord(rec, http.StatusBadRequest, "invalid_body", nil)
		httperrors.InvalidInputError(ctx, w, "body must be a JSON object")
		return
	}
	fillAPILogFromBody(rec, dict)

	dbg := NewProxyDebugSession(ctx, "openai-chat")
	dbg.ClientRequest(r, dict, nil, rec.Stream)

	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	up, err := models.ResolveChatUpstream(ctx, userCred, vk, dict)
	if err != nil {
		dbg.Error("resolve upstream: %v", err)
		failAPILogRecord(rec, http.StatusInternalServerError, "resolve_upstream", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	dbg.RoutingResolved(dict, up)
	fillAPILogFromUpstream(rec, up)

	if err := models.TakeVirtualKeyRequestsPerMinute(up.VirtualKeyId, up.RequestsPerMinute); err != nil {
		dbg.Error("rate limit: %v", err)
		failAPILogRecord(rec, http.StatusInternalServerError, "rate_limit", err)
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
		dbg.Error("max tokens: %v", err)
		failAPILogRecord(rec, http.StatusInternalServerError, "max_tokens", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	isStream := rec.Stream
	prov := providers.ChatProviderForUpstream(up.ProviderKey, up.APIMode)
	chatCtx := providers.ChatContextFromUpstream(up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode)
	if _, err := prov.BuildUpstreamRequest(chatCtx, dict, isStream); err != nil {
		dbg.Error("provider request: %v", err)
		failAPILogRecord(rec, http.StatusBadRequest, "provider_request", err)
		httperrors.InvalidInputError(ctx, w, "provider request: %v", err)
		return
	}
	timeout := 120 * time.Second
	if isStream {
		timeout = 2 * time.Hour
	}
	buildUpstream := func() (*upstream.Request, error) {
		req, err := buildProviderUpstream(up, dict, isStream)
		if err == nil {
			dbg.UpstreamRequest(req)
		}
		return req, err
	}

	if !isStream {
		resp, uerr := upstreamWithKeyFailover(ctx, up, timeout, buildUpstream)
		if uerr != nil {
			dbg.Error("upstream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
			recordUpstreamError(rec, uerr)
			writeUpstreamError(ctx, w, uerr)
			return
		}
		dbg.UpstreamResponse(resp.Body)
		out := resp.Body
		if norm, nerr := prov.NormalizeResponse(out); nerr == nil && len(norm) > 0 {
			out = norm
		}
		dbg.ClientResponse(out)
		chatlog.FillToolCallsFromJSON(rec, out)
		markAPILogSuccess(rec, out)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(out)
		return
	}

	ch, uerr := upstreamStreamWithKeyFailover(ctx, up, timeout, buildUpstream, func(reqCtx context.Context, upReq *upstream.Request) (<-chan upstream.StreamChunk, *upstream.Error) {
		return providerStreamChunks(reqCtx, up, upReq, prov, dbg)
	})
	if uerr != nil {
		dbg.Error("upstream stream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
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
	clientSeq := 0
	for chunk := range ch {
		if chunk.Done {
			break
		}
		if len(chunk.Data) == 0 {
			continue
		}
		if bytes.Contains(chunk.Data, []byte(`"usage"`)) {
			noteStreamUsage(rec, chunk.Data, &sawUsage)
		}
		chatlog.FillToolCallsFromJSON(rec, chunk.Data)
		if isUpstreamErrorChunk(chunk.Data) {
			dbg.Error("upstream stream error chunk: %s", truncateLogBytes(chunk.Data, proxyDebugLogMax))
			streamOK = false
			rec.Success = false
			rec.ErrorCode, rec.ErrorMessage = parseUpstreamErrorInfo(chunk.Data)
			models.RecordAiKeyFailure(up.AiKeyId, parseUpstreamErrorStatus(chunk.Data))
			clientSeq++
			dbg.ClientStreamData(clientSeq, chunk.Data)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk.Data)
			flushIf(w)
			break
		}
		clientSeq++
		dbg.ClientStreamData(clientSeq, chunk.Data)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk.Data)
		flushIf(w)
	}
	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
	flushIf(w)
	finishAPILogStream(rec, streamOK, sawUsage)
	if streamOK {
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

func buildProviderUpstream(up *models.ChatUpstream, dict *jsonutils.JSONDict, isStream bool) (*upstream.Request, error) {
	prov := providers.ChatProviderForUpstream(up.ProviderKey, up.APIMode)
	httpReq, err := prov.BuildUpstreamRequest(providers.ChatContextFromUpstream(
		up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode,
	), dict, isStream)
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
	dbg *ProxyDebugSession,
) (<-chan upstream.StreamChunk, *upstream.Error) {
	chatCtx := providers.ChatContextFromUpstream(up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode)
	if providers.OpenAIStreamPassthrough(prov, chatCtx) {
		ch, uerr := upstream.ChatCompletionStream(ctx, upReq)
		if uerr != nil {
			return nil, uerr
		}
		out := make(chan upstream.StreamChunk, 16)
		go func() {
			defer close(out)
			seq := 0
			for chunk := range ch {
				if len(chunk.Data) > 0 {
					seq++
					dbg.UpstreamStreamChunk(seq, chunk.Data)
				}
				out <- chunk
			}
		}()
		return out, nil
	}
	rawCh, uerr := upstream.ChatCompletionStreamRaw(ctx, upReq)
	if uerr != nil {
		return nil, uerr
	}
	out := make(chan upstream.StreamChunk, 16)
	go func() {
		defer close(out)
		state := &providers.StreamState{Model: up.UpstreamModel}
		seq := 0
		for evt := range rawCh {
			if len(evt.Data) > 0 {
				seq++
				dbg.UpstreamStreamChunk(seq, evt.Data)
			}
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
