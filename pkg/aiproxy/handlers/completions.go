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
	"fmt"
	"io"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// completionsHandler implements OpenAI-compatible POST /openai/v1/completions.
func completionsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httperrors.InvalidInputError(ctx, w, "only POST is supported")
		return
	}

	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "read body: %v", err)
		return
	}

	body, err := jsonutils.Parse(raw)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "invalid JSON body: %v", err)
		return
	}
	dict, ok := body.(*jsonutils.JSONDict)
	if !ok {
		httperrors.InvalidInputError(ctx, w, "body must be a JSON object")
		return
	}

	vk := extractVirtualKey(r)
	userCred := auth.AdminCredential()
	up, err := models.ResolveChatUpstream(ctx, userCred, vk, dict)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	if err := models.TakeVirtualKeyRequestsPerMinute(up.VirtualKeyId, up.RequestsPerMinute); err != nil {
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
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	compProv, err := providers.GetCompletions(up.ProviderKey)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "%v", err)
		return
	}

	isStream, _ := dict.Bool("stream")
	if _, err := compProv.BuildCompletionsRequest(&providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}, dict, isStream); err != nil {
		httperrors.InvalidInputError(ctx, w, "provider request: %v", err)
		return
	}

	timeout := 120 * time.Second
	if isStream {
		timeout = 2 * time.Hour
	}

	if !isStream {
		resp, uerr := completionsWithKeyFailover(ctx, up, dict, isStream, timeout)
		if uerr != nil {
			writeUpstreamError(w, uerr)
			return
		}
		out := resp.Body
		if norm, nerr := compProv.NormalizeCompletionsResponse(out); nerr == nil && len(norm) > 0 {
			out = norm
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(out)
		return
	}

	ch, uerr := completionsStreamWithKeyFailover(ctx, up, dict, isStream, compProv, timeout)
	if uerr != nil {
		writeUpstreamError(w, uerr)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flushIf(w)

	streamOK := true
	for chunk := range ch {
		if chunk.Done {
			break
		}
		if len(chunk.Data) == 0 {
			continue
		}
		if isUpstreamErrorChunk(chunk.Data) {
			streamOK = false
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
	if streamOK {
		models.RecordAiKeySuccess(up.AiKeyId)
	}
}

func completionsWithKeyFailover(
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
		upReq, err := buildCompletionsUpstream(up, dict, stream)
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

func completionsStreamWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	dict *jsonutils.JSONDict,
	stream bool,
	compProv providers.CompletionsProvider,
	timeout time.Duration,
) (<-chan upstream.StreamChunk, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := buildCompletionsUpstream(up, dict, stream)
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		var ch <-chan upstream.StreamChunk
		var uerr *upstream.Error
		if compProv.OpenAICompletionsStreamPassthrough() {
			ch, uerr = upstream.ChatCompletionStream(reqCtx, upReq)
		} else {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: "streaming completions not supported for provider"}
		}
		cancel()
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

func buildCompletionsUpstream(up *models.ChatUpstream, dict *jsonutils.JSONDict, isStream bool) (*upstream.Request, error) {
	compProv, err := providers.GetCompletions(up.ProviderKey)
	if err != nil {
		return nil, err
	}
	httpReq, err := compProv.BuildCompletionsRequest(&providers.ChatContext{
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
