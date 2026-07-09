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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/azure"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type azurePassthrough struct{}

func (azurePassthrough) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	return azure.BuildResponsesRequest(ctx, body, stream)
}

func (azurePassthrough) NormalizeResponse(_ providerapi.Provider, body []byte) ([]byte, error) {
	return body, nil
}

func (azurePassthrough) ResponsesStreamPassthrough() bool {
	return true
}

func (azurePassthrough) NewStreamState(string, *jsonutils.JSONDict) interface{} {
	return nil
}

func (azurePassthrough) ConvertStreamPayload(_ interface{}, _ []byte, _ bool) ([]providerapi.ResponsesStreamChunk, error) {
	return nil, nil
}

func (azurePassthrough) BuildSubResourceRequest(ctx *providerapi.ChatContext, method, responseID, subAction string, query url.Values) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	rawURL, err := azure.ResponsesSubResourceURL(ctx.BaseURL, responseID, subAction, nil)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		rawURL = openai.AppendQuery(rawURL, query)
	}
	return &providerapi.HTTPRequest{
		Method: strings.ToUpper(strings.TrimSpace(method)),
		URL:    rawURL,
		Headers: map[string]string{
			"api-key":      strings.TrimSpace(ctx.APIKey),
			"Content-Type": "application/json",
		},
	}, nil
}
