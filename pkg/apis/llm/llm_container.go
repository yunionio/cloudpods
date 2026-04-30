package llm

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

type LLMContainerType string

const (
	LLM_CONTAINER_OLLAMA       LLMContainerType = "ollama"
	LLM_CONTAINER_VLLM         LLMContainerType = "vllm"
	LLM_CONTAINER_DIFY         LLMContainerType = "dify"
	LLM_CONTAINER_COMFYUI      LLMContainerType = "comfyui"
	LLM_CONTAINER_OPENCLAW     LLMContainerType = "openclaw"
	LLM_CONTAINER_HERMES_AGENT LLMContainerType = "hermes-agent"
)

var (
	LLM_CONTAINER_TYPES = sets.NewString(
		string(LLM_CONTAINER_OLLAMA),
		string(LLM_CONTAINER_VLLM),
		string(LLM_CONTAINER_DIFY),
		string(LLM_CONTAINER_COMFYUI),
		string(LLM_CONTAINER_OPENCLAW),
		string(LLM_CONTAINER_HERMES_AGENT),
	)
	LLM_INSTANT_MODEL_TYPES = sets.NewString(
		string(LLM_CONTAINER_OLLAMA),
		string(LLM_CONTAINER_VLLM),
		string(LLM_CONTAINER_COMFYUI),
		string(LLM_CONTAINER_OPENCLAW),
	)
)

func IsLLMContainerType(t string) bool {
	return LLM_CONTAINER_TYPES.Has(t)
}

func IsLLMInstantModelType(t string) bool {
	return LLM_INSTANT_MODEL_TYPES.Has(t)
}

func GetLLMInstantModelContainerType(t LLMContainerType) LLMContainerType {
	return t
}

func IsLLMInstantModelCompatible(instantModelType LLMContainerType, containerType LLMContainerType) bool {
	return GetLLMInstantModelContainerType(instantModelType) == containerType
}

type LLMContainerCreateInput struct {
	apis.VirtualResourceCreateInput
	LLMId string `json:"llm_id"`
	Type  string `json:"type"`
	CmpId string `json:"cmp_id"`
}
