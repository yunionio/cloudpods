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

// ImagesCompat forwards OpenAI images/generations requests unchanged.
type ImagesCompat struct{}

// NewImagesCompat returns a new OpenAI-compatible images adapter.
func NewImagesCompat() *ImagesCompat {
	return &ImagesCompat{}
}

func (p *ImagesCompat) BuildImagesGenerationsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := CloneBodyWithModel(body, ctx.UpstreamModel)
	return &providerapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     ImagesGenerationsURL(ctx.BaseURL),
		Headers: BearerAuthHeaders(ctx.APIKey),
		Body:    []byte(dup.String()),
	}, nil
}

func (p *ImagesCompat) NormalizeImagesGenerationsResponse(body []byte) ([]byte, error) {
	return body, nil
}
