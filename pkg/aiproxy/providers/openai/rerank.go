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

// RerankCompat forwards OpenAI/Cohere/vLLM-compatible rerank requests.
type RerankCompat struct{}

// NewRerankCompat returns a new OpenAI-compatible rerank adapter.
func NewRerankCompat() *RerankCompat {
	return &RerankCompat{}
}

func (p *RerankCompat) BuildRerankRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := CloneBodyWithModel(body, ctx.UpstreamModel)
	return &providerapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     RerankURL(ctx.BaseURL),
		Headers: BearerAuthHeaders(ctx.APIKey),
		Body:    []byte(dup.String()),
	}, nil
}

// NormalizeRerankResponse fills in missing results[].index fields without
// touching relevance scores or document payloads.
func (p *RerankCompat) NormalizeRerankResponse(body []byte) ([]byte, error) {
	return NormalizeRerankBody(body)
}

// NormalizeRerankBody is the package-level form of the rerank normalizer.
func NormalizeRerankBody(body []byte) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return body, nil
	}
	raw, ok := root["results"]
	if !ok {
		return body, nil
	}
	var results []json.RawMessage
	if err := json.Unmarshal(raw, &results); err != nil {
		return body, nil
	}
	changed := false
	for i, itemRaw := range results {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			continue
		}
		if _, ok := item["index"]; !ok {
			item["index"] = json.RawMessage(fmt.Sprintf("%d", i))
			changed = true
		}
		patched, err := json.Marshal(item)
		if err != nil {
			return body, nil
		}
		results[i] = patched
	}
	if !changed {
		return body, nil
	}
	patchedResults, err := json.Marshal(results)
	if err != nil {
		return body, nil
	}
	root["results"] = patchedResults
	out, err := json.Marshal(root)
	if err != nil {
		return body, nil
	}
	return out, nil
}
