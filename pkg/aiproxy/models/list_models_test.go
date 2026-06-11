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
	routing := &SAiRouting{ModelPattern: "dep-gpt-4o-mini"}
	if got := clientFacingModelID(routing, &SAiRoutingModel{ModelPattern: "fast"}, mdl); got != "fast" {
		t.Fatalf("expected alias fast, got %q", got)
	}
	if got := clientFacingModelID(routing, &SAiRoutingModel{ModelPattern: "gpt-*"}, mdl); got != "dep-gpt-4o-mini" {
		t.Fatalf("expected routing model_pattern for wildcard pattern, got %q", got)
	}
	if got := clientFacingModelID(routing, &SAiRoutingModel{}, mdl); got != "dep-gpt-4o-mini" {
		t.Fatalf("expected routing model_pattern, got %q", got)
	}
	if got := clientFacingModelID(nil, &SAiRoutingModel{}, mdl); got != "gpt-4o-mini" {
		t.Fatalf("expected catalog model_key, got %q", got)
	}
}

func TestUniqueNonEmptyStrings(t *testing.T) {
	out := uniqueNonEmptyStrings([]string{"a", "a", "", "b", "b"})
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("unexpected dedupe result: %#v", out)
	}
}

func TestPickRoutingForRequestModelKeyPriority(t *testing.T) {
	routings := []SAiRouting{
		{Priority: 10, ModelPattern: ""},
		{Priority: 100, ModelKey: "lzx-test-Qwen3-0.6B"},
	}
	picked, err := pickRoutingForRequest(routings, "lzx-test-Qwen3-0.6B", "primary")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.ModelKey != "lzx-test-Qwen3-0.6B" {
		t.Fatalf("expected model_key routing, got %#v", picked)
	}
}

func TestPickRoutingForRequestModelKeyBeforePattern(t *testing.T) {
	routings := []SAiRouting{
		{Priority: 10, ModelPattern: "lzx-test-Qwen3-0.6B"},
		{Priority: 100, ModelKey: "lzx-test-Qwen3-0.6B"},
	}
	picked, err := pickRoutingForRequest(routings, "lzx-test-Qwen3-0.6B", "primary")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.Priority != 100 {
		t.Fatalf("expected model_key routing with priority 100, got %#v", picked)
	}
}

func TestPickRoutingForRequestFallbackPattern(t *testing.T) {
	routings := []SAiRouting{
		{Priority: 20, ModelPattern: "qwen-*"},
	}
	picked, err := pickRoutingForRequest(routings, "qwen-turbo", "primary")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.ModelPattern != "qwen-*" {
		t.Fatalf("expected pattern routing, got %#v", picked)
	}
}

func TestModelKeyMatches(t *testing.T) {
	if !modelKeyMatches("Foo", "foo") {
		t.Fatal("expected case-insensitive match")
	}
	if modelKeyMatches("", "foo") {
		t.Fatal("empty key should not match")
	}
}
