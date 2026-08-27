package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func sampleCloneSku() *SLLMSku {
	devices := api.Devices{
		{DevType: "GPU", SharingMode: "hami", Model: "NVIDIA-L20", MemoryMb: 20480},
	}
	volumes := api.Volumes{
		{SizeMB: 51200},
	}
	sku := &SLLMSku{
		SLLMSkuBase: SLLMSkuBase{
			Cpu:       8,
			Memory:    16384,
			Bandwidth: 200,
			Volumes:   &volumes,
			Devices:   &devices,
		},
		LLMImageId: "img-1",
		LLMType:    string(api.LLM_CONTAINER_VLLM),
		LLMSpec: &api.LLMSpec{
			Vllm: &api.LLMSpecVllm{PreferredModel: "Qwen3-8B"},
		},
		SMountedModelsResource: SMountedModelsResource{
			MountedModels: []string{"model-1", "model-2"},
		},
		Source:      api.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath:   "/data/models/Qwen3-8B",
		PreferHosts: []string{"host-1"},
		Categories:  []string{"llm"},
		BackendParameters: []string{
			`--max-model-len=8192`,
		},
	}
	sku.Description = "src desc"
	return sku
}

func TestBuildLLMSkuCloneCreateInputRejectsEmptyName(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.STATUS_READY
	_, err := buildLLMSkuCloneCreateInput(sku, api.LLMSkuCloneInput{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestBuildLLMSkuCloneCreateInputRejectsImporting(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL
	_, err := buildLLMSkuCloneCreateInput(sku, api.LLMSkuCloneInput{Name: "cloned"})
	if err == nil {
		t.Fatal("expected error while importing model")
	}
}

func TestBuildLLMSkuCloneCreateInputRejectsImportFailed(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED
	_, err := buildLLMSkuCloneCreateInput(sku, api.LLMSkuCloneInput{Name: "cloned"})
	if err == nil {
		t.Fatal("expected error after import model failed")
	}
}

func TestValidateLLMSkuCloneableImporting(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL
	if err := validateLLMSkuCloneable(sku); err == nil {
		t.Fatal("expected importing sku to be rejected")
	}
	sku.Status = api.STATUS_READY
	if err := validateLLMSkuCloneable(sku); err != nil {
		t.Fatalf("ready sku should be cloneable: %v", err)
	}
}

func TestValidateLLMSkuCloneableImportFailed(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED
	if err := validateLLMSkuCloneable(sku); err == nil {
		t.Fatal("expected import-failed sku to be rejected")
	}
}

func TestBuildLLMSkuCloneCreateInputCopiesFields(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.STATUS_READY
	create, err := buildLLMSkuCloneCreateInput(sku, api.LLMSkuCloneInput{Name: "cloned-sku"})
	if err != nil {
		t.Fatalf("buildLLMSkuCloneCreateInput: %v", err)
	}
	if create.Name != "cloned-sku" {
		t.Fatalf("unexpected name %q", create.Name)
	}
	if create.GenerateName != "cloned-sku" {
		t.Fatalf("unexpected generate_name %q", create.GenerateName)
	}
	if create.Description != "src desc" {
		t.Fatalf("unexpected description %q", create.Description)
	}
	if create.ModelSpec != nil {
		t.Fatal("clone must not set model_spec")
	}
	if create.Cpu != 8 || create.Memory != 16384 || create.Bandwidth != 200 {
		t.Fatalf("resource spec mismatch: cpu=%d mem=%d bw=%d", create.Cpu, create.Memory, create.Bandwidth)
	}
	if create.LLMImageId != "img-1" || create.LLMType != string(api.LLM_CONTAINER_VLLM) {
		t.Fatalf("llm identity mismatch: image=%s type=%s", create.LLMImageId, create.LLMType)
	}
	if create.LLMSpec == nil || create.LLMSpec.Vllm == nil || create.LLMSpec.Vllm.PreferredModel != "Qwen3-8B" {
		t.Fatalf("llm_spec not copied: %+v", create.LLMSpec)
	}
	if len(create.MountedModels) != 2 || create.MountedModels[0] != "model-1" {
		t.Fatalf("mounted_models not copied: %v", create.MountedModels)
	}
	if create.Source != api.LLM_MODEL_SOURCE_LOCAL_PATH || create.LocalPath != "/data/models/Qwen3-8B" {
		t.Fatalf("source not copied: source=%s path=%s", create.Source, create.LocalPath)
	}
	if len(create.PreferHosts) != 1 || create.PreferHosts[0] != "host-1" {
		t.Fatalf("prefer_hosts not copied: %v", create.PreferHosts)
	}
	if create.Devices == nil || len(*create.Devices) != 1 || (*create.Devices)[0].Model != "NVIDIA-L20" {
		t.Fatalf("devices not copied: %+v", create.Devices)
	}
	if create.Volumes == nil || len(*create.Volumes) != 1 || (*create.Volumes)[0].SizeMB != 51200 {
		t.Fatalf("volumes not copied: %+v", create.Volumes)
	}

	(*create.Devices)[0].Model = "mutated"
	create.MountedModels[0] = "mutated"
	create.PreferHosts[0] = "mutated"
	if (*sku.Devices)[0].Model != "NVIDIA-L20" {
		t.Fatal("clone must deep-copy devices")
	}
	if sku.MountedModels[0] != "model-1" {
		t.Fatal("clone must deep-copy mounted_models")
	}
	if sku.PreferHosts[0] != "host-1" {
		t.Fatal("clone must deep-copy prefer_hosts")
	}
}

func TestBuildLLMSkuCloneCreateInputGenerateName(t *testing.T) {
	sku := sampleCloneSku()
	sku.Status = api.STATUS_READY
	create, err := buildLLMSkuCloneCreateInput(sku, api.LLMSkuCloneInput{GenerateName: "cloned"})
	if err != nil {
		t.Fatalf("buildLLMSkuCloneCreateInput: %v", err)
	}
	if create.Name != "cloned" || create.GenerateName != "cloned" {
		t.Fatalf("expected generate_name to fill name, got name=%q generate_name=%q", create.Name, create.GenerateName)
	}
}
