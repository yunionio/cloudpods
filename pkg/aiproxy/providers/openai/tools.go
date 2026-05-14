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

// ToolCall is one OpenAI assistant tool invocation.
type ToolCall struct {
	Index    int          `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function ToolFunction `json:"function"`
}

// ToolFunction is the function payload inside a tool call.
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition describes one OpenAI function tool.
type ToolDefinition struct {
	Type     string          `json:"type"`
	Function ToolFunctionDef `json:"function"`
}

// ToolFunctionDef is the function schema in an OpenAI tools array.
type ToolFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// AssistantMessage is a normalized assistant response for OpenAI chat.completion.
type AssistantMessage struct {
	Content   string
	ToolCalls []ToolCall
}

// ExtractTools reads tools and tool_choice from an OpenAI chat body.
func ExtractTools(body *jsonutils.JSONDict) ([]ToolDefinition, json.RawMessage, error) {
	if body == nil {
		return nil, nil, nil
	}
	toolsRaw, err := body.Get("tools")
	if err != nil {
		return nil, nil, nil
	}
	var tools []ToolDefinition
	if err := json.Unmarshal([]byte(toolsRaw.String()), &tools); err != nil {
		return nil, nil, fmt.Errorf("invalid tools: %w", err)
	}
	var toolChoice json.RawMessage
	if tc, err := body.Get("tool_choice"); err == nil {
		toolChoice = []byte(tc.String())
	}
	return tools, toolChoice, nil
}

// ToolsToAnthropic converts OpenAI tools to Anthropic tools.
func ToolsToAnthropic(tools []ToolDefinition) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		if strings.TrimSpace(t.Type) != "" && t.Type != "function" {
			continue
		}
		name := strings.TrimSpace(t.Function.Name)
		if name == "" {
			continue
		}
		item := map[string]interface{}{
			"name": name,
		}
		if desc := strings.TrimSpace(t.Function.Description); desc != "" {
			item["description"] = desc
		}
		if len(t.Function.Parameters) > 0 && string(t.Function.Parameters) != "null" {
			var schema interface{}
			if json.Unmarshal(t.Function.Parameters, &schema) == nil {
				item["input_schema"] = schema
			}
		}
		out = append(out, item)
	}
	return out
}

// ToolChoiceToAnthropic converts OpenAI tool_choice to Anthropic tool_choice.
func ToolChoiceToAnthropic(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "", "auto":
			return map[string]interface{}{"type": "auto"}
		case "none":
			return map[string]interface{}{"type": "none"}
		case "required":
			return map[string]interface{}{"type": "any"}
		}
	}
	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		if strings.EqualFold(obj.Type, "function") && strings.TrimSpace(obj.Function.Name) != "" {
			return map[string]interface{}{
				"type": "tool",
				"name": strings.TrimSpace(obj.Function.Name),
			}
		}
	}
	return nil
}

// MessagesToAnthropic converts OpenAI messages to Anthropic message objects.
func MessagesToAnthropic(msgs []Message) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0, len(msgs))
	for _, m := range msgs {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "assistant":
			blocks := assistantContentToAnthropic(m)
			if len(blocks) == 0 {
				continue
			}
			out = append(out, map[string]interface{}{
				"role":    "assistant",
				"content": blocks,
			})
		case "tool":
			text := MessageTextContent(m.Content)
			if m.ToolCallID == "" && text == "" {
				continue
			}
			out = append(out, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type":        "tool_result",
						"tool_use_id": m.ToolCallID,
						"content":     text,
					},
				},
			})
		case "user":
			text := MessageTextContent(m.Content)
			if text == "" {
				continue
			}
			out = append(out, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": text},
				},
			})
		default:
			text := MessageTextContent(m.Content)
			if text == "" {
				continue
			}
			out = append(out, map[string]interface{}{
				"role": role,
				"content": []map[string]interface{}{
					{"type": "text", "text": text},
				},
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no convertible messages")
	}
	return out, nil
}

func assistantContentToAnthropic(m Message) []map[string]interface{} {
	blocks := make([]map[string]interface{}, 0, 1+len(m.ToolCalls))
	if text := MessageTextContent(m.Content); text != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "text",
			"text": text,
		})
	}
	for _, tc := range m.ToolCalls {
		if strings.TrimSpace(tc.Function.Name) == "" {
			continue
		}
		input := map[string]interface{}{}
		args := strings.TrimSpace(tc.Function.Arguments)
		if args != "" {
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
	return blocks
}

// AnthropicBlock is one Anthropic message content block.
type AnthropicBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// AnthropicBlocksToAssistant converts Anthropic content blocks to OpenAI assistant message fields.
func AnthropicBlocksToAssistant(blocks []AnthropicBlock) AssistantMessage {
	var out AssistantMessage
	for _, b := range blocks {
		switch b.Type {
		case "text":
			out.Content += b.Text
		case "tool_use":
			args, _ := json.Marshal(b.Input)
			id := strings.TrimSpace(b.ID)
			if id == "" {
				id = "call_" + strings.TrimSpace(b.Name)
			}
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   id,
				Type: "function",
				Function: ToolFunction{
					Name:      strings.TrimSpace(b.Name),
					Arguments: string(args),
				},
			})
		}
	}
	return out
}

// ToolsToGemini converts OpenAI tools to Gemini functionDeclarations.
func ToolsToGemini(tools []ToolDefinition) []map[string]interface{} {
	decls := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		if t.Type != "" && t.Type != "function" {
			continue
		}
		name := strings.TrimSpace(t.Function.Name)
		if name == "" {
			continue
		}
		decl := map[string]interface{}{
			"name": name,
		}
		if desc := strings.TrimSpace(t.Function.Description); desc != "" {
			decl["description"] = desc
		}
		if len(t.Function.Parameters) > 0 && string(t.Function.Parameters) != "null" {
			var params interface{}
			if json.Unmarshal(t.Function.Parameters, &params) == nil {
				decl["parameters"] = params
			}
		}
		decls = append(decls, decl)
	}
	if len(decls) == 0 {
		return nil
	}
	return []map[string]interface{}{{"functionDeclarations": decls}}
}

// MessagesToGemini converts OpenAI messages to Gemini contents entries.
func MessagesToGemini(msgs []Message) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0, len(msgs))
	for _, m := range msgs {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "assistant":
			parts := assistantPartsToGemini(m)
			if len(parts) == 0 {
				continue
			}
			out = append(out, map[string]interface{}{
				"role":  "model",
				"parts": parts,
			})
		case "tool":
			name := strings.TrimSpace(m.Name)
			if name == "" {
				name = "tool"
			}
			resp := toolResultToGeminiResponse(MessageTextContent(m.Content))
			out = append(out, map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"functionResponse": map[string]interface{}{
							"name":     name,
							"response": resp,
						},
					},
				},
			})
		case "user":
			text := MessageTextContent(m.Content)
			if text == "" {
				continue
			}
			out = append(out, map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": text},
				},
			})
		default:
			text := MessageTextContent(m.Content)
			if text == "" {
				continue
			}
			out = append(out, map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": text},
				},
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no convertible messages")
	}
	return out, nil
}

func assistantPartsToGemini(m Message) []map[string]interface{} {
	parts := make([]map[string]interface{}, 0, 1+len(m.ToolCalls))
	if text := MessageTextContent(m.Content); text != "" {
		parts = append(parts, map[string]interface{}{"text": text})
	}
	for _, tc := range m.ToolCalls {
		name := strings.TrimSpace(tc.Function.Name)
		if name == "" {
			continue
		}
		args := map[string]interface{}{}
		if raw := strings.TrimSpace(tc.Function.Arguments); raw != "" {
			_ = json.Unmarshal([]byte(raw), &args)
		}
		parts = append(parts, map[string]interface{}{
			"functionCall": map[string]interface{}{
				"name": name,
				"args": args,
			},
		})
	}
	return parts
}

func toolResultToGeminiResponse(content string) map[string]interface{} {
	content = strings.TrimSpace(content)
	if content == "" {
		return map[string]interface{}{}
	}
	var obj map[string]interface{}
	if json.Unmarshal([]byte(content), &obj) == nil {
		return obj
	}
	return map[string]interface{}{"output": content}
}

type geminiPart struct {
	Text         string `json:"text"`
	FunctionCall *struct {
		Name string                 `json:"name"`
		Args map[string]interface{} `json:"args"`
	} `json:"functionCall"`
}

// GeminiPart is one Gemini content part in a candidate response.
type GeminiPart = geminiPart

// GeminiPartsToAssistant converts Gemini candidate parts to OpenAI assistant fields.
func GeminiPartsToAssistant(parts []geminiPart) AssistantMessage {
	var out AssistantMessage
	for _, p := range parts {
		if p.Text != "" {
			out.Content += p.Text
		}
		if p.FunctionCall != nil && strings.TrimSpace(p.FunctionCall.Name) != "" {
			args, _ := json.Marshal(p.FunctionCall.Args)
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   "call_" + strings.TrimSpace(p.FunctionCall.Name),
				Type: "function",
				Function: ToolFunction{
					Name:      strings.TrimSpace(p.FunctionCall.Name),
					Arguments: string(args),
				},
			})
		}
	}
	return out
}

// NewChatCompletionWithTools builds an OpenAI chat.completion including tool_calls.
func NewChatCompletionWithTools(model, id string, msg AssistantMessage, finishReason string, promptTokens, completionTokens int) map[string]interface{} {
	message := map[string]interface{}{
		"role": "assistant",
	}
	if msg.Content != "" {
		message["content"] = msg.Content
	} else if len(msg.ToolCalls) > 0 {
		message["content"] = nil
	} else {
		message["content"] = ""
	}
	if len(msg.ToolCalls) > 0 {
		calls := make([]map[string]interface{}, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			typ := tc.Type
			if typ == "" {
				typ = "function"
			}
			calls[i] = map[string]interface{}{
				"id":   tc.ID,
				"type": typ,
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			}
		}
		message["tool_calls"] = calls
	}
	total := promptTokens + completionTokens
	reason := FinishReasonFromStop(finishReason)
	if len(msg.ToolCalls) > 0 && reason == "stop" {
		reason = "tool_calls"
	}
	return map[string]interface{}{
		"id":      id,
		"object":  "chat.completion",
		"created": jsonNowUnix(),
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"message":       message,
				"finish_reason": reason,
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      total,
		},
	}
}

// NewStreamChunkToolDelta builds an OpenAI stream chunk with tool_calls delta.
func NewStreamChunkToolDelta(model, id string, index int, tc ToolCall, finishReason string) map[string]interface{} {
	delta := map[string]interface{}{
		"role": "assistant",
	}
	call := map[string]interface{}{
		"index": index,
	}
	if tc.ID != "" {
		call["id"] = tc.ID
	}
	typ := tc.Type
	if typ == "" {
		typ = "function"
	}
	call["type"] = typ
	fn := map[string]interface{}{}
	if tc.Function.Name != "" {
		fn["name"] = tc.Function.Name
	}
	if tc.Function.Arguments != "" {
		fn["arguments"] = tc.Function.Arguments
	}
	if len(fn) > 0 {
		call["function"] = fn
	}
	delta["tool_calls"] = []map[string]interface{}{call}
	choice := map[string]interface{}{
		"index": 0,
		"delta": delta,
	}
	if finishReason != "" {
		choice["finish_reason"] = FinishReasonFromStop(finishReason)
	}
	return map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": jsonNowUnix(),
		"model":   model,
		"choices": []map[string]interface{}{choice},
	}
}

func jsonNowUnix() int64 {
	return time.Now().Unix()
}
