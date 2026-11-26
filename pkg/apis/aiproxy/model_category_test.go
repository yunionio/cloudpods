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

package aiproxy

import "testing"

func TestValidateModelCategory(t *testing.T) {
	for _, ok := range []string{"", "llm", "Embedding", " reranker ", "image"} {
		if err := ValidateModelCategory(ok); err != nil {
			t.Errorf("ValidateModelCategory(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"embed", "chat", "unknown"} {
		if err := ValidateModelCategory(bad); err == nil {
			t.Errorf("ValidateModelCategory(%q) = nil, want error", bad)
		}
	}
}

func TestModelCategoryHelpers(t *testing.T) {
	if !IsChatModelCategory("") || !IsChatModelCategory("llm") {
		t.Error("empty/llm must be treated as chat")
	}
	if IsChatModelCategory(ModelCategoryEmbedding) || IsChatModelCategory(ModelCategoryReranker) {
		t.Error("embedding/reranker must not be treated as chat")
	}
	if !IsEmbeddingModelCategory(" EMBEDDING ") {
		t.Error("embedding detection must be case/space insensitive")
	}
	if !IsRerankModelCategory("reranker") {
		t.Error("reranker detection failed")
	}
	if got := EndpointPathForCategory(ModelCategoryEmbedding); got != "/v1/embeddings" {
		t.Errorf("embedding endpoint = %q", got)
	}
	if got := EndpointPathForCategory(ModelCategoryReranker); got != "/v1/rerank" {
		t.Errorf("reranker endpoint = %q", got)
	}
	if got := EndpointPathForCategory(ModelCategoryLLM); got != "" {
		t.Errorf("llm endpoint = %q, want empty", got)
	}
}

func TestSAiModelConfigCategory(t *testing.T) {
	// A config that only sets category must not be treated as zero, otherwise
	// the column would be dropped on write.
	cfg := &SAiModelConfig{Category: ModelCategoryEmbedding}
	if cfg.IsZero() {
		t.Fatal("config with only category reported as zero")
	}
	if got := cfg.NormalizedCategory(); got != ModelCategoryEmbedding {
		t.Fatalf("NormalizedCategory = %q", got)
	}
	if got := (&SAiModelConfig{}).CategoryOrLLM(); got != ModelCategoryLLM {
		t.Fatalf("CategoryOrLLM fallback = %q", got)
	}

	// Embedding/reranker models default to text-only input modalities.
	if mods := cfg.InputModalitiesForCatalog(); len(mods) != 1 || mods[0] != "text" {
		t.Fatalf("InputModalitiesForCatalog = %v, want [text]", mods)
	}
	// Explicit modalities still win.
	explicit := &SAiModelConfig{
		Category:        ModelCategoryEmbedding,
		InputModalities: []string{"text", "image"},
	}
	if mods := explicit.InputModalitiesForCatalog(); len(mods) != 2 {
		t.Fatalf("explicit modalities overwritten: %v", mods)
	}
	// Legacy chat rows keep returning nil.
	if mods := (&SAiModelConfig{Category: ModelCategoryLLM}).InputModalitiesForCatalog(); mods != nil {
		t.Fatalf("llm modalities = %v, want nil", mods)
	}
}
