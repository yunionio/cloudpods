package llm_container

import (
	"context"
	"reflect"
	"strings"
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func TestVLLMGetContainerSpecHygonRuntime(t *testing.T) {
	v := newVLLM().(*vllm)
	hostPaths := api.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: api.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	hygonSku := &models.SLLMSku{
		SLLMSkuBase: models.SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_HYGON_DCU},
				{DevType: computeapi.CONTAINER_DEV_HYGON_DCU},
			},
		},
		LLMType:   string(api.LLM_CONTAINER_VLLM),
		Source:    api.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
	}
	hygonSku.HostPaths = &hostPaths
	image := &models.SLLMImage{}
	out := v.GetContainerSpec(context.Background(), nil, image, hygonSku, nil, nil, "")
	if out == nil {
		t.Fatal("expected container spec")
	}
	spec := &out.ContainerSpec
	if spec.Capabilities == nil || !reflect.DeepEqual(spec.Capabilities.Add, []string{"SYS_PTRACE"}) {
		t.Fatalf("capabilities = %#v, want SYS_PTRACE", spec.Capabilities)
	}
	if spec.SecurityContext == nil || !reflect.DeepEqual(spec.SecurityContext.SupplementalGroupNames, []string{"video"}) {
		t.Fatalf("security context = %#v, want video group", spec.SecurityContext)
	}
	if len(spec.Command) != 2 || spec.Command[0] != "/bin/bash" || spec.Command[1] != "-c" {
		t.Fatalf("command = %#v, want [/bin/bash -c]", spec.Command)
	}
	if len(spec.Args) == 0 || !strings.Contains(spec.Args[0], "/opt/dtk/env.sh") {
		t.Fatalf("args = %#v, want dtk env source in entrypoint", spec.Args)
	}
}

func TestVLLMGetContainerSpecNvidiaNoHygonRuntime(t *testing.T) {
	v := newVLLM().(*vllm)
	nvSku := &models.SLLMSku{
		SLLMSkuBase: models.SLLMSkuBase{
			Devices: &api.Devices{
				{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU},
			},
		},
	}
	image := &models.SLLMImage{}
	out := v.GetContainerSpec(context.Background(), nil, image, nvSku, nil, nil, "")
	if out == nil {
		t.Fatal("expected container spec")
	}
	spec := &out.ContainerSpec
	if spec.Capabilities != nil {
		t.Fatalf("expected nil capabilities, got %#v", spec.Capabilities)
	}
	if spec.SecurityContext != nil {
		t.Fatalf("expected nil security context, got %#v", spec.SecurityContext)
	}
	if len(spec.Command) != 2 || spec.Command[0] != "/bin/sh" {
		t.Fatalf("command = %#v, want [/bin/sh -c]", spec.Command)
	}
	if len(spec.Args) > 0 && strings.Contains(spec.Args[0], "/opt/dtk/env.sh") {
		t.Fatalf("args should not source dtk env for nvidia, got %#v", spec.Args)
	}
}
