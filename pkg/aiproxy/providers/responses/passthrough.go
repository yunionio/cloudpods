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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type passthroughAdapter struct{}

func (passthroughAdapter) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := body.Copy()
	dup.Set("model", jsonutils.NewString(ctx.UpstreamModel))
	base := strings.TrimSpace(ctx.BaseURL)
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    openai.ResponsesURL(base),
		Headers: map[string]string{
			"Authorization": "Bearer " + strings.TrimSpace(ctx.APIKey),
			"Content-Type":  "application/json",
		},
		Body: []byte(dup.String()),
	}, nil
}

func (passthroughAdapter) NormalizeResponse(_ providerapi.Provider, body []byte) ([]byte, error) {
	return body, nil
}

func (passthroughAdapter) ResponsesStreamPassthrough() bool {
	return true
}

func (passthroughAdapter) NewStreamState(string, *jsonutils.JSONDict) interface{} {
	return nil
}

func (passthroughAdapter) ConvertStreamPayload(_ interface{}, _ []byte, _ bool) ([]providerapi.ResponsesStreamChunk, error) {
	return nil, nil
}

func (passthroughAdapter) BuildSubResourceRequest(ctx *providerapi.ChatContext, method, responseID, subAction string, query url.Values) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	rawURL := openai.ResponsesSubResourceURL(ctx.BaseURL, responseID, subAction)
	if len(query) > 0 {
		rawURL = openai.AppendQuery(rawURL, query)
	}
	return &providerapi.HTTPRequest{
		Method: strings.ToUpper(strings.TrimSpace(method)),
		URL:    rawURL,
		Headers: map[string]string{
			"Authorization": "Bearer " + strings.TrimSpace(ctx.APIKey),
			"Content-Type":  "application/json",
		},
	}, nil
}
