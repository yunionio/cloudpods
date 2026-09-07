package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestApplySkuCgroupLimitDefaultsInference(t *testing.T) {
	for _, llmType := range []string{
		string(api.LLM_CONTAINER_VLLM),
		string(api.LLM_CONTAINER_SGLANG),
		string(api.LLM_CONTAINER_OLLAMA),
	} {
		input := &api.LLMSkuCreateInput{LLMType: llmType}
		applySkuCgroupLimitDefaults(input)
		if input.EnableCgroupCpu == nil || *input.EnableCgroupCpu {
			t.Fatalf("%s: expected enable_cgroup_cpu=false, got %v", llmType, input.EnableCgroupCpu)
		}
		if input.EnableCgroupMemory == nil || *input.EnableCgroupMemory {
			t.Fatalf("%s: expected enable_cgroup_memory=false, got %v", llmType, input.EnableCgroupMemory)
		}
	}
}

func TestApplySkuCgroupLimitDefaultsInferenceExplicitTrue(t *testing.T) {
	enabled := true
	input := &api.LLMSkuCreateInput{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		LLMSKuBaseCreateInput: api.LLMSKuBaseCreateInput{
			EnableCgroupCpu:    &enabled,
			EnableCgroupMemory: &enabled,
		},
	}
	applySkuCgroupLimitDefaults(input)
	if input.EnableCgroupCpu == nil || !*input.EnableCgroupCpu {
		t.Fatalf("expected explicit enable_cgroup_cpu=true, got %v", input.EnableCgroupCpu)
	}
	if input.EnableCgroupMemory == nil || !*input.EnableCgroupMemory {
		t.Fatalf("expected explicit enable_cgroup_memory=true, got %v", input.EnableCgroupMemory)
	}
}

func TestApplySkuCgroupLimitDefaultsNonInference(t *testing.T) {
	for _, llmType := range []string{
		string(api.LLM_CONTAINER_DIFY),
		string(api.LLM_CONTAINER_DESKTOP),
		string(api.LLM_CONTAINER_COMFYUI),
	} {
		input := &api.LLMSkuCreateInput{LLMType: llmType}
		applySkuCgroupLimitDefaults(input)
		if input.EnableCgroupCpu == nil || !*input.EnableCgroupCpu {
			t.Fatalf("%s: expected enable_cgroup_cpu=true, got %v", llmType, input.EnableCgroupCpu)
		}
		if input.EnableCgroupMemory == nil || !*input.EnableCgroupMemory {
			t.Fatalf("%s: expected enable_cgroup_memory=true, got %v", llmType, input.EnableCgroupMemory)
		}
	}
}

func TestSkuCgroupLimitEnabled(t *testing.T) {
	if !skuCgroupLimitEnabled(nil) {
		t.Fatal("nil should default to enabled (legacy SKU behavior)")
	}
	if !skuCgroupLimitEnabled(boolPtr(true)) {
		t.Fatal("true should be enabled")
	}
	if skuCgroupLimitEnabled(boolPtr(false)) {
		t.Fatal("false should be disabled")
	}
}
