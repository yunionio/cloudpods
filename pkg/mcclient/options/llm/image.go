package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMImageShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMImageShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMImageListOptions struct {
	options.BaseListOptions

	LLMType     string `json:"llm_type" choices:"ollama|dify|comfyui|vllm|sglang|hermes-agent|desktop|benchmark" help:"filter by llm type"`
	ImageName   string `json:"image_name" help:"filter by image name (substring match)"`
	ImageSource string `json:"image_source" choices:"manual|container_image" help:"filter by image source"`
}

func (o *LLMImageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMImageCreateOptions struct {
	apis.SharableVirtualResourceCreateInput
	IMAGE_NAME       string `json:"image_name"`
	IMAGE_LABEL      string `json:"image_label"`
	CredentialId     string `json:"credential_id"`
	LLM_TYPE         string `json:"llm_type" choices:"ollama|dify|comfyui|vllm|sglang|hermes-agent|desktop|benchmark" help:"llm type: ollama, comfyui, vllm, sglang, hermes-agent, desktop, benchmark or dify"`
	ImageSource      string `json:"image_source" choices:"manual|container_image" help:"image source: manual or container_image"`
	ContainerImageId string `json:"container_image_id" help:"linked container_image id when image_source is container_image"`
}

func (o *LLMImageCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMImageUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ID               string
	ImageName        string `json:"image_name"`
	ImageLabel       string `json:"image_label"`
	CredentialId     string `json:"credential_id"`
	LlmType          string `json:"llm_type" choices:"ollama|dify|vllm|sglang|comfyui|hermes-agent|desktop|benchmark" help:"llm type: ollama, comfyui, vllm, sglang, hermes-agent, desktop, benchmark or dify"`
	AppName          string `json:"app_name" help:"desktop application identifier (LinuxServer image id), e.g. firefox, chromium, steam, webtop-ubuntu-xfce"`
	ImageSource      string `json:"image_source" choices:"manual|container_image" help:"image source: manual or container_image"`
	ContainerImageId string `json:"container_image_id" help:"linked container_image id when image_source is container_image"`
}

func (o *LLMImageUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMImageUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMImageDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMImageDeleteOptions) GetId() string {
	return o.ID
}
