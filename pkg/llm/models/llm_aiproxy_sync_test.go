package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestMapLLMTypeToProviderKey(t *testing.T) {
	cases := []struct {
		in     string
		key    string
		ok     bool
	}{
		{string(api.LLM_CONTAINER_VLLM), "vllm", true},
		{string(api.LLM_CONTAINER_OLLAMA), "ollama", true},
		{string(api.LLM_CONTAINER_SGLANG), "sgl", true},
		{"dify", "", false},
	}
	for _, c := range cases {
		key, ok := mapLLMTypeToProviderKey(c.in)
		if ok != c.ok || key != c.key {
			t.Fatalf("mapLLMTypeToProviderKey(%q) = (%q, %v), want (%q, %v)", c.in, key, ok, c.key, c.ok)
		}
	}
}

func TestSlugModelKey(t *testing.T) {
	if got := slugModelKey("Qwen/Qwen2.5-7B-Instruct"); got != "qwen-qwen2-5-7b-instruct" {
		t.Fatalf("slugModelKey got %q", got)
	}
}

func TestClientModelAlias(t *testing.T) {
	llm := &SLLM{}
	llm.Name = "llm-0"
	if got := clientModelAlias(llm, "Qwen3-0.6B"); got != "llm-0-Qwen3-0.6B" {
		t.Fatalf("clientModelAlias got %q", got)
	}
	llm2 := &SLLM{}
	llm2.Id = "llm-id-1"
	if got := clientModelAlias(llm2, ""); got != "llm-id-1" {
		t.Fatalf("clientModelAlias without model_key got %q", got)
	}
}

func TestUpstreamModelKeyForBackend(t *testing.T) {
	cases := []struct {
		llmType    string
		modelName  string
		modelTag   string
		want       string
	}{
		{string(api.LLM_CONTAINER_VLLM), "Qwen/Qwen3-0.6B", "main", "Qwen3-0.6B"},
		{string(api.LLM_CONTAINER_SGLANG), "Qwen/Qwen2.5-7B-Instruct", "main", "Qwen2.5-7B-Instruct"},
		{string(api.LLM_CONTAINER_VLLM), "Qwen3-0.6B", "main", "Qwen3-0.6B"},
		{string(api.LLM_CONTAINER_OLLAMA), "qwen3", "8b", "qwen3:8b"},
		{string(api.LLM_CONTAINER_OLLAMA), "qwen3", "", "qwen3"},
	}
	for _, c := range cases {
		got := upstreamModelKeyForBackend(c.llmType, c.modelName, c.modelTag)
		if got != c.want {
			t.Fatalf("upstreamModelKeyForBackend(%q, %q, %q) = %q, want %q",
				c.llmType, c.modelName, c.modelTag, got, c.want)
		}
	}
}
