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
	"encoding/json"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

// EmbeddingCompat forwards OpenAI embeddings requests unchanged.
type EmbeddingCompat struct{}

// NewEmbeddingCompat returns a new OpenAI-compatible embeddings adapter.
func NewEmbeddingCompat() *EmbeddingCompat {
	return &EmbeddingCompat{}
}

func (p *EmbeddingCompat) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := CloneBodyWithModel(body, ctx.UpstreamModel)
	return &providerapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     EmbeddingsURL(ctx.BaseURL),
		Headers: BearerAuthHeaders(ctx.APIKey),
		Body:    []byte(dup.String()),
	}, nil
}

// NormalizeEmbeddingsResponse fills in missing OpenAI envelope fields
// (object / data[].object / data[].index) without touching the vector values:
// with encoding_format=base64 the vectors are strings and must pass through.
func (p *EmbeddingCompat) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	return NormalizeEmbeddingsBody(body)
}

// NormalizeEmbeddingsBody is the package-level form of the embeddings normalizer.
func NormalizeEmbeddingsBody(body []byte) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return body, nil
	}
	changed := false
	if _, ok := root["object"]; !ok {
		root["object"] = json.RawMessage(`"list"`)
		changed = true
	}
	var data []json.RawMessage
	if raw, ok := root["data"]; ok {
		if err := json.Unmarshal(raw, &data); err != nil {
			return body, nil
		}
		for i, itemRaw := range data {
			var item map[string]json.RawMessage
			if err := json.Unmarshal(itemRaw, &item); err != nil {
				continue
			}
			if _, ok := item["object"]; !ok {
				item["object"] = json.RawMessage(`"embedding"`)
				changed = true
			}
			if _, ok := item["index"]; !ok {
				item["index"] = json.RawMessage(fmt.Sprintf("%d", i))
				changed = true
			}
			patched, err := json.Marshal(item)
			if err != nil {
				return body, nil
			}
			data[i] = patched
		}
		patchedData, err := json.Marshal(data)
		if err != nil {
			return body, nil
		}
		root["data"] = patchedData
	}
	if !changed {
		return body, nil
	}
	out, err := json.Marshal(root)
	if err != nil {
		return body, nil
	}
	return out, nil
}
