package models

import (
	"testing"

	"yunion.io/x/onecloud/pkg/apis/llm"
)

func TestValidateLocalPathSkuCreate(t *testing.T) {
	hostPaths := llm.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: llm.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B", ReadOnly: true},
			},
		},
	}
	input := &llm.LLMSkuCreateInput{
		LLMSKuBaseCreateInput: llm.LLMSKuBaseCreateInput{
			HostPaths: &hostPaths,
		},
		LLMType:     string(llm.LLM_CONTAINER_VLLM),
		Source:      llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath:   "/data/models/Qwen3-8B",
		PreferHosts: []string{"host-1"},
	}
	if err := ValidateLocalPathSkuCreate(input); err != nil {
		t.Fatalf("expected valid local_path sku, got %v", err)
	}
	if input.Source != llm.LLM_MODEL_SOURCE_LOCAL_PATH {
		t.Fatalf("expected source local_path, got %q", input.Source)
	}
}

func TestValidateLocalPathSkuCreateRejectsModelSpec(t *testing.T) {
	hostPaths := llm.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: llm.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	input := &llm.LLMSkuCreateInput{
		LLMSKuBaseCreateInput: llm.LLMSKuBaseCreateInput{
			HostPaths: &hostPaths,
		},
		LLMType:     string(llm.LLM_CONTAINER_VLLM),
		Source:      llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath:   "/data/models/Qwen3-8B",
		PreferHosts: []string{"host-1"},
		ModelSpec:   &llm.InstantModelImportInput{ModelName: "x", ModelTag: "main"},
	}
	if err := ValidateLocalPathSkuCreate(input); err == nil {
		t.Fatal("expected error when model_spec is set")
	}
}

func TestValidateLocalPathSkuCreateRequiresPreferHosts(t *testing.T) {
	hostPaths := llm.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: llm.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	input := &llm.LLMSkuCreateInput{
		LLMSKuBaseCreateInput: llm.LLMSKuBaseCreateInput{
			HostPaths: &hostPaths,
		},
		LLMType:   string(llm.LLM_CONTAINER_VLLM),
		Source:    llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
	}
	if err := ValidateLocalPathSkuCreate(input); err == nil {
		t.Fatal("expected error when prefer_hosts is missing")
	}
}

func TestValidateLocalPathSkuCreateRequiresContainerMount(t *testing.T) {
	hostPaths := llm.HostPaths{
		{Type: "directory", Path: "/data/models/Qwen3-8B"},
	}
	input := &llm.LLMSkuCreateInput{
		LLMSKuBaseCreateInput: llm.LLMSKuBaseCreateInput{
			HostPaths: &hostPaths,
		},
		LLMType:     string(llm.LLM_CONTAINER_SGLANG),
		Source:      llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath:   "/data/models/Qwen3-8B",
		PreferHosts: []string{"host-1"},
	}
	if err := ValidateLocalPathSkuCreate(input); err == nil {
		t.Fatal("expected error when container 0 mount is missing")
	}
}

func TestSkuHasLocalHostPathModel(t *testing.T) {
	hostPaths := llm.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: llm.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	sku := &SLLMSku{
		Source:    llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
	}
	sku.HostPaths = &hostPaths
	if !SkuHasLocalHostPathModel(sku) {
		t.Fatal("expected local host path model sku")
	}
}

func TestValidateRequireMountedModelsSkipsLocalPathSku(t *testing.T) {
	hostPaths := llm.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: llm.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	sku := &SLLMSku{
		LLMType:   string(llm.LLM_CONTAINER_VLLM),
		Source:    llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
	}
	sku.HostPaths = &hostPaths
	if err := ValidateRequireMountedModels(string(llm.LLM_CONTAINER_VLLM), nil, nil, sku); err != nil {
		t.Fatalf("expected mounted_models not required for local_path sku, got %v", err)
	}
}
