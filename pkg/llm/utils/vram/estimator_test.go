package vram

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestEstimateClaimMb(t *testing.T) {
	gib := int64(1024 * 1024 * 1024)
	cases := []struct {
		name        string
		weightBytes int64
		llmType     string
		// minMb / maxMb form an acceptance band (the formula is heuristic;
		// we don't pin exact bytes, just confirm we're in GPUStack's ballpark).
		minMb int
		maxMb int
	}{
		// Reference numbers from GPUStack's docstring (bf16):
		//   0.5B → ~3.1 GiB
		//   3B   → ~8.9 GiB
		//   7B   → ~19.0 GiB
		//   72B  → ~164.5 GiB
		{
			name:        "0.5B bf16 vllm",
			weightBytes: 1 * gib,
			llmType:     string(api.LLM_CONTAINER_VLLM),
			minMb:       3000, maxMb: 3500, // 1024*1.2 + 2048 = 3276
		},
		{
			name:        "7B bf16 vllm",
			weightBytes: 14 * gib,
			llmType:     string(api.LLM_CONTAINER_VLLM),
			minMb:       18500, maxMb: 19500, // 14336*1.2 + 2048 = 19251
		},
		{
			name:        "72B bf16 vllm",
			weightBytes: 145 * gib,
			llmType:     string(api.LLM_CONTAINER_VLLM),
			minMb:       180000, maxMb: 180500, // 148480*1.2 + 2048 = 180224
		},
		{
			name:        "ollama same formula as vllm",
			weightBytes: 1 * gib,
			llmType:     string(api.LLM_CONTAINER_OLLAMA),
			minMb:       3000, maxMb: 3500,
		},
		{
			name:        "comfyui (image) — no factor, no overhead",
			weightBytes: 2 * gib,
			llmType:     string(api.LLM_CONTAINER_COMFYUI),
			minMb:       2000, maxMb: 2100, // 2048
		},
		{
			name:        "unknown backend → non-llm 512 MiB overhead",
			weightBytes: 1 * gib,
			llmType:     "dify",
			minMb:       1700, maxMb: 1800, // 1024*1.2 + 512 = 1740
		},
		{
			name:        "weight unknown → 0",
			weightBytes: 0,
			llmType:     string(api.LLM_CONTAINER_VLLM),
			minMb:       0, maxMb: 0,
		},
		{
			name:        "negative weight → 0",
			weightBytes: -1,
			llmType:     string(api.LLM_CONTAINER_VLLM),
			minMb:       0, maxMb: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := EstimateClaimMb(c.weightBytes, c.llmType)
			if got < c.minMb || got > c.maxMb {
				t.Errorf("got %d MiB, want in [%d, %d]", got, c.minMb, c.maxMb)
			}
		})
	}
}

func TestEstimateKvCacheReserveMb(t *testing.T) {
	if got := EstimateKvCacheReserveMb(8192); got != 2048 {
		t.Fatalf("8192 → %d, want 2048", got)
	}
	if got := EstimateKvCacheReserveMb(0); got != api.LLM_DEFAULT_CONTEXT_TOKENS/4 {
		t.Fatalf("0 → %d, want default/4=%d", got, api.LLM_DEFAULT_CONTEXT_TOKENS/4)
	}
	if got := EstimateKvCacheReserveMb(4096); got != 1024 {
		t.Fatalf("4096 → %d, want 1024", got)
	}
}

func TestEstimateClaimMbWithContext(t *testing.T) {
	weightBytes := int64(1024 * 1024 * 1000) // 1000 MiB
	base := EstimateClaimMb(weightBytes, string(api.LLM_CONTAINER_VLLM))
	got := EstimateClaimMbWithContext(weightBytes, string(api.LLM_CONTAINER_VLLM), 8192)
	if got != base+2048 {
		t.Fatalf("with context = %d, want base+2048=%d (base=%d)", got, base+2048, base)
	}
	// ComfyUI: no KV reserve
	imgWeight := int64(2 * 1024 * 1024 * 1024)
	imgBase := EstimateClaimMb(imgWeight, string(api.LLM_CONTAINER_COMFYUI))
	imgGot := EstimateClaimMbWithContext(imgWeight, string(api.LLM_CONTAINER_COMFYUI), 8192)
	if imgGot != imgBase {
		t.Fatalf("comfyui should ignore context, got %d base %d", imgGot, imgBase)
	}
	if EstimateClaimMbWithContext(0, string(api.LLM_CONTAINER_VLLM), 8192) != 0 {
		t.Fatal("zero weight should stay 0")
	}
}
