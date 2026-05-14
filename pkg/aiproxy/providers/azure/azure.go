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

package azure

import (
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type provider struct{}

// New returns the Azure OpenAI provider adapter.
func New() providerapi.Provider {
	return &provider{}
}

func (p *provider) Key() string {
	return "azure"
}

func (p *provider) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	return p.buildRequest(ctx, body, "chat/completions")
}

func (p *provider) NormalizeResponse(body []byte) ([]byte, error) {
	return body, nil
}

func (p *provider) OpenAIStreamPassthrough() bool {
	return true
}

func (p *provider) ConvertStreamEvent(eventType string, payload []byte, state *providerapi.StreamState) ([]providerapi.StreamChunk, error) {
	return nil, nil
}

func (p *provider) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	return p.buildRequest(ctx, body, "embeddings")
}

func (p *provider) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	return body, nil
}

func (p *provider) BuildImagesGenerationsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	return p.buildRequest(ctx, body, "images/generations")
}

func (p *provider) NormalizeImagesGenerationsResponse(body []byte) ([]byte, error) {
	return body, nil
}

func (p *provider) buildRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, action string) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := openai.CloneBodyWithModel(body, ctx.UpstreamModel)
	url, err := deploymentURL(ctx.BaseURL, ctx.UpstreamModel, body, action)
	if err != nil {
		return nil, err
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    url,
		Headers: map[string]string{
			"api-key":      strings.TrimSpace(ctx.APIKey),
			"Content-Type": "application/json",
		},
		Body: []byte(dup.String()),
	}, nil
}

func deploymentURL(baseURL, deployment string, body *jsonutils.JSONDict, action string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("azure provider requires base_url (resource endpoint)")
	}
	url := base
	suffix := "/" + action
	if strings.Contains(url, "/"+action) {
		return url, nil
	}
	if strings.HasSuffix(url, "/v1") {
		return url + suffix, nil
	}
	if strings.Contains(url, "/openai/deployments/") {
		return url + suffix, nil
	}
	url = openai.JoinURL(url, fmt.Sprintf("openai/deployments/%s/%s", deployment, action))
	if body != nil {
		if v, err := body.Get("api-version"); err == nil {
			ver, _ := v.GetString()
			if ver != "" {
				url = url + "?api-version=" + ver
			}
		}
	}
	return url, nil
}
