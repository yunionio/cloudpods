package models

import (
	"strings"
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestValidateDeploymentParallelMode(t *testing.T) {
	cases := []struct {
		name        string
		mode        string
		llmType     string
		backendParams []string
		deploymentPP int
		netType     string
		distributed bool
		nnodes      int
		wantErr     bool
		wantContain string
	}{
		{
			name:    "default pp valid",
			mode:    "",
			llmType: string(api.LLM_CONTAINER_VLLM),
		},
		{
			name:    "explicit dp valid",
			mode:    "dp",
			llmType: string(api.LLM_CONTAINER_VLLM),
		},
		{
			name:        "sglang dp rejected",
			mode:        "dp",
			llmType:     string(api.LLM_CONTAINER_SGLANG),
			wantErr:     true,
			wantContain: "SGLang 不支持数据并行",
		},
		{
			name:        "hostlocal pp rejected",
			mode:        "pp",
			llmType:     string(api.LLM_CONTAINER_VLLM),
			netType:     string(computeapi.NETWORK_TYPE_HOSTLOCAL),
			distributed: true,
			nnodes:      2,
			wantErr:     true,
			wantContain: "hostlocal 网络不支持跨机流水线并行",
		},
		{
			name:        "hostlocal dp allowed",
			mode:        "dp",
			llmType:     string(api.LLM_CONTAINER_VLLM),
			netType:     string(computeapi.NETWORK_TYPE_HOSTLOCAL),
			distributed: true,
			nnodes:      2,
		},
		{
			name:        "dp conflicts with deployment pp pin",
			mode:        "dp",
			llmType:     string(api.LLM_CONTAINER_VLLM),
			deploymentPP: 2,
			wantErr:     true,
			wantContain: "parallel_mode=dp 与 pipeline_parallel_size=2 冲突",
		},
		{
			name:        "dp conflicts with sku pp pin",
			mode:        "dp",
			llmType:     string(api.LLM_CONTAINER_VLLM),
			backendParams: []string{"--pipeline-parallel-size=2"},
			wantErr:     true,
			wantContain: "parallel_mode=dp 与 SKU backend_parameters 中的 --pipeline-parallel-size 2 冲突",
		},
		{
			name:        "pp conflicts with sku dp pin",
			mode:        "pp",
			llmType:     string(api.LLM_CONTAINER_VLLM),
			backendParams: []string{"--data-parallel-size=2"},
			wantErr:     true,
			wantContain: "parallel_mode=pp 与 SKU backend_parameters 中的 --data-parallel-size 2 冲突",
		},
		{
			name:    "explicit pins together allowed if no mode conflict",
			mode:    "pp",
			llmType: string(api.LLM_CONTAINER_VLLM),
			backendParams: []string{"--tensor-parallel-size=4"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateDeploymentParallelMode(c.mode, c.llmType, c.backendParams, c.deploymentPP, c.netType, c.distributed, c.nnodes)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if c.wantContain != "" && (err == nil || !strings.Contains(err.Error(), c.wantContain)) {
				t.Fatalf("expected error containing %q, got %v", c.wantContain, err)
			}
		})
	}
}
