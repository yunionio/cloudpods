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

package upstream

import "testing"

func TestModelsURL(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{"https://api.openai.com", "https://api.openai.com/v1/models"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/models"},
		{"https://generativelanguage.googleapis.com/v1beta", "https://generativelanguage.googleapis.com/v1beta/models"},
		{"https://api.deepseek.com/anthropic", "https://api.deepseek.com/anthropic/v1/models"},
		{"https://open.bigmodel.cn/api/paas/v4", "https://open.bigmodel.cn/api/paas/v4/models"},
	}
	for _, tc := range cases {
		if got := ModelsURL(tc.base); got != tc.want {
			t.Fatalf("ModelsURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}

func TestValidateModelsListBody(t *testing.T) {
	if err := validateModelsListBody([]byte(`{"object":"list","data":[]}`)); err != nil {
		t.Fatalf("expected valid data field: %v", err)
	}
	if err := validateModelsListBody([]byte(`{"models":[]}`)); err != nil {
		t.Fatalf("expected valid models field: %v", err)
	}
	if err := validateModelsListBody([]byte(`{"object":"list"}`)); err == nil {
		t.Fatal("expected error for missing data/models")
	}
}

func TestParseModelsListBodyOpenAI(t *testing.T) {
	keys, err := ParseModelsListBody([]byte(`{"object":"list","data":[{"id":"gpt-4o-mini"},{"id":"gpt-4o"}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(keys) != 2 || keys[0] != "gpt-4o" || keys[1] != "gpt-4o-mini" {
		t.Fatalf("keys: %#v", keys)
	}
}

func TestParseModelsListBodyGemini(t *testing.T) {
	keys, err := ParseModelsListBody([]byte(`{"models":[{"name":"models/gemini-2.0-flash"},{"name":"models/gemini-pro"}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(keys) != 2 || keys[0] != "gemini-2.0-flash" || keys[1] != "gemini-pro" {
		t.Fatalf("keys: %#v", keys)
	}
}
