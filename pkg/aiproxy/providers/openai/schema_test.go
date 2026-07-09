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

import "testing"

func TestChatCompletionsURLMoonshot(t *testing.T) {
	got := ChatCompletionsURL("https://api.moonshot.cn")
	want := "https://api.moonshot.cn/v1/chat/completions"
	if got != want {
		t.Fatalf("ChatCompletionsURL() = %q, want %q", got, want)
	}
}
