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

package cohere

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type provider struct {
	*openai.Compat
}

// New returns the Cohere provider adapter (OpenAI-compatible chat, native embeddings).
func New() providerapi.Provider {
	return &provider{Compat: openai.NewCompat("cohere")}
}

func (p *provider) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	texts, _, isText, err := openai.ParseEmbeddingInput(body)
	if err != nil {
		return nil, err
	}
	if !isText {
		return nil, fmt.Errorf("cohere embeddings requires string or string[] input")
	}
	reqBody := map[string]interface{}{
		"model":           ctx.UpstreamModel,
		"texts":           texts,
		"input_type":      embeddingInputType(body),
		"embedding_types": []string{"float"},
	}
	raw, err := openai.MarshalJSON(reqBody)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(ctx.BaseURL)
	if base == "" {
		base = "https://api.cohere.ai"
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    openai.JoinURL(base, "/v2/embed"),
		Headers: map[string]string{
			"Authorization": "Bearer " + strings.TrimSpace(ctx.APIKey),
			"Content-Type":  "application/json",
		},
		Body: raw,
	}, nil
}

func embeddingInputType(body *jsonutils.JSONDict) string {
	if body == nil {
		return "search_document"
	}
	if v, err := body.GetString("input_type"); err == nil && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return "search_document"
}

func (p *provider) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	var resp struct {
		Embeddings struct {
			Float [][]float64 `json:"float"`
		} `json:"embeddings"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}
	return openai.NewEmbeddingsResponse("", resp.Embeddings.Float, 0)
}
