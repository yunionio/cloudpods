package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func localPathTestSku() *SLLMSku {
	return &SLLMSku{
		LLMType:   string(api.LLM_CONTAINER_VLLM),
		Source:    api.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
		SLLMSkuBase: SLLMSkuBase{
			HostPaths: &api.HostPaths{{
				Type: "directory",
				Path: "/data/models/Qwen3-8B",
				Containers: map[string]*api.ContainerHostPathRelation{
					"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
				},
			}},
		},
	}
}

func TestSkuCanAutoGpuMemoryUtilizationLocalPath(t *testing.T) {
	localSku := localPathTestSku()
	if skuCanAutoGpuMemoryUtilization(localSku) {
		t.Fatal("local_path SKU should not allow auto GPU util")
	}
	localSku.VramClaimMb = 16384
	if skuCanAutoGpuMemoryUtilization(localSku) {
		t.Fatal("local_path SKU should not allow auto GPU util even with vram_claim_mb")
	}
	if !skuCanAutoGpuMemoryUtilization(nil) {
		t.Fatal("nil sku should keep legacy default true")
	}
}

func TestDefaultDeploymentAutoGpuMemoryUtilizationSkipsLocalPath(t *testing.T) {
	input := &api.LLMDeploymentCreateInput{}
	sku := localPathTestSku()
	defaultDeploymentAutoGpuMemoryUtilization(input, sku, string(api.LLM_CONTAINER_VLLM))
	if input.AutoGpuMemoryUtilization != nil {
		t.Fatalf("expected auto_gpu_memory_utilization unset for local_path SKU, got %v", *input.AutoGpuMemoryUtilization)
	}
}

func TestDisableDeploymentAutoGpuMemoryUtilizationForLocalPath(t *testing.T) {
	input := &api.LLMDeploymentCreateInput{
		AutoGpuMemoryUtilization: func() *bool { v := true; return &v }(),
	}
	disableDeploymentAutoGpuMemoryUtilizationForLocalPath(input, localPathTestSku())
	if input.AutoGpuMemoryUtilization == nil || *input.AutoGpuMemoryUtilization {
		t.Fatalf("expected auto_gpu_memory_utilization forced false, got %v", input.AutoGpuMemoryUtilization)
	}
}

func TestMaxMountedModelVramRequirementMBLocalPathRejected(t *testing.T) {
	sku := localPathTestSku()
	sku.VramClaimMb = 20480
	_, err := maxMountedModelVramRequirementMB(sku)
	if err == nil {
		t.Fatal("expected error for local_path SKU auto GPU util")
	}
}
