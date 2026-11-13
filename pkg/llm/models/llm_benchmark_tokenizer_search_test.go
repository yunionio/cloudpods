package models

import (
	"context"
	"testing"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestFilterTokenizerSearchResults(t *testing.T) {
	got := filterTokenizerSearchResults([]api.InstantModelModelScopeSearchResult{
		{ModelId: "Qwen/Qwen3-8B", Name: " Qwen3-8B ", Downloads: 12, Supported: true},
		{ModelId: "private/model", Supported: false, UnsupportedReason: "private"},
		{ModelId: "  ", Name: "blank", Supported: true},
		{ModelId: "org/gguf", Supported: false},
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ModelId != "Qwen/Qwen3-8B" || got[0].Name != "Qwen3-8B" || got[0].Downloads != 12 {
		t.Fatalf("got %+v", got[0])
	}
}

func TestFilterTokenizerSearchResultsEmpty(t *testing.T) {
	got := filterTokenizerSearchResults(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v, want empty slice", got)
	}
}

func TestNormalizeTokenizerSearchInput(t *testing.T) {
	if _, err := normalizeTokenizerSearchInput(api.LLMBenchmarkTokenizerSearchInput{Q: "  "}); err == nil {
		t.Fatal("expected missing q")
	}
	got, err := normalizeTokenizerSearchInput(api.LLMBenchmarkTokenizerSearchInput{Q: " Qwen3 ", PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if got.Q != "Qwen3" || got.PageSize != tokenizerSearchPageSize {
		t.Fatalf("got %+v", got)
	}
	got, err = normalizeTokenizerSearchInput(api.LLMBenchmarkTokenizerSearchInput{Q: "Qwen3", PageSize: 5})
	if err != nil {
		t.Fatal(err)
	}
	if got.PageSize != 5 {
		t.Fatalf("page size = %d", got.PageSize)
	}
}

func TestGetPropertyTokenizerSearchMissingQuery(t *testing.T) {
	man := &SLLMBenchmarkManager{}
	_, err := man.GetPropertyTokenizerSearch(context.Background(), nil, jsonutils.NewDict())
	if err == nil {
		t.Fatal("expected missing q")
	}
}
