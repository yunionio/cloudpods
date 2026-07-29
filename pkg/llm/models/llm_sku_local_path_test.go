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

func TestValidateLocalPathHamiDevicesRequireMemoryMb(t *testing.T) {
	t.Run("hami without memory_mb fails", func(t *testing.T) {
		devs := llm.Devices{{Model: "A100", SharingMode: "HAMI"}}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(&devs); err == nil {
			t.Fatal("expected error when HAMi memory_mb is missing")
		}
	})
	t.Run("empty sharing_mode defaults to hami and requires memory_mb", func(t *testing.T) {
		devs := llm.Devices{{Model: "A100"}}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(&devs); err == nil {
			t.Fatal("expected error when default HAMi memory_mb is missing")
		}
	})
	t.Run("legacy NVIDIA_HAMI without memory_mb fails", func(t *testing.T) {
		devs := llm.Devices{{Model: "A100", DevType: "NVIDIA_HAMI"}}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(&devs); err == nil {
			t.Fatal("expected error when legacy NVIDIA_HAMI memory_mb is missing")
		}
	})
	t.Run("hami with memory_mb passes", func(t *testing.T) {
		devs := llm.Devices{{Model: "A100", SharingMode: "HAMI", MemoryMb: 8192}}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(&devs); err != nil {
			t.Fatalf("expected valid HAMi devices, got %v", err)
		}
	})
	t.Run("empty sharing_mode with memory_mb passes", func(t *testing.T) {
		devs := llm.Devices{{Model: "A100", MemoryMb: 8192}}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(&devs); err != nil {
			t.Fatalf("expected default HAMi with memory_mb to pass, got %v", err)
		}
	})
	t.Run("exclusive without memory_mb passes", func(t *testing.T) {
		devs := llm.Devices{{Model: "A100", SharingMode: "EXCLUSIVE"}}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(&devs); err != nil {
			t.Fatalf("expected non-HAMi devices without memory_mb to pass, got %v", err)
		}
	})
	t.Run("nil devices passes", func(t *testing.T) {
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(nil); err != nil {
			t.Fatalf("expected nil devices to pass, got %v", err)
		}
	})
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
