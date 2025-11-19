package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMInstantAppListOptions struct {
	options.BaseListOptions

	ModelName []string `help:"filter by model name"`
	Tag       []string `help:"filter by model tag"`
}

func (o *LLMInstantAppListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMInstantAppShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMInstantAppShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMInstantAppCreateOptions struct {
	options.BaseCreateOptions

	MODEL_NAME string `json:"model_name"`
	TAG        string `json:"tag"`

	ImageId string `json:"image_id"`

	Mounts []string `json:"mounts"`
}

func (o *LLMInstantAppCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
