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

package providers

import (
	"encoding/json"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestAliyunProviderEnableThinkingPatch(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("qwen-turbo"), "model")
	body.Add(jsonutils.NewArray(jsonutils.NewDict()), "messages")

	p := Get("aliyun")
	req, err := p.BuildUpstreamRequest(&ChatContext{
		ProviderKey:   "aliyun",
		BaseURL:       "https://dashscope.aliyuncs.com/compatible-mode",
		APIKey:        "sk-test",
		UpstreamModel: "qwen-turbo",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(req.Body, &wire); err != nil {
		t.Fatal(err)
	}
	if v, ok := wire["enable_thinking"].(bool); !ok || v {
		t.Fatalf("expected enable_thinking=false, got %#v", wire["enable_thinking"])
	}
}

func TestAnthropicProviderBuildRequest(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("claude-3-5-sonnet"), "model")
	body.Add(jsonutils.NewInt(1024), "max_tokens")
	sysMsg := jsonutils.NewDict()
	sysMsg.Set("role", jsonutils.NewString("system"))
	sysMsg.Set("content", jsonutils.NewString("You are helpful."))
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("Hi"))
	msgs := jsonutils.NewArray(sysMsg, userMsg)
	body.Add(msgs, "messages")

	p := Get("anthropic")
	req, err := p.BuildUpstreamRequest(&ChatContext{
		ProviderKey:   "anthropic",
		BaseURL:       "https://api.anthropic.com",
		APIKey:        "sk-ant",
		UpstreamModel: "claude-3-5-sonnet-20241022",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.anthropic.com/v1/messages" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
	if req.Headers["x-api-key"] != "sk-ant" {
		t.Fatalf("missing x-api-key header")
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(req.Body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["system"] != "You are helpful." {
		t.Fatalf("expected system prompt, got %#v", wire["system"])
	}
}

func TestAnthropicNormalizeResponse(t *testing.T) {
	p := Get("anthropic")
	raw := []byte(`{
		"id":"msg_1",
		"model":"claude-3-5-sonnet-20241022",
		"content":[{"type":"text","text":"Hello"}],
		"stop_reason":"end_turn",
		"usage":{"input_tokens":3,"output_tokens":1}
	}`)
	out, err := p.NormalizeResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["object"] != "chat.completion" {
		t.Fatalf("unexpected object: %#v", resp["object"])
	}
	choices := resp["choices"].([]interface{})
	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if msg["content"] != "Hello" {
		t.Fatalf("unexpected content: %#v", msg["content"])
	}
}

func TestAnthropicToolCalling(t *testing.T) {
	body := jsonutils.NewDict()
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("Weather in Boston?"))
	msgs := jsonutils.NewArray(userMsg)
	body.Add(msgs, "messages")
	tool := jsonutils.NewDict()
	tool.Set("type", jsonutils.NewString("function"))
	fn := jsonutils.NewDict()
	fn.Set("name", jsonutils.NewString("get_weather"))
	fn.Set("parameters", jsonutils.NewDict())
	tool.Set("function", fn)
	body.Add(jsonutils.NewArray(tool), "tools")

	p := Get("anthropic")
	req, err := p.BuildUpstreamRequest(&ChatContext{
		BaseURL:       "https://api.anthropic.com",
		APIKey:        "sk-ant",
		UpstreamModel: "claude-3-5-sonnet-20241022",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(req.Body, &wire); err != nil {
		t.Fatal(err)
	}
	tools, ok := wire["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("expected tools in request, got %#v", wire["tools"])
	}

	raw := []byte(`{
		"id":"msg_2",
		"model":"claude-3-5-sonnet-20241022",
		"content":[{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Boston"}}],
		"stop_reason":"tool_use",
		"usage":{"input_tokens":10,"output_tokens":5}
	}`)
	out, err := p.NormalizeResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	choices := resp["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	if choice["finish_reason"] != "tool_calls" {
		t.Fatalf("expected finish_reason tool_calls, got %#v", choice["finish_reason"])
	}
	msg := choice["message"].(map[string]interface{})
	if msg["tool_calls"] == nil {
		t.Fatal("expected tool_calls in normalized response")
	}
}

func TestOpenAICompatPassthrough(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("gpt-4o"), "model")
	body.Add(jsonutils.NewArray(jsonutils.NewDict()), "messages")

	p := Get("openai")
	req, err := p.BuildUpstreamRequest(&ChatContext{
		BaseURL:       "https://api.openai.com",
		APIKey:        "sk-test",
		UpstreamModel: "gpt-4o-mini",
	}, body, true)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.openai.com/v1/chat/completions" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestRegistryFallback(t *testing.T) {
	p := Get("unknown-provider-key")
	if !p.OpenAIStreamPassthrough() {
		t.Fatal("unknown provider should use openai-compatible passthrough")
	}
}
