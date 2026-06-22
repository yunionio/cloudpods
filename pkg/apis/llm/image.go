package llm

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

type LLMImageType string

const (
	LLM_IMAGE_TYPE_OLLAMA       LLMImageType = "ollama"
	LLM_IMAGE_TYPE_VLLM         LLMImageType = "vllm"
	LLM_IMAGE_TYPE_SGLANG       LLMImageType = "sglang"
	LLM_IMAGE_TYPE_DIFY         LLMImageType = "dify"
	LLM_IMAGE_TYPE_COMFYUI      LLMImageType = "comfyui"
	LLM_IMAGE_TYPE_OPENCLAW     LLMImageType = "openclaw"
	LLM_IMAGE_TYPE_HERMES_AGENT LLMImageType = "hermes-agent"
	LLM_IMAGE_TYPE_LLM_ROUTER   LLMImageType = "llm-router"
	LLM_IMAGE_TYPE_DESKTOP      LLMImageType = "desktop"
)

var (
	LLM_IMAGE_TYPES = sets.NewString(
		string(LLM_IMAGE_TYPE_OLLAMA),
		string(LLM_IMAGE_TYPE_VLLM),
		string(LLM_IMAGE_TYPE_SGLANG),
		string(LLM_IMAGE_TYPE_DIFY),
		string(LLM_IMAGE_TYPE_COMFYUI),
		string(LLM_IMAGE_TYPE_OPENCLAW),
		string(LLM_IMAGE_TYPE_HERMES_AGENT),
		string(LLM_IMAGE_TYPE_LLM_ROUTER),
		string(LLM_IMAGE_TYPE_DESKTOP),
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
	AppName    string `json:"app_name"`
}

type LLMImageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName     string                 `json:"image_name"`
	ImageLabel    string                 `json:"image_label"`
	CredentialId  string                 `json:"credential_id"`
	LLMType       string                 `json:"llm_type"`
	AppName       string                 `json:"app_name"`
	DesktopConfig *LLMImageDesktopConfig `json:"desktop_config,omitempty"`
}

type LLMImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName     *string                `json:"image_name,omitempty"`
	ImageLabel    *string                `json:"image_label,omitempty"`
	CredentialId  *string                `json:"credential_id,omitempty"`
	LLMType       *string                `json:"llm_type,omitempty"`
	AppName       *string                `json:"app_name,omitempty"`
	DesktopConfig *LLMImageDesktopConfig `json:"desktop_config,omitempty"`
}
