package llm_container

import (
	commonapi "yunion.io/x/onecloud/pkg/apis"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

// applyInferenceContainerShmAndCapabilities sets the /dev/shm size and
// IPC_LOCK capability recommended by the vLLM GPUDirect RDMA guidance
// (https://docs.vllm.ai/en/latest/serving/parallelism_scaling/): a large
// /dev/shm for NCCL IPC plus IPC_LOCK to allow locking memory pages.
// extraCaps (e.g. SYS_PTRACE for Hygon) are merged into the same list.
func applyInferenceContainerShmAndCapabilities(spec *commonapi.ContainerSpec, extraCaps ...string) {
	spec.ShmSizeMB = api.LLM_INFERENCE_SHM_SIZE_MB
	caps := append([]string{"IPC_LOCK"}, extraCaps...)
	spec.Capabilities = &commonapi.ContainerCapability{Add: caps}
}
