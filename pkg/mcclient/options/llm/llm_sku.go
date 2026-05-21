package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMSkuListOptions struct {
	options.BaseListOptions

	LLMType    string `json:"llm_type" choices:"ollama|vllm|sglang|dify|comfyui|openclaw|hermes-agent"`
	Source     string `json:"source" help:"filter by source (huggingface, model_scope, local_path)"`
	Categories string `json:"categories" help:"filter by category (llm, embedding, image, ...)"`
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
	LLM_TYPE     string `json:"llm_type" choices:"ollama|vllm|sglang|comfyui|hermes-agent"`

	// Model source
	Source              string `help:"model source: huggingface, model_scope, local_path" json:"source"`
	HuggingfaceRepoId   string `help:"HuggingFace repo ID" json:"huggingface_repo_id"`
	HuggingfaceFilename string `help:"HuggingFace filename" json:"huggingface_filename"`
	ModelScopeModelId   string `help:"ModelScope model ID" json:"model_scope_model_id"`
	LocalPath           string `help:"local model path" json:"local_path"`
	// Model metadata
	Categories     string `help:"model categories, comma-separated: llm,embedding,image" json:"-"`
	BackendVersion string `help:"inference backend version" json:"backend_version"`

	PreferredModel string   `help:"preferred model (vllm/sglang), sets llm_spec.<type>.preferred_model" json:"-"`
	VllmArg        []string `help:"vLLM args in format key=value; use key= for flags without values" json:"-"`

	SGLangPreferredModel string   `token:"sglang-preferred-model" help:"SGLang preferred model; overrides preferred-model when llm_type=sglang" json:"-"`
	SGLangArg            []string `token:"sglang-arg" help:"SGLang args in format key=value; use key= for flags without values" json:"-"`

	HermesLLMId         string `token:"hermes-llm-id" help:"target vLLM/SGLang/Ollama LLM id/name for Hermes; resolves llm_spec.hermes_agent.llm_url/model" json:"-"`
	HermesLLMUrl        string `token:"hermes-llm-url" help:"OpenAI-compatible base URL for Hermes, e.g. http://host:8000/v1" json:"-"`
	HermesModel         string `token:"hermes-model" help:"model name for Hermes" json:"-"`
	HermesApiKey        string `token:"hermes-api-key" help:"API key for Hermes custom provider; defaults to EMPTY when omitted" json:"-"`
	HermesContextLength int    `token:"hermes-context-length" help:"Hermes model.context_length" json:"-"`
}

func (o *LLMSkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if err := o.LLMSkuBaseCreateOptions.Params(dict); err != nil {
		return nil, err
	}
	fetchMountedModels(o.MountedModels, dict)
	switch o.LLM_TYPE {
	case string(api.LLM_CONTAINER_VLLM):
		vllmSpec, err := newVLLMSpecFromArgs(o.PreferredModel, o.VllmArg)
		if err != nil {
			return nil, err
		}
		if vllmSpec != nil {
			spec := &api.LLMSpec{
				Ollama: nil,
				Vllm:   vllmSpec,
				Dify:   nil,
			}
			dict.Set("llm_spec", jsonutils.Marshal(spec))
		}
	case string(api.LLM_CONTAINER_SGLANG):
		sglangSpec, err := newSGLangSpecFromArgs(firstNonEmpty(o.SGLangPreferredModel, o.PreferredModel), o.SGLangArg)
		if err != nil {
			return nil, err
		}
		if sglangSpec != nil {
			spec := &api.LLMSpec{SGLang: sglangSpec}
			dict.Set("llm_spec", jsonutils.Marshal(spec))
		}
	case string(api.LLM_CONTAINER_HERMES_AGENT):
		hermesSpec, err := newHermesAgentSpecFromArgs(o.HermesLLMId, o.HermesLLMUrl, o.HermesModel, o.HermesApiKey, o.HermesContextLength)
		if err != nil {
			return nil, err
		}
		if hermesSpec != nil {
			spec := &api.LLMSpec{HermesAgent: hermesSpec}
			dict.Set("llm_spec", jsonutils.Marshal(spec))
		}
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

	// For ollama/vllm/sglang; backend merges into LLMSpec. Use dify-sku update for dify type.
	LlmImageId string `json:"llm_image_id"`

	PreferredModel string   `help:"preferred model (vllm only), sets llm_spec.vllm.preferred_model" json:"-"`
	VllmArg        []string `help:"vLLM args in format key=value; use key= for flags without values" json:"-"`

	SGLangPreferredModel string   `token:"sglang-preferred-model" help:"preferred model (SGLang only), sets llm_spec.sglang.preferred_model" json:"-"`
	SGLangArg            []string `token:"sglang-arg" help:"SGLang args in format key=value; use key= for flags without values" json:"-"`

	HermesLLMId         string `token:"hermes-llm-id" help:"target vLLM/SGLang/Ollama LLM id/name for Hermes; resolves llm_spec.hermes_agent.llm_url/model" json:"-"`
	HermesLLMUrl        string `token:"hermes-llm-url" help:"OpenAI-compatible base URL for Hermes, e.g. http://host:8000/v1" json:"-"`
	HermesModel         string `token:"hermes-model" help:"model name for Hermes" json:"-"`
	HermesApiKey        string `token:"hermes-api-key" help:"API key for Hermes custom provider; defaults to EMPTY when omitted" json:"-"`
	HermesContextLength int    `token:"hermes-context-length" help:"Hermes model.context_length" json:"-"`
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
	vllmSpec, err := newVLLMSpecFromArgs(o.PreferredModel, o.VllmArg)
	if err != nil {
		return nil, err
	}
	sglangSpec, err := newSGLangSpecFromArgs(o.SGLangPreferredModel, o.SGLangArg)
	if err != nil {
		return nil, err
	}
	hermesSpec, err := newHermesAgentSpecFromArgs(o.HermesLLMId, o.HermesLLMUrl, o.HermesModel, o.HermesApiKey, o.HermesContextLength)
	if err != nil {
		return nil, err
	}
	specCount := 0
	if vllmSpec != nil {
		specCount++
	}
	if sglangSpec != nil {
		specCount++
	}
	if hermesSpec != nil {
		specCount++
	}
	if specCount > 1 {
		return nil, errors.Error("cannot specify multiple llm spec args")
	}
	if vllmSpec != nil {
		spec := &api.LLMSpec{
			Ollama: nil,
			Vllm:   vllmSpec,
			Dify:   nil,
		}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	if sglangSpec != nil {
		spec := &api.LLMSpec{SGLang: sglangSpec}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	if hermesSpec != nil {
		spec := &api.LLMSpec{HermesAgent: hermesSpec}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	return dict, nil
}
