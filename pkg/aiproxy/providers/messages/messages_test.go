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

package messages

import (
	"encoding/json"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

func TestPassthroughAdapterBuild(t *testing.T) {
	adapter, err := GetAdapter("anthropic", "")
	if err != nil {
		t.Fatal(err)
	}
	if !adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected passthrough stream")
	}
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("claude-sonnet-4-5"))
	body.Set("max_tokens", jsonutils.NewInt(100))
	user := jsonutils.NewDict()
	user.Set("role", jsonutils.NewString("user"))
	user.Set("content", jsonutils.NewString("hi"))
	body.Set("messages", jsonutils.NewArray(user))

	req, err := adapter.BuildUpstreamRequest(testChatCtx("anthropic", "https://api.anthropic.com", "sk-ant", "claude-sonnet-4-5"), body, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.anthropic.com/v1/messages" {
		t.Fatalf("url: %s", req.URL)
	}
	if req.Headers["x-api-key"] != "sk-ant" {
		t.Fatal("missing x-api-key")
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(req.Body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["model"] != "claude-sonnet-4-5" {
		t.Fatalf("model override: %#v", wire["model"])
	}
}

func TestTranslationAdapterBuildDeepSeek(t *testing.T) {
	adapter, err := GetAdapter("openai", "")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected translated stream")
	}
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("deepseek-chat"))
	body.Set("max_tokens", jsonutils.NewInt(256))
	user := jsonutils.NewDict()
	user.Set("role", jsonutils.NewString("user"))
	user.Set("content", jsonutils.NewString("hello"))
	body.Set("messages", jsonutils.NewArray(user))

	req, err := adapter.BuildUpstreamRequest(testChatCtx("openai", "https://api.deepseek.com", "ds-key", "deepseek-chat"), body, true)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.deepseek.com/v1/chat/completions" {
		t.Fatalf("url: %s", req.URL)
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(req.Body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["model"] != "deepseek-chat" {
		t.Fatalf("model: %#v", wire["model"])
	}
	if wire["stream"] != true {
		t.Fatalf("stream: %#v", wire["stream"])
	}
}

func TestGetAdapterDeepseekAnthropicPassthrough(t *testing.T) {
	adapter, err := GetAdapter("deepseek", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if !adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected passthrough stream for deepseek anthropic mode")
	}
}

func TestGetAdapterDeepseekOpenAITranslation(t *testing.T) {
	adapter, err := GetAdapter("deepseek", "openai")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected translated stream for deepseek openai mode")
	}
}

func TestGetAdapterCustomAnthropicPassthrough(t *testing.T) {
	adapter, err := GetAdapter("custom", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if !adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected passthrough stream for custom anthropic mode")
	}
}

func TestGetAdapterZhipuAnthropicPassthrough(t *testing.T) {
	adapter, err := GetAdapter("zhipu", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if !adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected passthrough stream for zhipu anthropic mode")
	}
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("glm-5.2"))
	body.Set("max_tokens", jsonutils.NewInt(100))
	user := jsonutils.NewDict()
	user.Set("role", jsonutils.NewString("user"))
	user.Set("content", jsonutils.NewString("hi"))
	body.Set("messages", jsonutils.NewArray(user))

	req, err := adapter.BuildUpstreamRequest(testChatCtx("zhipu", "https://open.bigmodel.cn/api/anthropic", "zhipu-key", "glm-5.2"), body, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://open.bigmodel.cn/api/anthropic/v1/messages"
	if req.URL != want {
		t.Fatalf("url: %s, want %s", req.URL, want)
	}
}

func TestGetAdapterBlocksGemini(t *testing.T) {
	if _, err := GetAdapter("gemini", ""); err == nil {
		t.Fatal("expected gemini to be unsupported")
	}
}

func TestGetAdapterVLLMTranslation(t *testing.T) {
	adapter, err := GetAdapter("vllm", "")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.AnthropicStreamPassthrough() {
		t.Fatal("expected translated stream for vllm")
	}
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("t-vllm"))
	body.Set("max_tokens", jsonutils.NewInt(256))
	user := jsonutils.NewDict()
	user.Set("role", jsonutils.NewString("user"))
	user.Set("content", jsonutils.NewString("hello"))
	body.Set("messages", jsonutils.NewArray(user))

	req, err := adapter.BuildUpstreamRequest(testChatCtx("vllm", "http://127.0.0.1:8000/v1", "sk-test", "t-vllm"), body, true)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "http://127.0.0.1:8000/v1/chat/completions" {
		t.Fatalf("url: %s", req.URL)
	}
}

func testChatCtx(providerKey, baseURL, apiKey, model string) *providerapi.ChatContext {
	return &providerapi.ChatContext{
		ProviderKey:   providerKey,
		BaseURL:       baseURL,
		APIKey:        apiKey,
		UpstreamModel: model,
	}
}
