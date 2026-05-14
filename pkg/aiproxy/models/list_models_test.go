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

package models

import "testing"

func TestClientFacingModelID(t *testing.T) {
	mdl := &SAiModel{ModelKey: "gpt-4o-mini"}
	if got := clientFacingModelID(&SAiRoutingModel{ModelPattern: "fast"}, mdl); got != "fast" {
		t.Fatalf("expected alias fast, got %q", got)
	}
	if got := clientFacingModelID(&SAiRoutingModel{ModelPattern: "gpt-*"}, mdl); got != "gpt-4o-mini" {
		t.Fatalf("expected catalog model_key for wildcard pattern, got %q", got)
	}
	if got := clientFacingModelID(&SAiRoutingModel{}, mdl); got != "gpt-4o-mini" {
		t.Fatalf("expected catalog model_key, got %q", got)
	}
}

func TestUniqueNonEmptyStrings(t *testing.T) {
	out := uniqueNonEmptyStrings([]string{"a", "a", "", "b", "b"})
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("unexpected dedupe result: %#v", out)
	}
}
