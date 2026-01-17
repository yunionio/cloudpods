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

	LLMType string `json:"llm_type" choices:"ollama|dify" help:"filter by llm type"`
}

func (o *LLMImageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMImageCreateOptions struct {
	apis.SharableVirtualResourceCreateInput
	IMAGE_NAME   string `json:"image_name"`
	IMAGE_LABEL  string `json:"image_label"`
	CredentialId string `json:"credential_id"`
	LLM_TYPE     string `json:"llm_type" choices:"ollama|dify" help:"llm type: ollama or dify"`
}

func (o *LLMImageCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMImageUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ID           string
	ImageName    string `json:"image_name"`
	ImageLabel   string `json:"image_label"`
	CredentialId string `json:"credential_id"`
	LlmType      string `json:"llm_type" choices:"ollama|dify" help:"llm type: ollama or dify"`
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
