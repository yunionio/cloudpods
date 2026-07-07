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
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
)

const messagesDebugLogMax = 4096

func newMessagesReqID() string {
	return fmt.Sprintf("%08x", time.Now().UnixNano())
}

func logMessagesClientRequest(reqID string, r *http.Request, body *jsonutils.JSONDict, up *models.ChatUpstream, stream bool) {
	model, _ := body.GetString("model")
	log.Debugf(
		"aiproxy messages [%s] client request method=%s path=%s query=%s stream=%v provider=%s upstream_model=%s vk=%s body=%s",
		reqID,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		stream,
		up.ProviderKey,
		up.UpstreamModel,
		maskSecret(extractVirtualKey(r)),
		truncateLogBytes([]byte(body.String()), messagesDebugLogMax),
	)
	if model != "" && model != up.UpstreamModel {
		log.Debugf("aiproxy messages [%s] client model=%q routed upstream_model=%q", reqID, model, up.UpstreamModel)
	}
}

func logMessagesUpstreamRequest(reqID string, req *upstream.Request) {
	if req == nil {
		return
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		url = strings.TrimSpace(req.BaseURL)
	}
	log.Debugf(
		"aiproxy messages [%s] upstream request url=%s body=%s",
		reqID,
		url,
		truncateLogBytes(req.Body, messagesDebugLogMax),
	)
}

func logMessagesUpstreamStreamChunk(reqID string, seq int, data []byte) {
	log.Debugf(
		"aiproxy messages [%s] upstream stream chunk #%d: %s",
		reqID,
		seq,
		truncateLogBytes(data, messagesDebugLogMax),
	)
}

func logMessagesClientStreamEvents(reqID string, seq int, events []providerapi.AnthropicStreamChunk) {
	if len(events) == 0 {
		log.Debugf("aiproxy messages [%s] client stream out #%d: (no anthropic events)", reqID, seq)
		return
	}
	for i, evt := range events {
		log.Debugf(
			"aiproxy messages [%s] client stream out #%d.%d event=%s data=%s",
			reqID,
			seq,
			i,
			evt.Event,
			truncateLogBytes(evt.Data, messagesDebugLogMax),
		)
	}
}

func logMessagesUpstreamResponse(reqID string, body []byte) {
	log.Debugf(
		"aiproxy messages [%s] upstream response body=%s",
		reqID,
		truncateLogBytes(body, messagesDebugLogMax),
	)
}

func logMessagesClientResponse(reqID string, body []byte) {
	log.Debugf(
		"aiproxy messages [%s] client response body=%s",
		reqID,
		truncateLogBytes(body, messagesDebugLogMax),
	)
}

func logMessagesClientStreamPassthrough(reqID string, seq int, event string, data []byte) {
	log.Debugf(
		"aiproxy messages [%s] client stream passthrough #%d event=%s data=%s",
		reqID,
		seq,
		event,
		truncateLogBytes(data, messagesDebugLogMax),
	)
}

func logMessagesError(reqID string, format string, args ...interface{}) {
	log.Debugf("aiproxy messages [%s] "+format, append([]interface{}{reqID}, args...)...)
}

func truncateLogBytes(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("...(truncated, total=%d)", len(s))
}

func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
