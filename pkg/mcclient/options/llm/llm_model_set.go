package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMModelSetListOptions struct {
	options.BaseListOptions

	Category string   `help:"filter by category" choices:"llm|embedding|reranker|image|speech_to_text|text_to_speech"`
	Backend  string   `help:"filter by backend type — matches sets that have at least one spec with this backend" choices:"ollama|vllm|sglang"`
	SizeMin  *float64 `help:"minimum parameter size in billions" json:"size_min"`
	SizeMax  *float64 `help:"maximum parameter size in billions" json:"size_max"`
}

func (o *LLMModelSetListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMModelSetShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMModelSetShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMModelSetSpecsOptions struct {
	ID string `help:"model set id"`
}

func (o *LLMModelSetSpecsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

func (o *LLMModelSetSpecsOptions) GetId() string { return o.ID }

type LLMModelSpecShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMModelSpecShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMModelSetRefreshOptions struct{}

func (o *LLMModelSetRefreshOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}
