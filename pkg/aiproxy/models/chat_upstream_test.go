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

import (
	"strings"
	"testing"
)

func TestParseClientModelRef(t *testing.T) {
	cases := []struct {
		in           string
		routeKey     string
		catalogPart  string
		hierarchical bool
	}{
		{"flat-alias", "", "flat-alias", false},
		{"my-route/gpt-4", "my-route", "gpt-4", true},
		{"my-route/claude-sonnet-4-6", "my-route", "claude-sonnet-4-6", true},
		{"/gpt-4", "", "/gpt-4", false},
		{"my-route/", "my-route", "", true},
	}
	for _, tc := range cases {
		ref := parseClientModelRef(tc.in)
		if ref.routeKey != tc.routeKey || ref.catalogPart != tc.catalogPart || ref.hierarchical != tc.hierarchical {
			t.Fatalf("parseClientModelRef(%q) = %#v, want routeKey=%q catalogPart=%q hierarchical=%v",
				tc.in, ref, tc.routeKey, tc.catalogPart, tc.hierarchical)
		}
	}
}

func TestNormalizeAiRoutingModelKeyRejectsSlash(t *testing.T) {
	_, err := normalizeAiRoutingModelKey("my/route")
	if err == nil {
		t.Fatal("expected error for model_key containing slash")
	}
	if _, err := normalizeAiRoutingModelKey("my-route"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestPickRoutingForRequestHierarchical(t *testing.T) {
	routings := []SAiRouting{
		{Priority: 10, ModelKey: "claude", ModelPattern: "claude-*"},
		{Priority: 20, ModelKey: "other"},
	}
	picked, err := pickRoutingForRequest(routings, "claude/claude-sonnet-4-6", "primary")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.ModelKey != "claude" {
		t.Fatalf("expected claude routing, got %#v", picked)
	}
}

func TestPickRoutingForRequestHierarchicalSkipsPattern(t *testing.T) {
	routings := []SAiRouting{
		{Priority: 10, ModelPattern: "claude-*"},
	}
	picked, err := pickRoutingForRequest(routings, "claude/claude-sonnet-4-6", "primary")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked != nil {
		t.Fatalf("hierarchical request should not match pattern-only routing, got %#v", picked)
	}
}

func TestCatalogRequestPart(t *testing.T) {
	ref := parseClientModelRef("claude/claude-sonnet-4-6")
	if got := catalogRequestPart(ref, false); got != "claude-sonnet-4-6" {
		t.Fatalf("expected catalog part, got %q", got)
	}
	ref = parseClientModelRef("claude")
	if got := catalogRequestPart(ref, true); got != "" {
		t.Fatalf("flat model_key match should use empty catalog part, got %q", got)
	}
	if got := catalogRequestPart(ref, false); got != "claude" {
		t.Fatalf("flat pattern match should use raw model, got %q", got)
	}
}

func TestRouterRequestModel(t *testing.T) {
	ref := parseClientModelRef("claude/claude-sonnet-4-6")
	if got := routerRequestModel(ref); got != "claude-sonnet-4-6" {
		t.Fatalf("expected catalog part for router, got %q", got)
	}
	ref = parseClientModelRef("claude")
	if got := routerRequestModel(ref); got != "claude" {
		t.Fatalf("expected raw model for flat router request, got %q", got)
	}
}

func TestPickRoutingForRequestBoundElsewhereHierarchical(t *testing.T) {
	routings := []SAiRouting{
		{Priority: 10, ModelKey: "claude", AiProxyNodeId: "node-b"},
	}
	_, err := pickRoutingForRequest(routings, "claude/gpt-4", "node-a")
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if !strings.Contains(err.Error(), "bound to ai_proxy_node") {
		t.Fatalf("expected node binding error, got %v", err)
	}
}
