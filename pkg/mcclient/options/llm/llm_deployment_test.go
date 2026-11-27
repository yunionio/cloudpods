package llm

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestBuildSkuSpecBackendArgs(t *testing.T) {
	o := &LLMDeploymentCreateOptions{
		SkuLLMType:        string(api.LLM_CONTAINER_SGLANG),
		SkuCategories:     "reranker",
		SkuSGLangArg:      []string{"is-embedding=", "disable-radix-cache="},
		SkuPreferredModel: "bge-reranker-v2-m3",
	}
	if !o.hasSkuSpec() {
		t.Fatal("backend args should imply a sku_spec")
	}
	dict, err := o.buildSkuSpec()
	if err != nil {
		t.Fatalf("buildSkuSpec: %v", err)
	}
	spec := &api.LLMSpec{}
	if err := dict.Unmarshal(spec, "llm_spec"); err != nil {
		t.Fatalf("unmarshal llm_spec: %v", err)
	}
	if spec.SGLang == nil {
		t.Fatalf("sglang spec missing: %s", dict.String())
	}
	if spec.SGLang.PreferredModel != "bge-reranker-v2-m3" {
		t.Fatalf("preferred_model = %q", spec.SGLang.PreferredModel)
	}
	if len(spec.SGLang.CustomizedArgs) != 2 {
		t.Fatalf("customized_args = %+v", spec.SGLang.CustomizedArgs)
	}
	if spec.SGLang.CustomizedArgs[0].Key != "is-embedding" || spec.SGLang.CustomizedArgs[0].Value != "" {
		t.Fatalf("first arg = %+v", spec.SGLang.CustomizedArgs[0])
	}
	if !dict.Contains("categories") {
		t.Fatal("categories missing from sku_spec")
	}
}

func TestBuildSkuSpecRejectsArgsForWrongBackend(t *testing.T) {
	o := &LLMDeploymentCreateOptions{
		SkuLLMType:  string(api.LLM_CONTAINER_OLLAMA),
		SkuVllmArg:  []string{"runner=pooling"},
		SkuDiskSize: 1024,
	}
	if _, err := o.buildSkuSpec(); err == nil {
		t.Fatal("expected error for vllm args with ollama backend")
	}
}

func TestBuildSkuSpecVLLMArgs(t *testing.T) {
	o := &LLMDeploymentCreateOptions{
		SkuLLMType:  string(api.LLM_CONTAINER_VLLM),
		SkuVllmArg:  []string{"runner=pooling", "max-model-len=4096"},
		SkuDiskSize: 1024,
	}
	dict, err := o.buildSkuSpec()
	if err != nil {
		t.Fatalf("buildSkuSpec: %v", err)
	}
	spec := &api.LLMSpec{}
	if err := dict.Unmarshal(spec, "llm_spec"); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if spec.Vllm == nil || len(spec.Vllm.CustomizedArgs) != 2 {
		t.Fatalf("vllm spec = %+v", spec.Vllm)
	}
	if spec.Vllm.CustomizedArgs[0].Key != "runner" || spec.Vllm.CustomizedArgs[0].Value != "pooling" {
		t.Fatalf("args = %+v", spec.Vllm.CustomizedArgs)
	}
}
