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
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

const anthropicAPIVersion = "2023-06-01"

type passthroughAdapter struct{}

func (passthroughAdapter) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := jsonutils.NewDict()
	if body != nil {
		dup = body.Copy()
	}
	dup.Set("model", jsonutils.NewString(ctx.UpstreamModel))
	if stream {
		dup.Set("stream", jsonutils.JSONTrue)
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
		Body: []byte(dup.String()),
	}, nil
}

func (passthroughAdapter) NormalizeResponse(_ providerapi.Provider, body []byte) ([]byte, error) {
	return body, nil
}

func (passthroughAdapter) AnthropicStreamPassthrough() bool {
	return true
}

func (passthroughAdapter) NewStreamState(string) interface{} {
	return nil
}

func (passthroughAdapter) ConvertStreamPayload(_ interface{}, _ []byte, _ bool) ([]providerapi.AnthropicStreamChunk, error) {
	return nil, nil
}
