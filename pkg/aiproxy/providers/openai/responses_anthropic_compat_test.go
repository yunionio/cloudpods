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
)

func TestResponsesToAnthropicMessagesBasic(t *testing.T) {
	body, err := jsonutils.Parse([]byte(`{
		"model":"claude-test",
		"max_output_tokens":100,
		"instructions":"be helpful",
		"input":"hello"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := ResponsesToAnthropicMessages(body.(*jsonutils.JSONDict), "claude-up")
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := out.GetString("model"); m != "claude-up" {
		t.Fatalf("model=%q", m)
	}
	if sys, _ := out.GetString("system"); sys != "be helpful" {
		t.Fatalf("system=%q", sys)
	}
}

func TestResponsesToAnthropicMessagesDefaultMaxOutputTokens(t *testing.T) {
	body, err := jsonutils.Parse([]byte(`{"model":"claude-test","input":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := ResponsesToAnthropicMessages(body.(*jsonutils.JSONDict), "claude-up")
	if err != nil {
		t.Fatal(err)
	}
	mt, err := out.Int("max_tokens")
	if err != nil || mt != DefaultResponsesMaxOutputTokens {
		t.Fatalf("max_tokens = %d want %d", mt, DefaultResponsesMaxOutputTokens)
	}
}

func TestResponsesToAnthropicMessagesToolChoiceAuto(t *testing.T) {
	body, err := jsonutils.Parse([]byte(`{
		"model":"claude-test",
		"max_output_tokens":100,
		"input":"hello",
		"tool_choice":"auto"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := ResponsesToAnthropicMessages(body.(*jsonutils.JSONDict), "claude-up")
	if err != nil {
		t.Fatal(err)
	}
	tc, err := out.Get("tool_choice")
	if err != nil {
		t.Fatal(err)
	}
	if typ, _ := tc.GetString("type"); typ != "auto" {
		t.Fatalf("tool_choice.type = %q, want auto; raw=%s", typ, tc.String())
	}
}

func TestAnthropicMessagesToResponsesBasic(t *testing.T) {
	raw := []byte(`{
		"id":"msg_1",
		"model":"claude-test",
		"content":[{"type":"text","text":"hello"}],
		"stop_reason":"end_turn",
		"usage":{"input_tokens":5,"output_tokens":2}
	}`)
	out, err := AnthropicMessagesToResponses(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"object":"response"`) && !strings.Contains(string(out), `"object": "response"`) {
		t.Fatalf("out=%s", out)
	}
}
