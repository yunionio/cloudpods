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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/messages"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// messagesHandler implements Anthropic-compatible POST /ai/anthropic/v1/messages.
func messagesHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "only POST is supported")
		return
	}

	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "read body: %v", err)
		return
	}

	body, err := jsonutils.Parse(raw)
	if err != nil {
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body: %v", err)
		return
	}
	dict, ok := body.(*jsonutils.JSONDict)
	if !ok {
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "body must be a JSON object")
		return
	}

	dbg := NewProxyDebugSession(ctx, "anthropic-messages")
	isStream, _ := dict.Bool("stream")
	dbg.ClientRequest(r, dict, nil, isStream)

	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	up, err := models.ResolveChatUpstream(ctx, userCred, vk, dict)
	if err != nil {
		dbg.Error("resolve upstream: %v", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	dbg.RoutingResolved(dict, up)

	if err := models.TakeVirtualKeyRequestsPerMinute(up.VirtualKeyId, up.RequestsPerMinute); err != nil {
		dbg.Error("rate limit: %v", err)
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
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
		return
	}

	adapter, err := messages.GetAdapter(up.ProviderKey, up.APIMode)
	if err != nil {
		dbg.Error("adapter: %v", err)
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "%v", err)
		return
	}

	chatCtx := providers.ChatContextFromUpstream(up.ProviderKey, up.BaseURL, up.APIKey, up.UpstreamModel, up.APIMode)
	if _, err := adapter.BuildUpstreamRequest(chatCtx, dict, isStream); err != nil {
		dbg.Error("provider request: %v", err)
		writeAnthropicError(ctx, w, http.StatusBadRequest, "invalid_request_error", "provider request: %v", err)
		return
	}

	timeout := 120 * time.Second
	if isStream {
		timeout = 2 * time.Hour
	}

	build := func() (*upstream.Request, error) {
		req, err := buildMessagesUpstream(up, adapter, dict, isStream)
		if err == nil {
			dbg.UpstreamRequest(req)
		}
		return req, err
	}

	if !isStream {
		prov := providers.Get(up.ProviderKey)
		resp, uerr := upstreamWithKeyFailover(ctx, up, timeout, build)
		if uerr != nil {
			dbg.Error("upstream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
			writeMessagesUpstreamError(ctx, w, adapter, uerr)
			return
		}
		bodyOut := resp.Body
		dbg.UpstreamResponse(bodyOut)
		if norm, nerr := adapter.NormalizeResponse(prov, bodyOut); nerr == nil && len(norm) > 0 {
			bodyOut = norm
		}
		dbg.ClientResponse(bodyOut)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bodyOut)
		return
	}

	if adapter.AnthropicStreamPassthrough() {
		ch, uerr := upstreamRawStreamWithKeyFailover(ctx, up, timeout, build, upstream.ChatCompletionStreamRaw)
		if uerr != nil {
			dbg.Error("upstream stream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
			writeMessagesUpstreamError(ctx, w, adapter, uerr)
			return
		}
		writeAnthropicPassthroughStream(ctx, w, ch, up.AiKeyId, dbg)
		return
	}

	prov := providers.Get(up.ProviderKey)
	ch, uerr := upstreamStreamWithKeyFailover(ctx, up, timeout, build, func(reqCtx context.Context, upReq *upstream.Request) (<-chan upstream.StreamChunk, *upstream.Error) {
		return messagesOpenAIStreamChunks(reqCtx, up, upReq, prov, dbg)
	})
	if uerr != nil {
		dbg.Error("upstream stream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
		writeMessagesUpstreamError(ctx, w, adapter, uerr)
		return
	}
	writeAnthropicTranslatedStream(ctx, w, ch, adapter, up.UpstreamModel, up.AiKeyId, dbg)
}

func buildMessagesUpstream(
	up *models.ChatUpstream,
	adapter providerapi.MessagesAdapter,
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

func messagesOpenAIStreamChunks(
	ctx context.Context,
	up *models.ChatUpstream,
	upReq *upstream.Request,
	prov providers.Provider,
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
			seq++
			dbg.UpstreamStreamChunk(seq, evt.Data)
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

func writeAnthropicError(ctx context.Context, w http.ResponseWriter, status int, errType, format string, args ...interface{}) {
	if ctx.Err() != nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(openai.NewAnthropicErrorBody(errType, msg))
}

func writeMessagesUpstreamError(ctx context.Context, w http.ResponseWriter, adapter providerapi.MessagesAdapter, uerr *upstream.Error) {
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
		if adapter.AnthropicStreamPassthrough() {
			_, _ = w.Write(uerr.Body)
			return
		}
		_, _ = w.Write(openai.OpenAIErrorToAnthropic(uerr.Body, status))
		return
	}
	msg := "upstream request failed"
	if uerr != nil {
		msg = uerr.Error()
	}
	_, _ = w.Write(openai.NewAnthropicErrorBody("api_error", msg))
}

func writeAnthropicPassthroughStream(ctx context.Context, w http.ResponseWriter, ch <-chan upstream.RawSSEEvent, aiKeyId string, dbg *ProxyDebugSession) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	streamOK := true
	seq := 0
	for evt := range ch {
		seq++
		dbg.ClientStreamSSE(seq, evt.Event, evt.Data)
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
	if streamOK && aiKeyId != "" {
		models.RecordAiKeySuccess(aiKeyId)
	}
}

func writeAnthropicTranslatedStream(
	ctx context.Context,
	w http.ResponseWriter,
	ch <-chan upstream.StreamChunk,
	adapter providerapi.MessagesAdapter,
	requestModel string,
	aiKeyId string,
	dbg *ProxyDebugSession,
) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	state := adapter.NewStreamState(requestModel)
	streamOK := true
	outSeq := 0
	for chunk := range ch {
		if chunk.Done {
			events, err := adapter.ConvertStreamPayload(state, nil, true)
			if err != nil {
				dbg.Error("stream convert end error: %v", err)
				streamOK = false
				break
			}
			outSeq++
			dbg.ClientStreamEvents(outSeq, events)
			writeAnthropicSSEEvents(w, events)
			break
		}
		if len(chunk.Data) == 0 {
			continue
		}
		if isAnthropicUpstreamErrorChunk(chunk.Data) {
			dbg.Error("upstream stream error chunk: %s", truncateLogBytes(chunk.Data, proxyDebugLogMax))
			streamOK = false
			if aiKeyId != "" {
				models.RecordAiKeyFailure(aiKeyId, parseUpstreamErrorStatus(chunk.Data))
			}
			_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", openai.OpenAIErrorToAnthropic(chunk.Data, http.StatusBadGateway))
			flushIf(w)
			break
		}
		events, err := adapter.ConvertStreamPayload(state, chunk.Data, false)
		if err != nil {
			dbg.Error("stream convert error: %v upstream_chunk=%s", err, truncateLogBytes(chunk.Data, proxyDebugLogMax))
			streamOK = false
			break
		}
		outSeq++
		dbg.ClientStreamEvents(outSeq, events)
		writeAnthropicSSEEvents(w, events)
	}
	if streamOK && aiKeyId != "" {
		models.RecordAiKeySuccess(aiKeyId)
	}
}

func writeAnthropicSSEEvents(w http.ResponseWriter, events []providerapi.AnthropicStreamChunk) {
	for _, evt := range events {
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

func isAnthropicUpstreamErrorChunk(data []byte) bool {
	var wrap struct {
		Error interface{} `json:"error"`
	}
	return json.Unmarshal(data, &wrap) == nil && wrap.Error != nil
}
