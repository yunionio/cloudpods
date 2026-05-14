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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

// Message is one OpenAI chat message entry.
type Message struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// ImageItem is one generated image in an OpenAI images response.
type ImageItem struct {
	URL           string
	B64           string
	RevisedPrompt string
}

// MessageTextContent extracts plain text from an OpenAI message content field.
func MessageTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, p := range parts {
			if p.Type == "text" && p.Text != "" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(p.Text)
			}
		}
		return b.String()
	}
	return string(raw)
}

// ParseMessages splits OpenAI chat messages and hoists system prompts.
func ParseMessages(body *jsonutils.JSONDict) ([]Message, string, error) {
	if body == nil {
		return nil, "", fmt.Errorf("nil request body")
	}
	arr, err := body.Get("messages")
	if err != nil {
		return nil, "", fmt.Errorf("missing messages")
	}
	raw := []byte(arr.String())
	var msgs []Message
	if err := json.Unmarshal(raw, &msgs); err != nil {
		return nil, "", fmt.Errorf("invalid messages: %w", err)
	}
	var system strings.Builder
	out := make([]Message, 0, len(msgs))
	for i := range msgs {
		role := strings.ToLower(strings.TrimSpace(msgs[i].Role))
		switch role {
		case "system":
			text := MessageTextContent(msgs[i].Content)
			if text != "" {
				if system.Len() > 0 {
					system.WriteString("\n\n")
				}
				system.WriteString(text)
			}
		case "user", "assistant", "tool":
			out = append(out, msgs[i])
		default:
			out = append(out, msgs[i])
		}
	}
	return out, system.String(), nil
}

// IntParam reads the first positive int param from an OpenAI JSON body.
func IntParam(body *jsonutils.JSONDict, keys ...string) (int, bool) {
	for _, k := range keys {
		if v, err := body.Int(k); err == nil && v > 0 {
			return int(v), true
		}
	}
	return 0, false
}

// FloatParam reads a float param from an OpenAI JSON body.
func FloatParam(body *jsonutils.JSONDict, key string) (float64, bool) {
	if v, err := body.Float(key); err == nil {
		return v, true
	}
	return 0, false
}

// CloneBodyWithModel clones the request body and sets the upstream model id.
func CloneBodyWithModel(body *jsonutils.JSONDict, upstreamModel string) *jsonutils.JSONDict {
	dup := jsonutils.NewDict()
	if body != nil {
		dup = body.Copy()
	}
	dup.Set("model", jsonutils.NewString(upstreamModel))
	return dup
}

// ChatCompletionsURL builds an OpenAI-compatible chat completions endpoint.
func ChatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(base, "/v2") {
		return base + "/chat/completions"
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

// CompletionsURL builds an OpenAI-compatible legacy completions endpoint.
func CompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/completions") {
		return base
	}
	if strings.HasSuffix(base, "/v2") {
		return base + "/completions"
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/completions"
	}
	return base + "/v1/completions"
}

// EmbeddingsURL builds an OpenAI-compatible embeddings endpoint.
func EmbeddingsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/embeddings") {
		return base
	}
	if strings.HasSuffix(base, "/v2") {
		return base + "/embeddings"
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/embeddings"
	}
	return base + "/v1/embeddings"
}

// ImagesGenerationsURL builds an OpenAI-compatible images/generations endpoint.
func ImagesGenerationsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/images/generations") {
		return base
	}
	if strings.HasSuffix(base, "/v2") {
		return base + "/images/generations"
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/images/generations"
	}
	return base + "/v1/images/generations"
}

// BearerAuthHeaders returns standard OpenAI bearer auth headers.
func BearerAuthHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(apiKey),
		"Content-Type":  "application/json",
	}
}

// JoinURL joins a base URL and path segment.
func JoinURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = strings.TrimLeft(strings.TrimSpace(path), "/")
	if base == "" {
		return "/" + path
	}
	return base + "/" + path
}

// FinishReasonFromStop maps provider-specific stop reasons to OpenAI finish_reason values.
func FinishReasonFromStop(stop string) string {
	switch strings.TrimSpace(stop) {
	case "end_turn", "stop_sequence", "stop", "STOP":
		return "stop"
	case "max_tokens", "length":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		if stop == "" {
			return "stop"
		}
		return stop
	}
}

// NewChatCompletion builds an OpenAI chat.completion response object.
func NewChatCompletion(model, id, content, finishReason string, promptTokens, completionTokens int) map[string]interface{} {
	total := promptTokens + completionTokens
	return map[string]interface{}{
		"id":      id,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": FinishReasonFromStop(finishReason),
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      total,
		},
	}
}

// NewStreamChunk builds an OpenAI chat.completion.chunk object.
func NewStreamChunk(model, id string, index int, content string, finishReason string) map[string]interface{} {
	delta := map[string]interface{}{
		"role": "assistant",
	}
	if content != "" {
		delta["content"] = content
	}
	choice := map[string]interface{}{
		"index": index,
		"delta": delta,
	}
	if finishReason != "" {
		choice["finish_reason"] = FinishReasonFromStop(finishReason)
	}
	return map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]interface{}{choice},
	}
}

// MarshalJSON marshals v to JSON bytes.
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// ParseEmbeddingInput extracts string inputs from an OpenAI embeddings body.
func ParseEmbeddingInput(body *jsonutils.JSONDict) (texts []string, rawInput json.RawMessage, isText bool, err error) {
	if body == nil {
		return nil, nil, false, fmt.Errorf("nil request body")
	}
	inp, err := body.Get("input")
	if err != nil {
		return nil, nil, false, fmt.Errorf("missing input")
	}
	rawInput = []byte(inp.String())
	var s string
	if err := json.Unmarshal(rawInput, &s); err == nil {
		return []string{s}, rawInput, true, nil
	}
	var arr []string
	if err := json.Unmarshal(rawInput, &arr); err == nil {
		return arr, rawInput, true, nil
	}
	return nil, rawInput, false, nil
}

// NewEmbeddingsResponse builds an OpenAI embeddings list response.
func NewEmbeddingsResponse(model string, vectors [][]float64, promptTokens int) ([]byte, error) {
	data := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		data[i] = map[string]interface{}{
			"object":    "embedding",
			"index":     i,
			"embedding": v,
		}
	}
	if promptTokens <= 0 {
		promptTokens = 0
	}
	return MarshalJSON(map[string]interface{}{
		"object": "list",
		"data":   data,
		"model":  model,
		"usage": map[string]interface{}{
			"prompt_tokens": promptTokens,
			"total_tokens":  promptTokens,
		},
	})
}

// ParseImagePrompt reads the prompt from an OpenAI images/generations body.
func ParseImagePrompt(body *jsonutils.JSONDict) (string, error) {
	if body == nil {
		return "", fmt.Errorf("nil request body")
	}
	prompt, err := body.GetString("prompt")
	if err != nil || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("missing prompt")
	}
	return strings.TrimSpace(prompt), nil
}

// ImageCount reads n from an OpenAI images/generations body.
func ImageCount(body *jsonutils.JSONDict) int {
	if body == nil {
		return 1
	}
	if n, err := body.Int("n"); err == nil && n > 0 {
		return int(n)
	}
	return 1
}

// ImageSize reads size from an OpenAI images/generations body.
func ImageSize(body *jsonutils.JSONDict) string {
	if body == nil {
		return "1024x1024"
	}
	if s, err := body.GetString("size"); err == nil && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	return "1024x1024"
}

// SizeToAspectRatio maps OpenAI image size strings to provider aspect ratios.
func SizeToAspectRatio(size string) string {
	switch strings.TrimSpace(size) {
	case "1024x1792", "768x1344", "720x1280":
		return "9:16"
	case "1792x1024", "1344x768", "1280x720":
		return "16:9"
	case "256x256", "512x512", "1024x1024":
		return "1:1"
	default:
		return "1:1"
	}
}

// NewImagesGenerationsResponse builds an OpenAI images/generations response.
func NewImagesGenerationsResponse(items []ImageItem) ([]byte, error) {
	data := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		row := map[string]interface{}{}
		if item.URL != "" {
			row["url"] = item.URL
		}
		if item.B64 != "" {
			row["b64_json"] = item.B64
		}
		if item.RevisedPrompt != "" {
			row["revised_prompt"] = item.RevisedPrompt
		}
		data = append(data, row)
	}
	return MarshalJSON(map[string]interface{}{
		"created": time.Now().Unix(),
		"data":    data,
	})
}

// PatchFunc mutates an OpenAI request body before forwarding upstream.
type PatchFunc func(body *jsonutils.JSONDict, stream bool)

// PatchBody clones body and applies optional patches.
func PatchBody(body *jsonutils.JSONDict, stream bool, patches ...PatchFunc) *jsonutils.JSONDict {
	dup := jsonutils.NewDict()
	if body != nil {
		dup = body.Copy()
	}
	for _, patch := range patches {
		if patch != nil {
			patch(dup, stream)
		}
	}
	return dup
}
