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
