package llm

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

type LLMContainerType string

const (
	LLM_CONTAINER_OLLAMA   LLMContainerType = "ollama"
	LLM_CONTAINER_VLLM     LLMContainerType = "vllm"
	LLM_CONTAINER_DIFY     LLMContainerType = "dify"
	LLM_CONTAINER_COMFYUI  LLMContainerType = "comfyui"
	LLM_CONTAINER_OPENCLAW LLMContainerType = "openclaw"
)

var (
	LLM_CONTAINER_TYPES = sets.NewString(
		string(LLM_CONTAINER_OLLAMA),
		string(LLM_CONTAINER_VLLM),
		string(LLM_CONTAINER_DIFY),
		string(LLM_CONTAINER_COMFYUI),
		string(LLM_CONTAINER_OPENCLAW),
	)
)

func IsLLMContainerType(t string) bool {
	return LLM_CONTAINER_TYPES.Has(t)
}

type LLMContainerCreateInput struct {
	apis.VirtualResourceCreateInput
	LLMId string `json:"llm_id"`
	Type  string `json:"type"`
	CmpId string `json:"cmp_id"`
}
