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

package openai

import (
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

var (
	defaultEmbeddings = NewEmbeddingCompat()
	defaultImages     = NewImagesCompat()
)

// Compat forwards OpenAI-shaped JSON to OpenAI-compatible upstreams.
type Compat struct {
	ProviderKey string
	Patches     []PatchFunc
}

// NewCompat returns an OpenAI-compatible provider for the given catalog provider_key.
func NewCompat(providerKey string, patches ...PatchFunc) *Compat {
	return &Compat{ProviderKey: providerKey, Patches: patches}
}

func (p *Compat) Key() string {
	return p.ProviderKey
}

func (p *Compat) buildBody(body *jsonutils.JSONDict, upstreamModel string, stream bool) *jsonutils.JSONDict {
	if len(p.Patches) == 0 {
		return CloneBodyWithModel(body, upstreamModel)
	}
	return PatchBody(CloneBodyWithModel(body, upstreamModel), stream, p.Patches...)
}

func (p *Compat) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := p.buildBody(body, ctx.UpstreamModel, stream)
	return &providerapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     ChatCompletionsURL(ctx.BaseURL),
		Headers: BearerAuthHeaders(ctx.APIKey),
		Body:    []byte(dup.String()),
	}, nil
}

func (p *Compat) NormalizeResponse(body []byte) ([]byte, error) {
	return body, nil
}

func (p *Compat) OpenAIStreamPassthrough() bool {
	return true
}

func (p *Compat) ConvertStreamEvent(eventType string, payload []byte, state *providerapi.StreamState) ([]providerapi.StreamChunk, error) {
	return nil, nil
}

func (p *Compat) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	return defaultEmbeddings.BuildEmbeddingsRequest(ctx, body)
}

func (p *Compat) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	return defaultEmbeddings.NormalizeEmbeddingsResponse(body)
}

func (p *Compat) BuildImagesGenerationsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	return defaultImages.BuildImagesGenerationsRequest(ctx, body)
}

func (p *Compat) NormalizeImagesGenerationsResponse(body []byte) ([]byte, error) {
	return defaultImages.NormalizeImagesGenerationsResponse(body)
}

// DefaultEmbeddingCompat returns the shared OpenAI-compatible embeddings adapter.
func DefaultEmbeddingCompat() providerapi.EmbeddingsProvider {
	return defaultEmbeddings
}

// DefaultImagesCompat returns the shared OpenAI-compatible images adapter.
func DefaultImagesCompat() providerapi.ImagesProvider {
	return defaultImages
}
