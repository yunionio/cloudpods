package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestHasHygonDevices(t *testing.T) {
	hygonSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_HYGON_DCU},
			},
		},
	}
	if !HasHygonDevices(nil, hygonSku) {
		t.Fatal("expected Hygon DCU sku to be detected")
	}

	hamiSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_HYGON_DCU_HAMI},
			},
		},
	}
	if !HasHygonDevices(nil, hamiSku) {
		t.Fatal("expected Hygon DCU HAMI sku to be detected")
	}

	nvSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU},
			},
		},
	}
	if HasHygonDevices(nil, nvSku) {
		t.Fatal("expected NVIDIA sku not to be detected as Hygon")
	}

	llm := &SLLM{
		SLLMBase: SLLMBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_HYGON_DCU},
			},
		},
	}
	if !HasHygonDevices(llm, nvSku) {
		t.Fatal("expected llm device override to win over sku")
	}

	normalizedHygonSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.GPU_TYPE, Vendor: "HYGON", Model: "BW"},
			},
		},
	}
	if !HasHygonDevices(nil, normalizedHygonSku) {
		t.Fatal("expected normalized GPU+HYGON vendor sku to be detected as Hygon")
	}
}

func TestHasIluvatarDevices(t *testing.T) {
	iluvatarSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_ILUVATAR_GPU},
			},
		},
	}
	if !HasIluvatarDevices(nil, iluvatarSku) {
		t.Fatal("expected Iluvatar GPU sku to be detected")
	}

	nvSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU},
			},
		},
	}
	if HasIluvatarDevices(nil, nvSku) {
		t.Fatal("expected NVIDIA sku not to be detected as Iluvatar")
	}

	llm := &SLLM{
		SLLMBase: SLLMBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_ILUVATAR_GPU},
			},
		},
	}
	if !HasIluvatarDevices(llm, nvSku) {
		t.Fatal("expected llm device override to win over sku")
	}

	normalizedSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.GPU_TYPE, Vendor: "ILUVATAR", Model: "BI-V150S"},
			},
		},
	}
	if !HasIluvatarDevices(nil, normalizedSku) {
		t.Fatal("expected normalized GPU+ILUVATAR vendor sku to be detected as Iluvatar")
	}
}

func TestHasTHeadDevices(t *testing.T) {
	theadSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_THEAD_PPU},
			},
		},
	}
	if !HasTHeadDevices(nil, theadSku) {
		t.Fatal("expected T-Head PPU sku to be detected")
	}

	nvSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU},
			},
		},
	}
	if HasTHeadDevices(nil, nvSku) {
		t.Fatal("expected NVIDIA sku not to be detected as T-Head")
	}

	llm := &SLLM{
		SLLMBase: SLLMBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_THEAD_PPU},
			},
		},
	}
	if !HasTHeadDevices(llm, nvSku) {
		t.Fatal("expected llm device override to win over sku")
	}

	normalizedSku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.GPU_TYPE, Vendor: "THEAD", Model: "PPU-ZW810E"},
			},
		},
	}
	if !HasTHeadDevices(nil, normalizedSku) {
		t.Fatal("expected normalized GPU+THEAD vendor sku to be detected as T-Head")
	}
}
