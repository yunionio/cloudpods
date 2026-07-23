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
//
// EstimateClaimMbWithContext additionally reserves KV headroom for
// max_model_len (≈ 256 KiB/token → len/4 MiB) so HAMI slices can satisfy
// vLLM's KV check at the configured context length.
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

// EstimateKvCacheReserveMb returns a conservative KV-cache headroom in MiB
// for the given max_model_len. Uses ~256 KiB/token (len/4 MiB). When
// maxModelLen <= 0, defaults to api.LLM_DEFAULT_CONTEXT_TOKENS.
func EstimateKvCacheReserveMb(maxModelLen int) int {
	if maxModelLen <= 0 {
		maxModelLen = api.LLM_DEFAULT_CONTEXT_TOKENS
	}
	return maxModelLen / 4
}

// EstimateClaimMbWithContext is EstimateClaimMb plus KV reserve for LLM
// backends. Non-LLM / image types ignore maxModelLen (same as EstimateClaimMb).
func EstimateClaimMbWithContext(weightSizeBytes int64, llmType string, maxModelLen int) int {
	base := EstimateClaimMb(weightSizeBytes, llmType)
	if base <= 0 || !isLLMType(llmType) {
		return base
	}
	return base + EstimateKvCacheReserveMb(maxModelLen)
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
