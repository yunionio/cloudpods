package models

import (
	"testing"
)

func TestNormalizeModelScopeOpenAPISearchResults(t *testing.T) {
	items := []modelScopeOpenAPIModel{
		{Id: "Qwen/Qwen3-0.6B", Downloads: 10, Likes: 2, Tasks: []string{"text-generation"}},
		{Id: "org/gguf-model", Tags: []string{"library:gguf"}},
	}
	out := normalizeModelScopeOpenAPISearchResults(items)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].ModelId != "Qwen/Qwen3-0.6B" || !out[0].Supported {
		t.Fatalf("first=%+v", out[0])
	}
	if out[1].Supported {
		t.Fatalf("gguf should be unsupported: %+v", out[1])
	}
}

func TestResolveModelCatalogSourceCustomURL(t *testing.T) {
	custom := "https://example.com/catalog.yaml"
	if got := resolveModelCatalogSource(custom); got != custom {
		t.Fatalf("got=%q want=%q", got, custom)
	}
}
