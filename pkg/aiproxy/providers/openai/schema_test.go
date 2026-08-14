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
	"strings"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestChatCompletionsURLMoonshot(t *testing.T) {
	got := ChatCompletionsURL("https://api.moonshot.cn")
	want := "https://api.moonshot.cn/v1/chat/completions"
	if got != want {
		t.Fatalf("ChatCompletionsURL() = %q, want %q", got, want)
	}
}

func TestChatCompletionsURLZhipu(t *testing.T) {
	got := ChatCompletionsURL("https://open.bigmodel.cn/api/paas/v4")
	want := "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	if got != want {
		t.Fatalf("ChatCompletionsURL() = %q, want %q", got, want)
	}
}

func TestEnsureStreamIncludeUsage(t *testing.T) {
	body := jsonutils.NewDict()
	EnsureStreamIncludeUsage(body, false)
	if _, err := body.Get("stream_options"); err == nil {
		t.Fatal("non-stream request should not add stream_options")
	}

	EnsureStreamIncludeUsage(body, true)
	include, err := body.Bool("stream_options", "include_usage")
	if err != nil || !include {
		t.Fatalf("stream request should set include_usage, got err=%v include=%v body=%s", err, include, body)
	}

	body.Set("stream_options", jsonutils.Marshal(map[string]interface{}{
		"include_obfuscation": true,
	}))
	EnsureStreamIncludeUsage(body, true)
	include, err = body.Bool("stream_options", "include_usage")
	if err != nil || !include {
		t.Fatalf("should merge include_usage into existing stream_options: %s", body)
	}
	obfuscation, err := body.Bool("stream_options", "include_obfuscation")
	if err != nil || !obfuscation {
		t.Fatalf("should keep existing stream_options fields: %s", body)
	}
}

func TestMoonshotCompatStreamIncludeUsage(t *testing.T) {
	p := NewCompat(api.ProviderKeyMoonshot)
	body := jsonutils.NewDict()
	msg := jsonutils.NewDict()
	msg.Set("role", jsonutils.NewString("user"))
	msg.Set("content", jsonutils.NewString("hi"))
	body.Set("messages", jsonutils.NewArray(msg))
	ctx := &providerapi.ChatContext{
		BaseURL:       "https://api.moonshot.cn",
		UpstreamModel: "kimi-k2.5",
		APIKey:        "sk-test",
	}

	req, err := p.BuildUpstreamRequest(ctx, body, true)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jsonutils.Parse(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	include, err := parsed.Bool("stream_options", "include_usage")
	if err != nil || !include {
		t.Fatalf("moonshot stream body should include usage: %s", req.Body)
	}

	req, err = p.BuildUpstreamRequest(ctx, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(req.Body), "stream_options") {
		t.Fatalf("non-stream moonshot body should not add stream_options: %s", req.Body)
	}
}

func TestCompletionsCompatStreamIncludeUsage(t *testing.T) {
	p := NewCompletionsCompat()
	body := jsonutils.NewDict()
	body.Set("prompt", jsonutils.NewString("hello"))
	ctx := &providerapi.ChatContext{
		BaseURL:       "https://api.moonshot.cn",
		UpstreamModel: "kimi-k2.5",
		APIKey:        "sk-test",
	}
	req, err := p.BuildCompletionsRequest(ctx, body, true)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jsonutils.Parse(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	include, err := parsed.Bool("stream_options", "include_usage")
	if err != nil || !include {
		t.Fatalf("completions stream body should include usage: %s", req.Body)
	}
}
