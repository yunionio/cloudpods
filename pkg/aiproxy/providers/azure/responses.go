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

package azure

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

// BuildResponsesRequest builds an Azure OpenAI Responses API request.
func BuildResponsesRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := body.Copy()
	dup.Set("model", jsonutils.NewString(ctx.UpstreamModel))
	rawURL, err := ResponsesURL(ctx.BaseURL, dup)
	if err != nil {
		return nil, err
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    rawURL,
		Headers: map[string]string{
			"api-key":      strings.TrimSpace(ctx.APIKey),
			"Content-Type": "application/json",
		},
		Body: []byte(dup.String()),
	}, nil
}

// ResponsesURL builds Azure OpenAI Responses endpoint with api-version query.
func ResponsesURL(baseURL string, body *jsonutils.JSONDict) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("azure provider requires base_url (resource endpoint)")
	}
	path := "/openai/responses"
	if strings.Contains(base, "/openai/responses") {
		path = ""
	} else if strings.HasSuffix(base, "/openai") {
		path = "/responses"
	}
	raw := base + path
	ver := apiVersionFromBody(body)
	if ver == "" {
		ver = "2024-12-01-preview"
	}
	return appendAPIVersion(raw, ver), nil
}

// ResponsesSubResourceURL builds GET/cancel/delete URLs for Azure Responses.
func ResponsesSubResourceURL(baseURL, responseID, subAction string, body *jsonutils.JSONDict) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("azure provider requires base_url (resource endpoint)")
	}
	id := strings.Trim(strings.TrimSpace(responseID), "/")
	path := "/openai/responses"
	if id != "" {
		path += "/" + url.PathEscape(id)
	}
	if action := strings.Trim(strings.TrimSpace(subAction), "/"); action != "" {
		path += "/" + action
	}
	raw := base + path
	ver := apiVersionFromBody(body)
	if ver == "" {
		ver = "2024-12-01-preview"
	}
	return appendAPIVersion(raw, ver), nil
}

func apiVersionFromBody(body *jsonutils.JSONDict) string {
	if body == nil {
		return ""
	}
	if v, err := body.Get("api-version"); err == nil {
		if s, err := v.GetString(); err == nil {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func appendAPIVersion(path, apiVersion string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		if strings.HasSuffix(path, "?") || strings.HasSuffix(path, "&") {
			sep = ""
		} else {
			sep = "&"
		}
	}
	return path + sep + "api-version=" + url.QueryEscape(apiVersion)
}
