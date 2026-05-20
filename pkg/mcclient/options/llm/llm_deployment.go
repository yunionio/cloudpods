package llm

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMDeploymentListOptions struct {
	options.BaseListOptions

	PlacementStrategy string `help:"filter by placement strategy (spread, binpack)" json:"placement_strategy"`
	AccessPolicy      string `help:"filter by access policy (public, authed, allowed_users)" json:"access_policy"`
	LLMSku            string `help:"filter by SKU id or name" json:"llm_sku"`
}

func (o *LLMDeploymentListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMDeploymentShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMDeploymentShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

// LLMDeploymentCreateOptions supports three modes:
//
//	Mode A: --llm-sku-id <id>                 (reuse existing SKU)
//	Mode B: --sku-* flags                     (auto-create SKU)
//	Mode C: --sku-* + --model-* flags         (auto-import InstantModel + create SKU)
type LLMDeploymentCreateOptions struct {
	options.BaseCreateOptions

	// Mode A: existing SKU
	LLMSkuId string `help:"existing LLM SKU id or name (mode A)" json:"-"`

	// Mode B/C: SKU spec (all sku-* flags collapse into sku_spec)
	SkuName          string   `help:"SKU name (default: <deploy-name>-sku)" json:"-"`
	SkuLLMImageId    string   `help:"container image id (for SkuSpec)" json:"-"`
	SkuLLMType       string   `help:"container llm type" choices:"ollama|vllm|comfyui|sglang|dify" json:"-"`
	SkuCpu           int      `help:"SKU CPU cores" json:"-"`
	SkuMemory        int      `help:"SKU memory MB" json:"-"`
	SkuDiskSize      int      `help:"SKU disk size MB" json:"-"`
	SkuBandwidth     int      `help:"SKU bandwidth (Mb/s)" json:"-"`
	SkuStorageType   string   `help:"SKU storage type, e.g. local" json:"-"`
	SkuTemplateId    string   `help:"SKU storage template id" json:"-"`
	SkuPortMappings  []string `help:"port mapping protocol:port[:prefix][:offset][:envs]; repeatable" json:"-"`
	SkuDevices       []string `help:"device model[:path[:dev_type]]; repeatable" json:"-"`
	SkuEnv           []string `help:"env key=value; repeatable" json:"-"`
	SkuProperty      []string `help:"property key=value; repeatable" json:"-"`
	SkuMountedModels []string `help:"already-imported InstantModel ref, format name:tag; repeatable" json:"-"`
	SkuCategories    string   `help:"comma-separated categories: llm,embedding,image,..." json:"-"`
	SkuBackendVer    string   `help:"inference backend version" json:"-"`

	// Mode C: ModelSpec (auto-import InstantModel before SKU)
	ModelName     string `help:"model name to import (e.g. qwen3)" json:"-"`
	ModelTag      string `help:"model tag (e.g. 8b)" json:"-"`
	ModelLLMType  string `help:"InstantModel llm type" choices:"ollama|vllm|comfyui" json:"-"`
	ModelSource   string `help:"model source: huggingface (default if repo-id is set)" json:"-"`
	ModelRepoId   string `help:"HuggingFace repo id (when source=huggingface)" json:"-"`
	ModelRevision string `help:"model revision (e.g. main)" json:"-"`

	// Instance creation params
	Net        []string `help:"Network descriptions; repeatable" json:"-"`
	AutoStart  bool     `help:"auto start instances after creation" json:"auto_start"`
	PreferHost string   `help:"prefer specific host" json:"prefer_host"`

	// Deployment config
	Replicas             int    `help:"number of replicas" default:"1" json:"replicas"`
	PlacementStrategy    string `help:"placement strategy" choices:"spread|binpack" json:"placement_strategy"`
	CpuOffloading        *bool  `help:"enable CPU offloading" json:"cpu_offloading"`
	DistributedInference *bool  `help:"enable distributed inference" json:"distributed_inference"`
	RestartOnError       *bool  `help:"restart on error" json:"restart_on_error"`
	AccessPolicy         string `help:"access policy" choices:"public|authed|allowed_users" json:"access_policy"`
}

func (o *LLMDeploymentCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)

	// Mode A: reuse existing SKU
	if o.LLMSkuId != "" {
		params.Set("llm_sku_id", jsonutils.NewString(o.LLMSkuId))
	}

	// Mode B/C: build SkuSpec from --sku-* flags
	if o.hasSkuSpec() {
		skuSpec := o.buildSkuSpec()
		params.Set("sku_spec", skuSpec)
	}

	// Mode C: build ModelSpec from --model-* flags
	if o.ModelName != "" {
		modelSpec, err := o.buildModelSpec()
		if err != nil {
			return nil, err
		}
		params.Set("model_spec", jsonutils.Marshal(modelSpec))
	}

	// Network configs
	if len(o.Net) > 0 {
		nets := make([]*computeapi.NetworkConfig, 0)
		for i, n := range o.Net {
			net, err := cmdline.ParseNetworkConfig(n, i)
			if err != nil {
				return nil, errors.Wrapf(err, "parse network config %s", n)
			}
			nets = append(nets, net)
		}
		params.Set("nets", jsonutils.Marshal(nets))
	}

	return params, nil
}

func (o *LLMDeploymentCreateOptions) hasSkuSpec() bool {
	return o.SkuLLMImageId != "" || o.SkuLLMType != "" || o.SkuCpu > 0 || o.SkuMemory > 0 || o.SkuDiskSize > 0
}

func (o *LLMDeploymentCreateOptions) buildSkuSpec() *jsonutils.JSONDict {
	dict := jsonutils.NewDict()

	if o.SkuName != "" {
		dict.Set("name", jsonutils.NewString(o.SkuName))
	}
	if o.SkuLLMImageId != "" {
		dict.Set("llm_image_id", jsonutils.NewString(o.SkuLLMImageId))
	}
	if o.SkuLLMType != "" {
		dict.Set("llm_type", jsonutils.NewString(o.SkuLLMType))
	}
	if o.SkuCpu > 0 {
		dict.Set("cpu", jsonutils.NewInt(int64(o.SkuCpu)))
	}
	if o.SkuMemory > 0 {
		dict.Set("memory", jsonutils.NewInt(int64(o.SkuMemory)))
	}
	if o.SkuBandwidth > 0 {
		dict.Set("bandwidth", jsonutils.NewInt(int64(o.SkuBandwidth)))
	}

	// Disk size → volumes[0]
	if o.SkuDiskSize > 0 {
		vol := api.Volume{
			SizeMB:      o.SkuDiskSize,
			TemplateId:  o.SkuTemplateId,
			StorageType: o.SkuStorageType,
		}
		dict.Set("volumes", jsonutils.Marshal([]api.Volume{vol}))
	}

	// Port mappings (reuse parser from llm_sku_base.go)
	fetchPortmappings(o.SkuPortMappings, dict)
	fetchDevices(o.SkuDevices, dict)
	fetchEnvs(o.SkuEnv, dict)
	fetchProperties(o.SkuProperty, dict)

	if len(o.SkuMountedModels) > 0 {
		dict.Set("mounted_models", jsonutils.Marshal(o.SkuMountedModels))
	}

	if o.SkuCategories != "" {
		cats := jsonutils.NewArray()
		for _, c := range strings.Split(o.SkuCategories, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				cats.Add(jsonutils.NewString(c))
			}
		}
		dict.Set("categories", cats)
	}
	if o.SkuBackendVer != "" {
		dict.Set("backend_version", jsonutils.NewString(o.SkuBackendVer))
	}

	return dict
}

func (o *LLMDeploymentCreateOptions) buildModelSpec() (*api.InstantModelImportInput, error) {
	if o.ModelTag == "" {
		return nil, errors.Error("--model-tag is required for model spec")
	}
	llmType := o.ModelLLMType
	if llmType == "" {
		llmType = o.SkuLLMType
	}
	if llmType == "" {
		return nil, errors.Error("--model-llm-type or --sku-llm-type is required when using --model-name")
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

type LLMDeploymentUpdateOptions struct {
	options.BaseIdOptions

	Name              string `help:"new name" json:"name"`
	Replicas          *int   `help:"number of replicas" json:"replicas"`
	PlacementStrategy string `help:"placement strategy" json:"placement_strategy"`
	AccessPolicy      string `help:"access policy" json:"access_policy"`
}

func (o *LLMDeploymentUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMDeploymentUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMDeploymentDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMDeploymentDeleteOptions) GetId() string {
	return o.ID
}

func (o *LLMDeploymentDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
