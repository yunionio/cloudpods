package models

import (
	"context"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestSkuHasExplicitGpuMemoryUtilization(t *testing.T) {
	if skuHasExplicitGpuMemoryUtilization(nil) {
		t.Fatal("nil sku should not have explicit GPU util")
	}

	plain := &SLLMSku{LLMType: string(api.LLM_CONTAINER_VLLM)}
	if skuHasExplicitGpuMemoryUtilization(plain) {
		t.Fatal("SKU without backend args should not have explicit GPU util")
	}

	byBackend := &SLLMSku{
		LLMType:           string(api.LLM_CONTAINER_VLLM),
		BackendParameters: []string{"--max-model-len=4096", "--gpu-memory-utilization=0.9"},
	}
	if !skuHasExplicitGpuMemoryUtilization(byBackend) {
		t.Fatal("backend_parameters gpu-memory-utilization should count as explicit")
	}

	bySpec := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		LLMSpec: &api.LLMSpec{
			Vllm: &api.LLMSpecVllm{
				CustomizedArgs: []*api.VllmCustomizedArg{{Key: "gpu-memory-utilization", Value: "0.85"}},
			},
		},
	}
	if !skuHasExplicitGpuMemoryUtilization(bySpec) {
		t.Fatal("vllm customized_args gpu-memory-utilization should count as explicit")
	}

	sglang := &SLLMSku{
		LLMType:           string(api.LLM_CONTAINER_SGLANG),
		BackendParameters: []string{"--mem-fraction-static=0.8"},
	}
	if !skuHasExplicitGpuMemoryUtilization(sglang) {
		t.Fatal("sglang mem-fraction-static should count as explicit")
	}
}

func TestBuildDeploymentResolvedGpuMemoryLLMSpecSkipsExplicitSkuArg(t *testing.T) {
	auto := true
	deploy := &SLLMDeployment{AutoGpuMemoryUtilization: &auto}
	sku := &SLLMSku{
		LLMType:           string(api.LLM_CONTAINER_VLLM),
		BackendParameters: []string{"--gpu-memory-utilization=0.9"},
	}
	spec, err := BuildDeploymentResolvedGpuMemoryLLMSpec(context.Background(), nil, deploy, sku)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec != nil {
		t.Fatalf("expected nil spec when SKU already sets gpu-memory-utilization, got %#v", spec)
	}
}

func TestApplyResolvedGpuMemoryLLMSpecStripsStaleAutoArg(t *testing.T) {
	current := &api.LLMSpec{
		Vllm: &api.LLMSpecVllm{
			PreferredModel: "qwen",
			CustomizedArgs: []*api.VllmCustomizedArg{
				{Key: "gpu-memory-utilization", Value: "0.98"},
				{Key: "max-model-len", Value: "4096"},
			},
		},
	}
	got := applyResolvedGpuMemoryLLMSpec(current, nil, string(api.LLM_CONTAINER_VLLM))
	if got == nil || got.Vllm == nil {
		t.Fatal("expected remaining spec after stripping gpu util")
	}
	if got.Vllm.PreferredModel != "qwen" {
		t.Fatalf("preferred model: %q", got.Vllm.PreferredModel)
	}
	if len(got.Vllm.CustomizedArgs) != 1 || got.Vllm.CustomizedArgs[0].Key != "max-model-len" {
		t.Fatalf("customized args after strip: %#v", got.Vllm.CustomizedArgs)
	}
}

func TestApplyResolvedGpuMemoryLLMSpecWritesResolvedArg(t *testing.T) {
	current := &api.LLMSpec{
		Vllm: &api.LLMSpecVllm{
			CustomizedArgs: []*api.VllmCustomizedArg{{Key: "gpu-memory-utilization", Value: "0.98"}},
		},
	}
	resolved := &api.LLMSpec{
		Vllm: &api.LLMSpecVllm{
			CustomizedArgs: []*api.VllmCustomizedArg{{Key: "gpu-memory-utilization", Value: "0.9"}},
		},
	}
	got := applyResolvedGpuMemoryLLMSpec(current, resolved, string(api.LLM_CONTAINER_VLLM))
	if got == nil || got.Vllm == nil || len(got.Vllm.CustomizedArgs) != 1 {
		t.Fatalf("unexpected spec: %#v", got)
	}
	if got.Vllm.CustomizedArgs[0].Value != "0.9" {
		t.Fatalf("expected resolved 0.9, got %q", got.Vllm.CustomizedArgs[0].Value)
	}
}
