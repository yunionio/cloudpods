package models

import (
	"math"
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestSkuDevicesHaveHami(t *testing.T) {
	hamiSku := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{{
				Model:   "NVIDIA A100",
				DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI,
			}},
		},
	}
	if !skuDevicesHaveHami(hamiSku) {
		t.Fatal("expected HAMI device to be detected")
	}

	exclusiveSku := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{{
				Model:       "NVIDIA A100",
				DevType:     computeapi.GPU_TYPE,
				SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			}},
		},
	}
	if skuDevicesHaveHami(exclusiveSku) {
		t.Fatal("exclusive device should not be HAMI")
	}
	if skuDevicesHaveHami(nil) {
		t.Fatal("nil sku should not be HAMI")
	}
}

func TestResolveHamiAllocatedMemoryMBWithClaim(t *testing.T) {
	sku := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{{
				Model:   "NVIDIA A100",
				DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI,
			}},
		},
	}
	got, err := resolveHamiAllocatedMemoryMBWithClaim(sku, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 4096 {
		t.Fatalf("allocated = %d, want 4096", got)
	}

	sku.Devices = &api.Devices{{
		Model:    "NVIDIA A100",
		DevType:  computeapi.CONTAINER_DEV_NVIDIA_HAMI,
		MemoryMb: 16384,
	}}
	got, err = resolveHamiAllocatedMemoryMBWithClaim(sku, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 16384 {
		t.Fatalf("manual memory_mb should win, got %d", got)
	}

	sku.Devices = &api.Devices{
		{Model: "A100", DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI, MemoryMb: 16384},
		{Model: "A100", DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI},
	}
	got, err = resolveHamiAllocatedMemoryMBWithClaim(sku, 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// second device falls back to ceil(4096/2)=2048; min is 2048
	if got != 2048 {
		t.Fatalf("min allocated = %d, want 2048", got)
	}

	sku.Devices = &api.Devices{{Model: "A100", DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI}}
	if _, err := resolveHamiAllocatedMemoryMBWithClaim(sku, 0); err == nil {
		t.Fatal("expected error when claim and memory_mb are both 0")
	}
}

func TestCalculateDeploymentAutoGpuMemoryUtilizationHAMI(t *testing.T) {
	sku := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{{
				Model:   "NVIDIA A100",
				DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI,
			}},
		},
	}
	// required ≈ allocated → clamp to 1.0 (not physical 80GiB; no longer 0.95)
	got, err := calculateDeploymentAutoGpuMemoryUtilization(sku, 4096, 4096, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != autoGpuMemoryUtilizationMax {
		t.Fatalf("HAMI util when claim≈allocated = %v, want %v", got, autoGpuMemoryUtilizationMax)
	}

	// larger manual slice → util below max
	got, err = calculateDeploymentAutoGpuMemoryUtilization(sku, 4096, 16384, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := math.Ceil(4096*autoGpuMemoryUtilizationSafetyFactor/16384*100) / 100
	if got != want {
		t.Fatalf("HAMI util with larger slice = %v, want %v", got, want)
	}
	if got >= autoGpuMemoryUtilizationMax {
		t.Fatalf("expected util below max, got %v", got)
	}

	sglangSku := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_SGLANG),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{{
				Model:   "NVIDIA A100",
				DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI,
			}},
		},
	}
	got, err = calculateDeploymentAutoGpuMemoryUtilization(sglangSku, 4096, 4096, 1)
	if err != nil {
		t.Fatalf("unexpected sglang error: %v", err)
	}
	if got != autoGpuMemoryUtilizationMax {
		t.Fatalf("HAMI sglang util = %v, want %v", got, autoGpuMemoryUtilizationMax)
	}
}

func TestCalculateDeploymentAutoGpuMemoryUtilizationExclusive(t *testing.T) {
	sku := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{{
				Model:       "NVIDIA A100",
				DevType:     computeapi.GPU_TYPE,
				SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			}},
		},
	}
	got, err := calculateDeploymentAutoGpuMemoryUtilization(sku, 4096, 80*1024, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, err := calculateAutoGpuMemoryUtilization(4096, 80*1024, 1)
	if err != nil {
		t.Fatalf("reference calc error: %v", err)
	}
	if got != want {
		t.Fatalf("exclusive util = %v, want %v", got, want)
	}
	if got >= autoGpuMemoryUtilizationMax {
		t.Fatalf("exclusive util on large GPU should be below max, got %v", got)
	}
}
