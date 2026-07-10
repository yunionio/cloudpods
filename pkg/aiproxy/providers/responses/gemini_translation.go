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

package responses

import (
	"net/url"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

type geminiTranslationAdapter struct {
	inner chatTranslationAdapter
}

func (g geminiTranslationAdapter) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	g.inner.providerKey = api.ProviderKeyGemini
	return g.inner.BuildUpstreamRequest(ctx, body, stream)
}

func (g geminiTranslationAdapter) NormalizeResponse(prov providerapi.Provider, body []byte) ([]byte, error) {
	return g.inner.NormalizeResponse(prov, body)
}

func (g geminiTranslationAdapter) ResponsesStreamPassthrough() bool {
	return false
}

func (g geminiTranslationAdapter) NewStreamState(requestModel string, body *jsonutils.JSONDict) interface{} {
	return g.inner.NewStreamState(requestModel, body)
}

func (g geminiTranslationAdapter) ConvertStreamPayload(state interface{}, payload []byte, endOfStream bool) ([]providerapi.ResponsesStreamChunk, error) {
	return g.inner.ConvertStreamPayload(state, payload, endOfStream)
}

func (g geminiTranslationAdapter) BuildSubResourceRequest(*providerapi.ChatContext, string, string, string, url.Values) (*providerapi.HTTPRequest, error) {
	return nil, providerapi.ErrResponsesSubResourceNotSupported
}
