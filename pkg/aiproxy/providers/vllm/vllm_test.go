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

package vllm

import (
	"strings"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

func TestVLLMProviderBuildCompletionsRequest(t *testing.T) {
	p := New()
	cp, ok := p.(providerapi.CompletionsProvider)
	if !ok {
		t.Fatal("vllm provider should implement CompletionsProvider")
	}
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("hello"), "prompt")
	streamOpts := jsonutils.NewDict()
	streamOpts.Add(jsonutils.JSONTrue, "include_usage")
	body.Add(streamOpts, "stream_options")

	req, err := cp.BuildCompletionsRequest(&providerapi.ChatContext{
		BaseURL:       "http://127.0.0.1:8000",
		UpstreamModel: "Qwen/Qwen2.5-7B-Instruct",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "http://127.0.0.1:8000/v1/completions" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
	if strings.Contains(string(req.Body), "stream_options") {
		t.Fatalf("stream_options should be stripped for non-stream requests: %s", req.Body)
	}
}

func TestVLLMProviderBuildChatCompletionsRequest(t *testing.T) {
	p := New()
	body := jsonutils.NewDict()
	msg := jsonutils.NewDict()
	msg.Add(jsonutils.NewString("user"), "role")
	msg.Add(jsonutils.NewString("hi"), "content")
	body.Add(jsonutils.NewArray(msg), "messages")
	streamOpts := jsonutils.NewDict()
	streamOpts.Add(jsonutils.JSONTrue, "include_usage")
	body.Add(streamOpts, "stream_options")

	req, err := p.BuildUpstreamRequest(&providerapi.ChatContext{
		BaseURL:       "http://127.0.0.1:8000",
		UpstreamModel: "Qwen/Qwen2.5-7B-Instruct",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "http://127.0.0.1:8000/v1/chat/completions" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
	if strings.Contains(string(req.Body), "stream_options") {
		t.Fatalf("stream_options should be stripped for non-stream requests: %s", req.Body)
	}
}
