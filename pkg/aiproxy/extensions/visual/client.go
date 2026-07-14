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

package visual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
)

const visualSystemPrompt = "You are a vision analysis model behind Cloudpods AI Proxy Visual. Analyze images carefully, state uncertainty, and do not invent visual facts."

// HTTPVisionClient calls a multimodal upstream using OpenAI chat/completions.
type HTTPVisionClient struct {
	up        *models.ChatUpstream
	maxTokens int
}

func NewHTTPVisionClient(up *models.ChatUpstream, maxTokens int) *HTTPVisionClient {
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	return &HTTPVisionClient{up: up, maxTokens: maxTokens}
}

func (c *HTTPVisionClient) Analyze(ctx context.Context, request AnalysisRequest) (string, error) {
	if c == nil || c.up == nil {
		return "", fmt.Errorf("visual client is nil")
	}
	parts := jsonutils.NewArray()
	textPart := jsonutils.NewDict()
	textPart.Set("type", jsonutils.NewString("text"))
	textPart.Set("text", jsonutils.NewString(request.Prompt))
	parts.Add(textPart)
	for _, image := range request.Images {
		if part := chatImagePart(image); part != nil {
			parts.Add(part)
		}
	}
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", parts)
	sysMsg := jsonutils.NewDict()
	sysMsg.Set("role", jsonutils.NewString("system"))
	sysMsg.Set("content", jsonutils.NewString(visualSystemPrompt))
	messages := jsonutils.NewArray(sysMsg, userMsg)
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString(c.up.UpstreamModel))
	body.Set("max_tokens", jsonutils.NewInt(int64(c.maxTokens)))
	body.Set("messages", messages)
	respBody, err := callChatCompletions(ctx, c.up, body)
	if err != nil {
		return "", err
	}
	text := textFromChatResponse(respBody)
	if text == "" {
		return "", fmt.Errorf("visual provider returned empty content")
	}
	return text, nil
}

func callChatCompletions(ctx context.Context, up *models.ChatUpstream, body *jsonutils.JSONDict) ([]byte, error) {
	if up == nil || body == nil {
		return nil, fmt.Errorf("nil upstream or body")
	}
	forceNonStreamChatBody(body)
	req := &upstream.Request{
		URL:    openai.ChatCompletionsURL(ChatBaseURL(up.BaseURL)),
		APIKey: up.APIKey,
		Body:   []byte(body.String()),
	}
	resp, uerr := upstream.HTTPDo(ctx, http.MethodPost, req)
	if uerr != nil {
		return nil, uerr
	}
	return resp.Body, nil
}

// ChatBaseURL strips Anthropic-compatible path suffixes so chat/completions
// can be built against the OpenAI-compatible root (e.g. deepseek.com vs deepseek.com/anthropic).
func ChatBaseURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return base
	}
	lower := strings.ToLower(base)
	for _, suffix := range []string{"/api/anthropic", "/anthropic"} {
		if strings.HasSuffix(lower, suffix) {
			return base[:len(base)-len(suffix)]
		}
	}
	return base
}

// forceNonStreamChatBody ensures visual orchestration always calls upstream chat/completions
// without stream=true (Codex Responses requests carry stream=true in the converted body).
func forceNonStreamChatBody(body *jsonutils.JSONDict) {
	if body == nil {
		return
	}
	body.Remove("stream")
	body.Remove("stream_options")
}

func finishReasonFromChatResponse(body []byte) string {
	var resp struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Choices[0].FinishReason)
}

func toolCallsFromChatResponse(body []byte) []openai.ToolCall {
	var resp struct {
		Choices []struct {
			Message struct {
				ToolCalls []openai.ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

func assistantMessageFromChatResponse(body []byte) *jsonutils.JSONDict {
	var resp struct {
		Choices []struct {
			Message json.RawMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Choices) == 0 {
		return nil
	}
	parsed, err := jsonutils.Parse(resp.Choices[0].Message)
	if err != nil {
		return nil
	}
	if d, ok := parsed.(*jsonutils.JSONDict); ok {
		return d
	}
	return nil
}
