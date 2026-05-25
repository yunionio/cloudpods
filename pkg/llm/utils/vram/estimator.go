package vram

import api "yunion.io/x/onecloud/pkg/apis/llm"

// VRAM-claim estimation, mirroring GPUStack's `estimate_model_vram()` in
// `gpustack/policies/utils.py`.
//
// Formula:
//
//	LLM:        VRAM ≈ weight_size * 1.2 + 2 GiB
//	Embedding:  VRAM ≈ weight_size * 1.2 + 512 MiB
//	Image:      VRAM ≈ weight_size           (no factor, no overhead)
//
// Reference for the 20% factor:
// https://blog.eleuther.ai/transformer-math/#total-inference-memory
//
// Reference numbers (bf16, llm overhead):
//
//	0.5B  →  ~3.1 GiB
//	3B    →  ~8.9 GiB
//	7B    →  ~19.0 GiB
//	72B   →  ~164.5 GiB
const (
	activationOverheadFactor  = 1.2
	llmFrameworkOverheadMB    = 2048 // 2 GiB
	nonLlmFrameworkOverheadMB = 512  // 512 MiB
)

// EstimateClaimMb returns the heuristic VRAM requirement in MiB.
// weightSizeBytes <= 0 (i.e. unknown) → returns 0; callers treat 0 as
// "no constraint" / "schedule without VRAM check".
func EstimateClaimMb(weightSizeBytes int64, llmType string) int {
	if weightSizeBytes <= 0 {
		return 0
	}
	weightMb := weightSizeBytes / (1024 * 1024)

	// Image / diffusion: weight only, no factor, no overhead.
	if isImageLLMType(llmType) {
		return int(weightMb)
	}

	overhead := llmFrameworkOverheadMB
	if !isLLMType(llmType) {
		overhead = nonLlmFrameworkOverheadMB
	}
	return int(float64(weightMb)*activationOverheadFactor) + overhead
}

// isLLMType reports whether the backend serves text-generation LLMs that get
// the larger framework overhead (CUDA graphs, runtime buffers, KV scratch).
func isLLMType(t string) bool {
	switch api.LLMContainerType(t) {
	case api.LLM_CONTAINER_VLLM, api.LLM_CONTAINER_OLLAMA,
		api.LLM_CONTAINER_SGLANG, api.LLM_CONTAINER_HERMES_AGENT:
		return true
	}
	return false
}

// isImageLLMType reports whether the backend is a diffusion / image generation
// runtime — these skip the activation factor + framework overhead because the
// inference shape is dominated by weight tensors alone.
func isImageLLMType(t string) bool {
	return api.LLMContainerType(t) == api.LLM_CONTAINER_COMFYUI
}
