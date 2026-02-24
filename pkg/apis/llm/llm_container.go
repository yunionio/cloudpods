package llm

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

type LLMContainerType string

const (
	LLM_CONTAINER_OLLAMA LLMContainerType = "ollama"
	LLM_CONTAINER_VLLM   LLMContainerType = "vllm"
)

var (
	LLM_CONTAINER_TYPES = sets.NewString(
		string(LLM_CONTAINER_OLLAMA),
		string(LLM_CONTAINER_VLLM),
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
