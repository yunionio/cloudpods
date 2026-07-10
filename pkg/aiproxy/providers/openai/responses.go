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
	"net/url"
	"strings"
)

// ResponsesURL builds the OpenAI Responses create endpoint from a provider base URL.
func ResponsesURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "https://api.openai.com/v1/responses"
	}
	if strings.HasSuffix(base, "/v1/responses") || strings.HasSuffix(base, "/responses") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/responses"
	}
	return JoinURL(base, "/v1/responses")
}

// ResponsesSubResourceURL builds GET/cancel/delete URLs for a stored response.
func ResponsesSubResourceURL(baseURL, responseID, subAction string) string {
	base := ResponsesURL(baseURL)
	id := strings.Trim(strings.TrimSpace(responseID), "/")
	if id == "" {
		return base
	}
	path := base + "/" + url.PathEscape(id)
	action := strings.Trim(strings.TrimSpace(subAction), "/")
	if action != "" {
		path += "/" + action
	}
	return path
}

// AppendQuery appends encoded query parameters to a URL.
func AppendQuery(rawURL string, query url.Values) string {
	if len(query) == 0 {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + query.Encode()
}

// NewResponsesErrorBody returns an OpenAI-style error JSON body.
func NewResponsesErrorBody(msg, errType, code string) []byte {
	body := map[string]interface{}{
		"error": map[string]interface{}{
			"message": msg,
			"type":    errType,
		},
	}
	if code != "" {
		body["error"].(map[string]interface{})["code"] = code
	}
	b, _ := MarshalJSON(body)
	return b
}

// ResponsesErrorMessage extracts message from an OpenAI error JSON body.
func ResponsesErrorMessage(body []byte) string {
	var wrap struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &wrap) == nil && wrap.Error.Message != "" {
		return wrap.Error.Message
	}
	return "upstream request failed"
}
