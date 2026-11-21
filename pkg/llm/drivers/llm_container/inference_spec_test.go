package llm_container

import (
	"testing"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestApplyInferenceContainerShmAndCapabilities(t *testing.T) {
	spec := &commonapi.ContainerSpec{}
	applyInferenceContainerShmAndCapabilities(spec)
	if spec.ShmSizeMB != api.LLM_INFERENCE_SHM_SIZE_MB {
		t.Fatalf("shm = %d, want %d", spec.ShmSizeMB, api.LLM_INFERENCE_SHM_SIZE_MB)
	}
	if spec.Capabilities == nil || len(spec.Capabilities.Add) != 1 || spec.Capabilities.Add[0] != "IPC_LOCK" {
		t.Fatalf("capabilities = %+v, want [IPC_LOCK]", spec.Capabilities)
	}
}

func TestApplyInferenceContainerShmAndCapabilitiesMergesExtraCaps(t *testing.T) {
	spec := &commonapi.ContainerSpec{}
	applyInferenceContainerShmAndCapabilities(spec, "SYS_PTRACE")
	if spec.Capabilities == nil || len(spec.Capabilities.Add) != 2 ||
		spec.Capabilities.Add[0] != "IPC_LOCK" || spec.Capabilities.Add[1] != "SYS_PTRACE" {
		t.Fatalf("capabilities = %+v, want [IPC_LOCK SYS_PTRACE]", spec.Capabilities)
	}
}
