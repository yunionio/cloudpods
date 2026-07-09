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
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
)

const proxyDebugLogMax = 4096

// ProxyDebugSession groups debug logs for one proxied request.
type ProxyDebugSession struct {
	Protocol string
	ReqID    string
}

func NewProxyDebugSession(ctx context.Context, protocol string) *ProxyDebugSession {
	reqID := strings.TrimSpace(appctx.AppContextRequestId(ctx))
	if reqID == "" {
		reqID = fmt.Sprintf("%08x", time.Now().UnixNano())
	}
	return &ProxyDebugSession{
		Protocol: protocol,
		ReqID:    reqID,
	}
}

func (s *ProxyDebugSession) prefix() string {
	if s == nil {
		return "aiproxy"
	}
	if s.Protocol == "" {
		return fmt.Sprintf("aiproxy [%s]", s.ReqID)
	}
	return fmt.Sprintf("aiproxy %s [%s]", s.Protocol, s.ReqID)
}

func (s *ProxyDebugSession) ClientRequest(r *http.Request, body *jsonutils.JSONDict, up *models.ChatUpstream, stream bool) {
	if s == nil || r == nil {
		return
	}
	bodyStr := ""
	if body != nil {
		bodyStr = truncateLogBytes([]byte(body.String()), proxyDebugLogMax)
	}
	provider := ""
	upstreamModel := ""
	if up != nil {
		provider = up.ProviderKey
		upstreamModel = up.UpstreamModel
	}
	log.Debugf(
		"%s client request method=%s path=%s query=%s stream=%v provider=%s upstream_model=%s vk=%s body=%s",
		s.prefix(),
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		stream,
		provider,
		upstreamModel,
		maskSecret(extractVirtualKey(r)),
		bodyStr,
	)
	if body != nil && up != nil {
		s.logModelRouting(body, up)
	}
}

func (s *ProxyDebugSession) RoutingResolved(body *jsonutils.JSONDict, up *models.ChatUpstream) {
	if s == nil || up == nil {
		return
	}
	log.Debugf(
		"%s routing provider=%s upstream_model=%s",
		s.prefix(),
		up.ProviderKey,
		up.UpstreamModel,
	)
	s.logModelRouting(body, up)
}

func (s *ProxyDebugSession) logModelRouting(body *jsonutils.JSONDict, up *models.ChatUpstream) {
	if body == nil || up == nil {
		return
	}
	model, _ := body.GetString("model")
	if model != "" && model != up.UpstreamModel {
		log.Debugf("%s client model=%q routed upstream_model=%q", s.prefix(), model, up.UpstreamModel)
	}
}

func (s *ProxyDebugSession) ClientRequestNoBody(r *http.Request) {
	if s == nil || r == nil {
		return
	}
	log.Debugf(
		"%s client request method=%s path=%s query=%s vk=%s",
		s.prefix(),
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		maskSecret(extractVirtualKey(r)),
	)
}

func (s *ProxyDebugSession) UpstreamRequest(req *upstream.Request) {
	if s == nil || req == nil {
		return
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		url = strings.TrimSpace(req.BaseURL)
	}
	log.Debugf(
		"%s upstream request url=%s body=%s",
		s.prefix(),
		url,
		truncateLogBytes(req.Body, proxyDebugLogMax),
	)
}

func (s *ProxyDebugSession) UpstreamResponse(body []byte) {
	if s == nil {
		return
	}
	log.Debugf(
		"%s upstream response body=%s",
		s.prefix(),
		truncateLogBytes(body, proxyDebugLogMax),
	)
}

func (s *ProxyDebugSession) ClientResponse(body []byte) {
	if s == nil {
		return
	}
	log.Debugf(
		"%s client response body=%s",
		s.prefix(),
		truncateLogBytes(body, proxyDebugLogMax),
	)
}

func (s *ProxyDebugSession) UpstreamStreamChunk(seq int, data []byte) {
	if s == nil {
		return
	}
	log.Debugf(
		"%s upstream stream chunk #%d: %s",
		s.prefix(),
		seq,
		truncateLogBytes(data, proxyDebugLogMax),
	)
}

func (s *ProxyDebugSession) ClientStreamData(seq int, data []byte) {
	if s == nil {
		return
	}
	log.Debugf(
		"%s client stream out #%d data=%s",
		s.prefix(),
		seq,
		truncateLogBytes(data, proxyDebugLogMax),
	)
}

func (s *ProxyDebugSession) ClientStreamSSE(seq int, event string, data []byte) {
	if s == nil {
		return
	}
	log.Debugf(
		"%s client stream out #%d event=%s data=%s",
		s.prefix(),
		seq,
		event,
		truncateLogBytes(data, proxyDebugLogMax),
	)
}

func (s *ProxyDebugSession) ClientStreamEvents(seq int, events []providerapi.AnthropicStreamChunk) {
	if s == nil {
		return
	}
	if len(events) == 0 {
		log.Debugf("%s client stream out #%d: (no anthropic events)", s.prefix(), seq)
		return
	}
	for i, evt := range events {
		log.Debugf(
			"%s client stream out #%d.%d event=%s data=%s",
			s.prefix(),
			seq,
			i,
			evt.Event,
			truncateLogBytes(evt.Data, proxyDebugLogMax),
		)
	}
}

func (s *ProxyDebugSession) Error(format string, args ...interface{}) {
	if s == nil {
		return
	}
	log.Debugf(s.prefix()+" "+format, args...)
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
