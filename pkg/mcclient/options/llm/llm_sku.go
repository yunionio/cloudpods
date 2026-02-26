package llm

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMSkuListOptions struct {
	options.BaseListOptions

	LLMType string `json:"llm_type" choices:"ollama|comfyui|openclaw"`
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

	MountedModels []string `help:"mounted models, <model_id> e.g. qwen2:0.5b-dup" json:"mounted_models"`

	LLM_IMAGE_ID string `json:"llm_image_id"`
	LLM_TYPE     string `json:"llm_type" choices:"ollama|vllm|comfyui"`

	PreferredModel string `help:"preferred model (vllm only), sets llm_spec.vllm.preferred_model" json:"-"`
}

func (o *LLMSkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if err := o.LLMSkuBaseCreateOptions.Params(dict); err != nil {
		return nil, err
	}
	fetchMountedModels(o.MountedModels, dict)
	if o.LLM_TYPE == string(api.LLM_CONTAINER_VLLM) && len(o.PreferredModel) > 0 {
		spec := &api.LLMSpec{
			Ollama: nil,
			Vllm: &api.LLMSpecVllm{
				PreferredModel: o.PreferredModel,
			},
			Dify: nil,
		}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
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

	MountedModels []string `help:"mounted models, <model_id> e.g. qwen2:0.5b-dup" json:"mounted_models"`

	// For ollama/vllm; backend merges into LLMSpec. Use dify-sku update for dify type.
	LlmImageId string `json:"llm_image_id"`

	PreferredModel string `help:"preferred model (vllm only), sets llm_spec.vllm.preferred_model" json:"-"`
}

func (o *LLMSkuUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMSkuUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if err := o.LLMSkuBaseUpdateOptions.Params(dict); err != nil {
		return nil, err
	}
	fetchMountedModels(o.MountedModels, dict)
	if len(o.PreferredModel) > 0 {
		spec := &api.LLMSpec{
			Ollama: nil,
			Vllm: &api.LLMSpecVllm{
				PreferredModel: o.PreferredModel,
			},
			Dify: nil,
		}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	return dict, nil
}
