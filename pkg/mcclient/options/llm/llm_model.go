package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMModelListOptions struct {
	options.BaseListOptions

	LLMType string `json:"llm_type" choices:"ollama"`
}

func (o *LLMModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMModelShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMModelShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMModelCreateOptions struct {
	LLMModelBaseCreateOptions

	LLM_IMAGE_ID   string `json:"llm_image_id"`
	LLM_TYPE       string `json:"llm_type" choices:"ollama"`
	LLM_MODEL_NAME string `help:"specific model of large language model, for example: qwen3:32b" json:"llm_model_name"`
}

func (o *LLMModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMModelBaseCreateOptions.Params(dict)
	return dict, nil
}

type LLMModelDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMModelDeleteOptions) GetId() string {
	return o.ID
}

func (o *LLMModelDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMModelUpdateOptions struct {
	LLMModelBaseUpdateOptions

	LlmImageId   string
	LlmModelName string
}

func (o *LLMModelUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMModelBaseUpdateOptions.Params(dict)
	return dict, nil
}
