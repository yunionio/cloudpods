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
	"io"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// imagesGenerationsHandler implements OpenAI-compatible POST /openai/v1/images/generations.
func imagesGenerationsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
	rec.Stream = false

	dbg := NewProxyDebugSession(ctx, "openai-images")
	dbg.ClientRequest(r, dict, nil, false)

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

	imgProv := providers.GetImages(up.ProviderKey)
	if _, err := imgProv.BuildImagesGenerationsRequest(&providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}, dict); err != nil {
		dbg.Error("provider request: %v", err)
		failAPILogRecord(rec, http.StatusBadRequest, "provider_request", err)
		httperrors.InvalidInputError(ctx, w, "provider request: %v", err)
		return
	}

	resp, uerr := imagesGenerationsWithKeyFailover(ctx, up, dict, 180*time.Second, dbg)
	if uerr != nil {
		dbg.Error("upstream error status=%d body=%s", uerr.StatusCode, truncateLogBytes(uerr.Body, proxyDebugLogMax))
		recordUpstreamError(rec, uerr)
		writeUpstreamError(ctx, w, uerr)
		return
	}
	dbg.UpstreamResponse(resp.Body)
	out := resp.Body
	if norm, nerr := imgProv.NormalizeImagesGenerationsResponse(out); nerr == nil && len(norm) > 0 {
		out = norm
	}
	dbg.ClientResponse(out)
	markAPILogSuccess(rec, out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func imagesGenerationsWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	dict *jsonutils.JSONDict,
	timeout time.Duration,
	dbg *ProxyDebugSession,
) (*upstream.Response, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := buildImagesGenerationsUpstream(up, dict)
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		dbg.UpstreamRequest(upReq)
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

func buildImagesGenerationsUpstream(up *models.ChatUpstream, dict *jsonutils.JSONDict) (*upstream.Request, error) {
	imgProv := providers.GetImages(up.ProviderKey)
	httpReq, err := imgProv.BuildImagesGenerationsRequest(&providers.ChatContext{
		ProviderKey:   up.ProviderKey,
		BaseURL:       up.BaseURL,
		APIKey:        up.APIKey,
		UpstreamModel: up.UpstreamModel,
	}, dict)
	if err != nil {
		return nil, err
	}
	return providers.ToUpstreamRequest(httpReq, up.APIKey), nil
}
