package llm

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMSkuListOptions struct {
	options.BaseListOptions

	LLMType    string `json:"llm_type" choices:"ollama|vllm|sglang|dify|comfyui|openclaw|hermes-agent|llm-router|desktop"`
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
	LLM_TYPE     string `json:"llm_type" choices:"ollama|vllm|sglang|comfyui|hermes-agent|llm-router|desktop"`

	// Model source
	Source              string `help:"model source: huggingface, model_scope, local_path" json:"source"`
	HuggingfaceRepoId   string `help:"HuggingFace repo ID" json:"huggingface_repo_id"`
	HuggingfaceFilename string `help:"HuggingFace filename" json:"huggingface_filename"`
	ModelScopeModelId   string `help:"ModelScope model ID" json:"model_scope_model_id"`
	LocalPath           string `help:"local model path" json:"local_path"`
	// Model metadata
	Categories     string `help:"model categories, comma-separated: llm,embedding,image" json:"-"`
	BackendVersion string `help:"inference backend version" json:"backend_version"`

	// ModelSpec import params. When set, SKU creation starts InstantModel import.
	ModelName     string `help:"model name to import, e.g. Qwen/Qwen3-Embedding-0.6B" json:"-"`
	ModelTag      string `help:"model tag or revision, e.g. main" json:"-"`
	ModelLLMType  string `help:"InstantModel llm type; defaults to llm_type" choices:"ollama|vllm|comfyui|sglang" json:"-"`
	ModelSource   string `help:"model source: huggingface or ollama" json:"-"`
	ModelRepoId   string `help:"HuggingFace repo id, e.g. Qwen/Qwen3-Embedding-0.6B" json:"-"`
	ModelRevision string `help:"HuggingFace revision, e.g. main" json:"-"`

	PreferredModel string   `help:"preferred model (vllm/sglang), sets llm_spec.<type>.preferred_model" json:"-"`
	VllmArg        []string `help:"vLLM args in format key=value; use key= for flags without values" json:"-"`

	SGLangPreferredModel string   `token:"sglang-preferred-model" help:"SGLang preferred model; overrides preferred-model when llm_type=sglang" json:"-"`
	SGLangArg            []string `token:"sglang-arg" help:"SGLang args in format key=value; use key= for flags without values" json:"-"`

	HermesLLMId         string `token:"hermes-llm-id" help:"target vLLM/SGLang/Ollama LLM id/name for Hermes; resolves llm_spec.hermes_agent.llm_url/model" json:"-"`
	HermesLLMUrl        string `token:"hermes-llm-url" help:"OpenAI-compatible base URL for Hermes, e.g. http://host:8000/v1" json:"-"`
	HermesModel         string `token:"hermes-model" help:"model name for Hermes" json:"-"`
	HermesApiKey        string `token:"hermes-api-key" help:"API key for Hermes custom provider; defaults to EMPTY when omitted" json:"-"`
	HermesContextLength int    `token:"hermes-context-length" help:"Hermes model.context_length" json:"-"`

	RouterMethod               string   `token:"router-method" help:"LLM router method, e.g. routerdc" json:"-"`
	RouterConfigPath           string   `token:"router-config-path" help:"LLM router config path inside container" json:"-"`
	RouterModelDir             string   `token:"router-model-dir" help:"LLM router model directory inside container" json:"-"`
	RouterRoutePath            string   `token:"router-route-path" help:"LLM router route API path, e.g. /v1/route" json:"-"`
	RouterHealthPath           string   `token:"router-health-path" help:"LLM router health API path, e.g. /health" json:"-"`
	RouterCandidateMappingPath string   `token:"router-candidate-mapping-path" help:"LLM router candidate mapping path inside container" json:"-"`
	RouterEnv                  []string `token:"router-env" help:"LLM router env in format key=value; repeatable" json:"-"`
}

func (o *LLMSkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if err := o.LLMSkuBaseCreateOptions.Params(dict); err != nil {
		return nil, err
	}
	fetchMountedModels(o.MountedModels, dict)
	if o.ModelName != "" {
		modelSpec, err := o.buildModelSpec()
		if err != nil {
			return nil, err
		}
		dict.Set("model_spec", jsonutils.Marshal(modelSpec))
	}
	if o.Categories != "" {
		cats := jsonutils.NewArray()
		for _, c := range strings.Split(o.Categories, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				cats.Add(jsonutils.NewString(c))
			}
		}
		dict.Set("categories", cats)
	}
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
	case string(api.LLM_CONTAINER_LLM_ROUTER):
		routerSpec, err := newLLMRouterSpecFromArgs(o)
		if err != nil {
			return nil, err
		}
		dict.Set("llm_spec", jsonutils.Marshal(&api.LLMSpec{LLMRouter: routerSpec}))
	}
	return dict, nil
}

func newLLMRouterSpecFromArgs(o *LLMSkuCreateOptions) (*api.LLMSpecLLMRouter, error) {
	method := strings.TrimSpace(o.RouterMethod)
	if method == "" {
		return nil, errors.Error("--router-method is required when llm_type=llm-router")
	}
	envs, err := newLLMRouterEnvsFromArgs(o.RouterEnv)
	if err != nil {
		return nil, err
	}
	return &api.LLMSpecLLMRouter{
		RouterMethod:         method,
		ConfigPath:           strings.TrimSpace(o.RouterConfigPath),
		ModelDir:             strings.TrimSpace(o.RouterModelDir),
		RoutePath:            strings.TrimSpace(o.RouterRoutePath),
		HealthPath:           strings.TrimSpace(o.RouterHealthPath),
		CandidateMappingPath: strings.TrimSpace(o.RouterCandidateMappingPath),
		CustomizedEnvs:       envs,
	}, nil
}

func newLLMRouterEnvsFromArgs(args []string) ([]*api.LLMRouterEnv, error) {
	envs := make([]*api.LLMRouterEnv, 0, len(args))
	for _, arg := range args {
		key, val, ok := strings.Cut(arg, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, errors.Errorf("invalid --router-env %q, expected key=value", arg)
		}
		envs = append(envs, &api.LLMRouterEnv{Key: key, Value: strings.TrimSpace(val)})
	}
	return envs, nil
}

func (o *LLMSkuCreateOptions) buildModelSpec() (*api.InstantModelImportInput, error) {
	if o.ModelTag == "" {
		return nil, errors.Error("--model-tag is required for model spec")
	}
	llmType := o.ModelLLMType
	if llmType == "" {
		llmType = o.LLM_TYPE
	}
	if llmType == "" {
		return nil, errors.Error("--model-llm-type or --llm-type is required when using --model-name")
	}

	input := &api.InstantModelImportInput{
		ModelName: o.ModelName,
		ModelTag:  o.ModelTag,
		LlmType:   api.LLMContainerType(llmType),
		RepoId:    o.ModelRepoId,
		Revision:  o.ModelRevision,
	}
	if o.ModelSource != "" {
		input.Source = o.ModelSource
	} else if o.ModelRepoId != "" {
		input.Source = api.InstantModelSourceHuggingFace
	}
	return input, nil
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

type LLMSkuSchedulableCheckOptions struct {
	options.BaseIdOptions
	GpuCount int `help:"Number of GPUs to validate (default 1)" json:"gpu_count"`
}
