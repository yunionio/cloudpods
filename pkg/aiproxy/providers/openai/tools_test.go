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
	"testing"

	"yunion.io/x/jsonutils"
)

func TestToolsToAnthropic(t *testing.T) {
	tools := []ToolDefinition{{
		Type: "function",
		Function: ToolFunctionDef{
			Name:        "get_weather",
			Description: "Get weather",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
		},
	}}
	out := ToolsToAnthropic(tools)
	if len(out) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(out))
	}
	if out[0]["name"] != "get_weather" {
		t.Fatalf("unexpected name: %v", out[0]["name"])
	}
}

func TestMessagesToAnthropicToolRoundTrip(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: json.RawMessage(`"hello"`)},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: ToolFunction{
					Name:      "get_weather",
					Arguments: `{"city":"Boston"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Content: json.RawMessage(`"72F"`)},
	}
	anthropicMsgs, err := MessagesToAnthropic(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(anthropicMsgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(anthropicMsgs))
	}
	blocks, ok := anthropicMsgs[1]["content"].([]map[string]interface{})
	if !ok || len(blocks) != 1 || blocks[0]["type"] != "tool_use" {
		t.Fatalf("expected assistant tool_use block, got %#v", anthropicMsgs[1])
	}
}

func TestAnthropicBlocksToAssistant(t *testing.T) {
	msg := AnthropicBlocksToAssistant([]AnthropicBlock{
		{Type: "text", Text: "Checking"},
		{Type: "tool_use", ID: "toolu_1", Name: "get_weather", Input: map[string]interface{}{"city": "Boston"}},
	})
	if msg.Content != "Checking" {
		t.Fatalf("unexpected content: %q", msg.Content)
	}
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Name != "get_weather" {
		t.Fatalf("unexpected tool calls: %#v", msg.ToolCalls)
	}
}

func TestMessagesToGemini(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: json.RawMessage(`"hi"`)},
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				Function: ToolFunction{Name: "fn", Arguments: `{"a":1}`},
			}},
		},
		{Role: "tool", Name: "fn", Content: json.RawMessage(`{"result":"ok"}`)},
	}
	contents, err := MessagesToGemini(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(contents))
	}
}

func TestExtractTools(t *testing.T) {
	body, _ := jsonutils.Parse([]byte(`{
		"tools":[{"type":"function","function":{"name":"fn","parameters":{"type":"object"}}}],
		"tool_choice":"auto"
	}`))
	tools, choice, err := ExtractTools(body.(*jsonutils.JSONDict))
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Function.Name != "fn" {
		t.Fatalf("unexpected tools: %#v", tools)
	}
	if string(choice) != `"auto"` {
		t.Fatalf("unexpected tool_choice: %s", choice)
	}
}

func TestNewChatCompletionWithTools(t *testing.T) {
	out := NewChatCompletionWithTools("m", "id", AssistantMessage{
		ToolCalls: []ToolCall{{
			ID: "call_1", Type: "function",
			Function: ToolFunction{Name: "fn", Arguments: `{}`},
		}},
	}, "tool_use", 1, 2)
	choices := out["choices"].([]map[string]interface{})
	msg := choices[0]["message"].(map[string]interface{})
	if msg["tool_calls"] == nil {
		t.Fatal("expected tool_calls in message")
	}
	if choices[0]["finish_reason"] != "tool_calls" {
		t.Fatalf("expected finish_reason tool_calls, got %v", choices[0]["finish_reason"])
	}
}
