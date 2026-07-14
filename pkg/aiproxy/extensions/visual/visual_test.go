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

package visual

import (
	"strings"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestStripImagesFromChat(t *testing.T) {
	body := jsonutils.NewDict()
	msgs := jsonutils.NewArray()
	user := jsonutils.NewDict()
	user.Set("role", jsonutils.NewString("user"))
	parts := jsonutils.NewArray()
	text := jsonutils.NewDict()
	text.Set("type", jsonutils.NewString("text"))
	text.Set("text", jsonutils.NewString("describe"))
	parts.Add(text)
	img := jsonutils.NewDict()
	img.Set("type", jsonutils.NewString("image_url"))
	imgURL := jsonutils.NewDict()
	imgURL.Set("url", jsonutils.NewString("data:image/png;base64,abc"))
	img.Set("image_url", imgURL)
	parts.Add(img)
	user.Set("content", parts)
	msgs.Add(user)
	body.Set("messages", msgs)

	images, err := StripImagesFromChat(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("images = %d", len(images))
	}
	s := body.String()
	if strings.Contains(s, "image_url") {
		t.Fatalf("expected stripped image_url, got %s", s)
	}
	if !strings.Contains(s, "Image #1") {
		t.Fatalf("expected placeholder, got %s", s)
	}
}

func TestEnabled(t *testing.T) {
	cfg := &api.SAiModelConfig{
		Extensions: &api.SAiModelExtensions{
			Visual: &api.SAiModelVisualConfig{Enabled: true},
		},
	}
	if !Enabled(cfg) {
		t.Fatal("expected enabled")
	}
}

func TestInjectChatTools(t *testing.T) {
	body := jsonutils.NewDict()
	InjectChatTools(body)
	tools, err := body.Get("tools")
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := tools.(*jsonutils.JSONArray)
	if !ok || arr.Size() < 2 {
		t.Fatalf("tools = %v", tools)
	}
}

func TestNormalizeImagesUsesAttached(t *testing.T) {
	available := []ImageInput{{URL: "data:image/png;base64,abc"}}
	images := normalizeImages("", nil, nil, []string{"Image #1"}, available)
	if len(images) != 1 || images[0].URL == "" {
		t.Fatalf("images = %#v", images)
	}
}

func TestChatBaseURLStripsAnthropicSuffix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://api.deepseek.com/anthropic", "https://api.deepseek.com"},
		{"https://api.deepseek.com/anthropic/", "https://api.deepseek.com"},
		{"https://open.bigmodel.cn/api/anthropic", "https://open.bigmodel.cn"},
		{"https://api.moonshot.cn", "https://api.moonshot.cn"},
	}
	for _, c := range cases {
		if got := ChatBaseURL(c.in); got != c.want {
			t.Fatalf("ChatBaseURL(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestAnthropicMessagesHasImage(t *testing.T) {
	body, _ := jsonutils.Parse([]byte(`{"messages":[{"role":"user","content":[{"type":"image","source":{"type":"url","url":"https://x/a.png"}}]}]}`))
	if !AnthropicMessagesHasImage(body.(*jsonutils.JSONDict)) {
		t.Fatal("expected image")
	}
	body2, _ := jsonutils.Parse([]byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if AnthropicMessagesHasImage(body2.(*jsonutils.JSONDict)) {
		t.Fatal("expected no image")
	}
}

func TestShouldHandleResponsesIgnoresAPIMode(t *testing.T) {
	cfg := &api.SAiModelConfig{
		Extensions: &api.SAiModelExtensions{
			Visual: &api.SAiModelVisualConfig{Enabled: true},
		},
	}
	up := &models.ChatUpstream{
		ModelConfig: cfg,
		APIMode:     "anthropic",
	}
	body, _ := jsonutils.Parse([]byte(`{"input":[{"role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,x"}]}]}`))
	if !ShouldHandle(body.(*jsonutils.JSONDict), up, false) {
		t.Fatal("expected visual handle even with anthropic api_mode")
	}
	if ShouldHandle(body.(*jsonutils.JSONDict), up, true) {
		t.Fatal("stream should not use visual orchestrator")
	}
}
