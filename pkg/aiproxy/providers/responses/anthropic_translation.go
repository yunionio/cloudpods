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

const anthropicAPIVersion = "2023-06-01"

type anthropicTranslationStreamState struct {
	conv   *openai.ResponsesAnthropicStreamConverter
	convSt *openai.ResponsesConvertState
}

type anthropicTranslationAdapter struct{}

func (anthropicTranslationAdapter) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	anthropicBody, state, err := openai.ResponsesToAnthropicMessages(body, ctx.UpstreamModel)
	if err != nil {
		return nil, err
	}
	_ = state
	if stream {
		anthropicBody.Set("stream", jsonutils.JSONTrue)
	}
	base := strings.TrimSpace(ctx.BaseURL)
	if base == "" {
		base = "https://api.anthropic.com"
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    openai.JoinURL(base, "/v1/messages"),
		Headers: map[string]string{
			"x-api-key":         strings.TrimSpace(ctx.APIKey),
			"anthropic-version": anthropicAPIVersion,
			"Content-Type":      "application/json",
		},
		Body: []byte(anthropicBody.String()),
	}, nil
}

func (anthropicTranslationAdapter) NormalizeResponse(_ providerapi.Provider, body []byte) ([]byte, error) {
	return openai.AnthropicMessagesToResponses(body, nil)
}

func (anthropicTranslationAdapter) ResponsesStreamPassthrough() bool {
	return false
}

func (anthropicTranslationAdapter) NewStreamState(requestModel string, body *jsonutils.JSONDict) interface{} {
	toolMap := openai.CodexToolMap{}
	if body != nil {
		if _, state, err := openai.ResponsesToAnthropicMessages(body, requestModel); err == nil && state != nil {
			toolMap = state.ToolMap
		}
	}
	return &anthropicTranslationStreamState{
		conv: openai.NewResponsesAnthropicStreamConverter(requestModel, toolMap),
	}
}

func (anthropicTranslationAdapter) ConvertStreamPayload(state interface{}, payload []byte, endOfStream bool) ([]providerapi.ResponsesStreamChunk, error) {
	st, ok := state.(*anthropicTranslationStreamState)
	if !ok || st == nil || st.conv == nil {
		return nil, nil
	}
	// payload is raw anthropic data line when called from translated stream with event in separate path
	events, err := st.conv.Feed("", payload, endOfStream)
	if err != nil {
		return nil, err
	}
	return openai.AnthropicToProviderChunks(events), nil
}

func (anthropicTranslationAdapter) BuildSubResourceRequest(*providerapi.ChatContext, string, string, string, url.Values) (*providerapi.HTTPRequest, error) {
	return nil, providerapi.ErrResponsesSubResourceNotSupported
}

// ConvertAnthropicStreamEvent converts one Anthropic SSE event to Responses chunks.
func ConvertAnthropicStreamEvent(state interface{}, eventType string, payload []byte, endOfStream bool) ([]providerapi.ResponsesStreamChunk, error) {
	st, ok := state.(*anthropicTranslationStreamState)
	if !ok || st == nil || st.conv == nil {
		return nil, nil
	}
	events, err := st.conv.Feed(eventType, payload, endOfStream)
	if err != nil {
		return nil, err
	}
	return openai.AnthropicToProviderChunks(events), nil
}
