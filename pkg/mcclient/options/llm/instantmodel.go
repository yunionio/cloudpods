package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMInstantModelListOptions struct {
	options.BaseListOptions

	ModelName []string `help:"filter by model name"`
	Tag       []string `help:"filter by model tag"`
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

	MODEL_NAME string `json:"model_name"`
	TAG        string `json:"tag"`

	ImageId string `json:"image_id"`

	Mounts []string `json:"mounts"`
}

func (o *LLMInstantModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
