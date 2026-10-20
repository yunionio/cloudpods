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
	LLM_IMAGE_TYPE_DESKTOP      LLMImageType = "desktop"
	LLM_IMAGE_TYPE_BENCHMARK    LLMImageType = "benchmark"
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
		string(LLM_IMAGE_TYPE_DESKTOP),
		string(LLM_IMAGE_TYPE_BENCHMARK),
	)
)

func IsLLMImageType(t string) bool {
	return LLM_IMAGE_TYPES.Has(t)
}

// LLM image source: how image_name/image_label are provided.
const (
	LLM_IMAGE_SOURCE_MANUAL          = "manual"
	LLM_IMAGE_SOURCE_CONTAINER_IMAGE = "container_image"
)

type LLMImageListInput struct {
	apis.SharableVirtualResourceListInput

	ImageLabel       string `json:"image_label"`
	ImageName        string `json:"image_name"`
	LLMType          string `json:"llm_type"`
	AppName          string `json:"app_name"`
	ImageSource      string `json:"image_source"`
	ContainerImageId string `json:"container_image_id"`
}

type LLMImageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName        string                 `json:"image_name"`
	ImageLabel       string                 `json:"image_label"`
	CredentialId     string                 `json:"credential_id"`
	LLMType          string                 `json:"llm_type"`
	AppName          string                 `json:"app_name"`
	DesktopConfig    *LLMImageDesktopConfig `json:"desktop_config,omitempty"`
	ImageSource      string                 `json:"image_source"`
	ContainerImageId string                 `json:"container_image_id"`
}

type LLMImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName        *string                `json:"image_name,omitempty"`
	ImageLabel       *string                `json:"image_label,omitempty"`
	CredentialId     *string                `json:"credential_id,omitempty"`
	LLMType          *string                `json:"llm_type,omitempty"`
	AppName          *string                `json:"app_name,omitempty"`
	DesktopConfig    *LLMImageDesktopConfig `json:"desktop_config,omitempty"`
	ImageSource      *string                `json:"image_source,omitempty"`
	ContainerImageId *string                `json:"container_image_id,omitempty"`
}

type LLMImageDetails struct {
	apis.SharableVirtualResourceDetails

	// ContainerImage is the linked container_image resource name (for UI link text).
	ContainerImage string `json:"container_image"`
}
