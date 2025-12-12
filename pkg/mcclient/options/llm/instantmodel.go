package llm

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMInstantModelListOptions struct {
	options.BaseListOptions

	ModelName []string `help:"filter by model name"`
	ModelTag  []string `help:"filter by model tag"`
}

func (o *LLMInstantModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMInstantModelShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMInstantModelShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMInstantModelCreateOptions struct {
	options.BaseCreateOptions

	LLM_TYPE   string `help:"llm container type" choices:"ollama" json:"llm_type"`
	MODEL_NAME string `json:"model_name"`
	MODEL_TAG  string `json:"model_tag"`

	ImageId string `json:"image_id"`

	Mounts []string `json:"mounts"`
}

func (o *LLMInstantModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMInstantModelUpdateOptions struct {
	options.BaseIdOptions

	ImageId string `json:"image_id"`

	Mounts []string `json:"mounts"`
}

func (o *LLMInstantModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMInstantModelDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMInstantModelDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMInstantModelImportOptions struct {
	LLM_TYPE   string `help:"llm container type" choices:"ollama" json:"llm_type"`
	MODEL_NAME string `help:"model name to import, e.g. qwen3" json:"model_name"`
	MODEL_TAG  string `help:"model tag to import, e.g. 8b" json:"model_tag"`
}

func (o *LLMInstantModelImportOptions) Params() (jsonutils.JSONObject, error) {
	input := api.InstantModelImportInput{
		ModelName: o.MODEL_NAME,
		ModelTag:  o.MODEL_TAG,
		LlmType:   api.LLMContainerType(o.LLM_TYPE),
	}
	return jsonutils.Marshal(input), nil
}
