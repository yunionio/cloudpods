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

// CompletionsCompat forwards OpenAI legacy completions requests unchanged.
type CompletionsCompat struct {
	Patches []PatchFunc
}

// NewCompletionsCompat returns a new OpenAI-compatible completions adapter.
func NewCompletionsCompat(patches ...PatchFunc) *CompletionsCompat {
	return &CompletionsCompat{Patches: patches}
}

func (p *CompletionsCompat) buildBody(body *jsonutils.JSONDict, upstreamModel string, stream bool) *jsonutils.JSONDict {
	dup := CloneBodyWithModel(body, upstreamModel)
	if len(p.Patches) == 0 {
		return dup
	}
	return PatchBody(dup, stream, p.Patches...)
}

func (p *CompletionsCompat) BuildCompletionsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := p.buildBody(body, ctx.UpstreamModel, stream)
	return &providerapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     CompletionsURL(ctx.BaseURL),
		Headers: BearerAuthHeaders(ctx.APIKey),
		Body:    []byte(dup.String()),
	}, nil
}

func (p *CompletionsCompat) NormalizeCompletionsResponse(body []byte) ([]byte, error) {
	return body, nil
}

func (p *CompletionsCompat) OpenAICompletionsStreamPassthrough() bool {
	return true
}
