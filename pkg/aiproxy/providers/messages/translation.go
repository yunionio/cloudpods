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

package messages

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type translationStreamState struct {
	conv *openai.AnthropicStreamConverter
}

type translationAdapter struct {
	providerKey string
}

func (a translationAdapter) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	openaiBody, err := openai.AnthropicToChatCompletions(body, ctx.UpstreamModel)
	if err != nil {
		return nil, err
	}
	if stream {
		openaiBody.Set("stream", jsonutils.JSONTrue)
		streamOpts := jsonutils.NewDict()
		streamOpts.Set("include_usage", jsonutils.JSONTrue)
		openaiBody.Set("stream_options", streamOpts)
	}
	prov := providers.Get(a.providerKey)
	return prov.BuildUpstreamRequest(&providerapi.ChatContext{
		ProviderKey:   ctx.ProviderKey,
		BaseURL:       ctx.BaseURL,
		APIKey:        ctx.APIKey,
		UpstreamModel: ctx.UpstreamModel,
	}, openaiBody, stream)
}

func (a translationAdapter) NormalizeResponse(prov providerapi.Provider, body []byte) ([]byte, error) {
	norm, err := prov.NormalizeResponse(body)
	if err != nil {
		return nil, err
	}
	if len(norm) > 0 {
		body = norm
	}
	return openai.ChatCompletionToAnthropic(body)
}

func (translationAdapter) AnthropicStreamPassthrough() bool {
	return false
}

func (translationAdapter) NewStreamState(requestModel string) interface{} {
	return &translationStreamState{
		conv: openai.NewAnthropicStreamConverter(requestModel),
	}
}

func (translationAdapter) ConvertStreamPayload(state interface{}, payload []byte, endOfStream bool) ([]providerapi.AnthropicStreamChunk, error) {
	st, ok := state.(*translationStreamState)
	if !ok || st == nil || st.conv == nil {
		return nil, nil
	}
	events, err := st.conv.Feed(payload, endOfStream)
	if err != nil {
		return nil, err
	}
	out := make([]providerapi.AnthropicStreamChunk, 0, len(events))
	for _, evt := range events {
		out = append(out, providerapi.AnthropicStreamChunk{
			Event: evt.Event,
			Data:  evt.Data,
		})
	}
	return out, nil
}
