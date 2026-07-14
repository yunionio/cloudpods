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

	"yunion.io/x/jsonutils"
)

// AnthropicToChatCompletions converts an Anthropic Messages request body to OpenAI chat/completions shape.
func AnthropicToChatCompletions(body *jsonutils.JSONDict, upstreamModel string) (*jsonutils.JSONDict, error) {
	if body == nil {
		return nil, fmt.Errorf("nil request body")
	}
	out := jsonutils.NewDict()
	model := strings.TrimSpace(upstreamModel)
	if model == "" {
		if m, err := body.GetString("model"); err == nil {
			model = strings.TrimSpace(m)
		}
	}
	if model == "" {
		return nil, fmt.Errorf("missing model")
	}
	out.Set("model", jsonutils.NewString(model))

	maxTokens, err := body.Int("max_tokens")
	if err != nil || maxTokens <= 0 {
		return nil, fmt.Errorf("max_tokens is required")
	}
	out.Set("max_tokens", jsonutils.NewInt(maxTokens))

	if stream, _ := body.Bool("stream"); stream {
		out.Set("stream", jsonutils.JSONTrue)
		streamOpts := jsonutils.NewDict()
		streamOpts.Set("include_usage", jsonutils.JSONTrue)
		out.Set("stream_options", streamOpts)
	}
	if v, ok := FloatParam(body, "temperature"); ok {
		out.Set("temperature", jsonutils.NewFloat64(v))
	}
	if v, ok := FloatParam(body, "top_p"); ok {
		out.Set("top_p", jsonutils.NewFloat64(v))
	}
	if stops, err := body.Get("stop_sequences"); err == nil {
		out.Set("stop", stops)
	}

	msgs, err := anthropicMessagesToOpenAI(body)
	if err != nil {
		return nil, err
	}
	out.Set("messages", msgs)

	if tools, toolChoice, err := anthropicToolsToOpenAI(body); err != nil {
		return nil, err
	} else if tools != nil && tools.Length() > 0 {
		out.Set("tools", tools)
		if toolChoice != nil {
			out.Set("tool_choice", toolChoice)
		}
	}
	return out, nil
}

func anthropicMessagesToOpenAI(body *jsonutils.JSONDict) (*jsonutils.JSONArray, error) {
	var systemParts []string
	if sysRaw, err := body.Get("system"); err == nil {
		if sysText := anthropicSystemText(sysRaw); sysText != "" {
			systemParts = append(systemParts, sysText)
		}
	}
	rawMsgs, err := body.Get("messages")
	if err != nil {
		return nil, fmt.Errorf("missing messages")
	}
	var messages []json.RawMessage
	if err := json.Unmarshal([]byte(rawMsgs.String()), &messages); err != nil {
		return nil, fmt.Errorf("invalid messages: %w", err)
	}
	converted := jsonutils.NewArray()
	for _, raw := range messages {
		var msg struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, fmt.Errorf("invalid message: %w", err)
		}
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "system":
			sysText, err := anthropicMessageContentText(msg.Content)
			if err != nil {
				return nil, err
			}
			if sysText != "" {
				systemParts = append(systemParts, sysText)
			}
		case "user":
			parts, tools, err := parseAnthropicUserContent(msg.Content)
			if err != nil {
				return nil, err
			}
			for _, tr := range tools {
				toolMsg := jsonutils.NewDict()
				toolMsg.Set("role", jsonutils.NewString("tool"))
				toolMsg.Set("tool_call_id", jsonutils.NewString(tr.ID))
				toolMsg.Set("content", jsonutils.NewString(tr.Content))
				converted.Add(toolMsg)
			}
			if parts != nil {
				userMsg := jsonutils.NewDict()
				userMsg.Set("role", jsonutils.NewString("user"))
				userMsg.Set("content", parts)
				converted.Add(userMsg)
			}
		case "assistant":
			assistant, err := parseAnthropicAssistantContent(msg.Content)
			if err != nil {
				return nil, err
			}
			if assistant == nil {
				continue
			}
			converted.Add(assistant)
		default:
			return nil, fmt.Errorf("unsupported message role %q", role)
		}
	}
	if converted.Size() == 0 {
		return nil, fmt.Errorf("no convertible messages")
	}
	arr := jsonutils.NewArray()
	if len(systemParts) > 0 {
		sysMsg := jsonutils.NewDict()
		sysMsg.Set("role", jsonutils.NewString("system"))
		sysMsg.Set("content", jsonutils.NewString(strings.Join(systemParts, "\n\n")))
		arr.Add(sysMsg)
	}
	for i := 0; i < converted.Size(); i++ {
		obj, err := converted.GetAt(i)
		if err != nil {
			return nil, err
		}
		arr.Add(obj)
	}
	return arr, nil
}

func anthropicMessageContentText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	parsed, err := jsonutils.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid message content: %w", err)
	}
	return anthropicSystemText(parsed), nil
}

type anthropicToolResult struct {
	ID      string
	Content string
}

func anthropicSystemText(raw jsonutils.JSONObject) string {
	if raw == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal([]byte(raw.String()), &s); err == nil {
		return strings.TrimSpace(s)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw.String()), &blocks); err == nil {
		var b strings.Builder
		for _, blk := range blocks {
			if blk.Type == "text" && blk.Text != "" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(blk.Text)
			}
		}
		return b.String()
	}
	return ""
}

func parseAnthropicUserContent(raw json.RawMessage) (jsonutils.JSONObject, []anthropicToolResult, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if strings.TrimSpace(s) == "" {
			return nil, nil, nil
		}
		return jsonutils.NewString(s), nil, nil
	}
	var blocks []map[string]interface{}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, nil, fmt.Errorf("invalid user content: %w", err)
	}
	parts := jsonutils.NewArray()
	var tools []anthropicToolResult
	for _, blk := range blocks {
		typ, _ := blk["type"].(string)
		switch typ {
		case "text":
			if t, _ := blk["text"].(string); t != "" {
				part := jsonutils.NewDict()
				part.Set("type", jsonutils.NewString("text"))
				part.Set("text", jsonutils.NewString(t))
				parts.Add(part)
			}
		case "image":
			if part := anthropicImageBlockToChatPart(blk); part != nil {
				parts.Add(part)
			}
		case "tool_result":
			id, _ := blk["tool_use_id"].(string)
			content := anthropicBlockContentText(blk["content"])
			tools = append(tools, anthropicToolResult{ID: id, Content: content})
		}
	}
	if parts.Size() == 0 {
		return nil, tools, nil
	}
	// Single text-only part can stay a plain string for compatibility.
	if parts.Size() == 1 {
		first, _ := parts.GetAt(0)
		if d, ok := first.(*jsonutils.JSONDict); ok {
			if typ, _ := d.GetString("type"); typ == "text" {
				if text, _ := d.GetString("text"); text != "" {
					return jsonutils.NewString(text), tools, nil
				}
			}
		}
	}
	return parts, tools, nil
}

func anthropicImageBlockToChatPart(blk map[string]interface{}) *jsonutils.JSONDict {
	srcRaw, ok := blk["source"].(map[string]interface{})
	if !ok || srcRaw == nil {
		return nil
	}
	srcType, _ := srcRaw["type"].(string)
	var url string
	switch strings.ToLower(strings.TrimSpace(srcType)) {
	case "url":
		url, _ = srcRaw["url"].(string)
	case "base64":
		data, _ := srcRaw["data"].(string)
		mediaType, _ := srcRaw["media_type"].(string)
		if strings.TrimSpace(data) == "" {
			return nil
		}
		if mediaType == "" {
			mediaType = "image/png"
		}
		if strings.HasPrefix(data, "data:") {
			url = data
		} else {
			url = "data:" + mediaType + ";base64," + data
		}
	default:
		if u, _ := srcRaw["url"].(string); strings.TrimSpace(u) != "" {
			url = u
		} else if data, _ := srcRaw["data"].(string); strings.TrimSpace(data) != "" {
			mediaType, _ := srcRaw["media_type"].(string)
			if mediaType == "" {
				mediaType = "image/png"
			}
			url = "data:" + mediaType + ";base64," + data
		}
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	imgURL := jsonutils.NewDict()
	imgURL.Set("url", jsonutils.NewString(url))
	part := jsonutils.NewDict()
	part.Set("type", jsonutils.NewString("image_url"))
	part.Set("image_url", imgURL)
	return part
}

func anthropicBlockContentText(v interface{}) string {
	switch c := v.(type) {
	case string:
		return c
	case []interface{}:
		var parts []string
		for _, item := range c {
			if m, ok := item.(map[string]interface{}); ok {
				if t, _ := m["text"].(string); t != "" {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprint(v)
	}
}

func parseAnthropicAssistantContent(raw json.RawMessage) (*jsonutils.JSONDict, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if strings.TrimSpace(s) == "" {
			return nil, nil
		}
		msg := jsonutils.NewDict()
		msg.Set("role", jsonutils.NewString("assistant"))
		msg.Set("content", jsonutils.NewString(s))
		return msg, nil
	}
	var blocks []AnthropicBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("invalid assistant content: %w", err)
	}
	assistant := AnthropicBlocksToAssistant(blocks)
	msg := jsonutils.NewDict()
	msg.Set("role", jsonutils.NewString("assistant"))
	if assistant.Content != "" {
		msg.Set("content", jsonutils.NewString(assistant.Content))
	}
	if len(assistant.ToolCalls) > 0 {
		calls := jsonutils.NewArray()
		for _, tc := range assistant.ToolCalls {
			call := jsonutils.NewDict()
			call.Set("id", jsonutils.NewString(tc.ID))
			call.Set("type", jsonutils.NewString("function"))
			fn := jsonutils.NewDict()
			fn.Set("name", jsonutils.NewString(tc.Function.Name))
			fn.Set("arguments", jsonutils.NewString(tc.Function.Arguments))
			call.Set("function", fn)
			calls.Add(call)
		}
		msg.Set("tool_calls", calls)
	}
	return msg, nil
}

func anthropicToolsToOpenAI(body *jsonutils.JSONDict) (*jsonutils.JSONArray, jsonutils.JSONObject, error) {
	rawTools, err := body.Get("tools")
	if err != nil {
		return nil, nil, nil
	}
	var toolsIn []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"input_schema"`
	}
	if err := json.Unmarshal([]byte(rawTools.String()), &toolsIn); err != nil {
		return nil, nil, fmt.Errorf("invalid tools: %w", err)
	}
	if len(toolsIn) == 0 {
		return nil, nil, nil
	}
	out := jsonutils.NewArray()
	for _, t := range toolsIn {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		tool := jsonutils.NewDict()
		tool.Set("type", jsonutils.NewString("function"))
		fn := jsonutils.NewDict()
		fn.Set("name", jsonutils.NewString(name))
		if desc := strings.TrimSpace(t.Description); desc != "" {
			fn.Set("description", jsonutils.NewString(desc))
		}
		if len(t.InputSchema) > 0 && string(t.InputSchema) != "null" {
			if params, err := jsonutils.Parse(t.InputSchema); err == nil {
				fn.Set("parameters", params)
			}
		}
		tool.Set("function", fn)
		out.Add(tool)
	}
	var toolChoice jsonutils.JSONObject
	if tcRaw, err := body.Get("tool_choice"); err == nil {
		toolChoice = anthropicToolChoiceToOpenAI(tcRaw)
	}
	return out, toolChoice, nil
}

func anthropicToolChoiceToOpenAI(raw jsonutils.JSONObject) jsonutils.JSONObject {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw.String()), &obj); err != nil {
		return nil
	}
	typ, _ := obj["type"].(string)
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "auto", "":
		return jsonutils.NewString("auto")
	case "none":
		return jsonutils.NewString("none")
	case "any":
		return jsonutils.NewString("required")
	case "tool":
		name, _ := obj["name"].(string)
		if strings.TrimSpace(name) == "" {
			return jsonutils.NewString("required")
		}
		choice := jsonutils.NewDict()
		choice.Set("type", jsonutils.NewString("function"))
		fn := jsonutils.NewDict()
		fn.Set("name", jsonutils.NewString(strings.TrimSpace(name)))
		choice.Set("function", fn)
		return choice
	default:
		return nil
	}
}

// ChatCompletionToAnthropic converts an OpenAI chat.completion JSON body to Anthropic Messages response.
func ChatCompletionToAnthropic(body []byte) ([]byte, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string          `json:"role"`
				Content   json.RawMessage `json:"content"`
				ToolCalls []ToolCall      `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid OpenAI response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty OpenAI choices")
	}
	choice := resp.Choices[0]
	blocks := make([]map[string]interface{}, 0, 1+len(choice.Message.ToolCalls))
	if text := MessageTextContent(choice.Message.Content); text != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "text",
			"text": text,
		})
	}
	for _, tc := range choice.Message.ToolCalls {
		input := map[string]interface{}{}
		if args := strings.TrimSpace(tc.Function.Arguments); args != "" {
			_ = json.Unmarshal([]byte(args), &input)
		}
		id := strings.TrimSpace(tc.ID)
		if id == "" {
			id = "toolu_" + strings.TrimSpace(tc.Function.Name)
		}
		blocks = append(blocks, map[string]interface{}{
			"type":  "tool_use",
			"id":    id,
			"name":  strings.TrimSpace(tc.Function.Name),
			"input": input,
		})
	}
	stopReason := openAIFinishReasonToAnthropic(choice.FinishReason)
	out := map[string]interface{}{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"model":       resp.Model,
		"content":     blocks,
		"stop_reason": stopReason,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
		},
	}
	return json.Marshal(out)
}

func openAIFinishReasonToAnthropic(reason string) string {
	switch strings.TrimSpace(reason) {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls", "function_call":
		return "tool_use"
	default:
		if reason == "" {
			return "end_turn"
		}
		return reason
	}
}

// OpenAIErrorToAnthropic converts an OpenAI-style error JSON body to Anthropic error format.
func OpenAIErrorToAnthropic(body []byte, statusCode int) []byte {
	msg := "upstream request failed"
	var wrap struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &wrap) == nil && wrap.Error.Message != "" {
		msg = wrap.Error.Message
	}
	errType := "api_error"
	if statusCode == 400 {
		errType = "invalid_request_error"
	} else if statusCode == 401 {
		errType = "authentication_error"
	} else if statusCode == 429 {
		errType = "rate_limit_error"
	}
	out, _ := json.Marshal(map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    errType,
			"message": msg,
		},
	})
	return out
}

// NewAnthropicErrorBody builds an Anthropic-style error response body.
func NewAnthropicErrorBody(errType, message string) []byte {
	out, _ := json.Marshal(map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    errType,
			"message": message,
		},
	})
	return out
}
