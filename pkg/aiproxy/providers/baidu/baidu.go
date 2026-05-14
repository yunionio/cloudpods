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

package baidu

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type provider struct {
	v2 *openai.Compat
}

// New returns the Baidu Wenxin / Qianfan provider adapter.
func New() providerapi.Provider {
	return &provider{v2: openai.NewCompat("baidu")}
}

func (p *provider) Key() string {
	return "baidu"
}

func (p *provider) useV2(ctx *providerapi.ChatContext) bool {
	if ctx == nil {
		return true
	}
	return useQianfanV2(ctx.BaseURL)
}

func (p *provider) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if p.useV2(ctx) {
		return p.v2.BuildUpstreamRequest(ctx, body, stream)
	}
	return buildWenxinV1ChatRequest(ctx, body, stream)
}

func (p *provider) NormalizeResponse(body []byte) ([]byte, error) {
	if p.v2 != nil {
		// Try wenxin v1 only when response looks non-OpenAI.
		if out, err := normalizeWenxinV1ChatResponse(body); err != nil {
			return nil, err
		} else if string(out) != string(body) {
			return out, nil
		}
	}
	return body, nil
}

func (p *provider) OpenAIStreamPassthrough() bool {
	return false
}

func (p *provider) ConvertStreamEvent(eventType string, payload []byte, state *providerapi.StreamState) ([]providerapi.StreamChunk, error) {
	return convertWenxinV1StreamEvent(payload, state)
}

func (p *provider) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if p.useV2(ctx) {
		return p.v2.BuildEmbeddingsRequest(ctx, body)
	}
	return buildWenxinV1EmbeddingsRequest(ctx, body)
}

func (p *provider) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	if out, err := normalizeWenxinV1EmbeddingsResponse(body); err != nil {
		return nil, err
	} else if len(out) > 0 && string(out) != string(body) {
		return out, nil
	}
	return body, nil
}

func (p *provider) BuildImagesGenerationsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if p.useV2(ctx) {
		return p.v2.BuildImagesGenerationsRequest(ctx, body)
	}
	return nil, fmt.Errorf("provider %q wenxin v1 does not support images/generations; use qianfan v2 base_url", p.Key())
}

func (p *provider) NormalizeImagesGenerationsResponse(body []byte) ([]byte, error) {
	return body, nil
}

// OpenAIStreamPassthroughForContext reports whether upstream SSE is already OpenAI-compatible.
func (p *provider) OpenAIStreamPassthroughForContext(ctx *providerapi.ChatContext) bool {
	return p.useV2(ctx)
}
