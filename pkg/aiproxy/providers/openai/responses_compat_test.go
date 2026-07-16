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
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestResponsesURL(t *testing.T) {
	if got := ResponsesURL("https://api.openai.com/v1"); got != "https://api.openai.com/v1/responses" {
		t.Fatalf("ResponsesURL = %q", got)
	}
	if got := ResponsesSubResourceURL("https://api.openai.com/v1", "resp_123", "cancel"); got != "https://api.openai.com/v1/responses/resp_123/cancel" {
		t.Fatalf("ResponsesSubResourceURL = %q", got)
	}
}

func TestResponsesToChatCompletionsStringInput(t *testing.T) {
	body, err := jsonutils.Parse([]byte(`{"model":"gpt-test","input":"hello","max_output_tokens":128}`))
	if err != nil {
		t.Fatal(err)
	}
	dict := body.(*jsonutils.JSONDict)
	out, state, err := ResponsesToChatCompletions(dict, "upstream-model")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("expected state")
	}
	if m, _ := out.GetString("model"); m != "upstream-model" {
		t.Fatalf("model = %q", m)
	}
	if mt, _ := out.Int("max_tokens"); mt != 128 {
		t.Fatalf("max_tokens = %d", mt)
	}
	msgs, _ := out.Get("messages")
	arr, ok := msgs.(*jsonutils.JSONArray)
	if !ok || arr.Size() < 1 {
		t.Fatalf("messages = %v", msgs)
	}
}

func TestResponsesToChatCompletionsFunctionCallInput(t *testing.T) {
	raw := `{
		"model":"gpt-test",
		"max_output_tokens":64,
		"input":[{"type":"function_call","name":"get_weather","call_id":"call_1","arguments":"{\"city\":\"bj\"}"},
		         {"type":"function_call_output","call_id":"call_1","output":"sunny"}]
	}`
	body, _ := jsonutils.Parse([]byte(raw))
	out, _, err := ResponsesToChatCompletions(body.(*jsonutils.JSONDict), "m")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "tool_calls") || !strings.Contains(s, "tool_call_id") {
		t.Fatalf("expected tool messages in %s", s)
	}
}

func TestChatCompletionToResponsesBasic(t *testing.T) {
	raw := []byte(`{
		"id":"chatcmpl-1",
		"model":"gpt-test",
		"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}
	}`)
	out, err := ChatCompletionToResponses(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["object"] != "response" {
		t.Fatalf("object = %v", resp["object"])
	}
	if resp["output_text"] != "hi" {
		t.Fatalf("output_text = %v", resp["output_text"])
	}
}

func TestFlattenResponsesNamespaceTool(t *testing.T) {
	arr := jsonutils.NewArray()
	arr.Add(jsonutils.NewString(`{
		"type":"namespace",
		"name":"myns",
		"tools":[{"type":"function","name":"search","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}]
	}`))
	// parse as proper dict
	arr = jsonutils.NewArray()
	ns := jsonutils.NewDict()
	ns.Set("type", jsonutils.NewString("namespace"))
	ns.Set("name", jsonutils.NewString("myns"))
	sub := jsonutils.NewDict()
	sub.Set("type", jsonutils.NewString("function"))
	sub.Set("name", jsonutils.NewString("search"))
	sub.Set("parameters", jsonutils.NewString(`{"type":"object","properties":{"q":{"type":"string"}}}`))
	subs := jsonutils.NewArray(sub)
	ns.Set("tools", subs)
	arr.Add(ns)

	tools, toolMap, err := FlattenResponsesTools(arr)
	if err != nil {
		t.Fatal(err)
	}
	if tools == nil || tools.Length() != 1 {
		t.Fatalf("tools = %v", tools)
	}
	if _, ok := toolMap["myns"]; !ok {
		t.Fatalf("toolMap = %v", toolMap)
	}
}

func TestResponsesToChatCompletionsWithImageInput(t *testing.T) {
	raw := `{
		"model":"kimi-k2.6",
		"input":[{"role":"user","content":[
			{"type":"input_text","text":"describe this image"},
			{"type":"input_image","image_url":"data:image/png;base64,abc123"}
		]}]
	}`
	body, _ := jsonutils.Parse([]byte(raw))
	out, _, err := ResponsesToChatCompletions(body.(*jsonutils.JSONDict), "kimi-k2.6")
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "image_url") {
		t.Fatalf("expected image_url in messages, got %s", s)
	}
	if !strings.Contains(s, "data:image/png;base64,abc123") {
		t.Fatalf("expected image data url in messages, got %s", s)
	}
	if !strings.Contains(s, "describe this image") {
		t.Fatalf("expected text in messages, got %s", s)
	}
}

func TestResponsesInputHasImage(t *testing.T) {
	body, _ := jsonutils.Parse([]byte(`{"input":[{"role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,x"}]}]}`))
	if !ResponsesInputHasImage(body.(*jsonutils.JSONDict)) {
		t.Fatal("expected image input")
	}
	body2, _ := jsonutils.Parse([]byte(`{"input":"hello"}`))
	if ResponsesInputHasImage(body2.(*jsonutils.JSONDict)) {
		t.Fatal("expected no image for string input")
	}
}

func TestResponsesStreamConverterReasoningAndToolOutputIndex(t *testing.T) {
	conv := NewResponsesStreamConverter("kimi-k2.6", nil)
	chunk1 := []byte(`{"id":"chatcmpl-1","model":"kimi-k2.6","choices":[{"delta":{"reasoning_content":"think"}}]}`)
	events1, err := conv.Feed(chunk1, false)
	if err != nil {
		t.Fatal(err)
	}
	chunk2 := []byte(`{"id":"chatcmpl-1","model":"kimi-k2.6","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"view_image","arguments":"{}"}}]}}]}`)
	events2, err := conv.Feed(chunk2, false)
	if err != nil {
		t.Fatal(err)
	}
	events := append(events1, events2...)
	var addedIndex, deltaIndex int
	for _, e := range events {
		if e.Event != "response.output_item.added" && e.Event != "response.function_call_arguments.delta" {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(e.Data, &payload); err != nil {
			t.Fatal(err)
		}
		idx, _ := payload["output_index"].(float64)
		if e.Event == "response.output_item.added" {
			item, _ := payload["item"].(map[string]interface{})
			if item["type"] == "function_call" {
				addedIndex = int(idx)
			}
		}
		if e.Event == "response.function_call_arguments.delta" {
			deltaIndex = int(idx)
		}
	}
	if addedIndex == 0 || deltaIndex == 0 {
		t.Fatalf("expected function_call indices, added=%d delta=%d events=%+v", addedIndex, deltaIndex, events)
	}
	if addedIndex != deltaIndex {
		t.Fatalf("output_index mismatch: added=%d delta=%d", addedIndex, deltaIndex)
	}
}

func TestResponsesStreamConverterText(t *testing.T) {
	conv := NewResponsesStreamConverter("gpt-test", nil)
	chunk := []byte(`{"id":"chatcmpl-1","model":"gpt-test","choices":[{"delta":{"content":"Hi"}}]}`)
	events, err := conv.Feed(chunk, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if events[0].Event != "response.created" {
		t.Fatalf("first event = %q", events[0].Event)
	}
	end, err := conv.Feed(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	foundCompleted := false
	foundOutputItemDone := false
	for _, e := range end {
		if e.Event == "response.completed" {
			foundCompleted = true
		}
		if e.Event == "response.output_item.done" {
			foundOutputItemDone = true
		}
	}
	if !foundCompleted {
		t.Fatalf("events = %+v", end)
	}
	if !foundOutputItemDone {
		t.Fatalf("expected response.output_item.done in %v", responsesEventNames(end))
	}
}

func TestResponsesStreamConverterReasoningAndToolDoneEvents(t *testing.T) {
	conv := NewResponsesStreamConverter("deepseek-v4-flash", nil)
	chunks := [][]byte{
		[]byte(`{"id":"chatcmpl-1","model":"deepseek-v4-flash","choices":[{"delta":{"role":"assistant","reasoning_content":"think"}}]}`),
		[]byte(`{"id":"chatcmpl-1","model":"deepseek-v4-flash","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"exec_command","arguments":"{\"cmd\":\"ls\"}"}}]}}]}`),
		[]byte(`{"id":"chatcmpl-1","model":"deepseek-v4-flash","choices":[{"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`),
	}
	var events []ResponsesStreamEvent
	for _, chunk := range chunks {
		evs, err := conv.Feed(chunk, false)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, evs...)
	}
	end, err := conv.Feed(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	events = append(events, end...)

	counts := map[string]int{}
	for _, e := range events {
		counts[e.Event]++
	}
	if counts["response.reasoning_summary_part.done"] != 1 {
		t.Fatalf("reasoning_summary_part.done = %d, events=%v", counts["response.reasoning_summary_part.done"], responsesEventNames(events))
	}
	if counts["response.function_call_arguments.done"] != 1 {
		t.Fatalf("function_call_arguments.done = %d, events=%v", counts["response.function_call_arguments.done"], responsesEventNames(events))
	}
	if counts["response.output_item.done"] < 2 {
		t.Fatalf("output_item.done = %d, want at least 2, events=%v", counts["response.output_item.done"], responsesEventNames(events))
	}
	if counts["response.completed"] != 1 {
		t.Fatalf("completed = %d, events=%v", counts["response.completed"], responsesEventNames(events))
	}

	// output_item.done for tools must appear before response.completed
	lastToolDone := -1
	completedAt := -1
	for i, e := range events {
		if e.Event == "response.output_item.done" {
			var payload map[string]interface{}
			if err := json.Unmarshal(e.Data, &payload); err != nil {
				t.Fatal(err)
			}
			item, _ := payload["item"].(map[string]interface{})
			if item["type"] == "function_call" {
				lastToolDone = i
			}
		}
		if e.Event == "response.completed" {
			completedAt = i
		}
	}
	if lastToolDone < 0 || completedAt < 0 || lastToolDone >= completedAt {
		t.Fatalf("tool output_item.done must precede completed: toolDone=%d completed=%d", lastToolDone, completedAt)
	}
}

func responsesEventNames(events []ResponsesStreamEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Event
	}
	return out
}

func TestNonStreamChatCompletionToStreamPayloads(t *testing.T) {
	raw := []byte(`{
		"id":"chatcmpl-visual-1",
		"model":"deepseek-v4-flash",
		"choices":[{
			"message":{
				"role":"assistant",
				"reasoning_content":"thinking",
				"content":"Hello from visual"
			},
			"finish_reason":"stop"
		}],
		"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
	}`)
	payloads, err := NonStreamChatCompletionToStreamPayloads(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(payloads) < 3 {
		t.Fatalf("payloads = %d, want at least 3", len(payloads))
	}

	conv := NewResponsesStreamConverter("deepseek-v4-flash", nil)
	var events []ResponsesStreamEvent
	for _, payload := range payloads {
		chunkEvents, err := conv.Feed(payload, false)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, chunkEvents...)
	}
	endEvents, err := conv.Feed(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	events = append(events, endEvents...)

	foundCreated := false
	foundText := false
	foundCompleted := false
	for _, e := range events {
		switch e.Event {
		case "response.created":
			foundCreated = true
		case "response.output_text.delta":
			foundText = true
		case "response.completed":
			foundCompleted = true
		}
	}
	if !foundCreated || !foundText || !foundCompleted {
		t.Fatalf("events missing lifecycle: created=%v text=%v completed=%v all=%+v", foundCreated, foundText, foundCompleted, events)
	}
}
