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

// ResponsesToAnthropicMessages converts a Responses API request to Anthropic Messages shape.
func ResponsesToAnthropicMessages(body *jsonutils.JSONDict, upstreamModel string) (*jsonutils.JSONDict, *ResponsesConvertState, error) {
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

	maxOut := ResponsesMaxOutputTokens(body)
	if maxOut <= 0 {
		maxOut = DefaultResponsesMaxOutputTokens
	}
	out.Set("max_tokens", jsonutils.NewInt(maxOut))

	if stream, _ := body.Bool("stream"); stream {
		out.Set("stream", jsonutils.JSONTrue)
	}
	if v, ok := FloatParam(body, "temperature"); ok {
		out.Set("temperature", jsonutils.NewFloat64(v))
	}
	if v, ok := FloatParam(body, "top_p"); ok {
		out.Set("top_p", jsonutils.NewFloat64(v))
	}
	if stops, err := body.Get("stop"); err == nil {
		out.Set("stop_sequences", stops)
	}

	var systemParts []string
	if instr, _ := body.GetString("instructions"); strings.TrimSpace(instr) != "" {
		systemParts = append(systemParts, instr)
	}
	chatMsgs, sysFromInput, err := responsesInputToMessages(body)
	if err != nil {
		return nil, nil, err
	}
	systemParts = append(systemParts, sysFromInput...)
	if len(systemParts) > 0 {
		out.Set("system", jsonutils.NewString(strings.Join(systemParts, "\n")))
	}

	anthropicMsgs, err := chatMessagesToAnthropic(chatMsgs)
	if err != nil {
		return nil, nil, err
	}
	out.Set("messages", anthropicMsgs)

	if toolsRaw, err := body.Get("tools"); err == nil {
		if arr, ok := toolsRaw.(*jsonutils.JSONArray); ok {
			chatTools, toolMap, err := FlattenResponsesTools(arr)
			if err != nil {
				return nil, nil, err
			}
			state.ToolMap = toolMap
			if chatTools != nil && chatTools.Length() > 0 {
				anthTools := jsonutils.NewArray()
				for i := 0; i < chatTools.Length(); i++ {
					toolWrap, _ := chatTools.GetAt(i)
					fn, _ := toolWrap.Get("function")
					if fn == nil {
						continue
					}
					name, _ := fn.GetString("name")
					desc, _ := fn.GetString("description")
					params, _ := fn.Get("parameters")
					at := jsonutils.NewDict()
					at.Set("name", jsonutils.NewString(name))
					if desc != "" {
						at.Set("description", jsonutils.NewString(desc))
					}
					if params != nil {
						at.Set("input_schema", params)
					}
					anthTools.Add(at)
				}
				if anthTools.Length() > 0 {
					out.Set("tools", anthTools)
				}
			}
		}
	}
	if tc, err := body.Get("tool_choice"); err == nil {
		if choice, err := responsesToolChoiceToAnthropic(tc); err == nil && choice != nil {
			out.Set("tool_choice", choice)
		}
	}
	return out, state, nil
}

func chatMessagesToAnthropic(chatMsgs []*jsonutils.JSONDict) (*jsonutils.JSONArray, error) {
	out := jsonutils.NewArray()
	for _, msg := range chatMsgs {
		role, _ := msg.GetString("role")
		switch strings.ToLower(role) {
		case "system":
			continue
		case "tool":
			id, _ := msg.GetString("tool_call_id")
			content, _ := msg.GetString("content")
			block := jsonutils.NewDict()
			block.Set("type", jsonutils.NewString("tool_result"))
			block.Set("tool_use_id", jsonutils.NewString(id))
			block.Set("content", jsonutils.NewString(content))
			user := jsonutils.NewDict()
			user.Set("role", jsonutils.NewString("user"))
			user.Set("content", jsonutils.NewArray(block))
			out.Add(user)
		case "assistant":
			blocks := jsonutils.NewArray()
			if rc, _ := msg.GetString("reasoning_content"); rc != "" {
				think := jsonutils.NewDict()
				think.Set("type", jsonutils.NewString("thinking"))
				think.Set("thinking", jsonutils.NewString(rc))
				blocks.Add(think)
			}
			if content, _ := msg.GetString("content"); content != "" {
				text := jsonutils.NewDict()
				text.Set("type", jsonutils.NewString("text"))
				text.Set("text", jsonutils.NewString(content))
				blocks.Add(text)
			}
			if tools, _ := msg.Get("tool_calls"); tools != nil {
				var calls []ToolCall
				_ = json.Unmarshal([]byte(tools.String()), &calls)
				for _, tc := range calls {
					input := map[string]interface{}{}
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
					tu := jsonutils.NewDict()
					tu.Set("type", jsonutils.NewString("tool_use"))
					tu.Set("id", jsonutils.NewString(firstNonEmptyStr(tc.ID, "toolu_"+tc.Function.Name)))
					tu.Set("name", jsonutils.NewString(tc.Function.Name))
					tu.Set("input", jsonutils.NewString(string(mustMarshal(input))))
					blocks.Add(tu)
				}
			}
			if blocks.Size() == 0 {
				continue
			}
			asst := jsonutils.NewDict()
			asst.Set("role", jsonutils.NewString("assistant"))
			asst.Set("content", blocks)
			out.Add(asst)
		default:
			content, _ := msg.GetString("content")
			user := jsonutils.NewDict()
			user.Set("role", jsonutils.NewString("user"))
			if content != "" {
				user.Set("content", jsonutils.NewString(content))
			} else {
				user.Set("content", jsonutils.NewString(""))
			}
			out.Add(user)
		}
	}
	if out.Size() == 0 {
		return nil, fmt.Errorf("no convertible messages")
	}
	return out, nil
}

func responsesToolChoiceToAnthropic(tc jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if tc == nil {
		return nil, nil
	}
	s := strings.TrimSpace(tc.String())
	if s == "" || s == "null" {
		return nil, nil
	}
	converted := ToolChoiceToAnthropic([]byte(s))
	if converted == nil {
		return nil, nil
	}
	data, err := json.Marshal(converted)
	if err != nil {
		return nil, err
	}
	obj, err := jsonutils.Parse(data)
	if err != nil {
		return nil, err
	}
	if dict, ok := obj.(*jsonutils.JSONDict); ok {
		return dict, nil
	}
	return nil, nil
}

// AnthropicMessagesToResponses converts an Anthropic Messages response to Responses API format.
func AnthropicMessagesToResponses(body []byte, state *ResponsesConvertState) ([]byte, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type      string          `json:"type"`
			Text      string          `json:"text"`
			Thinking  string          `json:"thinking"`
			Signature string          `json:"signature"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid anthropic response: %w", err)
	}
	id := resp.ID
	if id == "" {
		id = "resp_" + uuid.New().String()
	}
	status := "completed"
	if resp.StopReason == "max_tokens" {
		status = "incomplete"
	}
	toolMap := CodexToolMap{}
	if state != nil {
		toolMap = state.ToolMap
	}
	output := make([]map[string]interface{}, 0)
	var textParts []string
	for _, block := range resp.Content {
		switch block.Type {
		case "thinking":
			output = append(output, map[string]interface{}{
				"type":   "reasoning",
				"status": "completed",
				"summary": []map[string]interface{}{
					{"type": "text", "text": block.Thinking, "signature": block.Signature},
				},
			})
		case "text":
			textParts = append(textParts, block.Text)
			output = append(output, map[string]interface{}{
				"type":    "message",
				"status":  "completed",
				"role":    "assistant",
				"content": []map[string]interface{}{{"type": "text", "text": block.Text}},
			})
		case "tool_use":
			args := string(block.Input)
			if args == "" {
				args = "{}"
			}
			ns, name, decodedArgs := DecodeNamespaceToolCall(toolMap, block.Name, args)
			item := map[string]interface{}{
				"type":      "function_call",
				"status":    "completed",
				"call_id":   firstNonEmptyStr(block.ID, "call_"+uuid.New().String()),
				"name":      name,
				"arguments": decodedArgs,
			}
			if ns != "" {
				item["namespace"] = ns
			}
			output = append(output, item)
		}
	}
	out := map[string]interface{}{
		"id":          id,
		"object":      "response",
		"created_at":  time.Now().Unix(),
		"status":      status,
		"model":       resp.Model,
		"output":      output,
		"output_text": strings.Join(textParts, ""),
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
			"total_tokens":  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
	return json.Marshal(out)
}
