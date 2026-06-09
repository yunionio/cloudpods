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

package baidu

import (
	"encoding/json"
	"strings"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

func TestQianfanV2ChatBuild(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("ernie-3.5-8k"), "model")
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("你好"))
	body.Add(jsonutils.NewArray(userMsg), "messages")

	p := New()
	req, err := p.BuildUpstreamRequest(&providerapi.ChatContext{
		BaseURL:       "https://qianfan.baidubce.com/v2",
		APIKey:        "bce-v3/ALTAK-test",
		UpstreamModel: "ernie-3.5-8k",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://qianfan.baidubce.com/v2/chat/completions" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
	if req.Headers["Authorization"] != "Bearer bce-v3/ALTAK-test" {
		t.Fatalf("missing bearer auth: %#v", req.Headers)
	}
}

func TestWenxinV1ChatBuild(t *testing.T) {
	body := jsonutils.NewDict()
	userMsg := jsonutils.NewDict()
	userMsg.Set("role", jsonutils.NewString("user"))
	userMsg.Set("content", jsonutils.NewString("你好"))
	body.Add(jsonutils.NewArray(userMsg), "messages")

	p := New()
	req, err := p.BuildUpstreamRequest(&providerapi.ChatContext{
		BaseURL:       "https://aip.baidubce.com",
		APIKey:        "test-access-token",
		UpstreamModel: "eb-instant",
	}, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(req.URL, "wenxinworkshop/chat/eb-instant") {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestWenxinV1NormalizeResponse(t *testing.T) {
	p := New()
	out, err := p.NormalizeResponse([]byte(`{
		"id":"as-1",
		"result":"你好",
		"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}
	}`))
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
}

func TestStreamPassthroughV2(t *testing.T) {
	p := New().(providerapi.ContextualStreamPassthrough)
	if !p.OpenAIStreamPassthroughForContext(&providerapi.ChatContext{
		BaseURL: "https://qianfan.baidubce.com/v2",
	}) {
		t.Fatal("qianfan v2 should passthrough SSE")
	}
	if p.OpenAIStreamPassthroughForContext(&providerapi.ChatContext{
		BaseURL: "https://aip.baidubce.com",
	}) {
		t.Fatal("wenxin v1 should not passthrough SSE")
	}
}

func TestUseQianfanV2(t *testing.T) {
	if !useQianfanV2("") {
		t.Fatal("empty base should default to v2")
	}
	if !useQianfanV2("https://qianfan.baidubce.com/v2") {
		t.Fatal("qianfan host should be v2")
	}
	if useQianfanV2("https://aip.baidubce.com") {
		t.Fatal("aip host should be v1")
	}
}
