package models

import (
	"context"
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

func TestUpstreamModelKeyFromLocalPathSku(t *testing.T) {
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

	got := UpstreamModelKeyFromLocalPathSku(nil, sku)
	if got != "Qwen3-8B" {
		t.Fatalf("expected upstream model key Qwen3-8B, got %q", got)
	}

	skuWithPreferred := &SLLMSku{
		LLMType:   string(llm.LLM_CONTAINER_VLLM),
		Source:    llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
		LLMSpec:   &llm.LLMSpec{Vllm: &llm.LLMSpecVllm{PreferredModel: "Qwen3-8B"}},
	}
	skuWithPreferred.HostPaths = &hostPaths
	got = UpstreamModelKeyFromLocalPathSku(nil, skuWithPreferred)
	if got != "Qwen3-8B" {
		t.Fatalf("expected upstream model key with preferred model, got %q", got)
	}

	nonLocal := &SLLMSku{LLMType: string(llm.LLM_CONTAINER_VLLM)}
	if UpstreamModelKeyFromLocalPathSku(nil, nonLocal) != "" {
		t.Fatal("expected empty key for non local_path sku")
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

func localPathSkuForPreferHostsUpdate() *SLLMSku {
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
		LLMType:     string(llm.LLM_CONTAINER_VLLM),
		Source:      llm.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath:   "/data/models/Qwen3-8B",
		PreferHosts: []string{"host-1"},
	}
	sku.HostPaths = &hostPaths
	return sku
}

func TestValidateLocalPathSkuUpdatePreferHostsOmitted(t *testing.T) {
	sku := localPathSkuForPreferHostsUpdate()
	input := &llm.LLMSkuUpdateInput{}
	if err := validateLocalPathSkuUpdatePreferHosts(context.Background(), nil, sku, input); err != nil {
		t.Fatalf("expected omitted prefer_hosts to skip, got %v", err)
	}
	if input.PreferHosts != nil {
		t.Fatalf("expected prefer_hosts to stay omitted, got %v", input.PreferHosts)
	}
}

func TestValidateLocalPathSkuUpdatePreferHostsEmptyRejected(t *testing.T) {
	sku := localPathSkuForPreferHostsUpdate()
	input := &llm.LLMSkuUpdateInput{PreferHosts: []string{}}
	if err := validateLocalPathSkuUpdatePreferHosts(context.Background(), nil, sku, input); err == nil {
		t.Fatal("expected error when prefer_hosts is empty")
	}
	input.PreferHosts = []string{"  ", ""}
	if err := validateLocalPathSkuUpdatePreferHosts(context.Background(), nil, sku, input); err == nil {
		t.Fatal("expected error when prefer_hosts is whitespace only")
	}
}

func TestValidateLocalPathSkuUpdatePreferHostsRejectedOnNonLocalPath(t *testing.T) {
	sku := &SLLMSku{LLMType: string(llm.LLM_CONTAINER_VLLM)}
	input := &llm.LLMSkuUpdateInput{PreferHosts: []string{"host-1"}}
	if err := validateLocalPathSkuUpdatePreferHosts(context.Background(), nil, sku, input); err == nil {
		t.Fatal("expected error when prefer_hosts is set on non local_path sku")
	}
}
