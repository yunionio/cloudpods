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

	"github.com/google/uuid"

	"yunion.io/x/jsonutils"
)

// DefaultResponsesMaxOutputTokens is used when Codex/Responses clients omit max_output_tokens.
const DefaultResponsesMaxOutputTokens = int64(8192)

// ResponsesMaxOutputTokens reads max_output_tokens or max_tokens from a Responses body.
func ResponsesMaxOutputTokens(body *jsonutils.JSONDict) int64 {
	if body == nil {
		return 0
	}
	if mt, err := body.Int("max_output_tokens"); err == nil && mt > 0 {
		return mt
	}
	if mt, err := body.Int("max_tokens"); err == nil && mt > 0 {
		return mt
	}
	return 0
}

type responsesInputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	ID        string          `json:"id"`
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	Arguments string          `json:"arguments"`
	Input     string          `json:"input"`
	Output    json.RawMessage `json:"output"`
	Summary   []struct {
		Type      string `json:"type"`
		Text      string `json:"text"`
		Signature string `json:"signature"`
	} `json:"summary"`
}

// ResponsesConvertState carries per-request tool map for response reversal.
type ResponsesConvertState struct {
	ToolMap CodexToolMap
}

// ResponsesToChatCompletions converts an OpenAI Responses request to chat/completions shape.
func ResponsesToChatCompletions(body *jsonutils.JSONDict, upstreamModel string) (*jsonutils.JSONDict, *ResponsesConvertState, error) {
	if body == nil {
		return nil, nil, fmt.Errorf("nil request body")
	}
	state := &ResponsesConvertState{}
	out := jsonutils.NewDict()
	model := strings.TrimSpace(upstreamModel)
	if model == "" {
		if m, err := body.GetString("model"); err == nil {
			model = strings.TrimSpace(m)
		}
	}
	if model == "" {
		return nil, nil, fmt.Errorf("missing model")
	}
	out.Set("model", jsonutils.NewString(model))

	if mt := ResponsesMaxOutputTokens(body); mt > 0 {
		out.Set("max_tokens", jsonutils.NewInt(mt))
	}
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
	if stops, err := body.Get("stop"); err == nil {
		out.Set("stop", stops)
	}

	var systemParts []string
	if instr, _ := body.GetString("instructions"); strings.TrimSpace(instr) != "" {
		systemParts = append(systemParts, instr)
	}
	msgs, sysFromInput, err := responsesInputToMessages(body)
	if err != nil {
		return nil, nil, err
	}
	systemParts = append(systemParts, sysFromInput...)
	messages := jsonutils.NewArray()
	if len(systemParts) > 0 {
		sysMsg := jsonutils.NewDict()
		sysMsg.Set("role", jsonutils.NewString("system"))
		sysMsg.Set("content", jsonutils.NewString(strings.Join(systemParts, "\n")))
		messages.Add(sysMsg)
	}
	for _, m := range msgs {
		messages.Add(m)
	}
	if messages.Size() == 0 {
		return nil, nil, fmt.Errorf("no convertible input messages")
	}
	out.Set("messages", messages)

	if toolsRaw, err := body.Get("tools"); err == nil {
		if arr, ok := toolsRaw.(*jsonutils.JSONArray); ok {
			tools, toolMap, err := FlattenResponsesTools(arr)
			if err != nil {
				return nil, nil, err
			}
			state.ToolMap = toolMap
			if tools != nil && tools.Length() > 0 {
				out.Set("tools", tools)
			}
		}
	}
	if tc, err := body.Get("tool_choice"); err == nil {
		out.Set("tool_choice", tc)
	}
	return out, state, nil
}

func responsesInputToMessages(body *jsonutils.JSONDict) ([]*jsonutils.JSONDict, []string, error) {
	raw, err := body.Get("input")
	if err != nil {
		return nil, nil, nil
	}
	rawBytes := []byte(raw.String())
	trimmed := strings.TrimSpace(string(rawBytes))
	if trimmed == "" || trimmed == "null" {
		return nil, nil, nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var text string
		if err := json.Unmarshal(rawBytes, &text); err != nil {
			return nil, nil, fmt.Errorf("invalid input string: %w", err)
		}
		if text == "" {
			return nil, nil, nil
		}
		msg := jsonutils.NewDict()
		msg.Set("role", jsonutils.NewString("user"))
		msg.Set("content", jsonutils.NewString(text))
		return []*jsonutils.JSONDict{msg}, nil, nil
	}
	if !strings.HasPrefix(trimmed, "[") {
		return nil, nil, fmt.Errorf("input must be a string or array")
	}
	var items []responsesInputItem
	if err := json.Unmarshal(rawBytes, &items); err != nil {
		return nil, nil, fmt.Errorf("invalid input array: %w", err)
	}
	var system []string
	var messages []*jsonutils.JSONDict
	var pendingReasoning []string
	var pendingToolCalls []ToolCall

	flushAssistant := func() {
		if len(pendingToolCalls) == 0 && len(pendingReasoning) == 0 {
			return
		}
		msg := jsonutils.NewDict()
		msg.Set("role", jsonutils.NewString("assistant"))
		if len(pendingReasoning) > 0 {
			msg.Set("reasoning_content", jsonutils.NewString(strings.Join(pendingReasoning, "")))
		}
		if len(pendingToolCalls) > 0 {
			calls := jsonutils.NewArray()
			for _, tc := range pendingToolCalls {
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
		messages = append(messages, msg)
		pendingReasoning = nil
		pendingToolCalls = nil
	}

	for _, item := range items {
		if isResponsesToolOutputType(item.Type) {
			flushAssistant()
			toolMsg := jsonutils.NewDict()
			toolMsg.Set("role", jsonutils.NewString("tool"))
			toolMsg.Set("tool_call_id", jsonutils.NewString(firstNonEmptyStr(item.CallID, item.ID)))
			toolMsg.Set("content", jsonutils.NewString(responsesOutputToText(item.Output)))
			messages = append(messages, toolMsg)
			continue
		}
		if item.Type == "reasoning" {
			for _, s := range item.Summary {
				if s.Text != "" {
					pendingReasoning = append(pendingReasoning, s.Text)
				}
			}
			continue
		}
		if item.Type == "function_call" || item.Type == "custom_tool_call" || item.Type == "local_shell_call" {
			args := item.Arguments
			if args == "" && item.Input != "" {
				b, _ := json.Marshal(map[string]string{"input": item.Input})
				args = string(b)
			}
			if !json.Valid([]byte(args)) {
				args = "{}"
			}
			name := item.Name
			if item.Namespace != "" && name != "" {
				name = NamespacedToolName(item.Namespace, name)
			}
			pendingToolCalls = append(pendingToolCalls, ToolCall{
				ID:   firstNonEmptyStr(item.CallID, item.ID),
				Type: "function",
				Function: ToolFunction{
					Name:      name,
					Arguments: args,
				},
			})
			continue
		}
		flushAssistant()
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" {
			role = "user"
		}
		switch role {
		case "system", "developer":
			if text := responsesContentToText(item.Content); text != "" {
				system = append(system, text)
			}
		case "assistant":
			msg := jsonutils.NewDict()
			msg.Set("role", jsonutils.NewString("assistant"))
			if text := responsesContentToText(item.Content); text != "" {
				msg.Set("content", jsonutils.NewString(text))
			}
			messages = append(messages, msg)
		default:
			msg := jsonutils.NewDict()
			msg.Set("role", jsonutils.NewString("user"))
			if text := responsesContentToText(item.Content); text != "" {
				msg.Set("content", jsonutils.NewString(text))
			} else {
				msg.Set("content", jsonutils.NewString(""))
			}
			messages = append(messages, msg)
		}
	}
	flushAssistant()
	return messages, system, nil
}

func isResponsesToolOutputType(t string) bool {
	switch strings.TrimSpace(t) {
	case "function_call_output", "custom_tool_call_output", "local_shell_call_output":
		return true
	default:
		return false
	}
}

func responsesContentToText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var texts []string
		for _, p := range parts {
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, "")
	}
	return string(raw)
}

func responsesOutputToText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return responsesContentToText(raw)
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ChatCompletionToResponses converts an OpenAI chat.completion JSON body to Responses API format.
func ChatCompletionToResponses(body []byte, state *ResponsesConvertState) ([]byte, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role             string          `json:"role"`
				Content          json.RawMessage `json:"content"`
				ReasoningContent string          `json:"reasoning_content"`
				ToolCalls        []ToolCall      `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid OpenAI response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty OpenAI choices")
	}
	choice := resp.Choices[0]
	id := resp.ID
	if id == "" {
		id = "resp_" + uuid.New().String()
	}
	status := "completed"
	if choice.FinishReason == "length" {
		status = "incomplete"
	}
	output := make([]map[string]interface{}, 0)
	if rc := strings.TrimSpace(choice.Message.ReasoningContent); rc != "" {
		output = append(output, map[string]interface{}{
			"type":   "reasoning",
			"status": "completed",
			"summary": []map[string]interface{}{
				{"type": "text", "text": rc},
			},
		})
	}
	text := MessageTextContent(choice.Message.Content)
	if text != "" {
		output = append(output, map[string]interface{}{
			"type":    "message",
			"status":  "completed",
			"role":    "assistant",
			"content": []map[string]interface{}{{"type": "text", "text": text}},
		})
	}
	toolMap := CodexToolMap{}
	if state != nil {
		toolMap = state.ToolMap
	}
	for _, tc := range choice.Message.ToolCalls {
		args := strings.TrimSpace(tc.Function.Arguments)
		ns, name, decodedArgs := DecodeNamespaceToolCall(toolMap, tc.Function.Name, args)
		item := map[string]interface{}{
			"type":      "function_call",
			"status":    "completed",
			"call_id":   firstNonEmptyStr(tc.ID, "call_"+uuid.New().String()),
			"name":      name,
			"arguments": decodedArgs,
		}
		if ns != "" {
			item["namespace"] = ns
		}
		output = append(output, item)
	}
	out := map[string]interface{}{
		"id":          id,
		"object":      "response",
		"created_at":  time.Now().Unix(),
		"status":      status,
		"model":       resp.Model,
		"output":      output,
		"output_text": text,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
			"total_tokens":  resp.Usage.TotalTokens,
		},
	}
	return json.Marshal(out)
}
