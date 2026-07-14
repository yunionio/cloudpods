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

package codexconfig

import "testing"

func TestBuildCatalogVisualModalities(t *testing.T) {
	catalog := BuildCatalogFromIDs([]ModelListEntry{{
		ID:              "deepseek-v4-pro",
		OwnedBy:         "deepseek",
		InputModalities: []string{"text", "image"},
	}})
	if len(catalog) != 1 {
		t.Fatalf("catalog len = %d", len(catalog))
	}
	if len(catalog[0].InputModalities) != 2 || catalog[0].InputModalities[1] != "image" {
		t.Fatalf("modalities = %#v", catalog[0].InputModalities)
	}
}

func TestApplyVisualModalities(t *testing.T) {
	catalog := BuildCatalogFromIDs([]ModelListEntry{
		{ID: "text-only", OwnedBy: "deepseek"},
		{ID: "route/flash", OwnedBy: "deepseek", InputModalities: []string{"text"}},
		{ID: "already-image", OwnedBy: "deepseek", InputModalities: []string{"text", "image"}},
	})
	visual := map[string]struct{}{
		"route/flash":   {},
		"already-image": {},
		"missing-slug":  {},
	}
	out := ApplyVisualModalities(catalog, visual)
	if len(out) != 3 {
		t.Fatalf("len = %d", len(out))
	}
	bySlug := map[string]ModelInfo{}
	for _, m := range out {
		bySlug[m.Slug] = m
	}
	if len(bySlug["text-only"].InputModalities) != 1 || bySlug["text-only"].InputModalities[0] != "text" {
		t.Fatalf("text-only = %#v", bySlug["text-only"].InputModalities)
	}
	if !bySlug["route/flash"].SupportsImageDetailOriginal {
		t.Fatal("route/flash should support image detail original")
	}
	mods := bySlug["route/flash"].InputModalities
	if len(mods) != 2 || mods[0] != "text" || mods[1] != "image" {
		t.Fatalf("route/flash modalities = %#v", mods)
	}
	mods = bySlug["already-image"].InputModalities
	if len(mods) != 2 || mods[1] != "image" {
		t.Fatalf("already-image modalities = %#v", mods)
	}
}

func TestEnsureImageModality(t *testing.T) {
	if got := ensureImageModality(nil); len(got) != 2 || got[0] != "text" || got[1] != "image" {
		t.Fatalf("nil = %#v", got)
	}
	if got := ensureImageModality([]string{"text", "image"}); len(got) != 2 {
		t.Fatalf("already has image = %#v", got)
	}
	if got := ensureImageModality([]string{"audio"}); len(got) != 3 || got[0] != "text" || got[2] != "image" {
		t.Fatalf("preserve audio = %#v", got)
	}
}
