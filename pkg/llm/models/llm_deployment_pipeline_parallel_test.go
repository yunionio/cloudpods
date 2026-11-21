package models

import "testing"

func TestValidateDeploymentPipelineParallel(t *testing.T) {
	cases := []struct {
		name     string
		llmType  string
		pp       int
		gpuCount int
		wantErr  bool
	}{
		{"auto", "vllm", 0, 8, false},
		{"single stage normalized earlier", "vllm", 1, 8, false},
		{"vllm even split", "vllm", 2, 8, false},
		{"vllm indivisible", "vllm", 3, 8, true},
		{"vllm exceeds gpus", "vllm", 9, 8, true},
		{"vllm no gpus", "vllm", 2, 0, true},
		{"sglang indivisible ok", "sglang", 3, 8, false},
		{"sglang exceeds gpus", "sglang", 9, 8, true},
		{"unsupported backend", "ollama", 2, 8, true},
		{"unknown backend", "", 2, 8, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDeploymentPipelineParallel(tc.llmType, tc.pp, tc.gpuCount)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
