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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type chatTranslationStreamState struct {
	conv *openai.ResponsesStreamConverter
}

type chatTranslationAdapter struct {
	providerKey string
	apiMode     string
}

func (a chatTranslationAdapter) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	chatBody, state, err := openai.ResponsesToChatCompletions(body, ctx.UpstreamModel)
	if err != nil {
		return nil, err
	}
	_ = state
	if stream {
		chatBody.Set("stream", jsonutils.JSONTrue)
		streamOpts := jsonutils.NewDict()
		streamOpts.Set("include_usage", jsonutils.JSONTrue)
		chatBody.Set("stream_options", streamOpts)
	}
	prov := providers.ChatProviderForUpstream(a.providerKey, a.apiMode)
	return prov.BuildUpstreamRequest(&providerapi.ChatContext{
		ProviderKey:   ctx.ProviderKey,
		BaseURL:       ctx.BaseURL,
		APIKey:        ctx.APIKey,
		UpstreamModel: ctx.UpstreamModel,
		APIMode:       a.apiMode,
	}, chatBody, stream)
}

func (a chatTranslationAdapter) NormalizeResponse(prov providerapi.Provider, body []byte) ([]byte, error) {
	norm, err := prov.NormalizeResponse(body)
	if err != nil {
		return nil, err
	}
	if len(norm) > 0 {
		body = norm
	}
	return openai.ChatCompletionToResponses(body, nil)
}

func (chatTranslationAdapter) ResponsesStreamPassthrough() bool {
	return false
}

func (a chatTranslationAdapter) NewStreamState(requestModel string, body *jsonutils.JSONDict) interface{} {
	toolMap := openai.CodexToolMap{}
	if body != nil {
		if _, state, err := openai.ResponsesToChatCompletions(body, requestModel); err == nil && state != nil {
			toolMap = state.ToolMap
		}
	}
	return &chatTranslationStreamState{
		conv: openai.NewResponsesStreamConverter(requestModel, toolMap),
	}
}

func (chatTranslationAdapter) ConvertStreamPayload(state interface{}, payload []byte, endOfStream bool) ([]providerapi.ResponsesStreamChunk, error) {
	st, ok := state.(*chatTranslationStreamState)
	if !ok || st == nil || st.conv == nil {
		return nil, nil
	}
	events, err := st.conv.Feed(payload, endOfStream)
	if err != nil {
		return nil, err
	}
	return openai.ToProviderChunks(events), nil
}

func (chatTranslationAdapter) BuildSubResourceRequest(*providerapi.ChatContext, string, string, string, url.Values) (*providerapi.HTTPRequest, error) {
	return nil, providerapi.ErrResponsesSubResourceNotSupported
}
