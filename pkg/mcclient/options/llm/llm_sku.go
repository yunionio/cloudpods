package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMSkuListOptions struct {
	options.BaseListOptions

	LLMType string `json:"llm_type" choices:"ollama"`
}

func (o *LLMSkuListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMSkuShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMSkuShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMSkuCreateOptions struct {
	LLMSkuBaseCreateOptions

	LLM_IMAGE_ID   string `json:"llm_image_id"`
	LLM_TYPE       string `json:"llm_type" choices:"ollama"`
	LLM_MODEL_NAME string `help:"specific model of large language model, for example: qwen3:32b" json:"llm_model_name"`
}

func (o *LLMSkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMSkuBaseCreateOptions.Params(dict)
	return dict, nil
}

type LLMSkuDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMSkuDeleteOptions) GetId() string {
	return o.ID
}

func (o *LLMSkuDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMSkuUpdateOptions struct {
	LLMSkuBaseUpdateOptions

	MountedModels []string `help:"mounted models, <model_id>@<model_name>:<model_tag> e.g. 6f48b936a09f@qwen2:0.5b" json:"mounted_models"`

	LlmImageId   string
	LlmModelName string
}

func (o *LLMSkuUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMSkuUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMSkuBaseUpdateOptions.Params(dict)
	fetchMountedModels(o.MountedModels, dict)
	return dict, nil
}
