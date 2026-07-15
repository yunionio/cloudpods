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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
	"yunion.io/x/onecloud/pkg/aiproxy/extensions/visual"
	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/responses"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func responsesHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		start := time.Now()
		rec := newAPILogRecord(ctx, r, start)
		defer finishAPILogRecord(rec, start)
		failAPILogRecord(rec, http.StatusMethodNotAllowed, "invalid_method", nil)
		writeResponsesError(ctx, w, http.StatusMethodNotAllowed, "invalid_request_error", "only POST is supported")
		return
	}
	handleResponsesCreate(ctx, w, r)
}

func responsesRetrieveHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleResponsesSubResource(ctx, w, r, http.MethodGet, "", appsrv.AppContextGetParams(ctx))
}

func responsesCancelHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleResponsesSubResource(ctx, w, r, http.MethodPost, "cancel", appsrv.AppContextGetParams(ctx))
}

func responsesDeleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handleResponsesSubResource(ctx, w, r, http.MethodDelete, "", appsrv.AppContextGetParams(ctx))
}

func handleResponsesCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := newAPILogRecord(ctx, r, start)
	defer finishAPILogRecord(rec, start)

	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		failAPILogRecord(rec, http.StatusBadRequest, "read_body", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "read body: %v", err)
		return
	}
	body, err := jsonutils.Parse(raw)
	if err != nil {
		failAPILogRecord(rec, http.StatusBadRequest, "invalid_json", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body: %v", err)
		return
	}
	dict, ok := body.(*jsonutils.JSONDict)
	if !ok {
		failAPILogRecord(rec, http.StatusBadRequest, "invalid_body", nil)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "body must be a JSON object")
		return
	}
	fillAPILogFromBody(rec, dict)

	dbg := NewProxyDebugSession(ctx, "openai-responses")
	isStream := rec.Stream
	dbg.ClientRequest(r, dict, nil, isStream)

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
		vkLim = &api.SAiVirtualKeyLimits{MaxTokensPerRequest: up.MaxTokensPerRequest}
	}
	if err := models.EnsureResponsesMaxOutputTokens(dict, vkLim); err != nil {
		dbg.Error("max tokens: %v", err)
		failAPILogRecord(rec, http.StatusBadRequest, "max_tokens", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
		return
	}

	if visual.ShouldHandle(dict, up, isStream) {
		if !isStream {
			bodyOut, _, err := visual.HandleResponsesCreate(ctx, dict, up)
			if err != nil {
				dbg.Error("visual orchestration: %v", err)
				failAPILogRecord(rec, http.StatusBadGateway, "visual_orchestration", err)
				writeResponsesError(ctx, w, http.StatusBadGateway, "api_error", "visual orchestration: %v", err)
				return
			}
			dbg.ClientResponse(bodyOut)
			markAPILogSuccess(rec, bodyOut)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bodyOut)
			if up.AiKeyId != "" {
				models.RecordAiKeySuccess(up.AiKeyId)
			}
			return
		}

		chatBody, _, err := visual.HandleResponsesCreateChat(ctx, dict, up)
		if err != nil {
			dbg.Error("visual orchestration: %v", err)
			failAPILogRecord(rec, http.StatusBadGateway, "visual_orchestration", err)
			writeResponsesError(ctx, w, http.StatusBadGateway, "api_error", "visual orchestration: %v", err)
			return
		}
		payloads, err := openai.NonStreamChatCompletionToStreamPayloads(chatBody)
		if err != nil {
			dbg.Error("visual synthetic stream: %v", err)
			failAPILogRecord(rec, http.StatusBadGateway, "visual_synthetic_stream", err)
			writeResponsesError(ctx, w, http.StatusBadGateway, "api_error", "visual synthetic stream: %v", err)
			return
		}
		adapter, err := responses.GetAdapter(up.ProviderKey, up.APIMode)
		if err != nil {
			dbg.Error("adapter: %v", err)
			failAPILogRecord(rec, http.StatusBadRequest, "adapter", err)
			writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
			return
		}
		writeResponsesSyntheticStream(ctx, w, adapter, up.UpstreamModel, dict, payloads, up.AiKeyId, dbg, rec)
		return
	}

	adapter, err := responses.GetAdapter(up.ProviderKey, up.APIMode)
	if err != nil {
		dbg.Error("adapter: %v", err)
		failAPILogRecord(rec, http.StatusBadRequest, "adapter", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
		return
	}

	chatCtx := providers.ChatContextFromUpstream(up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode)
	if _, err := adapter.BuildUpstreamRequest(chatCtx, dict, isStream); err != nil {
		dbg.Error("provider request: %v", err)
		failAPILogRecord(rec, http.StatusBadRequest, "provider_request", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "provider request: %v", err)
		return
	}

	timeout := 120 * time.Second
	if isStream {
		timeout = 2 * time.Hour
	}
	build := func() (*upstream.Request, error) {
		req, err := buildResponsesUpstream(up, adapter, dict, isStream)
		if err == nil {
			dbg.UpstreamRequest(req)
		}
		return req, err
	}

	if !isStream {
		prov := providers.ChatProviderForUpstream(up.ProviderKey, up.APIMode)
		resp, uerr := upstreamWithKeyFailover(ctx, up, timeout, build)
		if uerr != nil {
			dbg.Error("upstream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
			recordUpstreamError(rec, uerr)
			writeResponsesUpstreamError(ctx, w, uerr)
			return
		}
		dbg.UpstreamResponse(resp.Body)
		bodyOut := resp.Body
		if adapter.ResponsesStreamPassthrough() {
			// passthrough body unchanged
		} else if responses.UsesAnthropicTranslation(up.ProviderKey, up.APIMode) {
			_, state, _ := openai.ResponsesToAnthropicMessages(dict, up.UpstreamModel)
			if norm, nerr := openai.AnthropicMessagesToResponses(bodyOut, state); nerr == nil && len(norm) > 0 {
				bodyOut = norm
			}
		} else {
			norm, nerr := prov.NormalizeResponse(bodyOut)
			if nerr == nil && len(norm) > 0 {
				bodyOut = norm
			}
			_, state, _ := openai.ResponsesToChatCompletions(dict, up.UpstreamModel)
			if converted, cerr := openai.ChatCompletionToResponses(bodyOut, state); cerr == nil && len(converted) > 0 {
				bodyOut = converted
			}
		}
		dbg.ClientResponse(bodyOut)
		markAPILogSuccess(rec, bodyOut)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bodyOut)
		return
	}

	if adapter.ResponsesStreamPassthrough() {
		ch, uerr := upstreamRawStreamWithKeyFailover(ctx, up, timeout, build, upstream.ChatCompletionStreamRaw)
		if uerr != nil {
			dbg.Error("upstream stream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
			recordUpstreamError(rec, uerr)
			writeResponsesUpstreamError(ctx, w, uerr)
			return
		}
		writeResponsesPassthroughStream(ctx, w, ch, up.AiKeyId, dbg, rec)
		return
	}

	if responses.UsesAnthropicTranslation(up.ProviderKey, up.APIMode) {
		ch, uerr := upstreamRawStreamWithKeyFailover(ctx, up, timeout, build, upstream.ChatCompletionStreamRaw)
		if uerr != nil {
			dbg.Error("upstream stream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
			recordUpstreamError(rec, uerr)
			writeResponsesUpstreamError(ctx, w, uerr)
			return
		}
		writeResponsesAnthropicTranslatedStream(ctx, w, ch, adapter, up.UpstreamModel, dict, up.AiKeyId, dbg, rec)
		return
	}

	prov := providers.ChatProviderForUpstream(up.ProviderKey, up.APIMode)
	ch, uerr := upstreamStreamWithKeyFailover(ctx, up, timeout, build, func(reqCtx context.Context, upReq *upstream.Request) (<-chan upstream.StreamChunk, *upstream.Error) {
		return responsesOpenAIStreamChunks(reqCtx, up, upReq, prov, adapter, dict, dbg)
	})
	if uerr != nil {
		dbg.Error("upstream stream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
		recordUpstreamError(rec, uerr)
		writeResponsesUpstreamError(ctx, w, uerr)
		return
	}
	writeResponsesTranslatedStream(ctx, w, ch, adapter, up.UpstreamModel, dict, up.AiKeyId, dbg, rec)
}

func handleResponsesSubResource(ctx context.Context, w http.ResponseWriter, r *http.Request, method, subAction string, params *appsrv.SAppParams) {
	dbg := NewProxyDebugSession(ctx, "openai-responses")
	dbg.ClientRequestNoBody(r)

	responseID := ""
	if params != nil {
		responseID = params.Params["<id>"]
	}
	if responseID == "" {
		dbg.Error("missing response id")
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "missing response id")
		return
	}

	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	probe := jsonutils.NewDict()
	if model := strings.TrimSpace(r.URL.Query().Get("model")); model != "" {
		probe.Set("model", jsonutils.NewString(model))
	} else {
		dbg.Error("model query parameter is required")
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "model query parameter is required")
		return
	}
	up, err := models.ResolveChatUpstream(ctx, userCred, vk, probe)
	if err != nil {
		dbg.Error("resolve upstream: %v", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	dbg.RoutingResolved(probe, up)

	adapter, err := responses.GetAdapter(up.ProviderKey, up.APIMode)
	if err != nil {
		dbg.Error("adapter: %v", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
		return
	}

	chatCtx := providers.ChatContextFromUpstream(up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode)
	httpReq, err := adapter.BuildSubResourceRequest(chatCtx, method, responseID, subAction, r.URL.Query())
	if err != nil {
		if errors.Is(err, providerapi.ErrResponsesSubResourceNotSupported) {
			dbg.Error("not supported: %v", err)
			writeResponsesError(ctx, w, http.StatusNotImplemented, "not_supported_error", "%v", err)
			return
		}
		dbg.Error("build sub resource: %v", err)
		writeResponsesError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
		return
	}

	if method == http.MethodPost {
		defer r.Body.Close()
		if raw, err := io.ReadAll(r.Body); err == nil && len(raw) > 0 {
			httpReq.Body = raw
		}
	}

	upReq := providers.ToUpstreamRequest(httpReq, up.APIKey)
	dbg.UpstreamRequest(upReq)
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	resp, uerr := upstream.HTTPDo(reqCtx, method, upReq)
	if uerr != nil {
		dbg.Error("upstream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
		writeResponsesUpstreamError(ctx, w, uerr)
		return
	}
	dbg.UpstreamResponse(resp.Body)
	if len(resp.Body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	dbg.ClientResponse(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Body)
}

func buildResponsesUpstream(
	up *models.ChatUpstream,
	adapter providerapi.ResponsesAdapter,
	dict *jsonutils.JSONDict,
	isStream bool,
) (*upstream.Request, error) {
	httpReq, err := adapter.BuildUpstreamRequest(providers.ChatContextFromUpstream(
		up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode,
	), dict, isStream)
	if err != nil {
		return nil, err
	}
	return providers.ToUpstreamRequest(httpReq, up.APIKey), nil
}

func responsesOpenAIStreamChunks(
	ctx context.Context,
	up *models.ChatUpstream,
	upReq *upstream.Request,
	prov providers.Provider,
	adapter providerapi.ResponsesAdapter,
	requestBody *jsonutils.JSONDict,
	dbg *ProxyDebugSession,
) (<-chan upstream.StreamChunk, *upstream.Error) {
	chatCtx := &providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}
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
	_ = adapter
	_ = requestBody
	return out, nil
}

func writeResponsesError(ctx context.Context, w http.ResponseWriter, status int, errType, format string, args ...interface{}) {
	if ctx.Err() != nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": msg,
			"type":    errType,
		},
	})
}

func writeResponsesUpstreamError(ctx context.Context, w http.ResponseWriter, uerr *upstream.Error) {
	writeUpstreamError(ctx, w, uerr)
}

func writeResponsesPassthroughStream(ctx context.Context, w http.ResponseWriter, ch <-chan upstream.RawSSEEvent, aiKeyId string, dbg *ProxyDebugSession, rec *chatlog.Record) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)
	streamOK := true
	sawUsage := false
	seq := 0
	for evt := range ch {
		seq++
		dbg.ClientStreamSSE(seq, evt.Event, evt.Data)
		if len(evt.Data) > 0 && bytes.Contains(evt.Data, []byte(`"usage"`)) {
			noteStreamUsage(rec, evt.Data, &sawUsage)
		}
		if evt.Event != "" {
			_, _ = fmt.Fprintf(w, "event: %s\n", evt.Event)
		}
		if len(evt.Data) > 0 {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", evt.Data)
		} else if evt.Event != "" {
			_, _ = fmt.Fprint(w, "\n")
		}
		flushIf(w)
	}
	finishAPILogStream(rec, streamOK, sawUsage)
	if streamOK && aiKeyId != "" {
		models.RecordAiKeySuccess(aiKeyId)
	}
}

func writeResponsesSyntheticStream(
	ctx context.Context,
	w http.ResponseWriter,
	adapter providerapi.ResponsesAdapter,
	requestModel string,
	requestBody *jsonutils.JSONDict,
	payloads [][]byte,
	aiKeyId string,
	dbg *ProxyDebugSession,
	rec *chatlog.Record,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	state := adapter.NewStreamState(requestModel, requestBody)
	streamOK := true
	sawUsage := false
	outSeq := 0
	for _, payload := range payloads {
		if len(payload) > 0 && bytes.Contains(payload, []byte(`"usage"`)) {
			noteStreamUsage(rec, payload, &sawUsage)
		}
		events, err := adapter.ConvertStreamPayload(state, payload, false)
		if err != nil {
			dbg.Error("visual synthetic stream convert error: %v", err)
			streamOK = false
			failAPILogRecord(rec, http.StatusOK, "stream_convert", err)
			break
		}
		outSeq++
		noteResponsesSSEUsage(rec, events, &sawUsage)
		writeResponsesSSEEvents(w, events, dbg, outSeq)
	}
	if streamOK {
		events, err := adapter.ConvertStreamPayload(state, nil, true)
		if err != nil {
			dbg.Error("visual synthetic stream end error: %v", err)
			streamOK = false
			failAPILogRecord(rec, http.StatusOK, "stream_convert", err)
		} else {
			outSeq++
			noteResponsesSSEUsage(rec, events, &sawUsage)
			writeResponsesSSEEvents(w, events, dbg, outSeq)
		}
	}
	finishAPILogStream(rec, streamOK, sawUsage)
	if streamOK && aiKeyId != "" {
		models.RecordAiKeySuccess(aiKeyId)
	}
}

func writeResponsesTranslatedStream(
	ctx context.Context,
	w http.ResponseWriter,
	ch <-chan upstream.StreamChunk,
	adapter providerapi.ResponsesAdapter,
	requestModel string,
	requestBody *jsonutils.JSONDict,
	aiKeyId string,
	dbg *ProxyDebugSession,
	rec *chatlog.Record,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	state := adapter.NewStreamState(requestModel, requestBody)
	streamOK := true
	sawUsage := false
	outSeq := 0
	for chunk := range ch {
		if chunk.Done {
			events, err := adapter.ConvertStreamPayload(state, nil, true)
			if err != nil {
				dbg.Error("stream convert end error: %v", err)
				streamOK = false
				failAPILogRecord(rec, http.StatusOK, "stream_convert", err)
				break
			}
			outSeq++
			noteResponsesSSEUsage(rec, events, &sawUsage)
			writeResponsesSSEEvents(w, events, dbg, outSeq)
			break
		}
		if len(chunk.Data) == 0 {
			continue
		}
		if bytes.Contains(chunk.Data, []byte(`"usage"`)) {
			noteStreamUsage(rec, chunk.Data, &sawUsage)
		}
		if isResponsesUpstreamErrorChunk(chunk.Data) {
			dbg.Error("upstream stream error chunk: %s", truncateLogBytes(chunk.Data, proxyDebugLogMax))
			streamOK = false
			rec.Success = false
			rec.ErrorCode, rec.ErrorMessage = parseUpstreamErrorInfo(chunk.Data)
			if aiKeyId != "" {
				models.RecordAiKeyFailure(aiKeyId, parseUpstreamErrorStatus(chunk.Data))
			}
			_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", chunk.Data)
			flushIf(w)
			break
		}
		events, err := adapter.ConvertStreamPayload(state, chunk.Data, false)
		if err != nil {
			dbg.Error("stream convert error: %v", err)
			streamOK = false
			failAPILogRecord(rec, http.StatusOK, "stream_convert", err)
			break
		}
		outSeq++
		noteResponsesSSEUsage(rec, events, &sawUsage)
		writeResponsesSSEEvents(w, events, dbg, outSeq)
	}
	finishAPILogStream(rec, streamOK, sawUsage)
	if streamOK && aiKeyId != "" {
		models.RecordAiKeySuccess(aiKeyId)
	}
}

func writeResponsesAnthropicTranslatedStream(
	ctx context.Context,
	w http.ResponseWriter,
	ch <-chan upstream.RawSSEEvent,
	adapter providerapi.ResponsesAdapter,
	requestModel string,
	requestBody *jsonutils.JSONDict,
	aiKeyId string,
	dbg *ProxyDebugSession,
	rec *chatlog.Record,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	state := adapter.NewStreamState(requestModel, requestBody)
	streamOK := true
	sawUsage := false
	outSeq := 0
	for evt := range ch {
		outSeq++
		dbg.UpstreamStreamChunk(outSeq, evt.Data)
		if len(evt.Data) > 0 && bytes.Contains(evt.Data, []byte(`"usage"`)) {
			noteStreamUsage(rec, evt.Data, &sawUsage)
		}
		events, err := responses.ConvertAnthropicStreamEvent(state, evt.Event, evt.Data, false)
		if err != nil {
			dbg.Error("stream convert error: %v", err)
			streamOK = false
			failAPILogRecord(rec, http.StatusOK, "stream_convert", err)
			break
		}
		noteResponsesSSEUsage(rec, events, &sawUsage)
		writeResponsesSSEEvents(w, events, dbg, outSeq)
	}
	events, err := responses.ConvertAnthropicStreamEvent(state, "", nil, true)
	if err == nil {
		outSeq++
		noteResponsesSSEUsage(rec, events, &sawUsage)
		writeResponsesSSEEvents(w, events, dbg, outSeq)
	}
	finishAPILogStream(rec, streamOK, sawUsage)
	if streamOK && aiKeyId != "" {
		models.RecordAiKeySuccess(aiKeyId)
	}
}

func writeResponsesSSEEvents(w http.ResponseWriter, events []providerapi.ResponsesStreamChunk, dbg *ProxyDebugSession, seq int) {
	for i, evt := range events {
		logSeq := seq
		if len(events) > 1 {
			dbg.ClientStreamSSE(logSeq, fmt.Sprintf("%s.%d", evt.Event, i), evt.Data)
		} else {
			dbg.ClientStreamSSE(logSeq, evt.Event, evt.Data)
		}
		if evt.Event != "" {
			_, _ = fmt.Fprintf(w, "event: %s\n", evt.Event)
		}
		if len(evt.Data) > 0 {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(evt.Data))
		} else if evt.Event != "" {
			_, _ = fmt.Fprint(w, "\n")
		}
		flushIf(w)
	}
}

func noteResponsesSSEUsage(rec *chatlog.Record, events []providerapi.ResponsesStreamChunk, sawUsage *bool) {
	for _, evt := range events {
		if len(evt.Data) > 0 && bytes.Contains(evt.Data, []byte(`"usage"`)) {
			noteStreamUsage(rec, evt.Data, sawUsage)
		}
	}
}

func isResponsesUpstreamErrorChunk(data []byte) bool {
	var wrap struct {
		Error interface{} `json:"error"`
	}
	return json.Unmarshal(data, &wrap) == nil && wrap.Error != nil
}
