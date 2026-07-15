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

package chatlog

import "testing"

func TestFillUsageFromJSONOpenAI(t *testing.T) {
	rec := &Record{}
	ok := FillUsageFromJSON(rec, []byte(`{"usage":{"prompt_tokens":11,"completion_tokens":22,"total_tokens":33}}`))
	if !ok || rec.UsageMissing {
		t.Fatalf("expected openai usage ok, got ok=%v missing=%v", ok, rec.UsageMissing)
	}
	if rec.PromptTokens != 11 || rec.CompletionTokens != 22 || rec.TotalTokens != 33 {
		t.Fatalf("unexpected tokens: %+v", rec)
	}
}

func TestFillUsageFromJSONAnthropicAliases(t *testing.T) {
	rec := &Record{}
	ok := FillUsageFromJSON(rec, []byte(`{"usage":{"input_tokens":7,"output_tokens":9}}`))
	if !ok || rec.UsageMissing {
		t.Fatalf("expected anthropic usage ok, got ok=%v missing=%v", ok, rec.UsageMissing)
	}
	if rec.PromptTokens != 7 || rec.CompletionTokens != 9 || rec.TotalTokens != 16 {
		t.Fatalf("unexpected tokens: prompt=%d completion=%d total=%d", rec.PromptTokens, rec.CompletionTokens, rec.TotalTokens)
	}
}

func TestFillUsageFromJSONMissing(t *testing.T) {
	rec := &Record{}
	ok := FillUsageFromJSON(rec, []byte(`{"id":"resp"}`))
	if ok || !rec.UsageMissing {
		t.Fatalf("expected missing usage, got ok=%v missing=%v", ok, rec.UsageMissing)
	}
}
