package llm_container

import (
	"strings"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestDefaultServingArgsSGLang(t *testing.T) {
	cases := []struct {
		name    string
		cats    []string
		wantLen int
	}{
		{"embedding", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, 1},
		{"llm", []string{api.LLM_MODEL_CATEGORY_LLM}, 0},
		{"empty", nil, 0},
		// SGLang rerank flags are ambiguous (cross-encoder vs decoder-only).
		{"reranker", []string{api.LLM_MODEL_CATEGORY_RERANKER}, 0},
	}
	for _, c := range cases {
		got := defaultServingArgs(string(api.LLM_CONTAINER_SGLANG), "", c.cats)
		if len(got) != c.wantLen {
			t.Fatalf("%s: got %v want len %d", c.name, got, c.wantLen)
		}
		if c.wantLen == 1 && (got[0].Key != "is-embedding" || got[0].Value != "") {
			t.Fatalf("%s: unexpected arg %+v", c.name, got[0])
		}
	}
}

func TestDefaultServingArgsVLLMVersionGate(t *testing.T) {
	cases := []struct {
		version string
		cats    []string
		want    bool
	}{
		{"0.11.0", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, true},
		{"v0.10.0", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, true},
		{"0.10", []string{api.LLM_MODEL_CATEGORY_RERANKER}, true},
		{"0.9.2", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, false},
		{"", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, false},
		{"garbage", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, false},
		{"1.0.0", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, true},
		{"0.11.0", []string{api.LLM_MODEL_CATEGORY_LLM}, false},
		{"0.11.0", nil, false},
	}
	for _, c := range cases {
		got := defaultServingArgs(string(api.LLM_CONTAINER_VLLM), c.version, c.cats)
		if c.want {
			if len(got) != 1 || got[0].Key != "runner" || got[0].Value != "pooling" {
				t.Fatalf("version=%q cats=%v: got %v", c.version, c.cats, got)
			}
			continue
		}
		if len(got) != 0 {
			t.Fatalf("version=%q cats=%v: expected no args, got %v", c.version, c.cats, got)
		}
	}
}

func TestDefaultServingArgsVLLMOmni(t *testing.T) {
	for _, cat := range []string{api.LLM_MODEL_CATEGORY_IMAGE, api.LLM_MODEL_CATEGORY_TEXT_TO_SPEECH} {
		got := defaultServingArgs(string(api.LLM_CONTAINER_VLLM), "", []string{cat})
		if len(got) != 1 || got[0].Key != "omni" || got[0].Value != "" {
			t.Fatalf("cats=%v: got %v want [--omni]", cat, got)
		}
	}
	got := prependDefaultServingArgs(
		string(api.LLM_CONTAINER_VLLM), "",
		[]string{api.LLM_MODEL_CATEGORY_TEXT_TO_SPEECH},
		[]string{"--omni", "--trust-remote-code"},
	)
	if len(got) < 2 || got[0] != "--omni" {
		t.Fatalf("unexpected prepend order: %v", got)
	}
}

func TestDefaultServingArgsOtherBackends(t *testing.T) {
	if got := defaultServingArgs(string(api.LLM_CONTAINER_OLLAMA), "0.11.0", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}); len(got) != 0 {
		t.Fatalf("ollama should not inject args, got %v", got)
	}
}

func TestPrependDefaultServingArgsUserOverride(t *testing.T) {
	user := []string{"--runner=generate", "--max-model-len=4096"}
	withDefaults := prependDefaultServingArgs(
		string(api.LLM_CONTAINER_VLLM), "0.11.0",
		[]string{api.LLM_MODEL_CATEGORY_EMBEDDING}, user,
	)
	if len(withDefaults) != 3 {
		t.Fatalf("expected default + user args, got %v", withDefaults)
	}
	// Merging is last-wins, so the user value must come after the default.
	if withDefaults[0] != "--runner=pooling" || withDefaults[1] != "--runner=generate" {
		t.Fatalf("unexpected order: %v", withDefaults)
	}

	args, err := parseBackendParameterArgs(withDefaults, validateVLLMArgKey)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	merged, err := mergeRuntimeArgs(args, nil, validateVLLMArgKey)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	for _, arg := range merged {
		if arg.Key == "runner" && arg.Value != "generate" {
			t.Fatalf("user runner value did not win: %+v", arg)
		}
	}
}

func TestNewLLMInferenceStartupProbe(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		cats     []string
		wantPath string
		wantGet  string
		wantBody string
	}{
		{"chat", "qwen3-8b", []string{api.LLM_MODEL_CATEGORY_LLM}, "/v1/models", "", ""},
		{"embedding", "Qwen3-Embedding-0.6B", []string{api.LLM_MODEL_CATEGORY_EMBEDDING},
			"/v1/embeddings", "POST", `{"model":"Qwen3-Embedding-0.6B","input":"test"}`},
		{"reranker", "bge-reranker-v2-m3", []string{api.LLM_MODEL_CATEGORY_RERANKER},
			"/v1/rerank", "POST", `{"model":"bge-reranker-v2-m3","query":"test","documents":["test"]}`},
		// Unknown served model name → cannot POST, fall back to the liveness check.
		{"embedding-no-model", "", []string{api.LLM_MODEL_CATEGORY_EMBEDDING}, "/v1/models", "", ""},
	}
	for _, c := range cases {
		p := newLLMInferenceStartupProbe(8000, c.model, c.cats)
		if p.HTTPGet == nil {
			t.Fatalf("%s: no http probe", c.name)
		}
		if p.HTTPGet.Path != c.wantPath {
			t.Fatalf("%s: path=%q want %q", c.name, p.HTTPGet.Path, c.wantPath)
		}
		if p.HTTPGet.Method != c.wantGet {
			t.Fatalf("%s: method=%q want %q", c.name, p.HTTPGet.Method, c.wantGet)
		}
		if p.HTTPGet.Body != c.wantBody {
			t.Fatalf("%s: body=%q want %q", c.name, p.HTTPGet.Body, c.wantBody)
		}
		if p.HTTPGet.Port != 8000 {
			t.Fatalf("%s: port=%d", c.name, p.HTTPGet.Port)
		}
		if p.TimeoutSeconds != llmStartupProbeTimeoutSeconds || p.FailureThreshold != llmStartupProbeFailureThreshold {
			t.Fatalf("%s: probe timing changed", c.name)
		}
	}
}

func TestServedModelNameForPath(t *testing.T) {
	cases := map[string]string{
		"/data/models/Qwen/Qwen3-Embedding-0.6B": "Qwen3-Embedding-0.6B",
		"/data/models/bge-m3/":                   "bge-m3",
		"":                                       "",
		"/":                                      "",
	}
	for in, want := range cases {
		if got := servedModelNameForPath(in); got != want {
			t.Fatalf("servedModelNameForPath(%q)=%q want %q", in, got, want)
		}
	}
}

func TestEscapeProbeJSON(t *testing.T) {
	got := buildEmbeddingProbeBody(`a"b\c`)
	if strings.Contains(got, `"model":"a"b`) {
		t.Fatalf("model id was not escaped: %s", got)
	}
	if !strings.Contains(got, `a\"b\\c`) {
		t.Fatalf("unexpected escaping: %s", got)
	}
}

func TestSkuCategoriesNeedNoToolCallDefaults(t *testing.T) {
	if skuCategoriesNeedNoToolCallDefaults(nil) {
		t.Fatal("empty categories must keep tool-call defaults")
	}
	if skuCategoriesNeedNoToolCallDefaults([]string{api.LLM_MODEL_CATEGORY_LLM}) {
		t.Fatal("llm category must keep tool-call defaults")
	}
	for _, c := range []string{
		api.LLM_MODEL_CATEGORY_EMBEDDING,
		api.LLM_MODEL_CATEGORY_RERANKER,
	} {
		if !skuCategoriesNeedNoToolCallDefaults([]string{c}) {
			t.Fatalf("%s must skip tool-call defaults", c)
		}
	}
}
