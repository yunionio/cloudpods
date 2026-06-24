package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestValidateDeploymentDevices(t *testing.T) {
	devices := api.Devices{{Model: "NVIDIA A100"}}
	skuWithGPU := &SLLMSku{
		LLMType: string(api.LLM_CONTAINER_VLLM),
		SLLMSkuBase: SLLMSkuBase{
			Devices: &devices,
		},
	}
	skuWithoutGPU := &SLLMSku{LLMType: string(api.LLM_CONTAINER_VLLM)}

	if err := ValidateDeploymentDevices(string(api.LLM_CONTAINER_VLLM), skuWithGPU); err != nil {
		t.Fatalf("expected no error with GPU devices, got %v", err)
	}
	if err := ValidateDeploymentDevices(string(api.LLM_CONTAINER_VLLM), skuWithoutGPU); err == nil {
		t.Fatal("expected error when vllm SKU has no GPU devices")
	}
	if err := ValidateDeploymentDevices(string(api.LLM_CONTAINER_DIFY), skuWithoutGPU); err != nil {
		t.Fatalf("dify deployment should not require GPU devices, got %v", err)
	}
}

func TestSkuFromLLMSkuCreateInput(t *testing.T) {
	devices := api.Devices{{Model: "NVIDIA A100"}}
	sku := skuFromLLMSkuCreateInput(&api.LLMSkuCreateInput{
		LLMType: string(api.LLM_CONTAINER_SGLANG),
		LLMSKuBaseCreateInput: api.LLMSKuBaseCreateInput{
			Devices: &devices,
		},
	})
	if sku == nil || sku.LLMType != string(api.LLM_CONTAINER_SGLANG) {
		t.Fatalf("unexpected sku: %#v", sku)
	}
	if sku.Devices == nil || len(*sku.Devices) != 1 {
		t.Fatalf("expected devices on sku, got %#v", sku.Devices)
	}
}
