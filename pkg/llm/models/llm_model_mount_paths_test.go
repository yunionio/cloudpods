package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestPickContainerModelMountPath(t *testing.T) {
	paths := []string{
		"/data/models/Other",
		"/data/models/huggingface/Qwen3-0.6B",
	}
	got := PickContainerModelMountPath(paths, "Qwen3-0.6B")
	if got != "/data/models/huggingface/Qwen3-0.6B" {
		t.Fatalf("basename preferred: got %q", got)
	}
	got = PickContainerModelMountPath(paths, "/data/models/Other")
	if got != "/data/models/Other" {
		t.Fatalf("exact preferred: got %q", got)
	}
	got = PickContainerModelMountPath(paths, "")
	if got != "/data/models/Other" {
		t.Fatalf("empty preferred first sorted: got %q", got)
	}
	if PickContainerModelMountPath(nil, "x") != "" {
		t.Fatal("empty paths should return empty")
	}
}

func TestCollectContainerModelMountPathsLocalHostPath(t *testing.T) {
	hostPaths := api.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: api.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	sku := &SLLMSku{
		LLMType:   string(api.LLM_CONTAINER_VLLM),
		Source:    api.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
	}
	sku.HostPaths = &hostPaths
	got := CollectContainerModelMountPaths(nil, sku)
	if len(got) != 1 || got[0] != "/data/models/huggingface/Qwen3-8B" {
		t.Fatalf("local_path mounts: got %#v", got)
	}
}
