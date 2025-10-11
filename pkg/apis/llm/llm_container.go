package llm

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"yunion.io/x/onecloud/pkg/apis"
)

type LLMContainerType string

const (
	LLM_CONTAINER_OLLAMA = "ollama"
)

var (
	LLM_CONTAINER_TYPES = sets.NewString(
		string(LLM_CONTAINER_OLLAMA),
	)
)

func IsLLMContainerType(t string) bool {
	return LLM_CONTAINER_TYPES.Has(t)
}

type LLMContainerCreateInput struct {
	apis.VirtualResourceCreateInput
	LLMId string `json:"llm_id"`
	Type  string `json:"type"`
	SvrId string `json:"cmp_id"`
}
