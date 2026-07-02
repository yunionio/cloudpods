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

package ft

import (
	"strings"
	"testing"
)

func TestCatalogModelID(t *testing.T) {
	if got := CatalogModelID("aliyun", "qwen-turbo"); got != "aliyun-qwen-turbo" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultModelForProvider(t *testing.T) {
	cases := map[string]string{
		"aliyun":    "qwen-turbo",
		"xiaomi":    "mimo-v2-flash",
		"anthropic": "claude-sonnet-4-5",
		"unknown":   "",
	}
	for provider, want := range cases {
		if got := DefaultModelForProvider(provider); got != want {
			t.Fatalf("%s: got %q want %q", provider, got, want)
		}
	}
}

func TestParseOpenAIStreamDelta(t *testing.T) {
	payload := `{"choices":[{"delta":{"content":"hello"}}]}`
	delta, err := parseOpenAIStreamDelta(payload)
	if err != nil || delta != "hello" {
		t.Fatalf("delta=%q err=%v", delta, err)
	}
	_, err = parseOpenAIStreamDelta(`{"error":{"message":"fail"}}`)
	if err == nil {
		t.Fatal("expected error event")
	}
}

func TestParseAnthropicStreamDelta(t *testing.T) {
	payload := `{"delta":{"text":"hi"}}`
	delta, err := parseAnthropicStreamDelta(payload)
	if err != nil || delta != "hi" {
		t.Fatalf("delta=%q err=%v", delta, err)
	}
}

func TestExtractOpenAIChatContent(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"answer"}}]}`)
	content, err := extractOpenAIChatContent(body)
	if err != nil || content != "answer" {
		t.Fatalf("content=%q err=%v", content, err)
	}
}

func TestExtractAnthropicTextContent(t *testing.T) {
	body := []byte(`{"content":[{"type":"text","text":"hello"}]}`)
	content, err := extractAnthropicTextContent(body)
	if err != nil || content != "hello" {
		t.Fatalf("content=%q err=%v", content, err)
	}
}

func TestAggregateSSEStreamOpenAI(t *testing.T) {
	input := strings.Join([]string{
		"data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}",
		"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}",
		"data: [DONE]",
	}, "\n")
	out, err := aggregateSSEStream(strings.NewReader(input), parseOpenAIStreamDelta)
	if err != nil || out != "Hello" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestOpenAIChatURL(t *testing.T) {
	got := openAIChatURL("https://aiproxy.example.com/")
	if got != "https://aiproxy.example.com/ai/openai/v1/chat/completions" {
		t.Fatalf("got %q", got)
	}
}

func TestAnthropicMessagesURL(t *testing.T) {
	got := anthropicMessagesURL("https://aiproxy.example.com")
	if got != "https://aiproxy.example.com/ai/anthropic/v1/messages" {
		t.Fatalf("got %q", got)
	}
}

func TestResourceTrackerHasCreated(t *testing.T) {
	t.Parallel()
	tr := NewResourceTracker(false)
	if tr.hasCreated() {
		t.Fatal("expected empty tracker")
	}
	tr.createdAiKey = "k1"
	if !tr.hasCreated() {
		t.Fatal("expected created")
	}
}

func TestDefaultAdminNames(t *testing.T) {
	names := DefaultAdminNames("aliyun", "")
	if names.KeyName != "aiproxy-test-aliyun" || names.VkName != "aiproxy-test-aliyun-vk" {
		t.Fatalf("%+v", names)
	}
}
