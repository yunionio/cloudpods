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

func TestAnthropicToChatCompletionsSystemInMessages(t *testing.T) {
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("deepseek-chat"))
	body.Set("max_tokens", jsonutils.NewInt(512))
	sysMsg := jsonutils.NewDict()
	sysMsg.Set("role", jsonutils.NewString("system"))
	sysMsg.Set("content", jsonutils.NewString("You are Claude Code."))
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("hi"))
	body.Set("messages", jsonutils.NewArray(sysMsg, userMsg))

	out, err := AnthropicToChatCompletions(body, "deepseek-chat")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := out.Get("messages")
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := msgs.(*jsonutils.JSONArray)
	if !ok || arr.Length() != 2 {
		t.Fatalf("expected system+user messages, got %#v", msgs)
	}
	role0, _ := arr.GetAt(0)
	if got, _ := role0.(*jsonutils.JSONDict).GetString("role"); got != "system" {
		t.Fatalf("first message role: got %q", got)
	}
}

func TestAnthropicToChatCompletionsBasic(t *testing.T) {
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("deepseek-chat"))
	body.Set("max_tokens", jsonutils.NewInt(512))
	body.Set("system", jsonutils.NewString("You are helpful."))
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("Hello"))
	body.Set("messages", jsonutils.NewArray(userMsg))

	out, err := AnthropicToChatCompletions(body, "deepseek-chat")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := out.GetString("model"); got != "deepseek-chat" {
		t.Fatalf("model: got %q", got)
	}
	msgs, err := out.Get("messages")
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := msgs.(*jsonutils.JSONArray)
	if !ok || arr.Length() != 2 {
		t.Fatalf("expected system+user messages, got %#v", msgs)
	}
}

func TestChatCompletionToAnthropicBasic(t *testing.T) {
	raw := []byte(`{
		"id":"chatcmpl-1",
		"model":"deepseek-chat",
		"choices":[{"message":{"role":"assistant","content":"Hi there"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":5,"completion_tokens":3}
	}`)
	out, err := ChatCompletionToAnthropic(raw)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["type"] != "message" {
		t.Fatalf("unexpected type: %#v", resp["type"])
	}
	if resp["stop_reason"] != "end_turn" {
		t.Fatalf("unexpected stop_reason: %#v", resp["stop_reason"])
	}
	content := resp["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("expected one content block, got %#v", content)
	}
}

func TestAnthropicStreamConverterMultipleTools(t *testing.T) {
	conv := NewAnthropicStreamConverter("deepseek-chat")
	textChunk, _ := json.Marshal(NewStreamChunk("deepseek-chat", "chatcmpl-1", 0, "Checking.", ""))
	if _, err := conv.Feed(textChunk, false); err != nil {
		t.Fatal(err)
	}

	tool1Start, _ := json.Marshal(NewStreamChunkToolDelta("deepseek-chat", "chatcmpl-1", 0, ToolCall{
		ID: "call_0", Type: "function", Function: ToolFunction{Name: "Bash"},
	}, ""))
	events, err := conv.Feed(tool1Start, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := blockIndexFromEvent(t, events[1]); got != 1 {
		t.Fatalf("tool1 start index: got %d", got)
	}

	tool1End, _ := json.Marshal(NewStreamChunkToolDelta("deepseek-chat", "chatcmpl-1", 0, ToolCall{
		Function: ToolFunction{Arguments: `{"command":"ls"}`},
	}, ""))
	if _, err := conv.Feed(tool1End, false); err != nil {
		t.Fatal(err)
	}

	tool2Start, _ := json.Marshal(NewStreamChunkToolDelta("deepseek-chat", "chatcmpl-1", 1, ToolCall{
		ID: "call_1", Type: "function", Function: ToolFunction{Name: "Bash"},
	}, ""))
	events, err = conv.Feed(tool2Start, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected stop+start for tool2, got %v", eventNames(events))
	}
	if events[0].Event != "content_block_stop" {
		t.Fatalf("expected stop before tool2, got %v", eventNames(events))
	}
	if got := blockIndexFromEvent(t, events[0]); got != 1 {
		t.Fatalf("tool1 stop index: got %d want 1", got)
	}
	if got := blockIndexFromEvent(t, events[1]); got != 2 {
		t.Fatalf("tool2 start index: got %d want 2", got)
	}

	finishChunk, _ := json.Marshal(map[string]interface{}{
		"id": "chatcmpl-1",
		"choices": []map[string]interface{}{
			{"delta": map[string]interface{}{}, "finish_reason": "tool_calls"},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     100,
			"completion_tokens": 50,
		},
	})
	events, err = conv.Feed(finishChunk, true)
	if err != nil {
		t.Fatal(err)
	}
	var sawStop2, sawMessageStop bool
	for _, evt := range events {
		switch evt.Event {
		case "content_block_stop":
			if blockIndexFromEvent(t, evt) == 2 {
				sawStop2 = true
			}
		case "message_stop":
			sawMessageStop = true
		}
	}
	if !sawStop2 {
		t.Fatalf("expected content_block_stop for tool2, got %v", eventNames(events))
	}
	if !sawMessageStop {
		t.Fatalf("expected message_stop, got %v", eventNames(events))
	}
	if conv.outputTokens != 50 {
		t.Fatalf("output tokens: got %d want 50", conv.outputTokens)
	}
}

func blockIndexFromEvent(t *testing.T, evt AnthropicStreamEvent) int {
	t.Helper()
	var wrap map[string]interface{}
	if err := json.Unmarshal(evt.Data, &wrap); err != nil {
		t.Fatal(err)
	}
	idx, ok := wrap["index"].(float64)
	if !ok {
		t.Fatalf("missing index in %s: %#v", evt.Event, wrap)
	}
	return int(idx)
}

func TestAnthropicStreamConverterTextToTool(t *testing.T) {
	conv := NewAnthropicStreamConverter("deepseek-chat")
	textChunk, _ := json.Marshal(NewStreamChunk("deepseek-chat", "chatcmpl-1", 0, "Let me check.", ""))
	events, err := conv.Feed(textChunk, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 3 {
		t.Fatalf("expected message_start + block start + delta, got %d events", len(events))
	}

	toolChunk, _ := json.Marshal(NewStreamChunkToolDelta("deepseek-chat", "chatcmpl-1", 0, ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: ToolFunction{
			Name: "Glob",
		},
	}, ""))
	events, err = conv.Feed(toolChunk, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected content_block_stop + content_block_start, got %d events: %v", len(events), eventNames(events))
	}
	if events[0].Event != "content_block_stop" || events[1].Event != "content_block_start" {
		t.Fatalf("unexpected events: %v", eventNames(events))
	}
	var start map[string]interface{}
	if err := json.Unmarshal(events[1].Data, &start); err != nil {
		t.Fatal(err)
	}
	if int(start["index"].(float64)) != 1 {
		t.Fatalf("tool block index: %#v", start["index"])
	}

	argsChunk, _ := json.Marshal(NewStreamChunkToolDelta("deepseek-chat", "chatcmpl-1", 0, ToolCall{
		Function: ToolFunction{Arguments: `{"pattern":"**/*.go"}`},
	}, ""))
	events, err = conv.Feed(argsChunk, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Event != "content_block_delta" {
		t.Fatalf("expected tool args delta, got %v", eventNames(events))
	}
	var delta map[string]interface{}
	if err := json.Unmarshal(events[0].Data, &delta); err != nil {
		t.Fatal(err)
	}
	if int(delta["index"].(float64)) != 1 {
		t.Fatalf("tool delta index: %#v", delta["index"])
	}
}

func eventNames(events []AnthropicStreamEvent) []string {
	out := make([]string, len(events))
	for i, evt := range events {
		out[i] = evt.Event
	}
	return out
}

func TestAnthropicStreamConverterText(t *testing.T) {
	conv := NewAnthropicStreamConverter("deepseek-chat")
	chunk, _ := json.Marshal(NewStreamChunk("deepseek-chat", "chatcmpl-1", 0, "Hello", ""))
	events, err := conv.Feed(chunk, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected stream events")
	}
	if events[0].Event != "message_start" {
		t.Fatalf("first event: %s", events[0].Event)
	}
	finishChunk, _ := json.Marshal(map[string]interface{}{
		"id": "chatcmpl-1",
		"choices": []map[string]interface{}{
			{"delta": map[string]interface{}{}, "finish_reason": "stop"},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     3,
			"completion_tokens": 2,
		},
	})
	events, err = conv.Feed(finishChunk, true)
	if err != nil {
		t.Fatal(err)
	}
	foundStop := false
	for _, evt := range events {
		if evt.Event == "message_stop" {
			foundStop = true
		}
	}
	if !foundStop {
		t.Fatal("expected message_stop event")
	}
}

func TestAnthropicToolRoundTrip(t *testing.T) {
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("claude-3-5-sonnet"))
	body.Set("max_tokens", jsonutils.NewInt(1024))
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("Weather?"))
	body.Set("messages", jsonutils.NewArray(userMsg))
	tool := jsonutils.NewDict()
	tool.Set("name", jsonutils.NewString("get_weather"))
	tool.Set("description", jsonutils.NewString("Get weather"))
	tool.Set("input_schema", jsonutils.NewDict())
	body.Set("tools", jsonutils.NewArray(tool))

	openaiBody, err := AnthropicToChatCompletions(body, "claude-3-5-sonnet")
	if err != nil {
		t.Fatal(err)
	}
	toolsObj, err := openaiBody.Get("tools")
	if err != nil {
		t.Fatal(err)
	}
	toolsArr, ok := toolsObj.(*jsonutils.JSONArray)
	if !ok || toolsArr.Length() != 1 {
		t.Fatalf("expected tools in OpenAI body: %#v", openaiBody)
	}

	raw := []byte(`{
		"id":"msg_1",
		"model":"claude-3-5-sonnet",
		"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Boston\"}"}}]},"finish_reason":"tool_calls"}],
		"usage":{"prompt_tokens":10,"completion_tokens":5}
	}`)
	out, err := ChatCompletionToAnthropic(raw)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["stop_reason"] != "tool_use" {
		t.Fatalf("stop_reason: %#v", resp["stop_reason"])
	}
}
