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

package vllm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type provider struct {
	*openai.Compat
	completions *openai.CompletionsCompat
}

func patchVLLMRequest(body *jsonutils.JSONDict, stream bool) {
	if body == nil {
		return
	}
	if !stream {
		body.Remove("stream_options")
	}
}

// New returns the vLLM OpenAI-compatible provider adapter.
func New() providerapi.Provider {
	patches := []openai.PatchFunc{patchVLLMRequest}
	return &provider{
		Compat:      openai.NewCompat("vllm", patches...),
		completions: openai.NewCompletionsCompat(patches...),
	}
}

func (p *provider) BuildCompletionsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	return p.completions.BuildCompletionsRequest(ctx, body, stream)
}

func (p *provider) NormalizeCompletionsResponse(body []byte) ([]byte, error) {
	return p.completions.NormalizeCompletionsResponse(body)
}

func (p *provider) OpenAICompletionsStreamPassthrough() bool {
	return p.completions.OpenAICompletionsStreamPassthrough()
}
