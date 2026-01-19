package llm

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

type LLMImageType string

const (
	LLM_IMAGE_TYPE_OLLAMA LLMImageType = "ollama"
	LLM_IMAGE_TYPE_DIFY   LLMImageType = "dify"
)

var (
	LLM_IMAGE_TYPES = sets.NewString(
		string(LLM_IMAGE_TYPE_OLLAMA),
		string(LLM_IMAGE_TYPE_DIFY),
	)
)

func IsLLMImageType(t string) bool {
	return LLM_IMAGE_TYPES.Has(t)
}

type LLMImageListInput struct {
	apis.SharableVirtualResourceListInput

	ImageLabel string `json:"image_label"`
	ImageName  string `json:"image_name"`
	LLMType    string `json:"llm_type"`
}

type LLMImageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    string `json:"image_name"`
	ImageLabel   string `json:"image_label"`
	CredentialId string `json:"credential_id"`
	LLMType      string `json:"llm_type"`
}

type LLMImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName    *string `json:"image_name,omitempty"`
	ImageLabel   *string `json:"image_label,omitempty"`
	CredentialId *string `json:"credential_id,omitempty"`
	LLMType      *string `json:"llm_type,omitempty"`
}
