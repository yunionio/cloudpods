package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/llm/scheduler"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetLLMDeploymentManager()
}

var llmDeploymentManager *SLLMDeploymentManager

func GetLLMDeploymentManager() *SLLMDeploymentManager {
	if llmDeploymentManager != nil {
		return llmDeploymentManager
	}
	llmDeploymentManager = &SLLMDeploymentManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLLMDeployment{},
			"llm_deployments_tbl",
			"llm_deployment",
			"llm_deployments",
		),
	}
	llmDeploymentManager.SetVirtualObject(llmDeploymentManager)
	return llmDeploymentManager
}

type SLLMDeploymentManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SLLMDeployment struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	// Default SKU for creating instances (SKU carries model source, backend, categories).
	// Nullable: may be empty during the brief window when a deployment auto-creates a SKU via SkuSpec.
	LLMSkuId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// SKU automatically created for this deployment and owned by its lifecycle.
	// Empty when the deployment reuses an existing SKU.
	ManagedLLMSkuId string `width:"128" charset:"ascii" nullable:"true" list:"user"`

	// Expected replica count
	Replicas int `nullable:"false" default:"1" list:"user" create:"optional" update:"user"`

	// Ready replica count (computed, periodically synced)
	ReadyReplicas int `nullable:"false" default:"0" list:"user"`

	// Placement strategy: spread or binpack
	PlacementStrategy string `width:"32" charset:"ascii" nullable:"true" default:"spread" list:"user" create:"optional" update:"user"`

	// Allow CPU offloading
	CpuOffloading *bool `nullable:"true" list:"user" create:"optional" update:"user"`

	// Allow distributed inference across hosts
	DistributedInference *bool `nullable:"true" list:"user" create:"optional" update:"user"`

	// Expected number of pods per logical replica: 0 = auto (single host
	// first), >= 2 = force multi-host with exactly this many nodes.
	Nnodes int `nullable:"false" default:"0" list:"user" create:"optional" update:"user"`

	// ParallelMode selects the distributed inference layout for multi-node
	// deployments: "pp" (pipeline parallel, default) or "dp" (data parallel).
	ParallelMode string `width:"8" charset:"ascii" nullable:"true" default:"pp" list:"user" create:"optional" update:"user"`

	// PipelineParallelSize pins the pipeline parallelism (vLLM
	// --pipeline-parallel-size / SGLang --pp-size): 0 = auto, >= 2 = pin.
	// Overrides the SKU backend parameters; valid without
	// distributed_inference (single-node pipeline parallelism).
	PipelineParallelSize int `nullable:"false" default:"0" list:"user" create:"optional" update:"user"`

	// ReconcileFailures counts consecutive distributed replica group creation
	// failures. Automatic reconcile pauses once it reaches
	// MaxDeploymentReconcileFailures so a broken runtime does not churn pods
	// forever; user actions (restart / scale / update) reset it.
	ReconcileFailures int `nullable:"false" default:"0" list:"user"`

	// GPU selector, JSON
	GpuSelector string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`

	// GPU memory utilization fraction passed to inference backend runtime args.
	GpuMemoryUtilization *float64 `nullable:"true" list:"user" create:"optional" update:"user"`

	// Auto-calculate GPU memory utilization from mounted model VRAM and GPU hardware memory.
	AutoGpuMemoryUtilization *bool `nullable:"true" list:"user" create:"optional" update:"user"`

	// Host label selector, JSON
	WorkerSelector string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`

	// Auto-restart on error (pointer so framework can distinguish unset / explicit false)
	RestartOnError *bool `default:"true" list:"user" create:"optional" update:"user"`

	// Extended KV cache config, JSON
	ExtendedKVCache string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`

	// Speculative decoding config, JSON
	SpeculativeConfig string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`

	// Access policy: public, authed, allowed_users
	AccessPolicy string `width:"32" charset:"ascii" nullable:"true" default:"authed" list:"user" create:"optional" update:"user"`

	// Instance template — captured at create time so scale-up can re-create
	// new SLLM replicas with the same network / start / mount settings as the originals.
	// Scale request bodies don't carry these fields.
	Nets        *api.LLMDeploymentNets `charset:"utf8" length:"medium" nullable:"true" list:"user"`
	HostPaths   *api.HostPaths         `charset:"utf8" length:"medium" nullable:"true" list:"user"`
	PreferHosts []string               `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional" update:"user"`

	// Aiproxy integration (llm sync writes aiproxy catalog via mcclient).
	AutoRegisterAiproxy bool                 `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	AiproxyModelPrefix  string               `width:"128" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	AiproxyRoutingId    string               `width:"128" charset:"ascii" nullable:"true" list:"user"`
	AiproxyBindings     *api.AiproxyBindings `charset:"utf8" length:"long" nullable:"true" list:"user"`
	// Gateway registration/sync progress (independent from replica-health status).
	AiproxySyncStatus string `width:"32" charset:"ascii" nullable:"false" default:"disabled" list:"user"`
}

func (man *SLLMDeploymentManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.LLMDeploymentCreateInput,
) (*api.LLMDeploymentCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate VirtualResourceCreateInput")
	}

	// Validate SKU mode: exactly one of LLMSkuId or SkuSpec must be set
	if len(input.LLMSkuId) > 0 && input.SkuSpec != nil {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_sku_id and sku_spec are mutually exclusive")
	}
	if len(input.LLMSkuId) == 0 && input.SkuSpec == nil {
		return input, errors.Wrap(httperrors.ErrMissingParameter, "either llm_sku_id or sku_spec is required")
	}

	switch {
	case len(input.LLMSkuId) > 0:
		// Mode A: reuse existing SKU
		skuObj, err := GetLLMSkuManager().FetchByIdOrName(ctx, userCred, input.LLMSkuId)
		if err != nil {
			return input, errors.Wrapf(err, "fetch LLMSku %s", input.LLMSkuId)
		}
		lSku := skuObj.(*SLLMSku)
		if err := ValidateLLMSkuReadyForUse(lSku); err != nil {
			return input, err
		}
		if err := validateDeploymentGpuMemoryUtilization(input.GpuMemoryUtilization, input.AutoGpuMemoryUtilization, lSku.LLMType); err != nil {
			return input, err
		}
		defaultDeploymentAutoGpuMemoryUtilization(input, lSku, lSku.LLMType)
		disableDeploymentAutoGpuMemoryUtilizationForLocalPath(input, lSku)
		if err := validateLocalPathDeploymentPreferHosts(ctx, userCred, input, lSku); err != nil {
			return input, err
		}
		if SkuHasLocalHostPathModel(lSku) {
			if err := ValidateLocalPathHamiDevicesRequireMemoryMb(lSku.Devices); err != nil {
				return input, err
			}
		}
		if err := ValidateDeploymentDevices(lSku.LLMType, lSku); err != nil {
			return input, err
		}
		input.LLMSkuId = lSku.GetId()
		// ModelSpec is meaningless here
		input.ModelSpec = nil
		if err := validateDeploymentParallelMode(
			input.ParallelMode,
			lSku.LLMType,
			lSku.BackendParameters,
			input.PipelineParallelSize,
			deploymentNetType(input.Nets),
			input.DistributedInference != nil && *input.DistributedInference,
			input.Nnodes,
		); err != nil {
			return input, err
		}

	case input.SkuSpec != nil:
		// Mode B/C: SKU will be created in the task
		// Basic SkuSpec sanity check (name is filled later by task using deployment name)
		if input.SkuSpec.LLMType == "" {
			return input, errors.Wrap(httperrors.ErrMissingParameter, "sku_spec.llm_type is required")
		}
		if input.SkuSpec.Cpu <= 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "sku_spec.cpu must be > 0")
		}
		if input.SkuSpec.Memory <= 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "sku_spec.memory must be > 0")
		}
		if input.ModelSpec != nil {
			// Mode C basic check
			if input.ModelSpec.ModelName == "" || input.ModelSpec.ModelTag == "" {
				return input, errors.Wrap(httperrors.ErrMissingParameter, "model_spec.model_name and model_spec.model_tag are required")
			}
			if input.ModelSpec.LlmType == "" {
				input.ModelSpec.LlmType = api.LLMContainerType(input.SkuSpec.LLMType)
			}
		}
		skuInput := skuFromLLMSkuCreateInput(input.SkuSpec)
		if err := validateDeploymentGpuMemoryUtilization(input.GpuMemoryUtilization, input.AutoGpuMemoryUtilization, input.SkuSpec.LLMType); err != nil {
			return input, err
		}
		defaultDeploymentAutoGpuMemoryUtilization(input, skuInput, input.SkuSpec.LLMType)
		disableDeploymentAutoGpuMemoryUtilizationForLocalPath(input, skuInput)
		if err := validateLocalPathDeploymentPreferHosts(ctx, userCred, input, skuInput); err != nil {
			return input, err
		}
		if SkuHasLocalHostPathModel(skuInput) {
			if err := ValidateLocalPathHamiDevicesRequireMemoryMb(skuInput.Devices); err != nil {
				return input, err
			}
		}
		if err := ValidateDeploymentDevices(input.SkuSpec.LLMType, skuInput); err != nil {
			return input, err
		}
		if err := validateDeploymentParallelMode(
			input.ParallelMode,
			input.SkuSpec.LLMType,
			input.SkuSpec.BackendParameters,
			input.PipelineParallelSize,
			deploymentNetType(input.Nets),
			input.DistributedInference != nil && *input.DistributedInference,
			input.Nnodes,
		); err != nil {
			return input, err
		}
	}

	// Validate placement strategy
	if len(input.PlacementStrategy) > 0 {
		if !api.LLM_MODEL_PLACEMENT_STRATEGIES.Has(input.PlacementStrategy) {
			return input, errors.Wrapf(httperrors.ErrInputParameter,
				"placement_strategy must be one of %s", strings.Join(api.LLM_MODEL_PLACEMENT_STRATEGIES.List(), ", "))
		}
	} else {
		input.PlacementStrategy = api.LLM_MODEL_PLACEMENT_SPREAD
	}

	// Validate access policy
	if len(input.AccessPolicy) > 0 {
		if !api.LLM_MODEL_ACCESS_POLICIES.Has(input.AccessPolicy) {
			return input, errors.Wrapf(httperrors.ErrInputParameter,
				"access_policy must be one of %s", strings.Join(api.LLM_MODEL_ACCESS_POLICIES.List(), ", "))
		}
	} else {
		input.AccessPolicy = api.LLM_MODEL_ACCESS_AUTHED
	}

	// Default replicas
	if input.Replicas <= 0 {
		input.Replicas = 1
	}

	if input.DistributedInference != nil && *input.DistributedInference {
		llmType := ""
		if len(input.LLMSkuId) > 0 {
			if skuObj, err := GetLLMSkuManager().FetchById(input.LLMSkuId); err == nil {
				llmType = skuObj.(*SLLMSku).LLMType
			}
		} else if input.SkuSpec != nil {
			llmType = input.SkuSpec.LLMType
		}
		switch api.LLMContainerType(llmType) {
		case api.LLM_CONTAINER_VLLM, api.LLM_CONTAINER_SGLANG:
		default:
			return input, errors.Wrapf(httperrors.ErrInputParameter,
				"distributed_inference is only supported for vllm and sglang, got %q", llmType)
		}
	}

	if input.Nnodes < 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "nnodes must be >= 0")
	}
	if input.Nnodes == 1 {
		return input, errors.Wrap(httperrors.ErrInputParameter,
			"nnodes=1 is not distributed inference; leave nnodes empty for auto planning or disable distributed_inference")
	}
	if input.Nnodes >= 2 {
		if input.DistributedInference == nil || !*input.DistributedInference {
			return input, errors.Wrap(httperrors.ErrInputParameter,
				"nnodes requires distributed_inference=true")
		}
		gpuCount := 0
		singleGPUPerNode := false
		if len(input.LLMSkuId) > 0 {
			if skuObj, err := GetLLMSkuManager().FetchById(input.LLMSkuId); err == nil {
				sku := skuObj.(*SLLMSku)
				gpuCount = skuDeviceCount(sku)
				singleGPUPerNode = skuForceSingleGPUPerNode(sku)
			}
		} else if input.SkuSpec != nil && input.SkuSpec.Devices != nil {
			gpuCount = len(*input.SkuSpec.Devices)
		}
		if err := validateDeploymentNnodes(input.Nnodes, gpuCount, singleGPUPerNode); err != nil {
			return input, err
		}
	}

	if input.PipelineParallelSize < 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "pipeline_parallel_size must be >= 0")
	}
	if input.PipelineParallelSize == 1 {
		// 1 = a single pipeline stage = the default; normalize to auto.
		input.PipelineParallelSize = 0
	}
	if input.PipelineParallelSize >= 2 {
		llmType := ""
		gpuCount := 0
		if len(input.LLMSkuId) > 0 {
			if skuObj, err := GetLLMSkuManager().FetchById(input.LLMSkuId); err == nil {
				sku := skuObj.(*SLLMSku)
				llmType = sku.LLMType
				gpuCount = skuDeviceCount(sku)
			}
		} else if input.SkuSpec != nil {
			llmType = input.SkuSpec.LLMType
			if input.SkuSpec.Devices != nil {
				gpuCount = len(*input.SkuSpec.Devices)
			}
		}
		if err := validateDeploymentPipelineParallel(llmType, input.PipelineParallelSize, gpuCount); err != nil {
			return input, err
		}
	}

	if input.AutoRegisterAiproxy == nil {
		t := true
		input.AutoRegisterAiproxy = &t
	}
	input.AiproxyModelPrefix = strings.TrimSpace(input.AiproxyModelPrefix)

	input.ParallelMode = api.NormalizeLLMDeploymentParallelMode(input.ParallelMode)

	input.Status = api.STATUS_READY
	return input, nil
}

func (model *SLLMDeployment) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMDeploymentUpdateInput,
) (api.LLMDeploymentUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = model.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate VirtualResourceBaseUpdateInput")
	}

	if input.PlacementStrategy != nil && len(*input.PlacementStrategy) > 0 {
		if !api.LLM_MODEL_PLACEMENT_STRATEGIES.Has(*input.PlacementStrategy) {
			return input, errors.Wrapf(httperrors.ErrInputParameter,
				"placement_strategy must be one of %s", strings.Join(api.LLM_MODEL_PLACEMENT_STRATEGIES.List(), ", "))
		}
	}

	if input.AccessPolicy != nil && len(*input.AccessPolicy) > 0 {
		if !api.LLM_MODEL_ACCESS_POLICIES.Has(*input.AccessPolicy) {
			return input, errors.Wrapf(httperrors.ErrInputParameter,
				"access_policy must be one of %s", strings.Join(api.LLM_MODEL_ACCESS_POLICIES.List(), ", "))
		}
	}

	if input.Replicas != nil && *input.Replicas < 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "replicas must be >= 0")
	}
	if input.AiproxyModelPrefix != nil {
		*input.AiproxyModelPrefix = strings.TrimSpace(*input.AiproxyModelPrefix)
	}

	if input.GpuMemoryUtilization != nil || input.AutoGpuMemoryUtilization != nil {
		llmType := ""
		if model.LLMSkuId != "" {
			skuObj, err := GetLLMSkuManager().FetchById(model.LLMSkuId)
			if err != nil {
				return input, errors.Wrapf(err, "fetch LLMSku %s", model.LLMSkuId)
			}
			llmType = skuObj.(*SLLMSku).LLMType
		}
		if err := validateDeploymentGpuMemoryUtilization(input.GpuMemoryUtilization, input.AutoGpuMemoryUtilization, llmType); err != nil {
			return input, err
		}
	}

	if input.DistributedInference != nil && *input.DistributedInference {
		llmType := ""
		if model.LLMSkuId != "" {
			skuObj, err := GetLLMSkuManager().FetchById(model.LLMSkuId)
			if err != nil {
				return input, errors.Wrapf(err, "fetch LLMSku %s", model.LLMSkuId)
			}
			llmType = skuObj.(*SLLMSku).LLMType
		}
		switch api.LLMContainerType(llmType) {
		case api.LLM_CONTAINER_VLLM, api.LLM_CONTAINER_SGLANG:
		default:
			return input, errors.Wrapf(httperrors.ErrInputParameter,
				"distributed_inference is only supported for vllm and sglang, got %q", llmType)
		}
	}

	if input.Nnodes != nil {
		if *input.Nnodes < 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "nnodes must be >= 0")
		}
		if *input.Nnodes == 1 {
			return input, errors.Wrap(httperrors.ErrInputParameter,
				"nnodes=1 is not distributed inference; clear nnodes for auto planning instead")
		}
		if *input.Nnodes >= 2 {
			enabled := input.DistributedInference != nil && *input.DistributedInference
			if input.DistributedInference == nil && model.DistributedInference != nil {
				enabled = *model.DistributedInference
			}
			if !enabled {
				return input, errors.Wrap(httperrors.ErrInputParameter,
					"nnodes requires distributed_inference=true")
			}
			gpuCount := 0
			singleGPUPerNode := false
			if model.LLMSkuId != "" {
				if skuObj, err := GetLLMSkuManager().FetchById(model.LLMSkuId); err == nil {
					sku := skuObj.(*SLLMSku)
					gpuCount = skuDeviceCount(sku)
					singleGPUPerNode = skuForceSingleGPUPerNode(sku)
				}
			}
			if err := validateDeploymentNnodes(*input.Nnodes, gpuCount, singleGPUPerNode); err != nil {
				return input, err
			}
		}
	}

	if input.PipelineParallelSize != nil {
		if *input.PipelineParallelSize < 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter, "pipeline_parallel_size must be >= 0")
		}
		if *input.PipelineParallelSize == 1 {
			// 1 = a single pipeline stage = the default; normalize to auto.
			zero := 0
			input.PipelineParallelSize = &zero
		}
		if *input.PipelineParallelSize >= 2 {
			llmType := ""
			gpuCount := 0
			if model.LLMSkuId != "" {
				if skuObj, err := GetLLMSkuManager().FetchById(model.LLMSkuId); err == nil {
					sku := skuObj.(*SLLMSku)
					llmType = sku.LLMType
					gpuCount = skuDeviceCount(sku)
				}
			}
			if err := validateDeploymentPipelineParallel(llmType, *input.PipelineParallelSize, gpuCount); err != nil {
				return input, err
			}
		}
	}

	// Validate parallel_mode and its cross-field constraints using effective
	// values (input overrides current model).
	{
		var sku *SLLMSku
		if model.LLMSkuId != "" {
			if skuObj, err := GetLLMSkuManager().FetchById(model.LLMSkuId); err == nil {
				sku = skuObj.(*SLLMSku)
			}
		}
		llmType := ""
		backendParams := []string{}
		if sku != nil {
			llmType = sku.LLMType
			backendParams = sku.BackendParameters
		}
		mode := model.ParallelMode
		if input.ParallelMode != nil {
			mode = *input.ParallelMode
		}
		distributed := false
		if model.DistributedInference != nil {
			distributed = *model.DistributedInference
		}
		if input.DistributedInference != nil {
			distributed = *input.DistributedInference
		}
		nnodes := model.Nnodes
		if input.Nnodes != nil {
			nnodes = *input.Nnodes
		}
		deploymentPP := model.PipelineParallelSize
		if input.PipelineParallelSize != nil {
			deploymentPP = *input.PipelineParallelSize
		}
		netType := ""
		if model.Nets != nil {
			netType = deploymentNetType(*model.Nets)
		}
		if err := validateDeploymentParallelMode(mode, llmType, backendParams, deploymentPP, netType, distributed, nnodes); err != nil {
			return input, err
		}
	}

	// Turning distributed inference off leaves no valid node count: reset it to
	// auto so later replica groups are not planned as multi-host. The pipeline
	// pin is deliberately NOT cleared here: single-node pipeline parallelism
	// stays valid without distributed inference.
	if input.DistributedInference != nil && !*input.DistributedInference {
		if input.Nnodes != nil && *input.Nnodes != 0 {
			return input, errors.Wrap(httperrors.ErrInputParameter,
				"nnodes requires distributed_inference=true; clear nnodes when disabling distributed inference")
		}
		zero := 0
		input.Nnodes = &zero
	}

	if input.ParallelMode != nil {
		mode := api.NormalizeLLMDeploymentParallelMode(*input.ParallelMode)
		input.ParallelMode = &mode
	}

	return input, nil
}

func (man *SLLMDeploymentManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LLMDeploymentListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	if len(input.PlacementStrategy) > 0 {
		q = q.Equals("placement_strategy", input.PlacementStrategy)
	}
	if len(input.AccessPolicy) > 0 {
		q = q.Equals("access_policy", input.AccessPolicy)
	}
	if input.DistributedInference != nil {
		q = q.Equals("distributed_inference", *input.DistributedInference)
	}
	if len(input.LLMSku) > 0 {
		skuObj, err := GetLLMSkuManager().FetchByIdOrName(ctx, userCred, input.LLMSku)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch LLMSku %s", input.LLMSku)
		}
		q = q.Equals("llm_sku_id", skuObj.GetId())
	}
	return q, nil
}

func (man *SLLMDeploymentManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMDeploymentDetails {
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	models := make([]SLLMDeployment, len(objs))
	jsonutils.Update(&models, objs)

	res := make([]api.LLMDeploymentDetails, len(objs))
	for i := range res {
		res[i].VirtualResourceDetails = virtRows[i]
		res[i].Replicas = models[i].Replicas
		res[i].ReadyReplicas = models[i].ReadyReplicas
		res[i].PlacementStrategy = models[i].PlacementStrategy
		res[i].CpuOffloading = models[i].CpuOffloading
		res[i].DistributedInference = models[i].DistributedInference
		res[i].Nnodes = models[i].Nnodes
		res[i].ParallelMode = api.NormalizeLLMDeploymentParallelMode(models[i].ParallelMode)
		res[i].PipelineParallelSize = models[i].PipelineParallelSize
		res[i].GpuMemoryUtilization = models[i].GpuMemoryUtilization
		res[i].AutoGpuMemoryUtilization = models[i].AutoGpuMemoryUtilization
		res[i].RestartOnError = models[i].RestartOnError
		res[i].AccessPolicy = models[i].AccessPolicy
		res[i].AutoRegisterAiproxy = models[i].AutoRegisterAiproxy
		res[i].AiproxyModelPrefix = models[i].AiproxyModelPrefix
		res[i].AiproxyRoutingId = models[i].AiproxyRoutingId
		res[i].AiproxySyncStatus = models[i].AiproxySyncStatus
		res[i].PreferHosts = models[i].PreferHosts
	}

	// Batch fetch SKU data for source/backend/categories info
	skuIds := make([]string, len(models))
	for i, m := range models {
		skuIds[i] = m.LLMSkuId
	}
	skuMap := make(map[string]SLLMSku)
	if err := db.FetchModelObjectsByIds(GetLLMSkuManager(), "id", skuIds, &skuMap); err == nil {
		for i, m := range models {
			res[i].LLMSkuId = m.LLMSkuId
			if sku, ok := skuMap[m.LLMSkuId]; ok {
				res[i].LLMSku = sku.Name
				res[i].Source = sku.Source
				res[i].HuggingfaceRepoId = sku.HuggingfaceRepoId
				res[i].HuggingfaceFilename = sku.HuggingfaceFilename
				res[i].ModelScopeModelId = sku.ModelScopeModelId
				res[i].ModelScopeFilePath = sku.ModelScopeFilePath
				res[i].LocalPath = sku.LocalPath
				res[i].Categories = sku.Categories
				res[i].Backend = sku.LLMType
				res[i].BackendVersion = sku.BackendVersion
				fillDeploymentDerivedParallelism(&res[i], models[i], sku)
			}
		}
	} else {
		log.Errorf("FetchCustomizeColumns fetch SKUs: %s", err)
	}

	// Batch compute running instance count from llms_tbl
	modelIds := make([]string, len(models))
	for i, m := range models {
		modelIds[i] = m.Id
	}
	if len(modelIds) > 0 {
		llmQ := GetLLMManager().Query()
		llmQ = llmQ.In("llm_deployment_id", modelIds)
		llmQ = llmQ.Equals("status", api.LLM_STATUS_RUNNING)
		type countResult struct {
			LLMDeploymentId string `json:"llm_deployment_id"`
			Cnt             int    `json:"cnt"`
		}
		var counts []countResult
		err := llmQ.GroupBy("llm_deployment_id").AppendField(
			llmQ.Field("llm_deployment_id"),
			sqlchemy.COUNT("cnt"),
		).All(&counts)
		if err != nil {
			log.Errorf("FetchCustomizeColumns count running instances: %s", err)
		} else {
			countMap := make(map[string]int, len(counts))
			for _, c := range counts {
				countMap[c.LLMDeploymentId] = c.Cnt
			}
			for i, m := range models {
				if cnt, ok := countMap[m.Id]; ok {
					// RunningInstances is physical (all running SLLM rows, including
					// distributed followers). ReadyReplicas is logical and is kept
					// from SyncReadyReplicas; do not overwrite it with pod counts.
					res[i].RunningInstances = cnt
				}
			}
		}
	}

	// Distributed topology observability: aggregate groups from SLLM rows so
	// the API can expose dist_nnodes, gpus_per_node and per-group rank health.
	distModelIds := make([]string, 0)
	for i := range models {
		if models[i].DistributedInference != nil && *models[i].DistributedInference {
			distModelIds = append(distModelIds, models[i].Id)
		}
	}
	if len(distModelIds) > 0 {
		var rows []distReplicaMemberRow
		err := GetLLMManager().Query(
			"id", "name", "status",
			"dist_role", "dist_rank", "dist_nnodes", "dist_leader_id", "dist_group",
			"cmp_id", "llm_ip", "llm_deployment_id",
		).In("llm_deployment_id", distModelIds).All(&rows)
		if err != nil {
			log.Errorf("FetchCustomizeColumns fetch distributed members: %s", err)
		} else {
			byDeployment := make(map[string][]distReplicaMemberRow)
			cmpIds := make([]string, 0, len(rows))
			for _, r := range rows {
				if !r.isDistributed() {
					continue
				}
				byDeployment[r.LLMDeploymentId] = append(byDeployment[r.LLMDeploymentId], r)
				if r.CmpId != "" {
					cmpIds = append(cmpIds, r.CmpId)
				}
			}
			servers := fetchDeploymentServerMap(ctx, cmpIds)
			for i := range models {
				if models[i].DistributedInference == nil || !*models[i].DistributedInference {
					continue
				}
				members := byDeployment[models[i].Id]
				if len(members) == 0 {
					continue
				}
				maxNodes := 0
				for _, m := range members {
					if m.DistNnodes > maxNodes {
						maxNodes = m.DistNnodes
					}
				}
				// The SKU holds the GPUs of one full logical replica, so the
				// per-node share is total / nnodes for a homogeneous plan.
				totalGpus := 0
				if sku, ok := skuMap[models[i].LLMSkuId]; ok && sku.Devices != nil {
					totalGpus = len(*sku.Devices)
				}
				gpusPerNode := totalGpus
				if maxNodes > 1 && totalGpus > 0 && totalGpus%maxNodes == 0 {
					gpusPerNode = totalGpus / maxNodes
				}
				res[i].DistributedNnodes = maxNodes
				res[i].GpusPerNode = gpusPerNode
				res[i].DistGroups = buildDistGroupInfos(members, totalGpus, servers)
			}
		}
	}

	return res
}

// fetchDeploymentServerMap resolves the compute server rows behind distributed
// replicas so group members can report their host. Failures degrade to empty
// host fields instead of breaking the deployment list.
func fetchDeploymentServerMap(ctx context.Context, ids []string) map[string]computeapi.ServerDetails {
	out := make(map[string]computeapi.ServerDetails)
	if len(ids) == 0 {
		return out
	}
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	params := computeoptions.ServerListOptions{}
	limit := 1000
	params.Limit = &limit
	details := true
	params.Details = &details
	params.Scope = "maxallowed"
	for offset := 0; offset < len(unique); offset += limit {
		last := offset + limit
		if last > len(unique) {
			last = len(unique)
		}
		params.Id = unique[offset:last]
		results, err := compute.Servers.List(s, jsonutils.Marshal(params))
		if err != nil {
			log.Errorf("fetchDeploymentServerMap list servers: %s", err)
			break
		}
		for i := range results.Data {
			guest := computeapi.ServerDetails{}
			if err := results.Data[i].Unmarshal(&guest); err == nil {
				out[guest.Id] = guest
			}
		}
	}
	return out
}

// ValidateDeleteCondition is intentionally permissive.
// CustomizeDelete handles cascade deletion of SLLM instances via task.
func (model *SLLMDeployment) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return nil
}

func (model *SLLMDeployment) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (model *SLLMDeployment) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return model.SVirtualResourceBase.Delete(ctx, userCred)
}

// SyncReadyReplicasOptions configures SyncReadyReplicas behavior.
type SyncReadyReplicasOptions struct {
	// SkipAiproxySync avoids scheduling LLMAiproxySyncTask (e.g. create task uses a child sync instead).
	SkipAiproxySync bool
	// ForceHealthStatus applies replica-health status even when the current
	// value would otherwise be protected (e.g. restarting after the restart task).
	ForceHealthStatus bool
}

// SyncReadyReplicas recomputes ReadyReplicas from SLLM instances, persists it
// to the deployment row, and transitions the deployment status based on replica
// health:
//
//	desired <= 0 && replicas still running or probing → deploying
//	desired <= 0 && no running replicas              → stopped
//	ready_replicas == 0 && desired > 0               → deploying (or create_fail / start_fail)
//	0 < ready_replicas < desired                     → partial
//	ready_replicas == desired                        → ready
//	replica create_fail with 0 running               → create_fail
//	replica start_fail with 0 running                → start_fail
//
// Failure / lifecycle statuses (deleting, importing_model, etc.) are not
// overridden once set. create_fail and start_fail may recover to ready/partial
// when replicas become healthy again (e.g. after syncstatus or auto-restart).
//
// Call after create/scale tasks finish and on every instance status change.
func (model *SLLMDeployment) SyncReadyReplicas(ctx context.Context, userCred mcclient.TokenCredential, opts ...SyncReadyReplicasOptions) error {
	skipAiproxySync := false
	forceHealthStatus := false
	if len(opts) > 0 {
		skipAiproxySync = opts[0].SkipAiproxySync
		forceHealthStatus = opts[0].ForceHealthStatus
	}
	var rows []distReplicaMemberRow
	err := GetLLMManager().Query("id", "name", "status", "dist_role", "dist_rank", "dist_nnodes", "dist_leader_id", "dist_group").
		Equals("llm_deployment_id", model.Id).
		All(&rows)
	if err != nil {
		return errors.Wrap(err, "fetch replica statuses")
	}
	summary := summarizeDistReplicaGroups(rows)
	if model.ReadyReplicas != summary.Running {
		if _, err = db.Update(model, func() error {
			model.ReadyReplicas = summary.Running
			return nil
		}); err != nil {
			return errors.Wrap(err, "update ready_replicas")
		}
	}

	// A deployment with recorded group failures keeps its failure status
	// instead of being flipped back to deploying/partial by later instance
	// churn; the broken group is preserved for debugging. User actions
	// (syncstatus / scale / start / stop) reset the counter or pass
	// ForceHealthStatus to recover.
	if !forceHealthStatus && model.ReconcileFailures > 0 && model.Status == api.LLM_STATUS_CREATE_FAIL {
		return nil
	}

	desired := computeDeploymentReplicaStatus(summary, model.Replicas)
	if desired == "" {
		return nil
	}
	if !forceHealthStatus && !canUpdateReplicaHealthStatus(model.Status, desired) {
		return nil
	}
	oldStatus := model.Status
	if err := model.transitionReplicaHealthStatus(ctx, userCred, desired, summary.Running); err != nil {
		return err
	}
	if !skipAiproxySync && model.AutoRegisterAiproxy && (desired == api.STATUS_READY || desired == api.LLM_DEPLOYMENT_STATUS_PARTIAL) {
		if oldStatus != desired || desired == api.STATUS_READY {
			if err := model.StartAiproxySyncTask(ctx, userCred, "", ""); err != nil {
				log.Warningf("SyncReadyReplicas: start aiproxy sync for %s: %v", model.Name, err)
			}
		}
	}
	return nil
}

func (model *SLLMDeployment) transitionReplicaHealthStatus(ctx context.Context, userCred mcclient.TokenCredential, desired string, running int) error {
	if desired == "" {
		return nil
	}
	if model.Status == desired {
		return nil
	}
	return model.SetStatus(ctx, userCred, desired, fmt.Sprintf("ready_replicas=%d/%d", running, model.Replicas))
}

type deploymentReplicaStatusSummary struct {
	Running          int
	HasCreateFailure bool
	HasStartFailure  bool
	HasProbing       bool
}

func computeDeploymentReplicaStatus(summary deploymentReplicaStatusSummary, desired int) string {
	if desired <= 0 {
		// Scale-to-zero is reconciled once nothing is still coming up.
		// An empty string used to leave StartSyncReplicasTask's "syncing" in place.
		if summary.Running > 0 || summary.HasProbing {
			return api.LLM_DEPLOYMENT_STATUS_DEPLOYING
		}
		return api.LLM_DEPLOYMENT_STATUS_STOPPED
	}
	switch {
	case summary.Running >= desired:
		return api.STATUS_READY
	case summary.Running > 0:
		return api.LLM_DEPLOYMENT_STATUS_PARTIAL
	case summary.HasCreateFailure:
		return api.LLM_STATUS_CREATE_FAIL
	case summary.HasStartFailure:
		return api.LLM_STATUS_START_FAIL
	case summary.HasProbing:
		return api.LLM_DEPLOYMENT_STATUS_DEPLOYING
	default:
		return api.LLM_DEPLOYMENT_STATUS_DEPLOYING
	}
}

func isDeploymentReplicaFailureStatus(status string) bool {
	switch status {
	case api.LLM_STATUS_CREATE_FAIL,
		api.LLM_STATUS_START_FAIL:
		return true
	}
	return false
}

// canUpdateReplicaHealthStatus reports whether replica-health-driven status can
// override the current deployment value. Early lifecycle, restarting, and
// terminal failure / delete states must not be clobbered. create_fail /
// start_fail may recover to ready/partial when replicas are healthy again, or
// to deploying/stopped when there are no running replicas (empty recover).
// Restarting is cleared only by SyncReadyReplicas with ForceHealthStatus after
// the restart task.
func canUpdateReplicaHealthStatus(current, desired string) bool {
	if current == api.LLM_DEPLOYMENT_STATUS_RESTARTING {
		return false
	}
	if current == api.LLM_STATUS_CREATE_FAIL || current == api.LLM_STATUS_START_FAIL {
		switch desired {
		case api.STATUS_READY,
			api.LLM_DEPLOYMENT_STATUS_PARTIAL,
			api.LLM_DEPLOYMENT_STATUS_DEPLOYING,
			api.LLM_DEPLOYMENT_STATUS_STOPPED:
			return true
		default:
			return false
		}
	}
	switch current {
	case api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL,
		api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED,
		api.LLM_DEPLOYMENT_STATUS_CREATING_SKU,
		api.LLM_DEPLOYMENT_STATUS_CREATE_SKU_FAILED,
		api.LLM_STATUS_DELETING,
		api.LLM_STATUS_DELETE_FAILED:
		return false
	}
	return true
}

// TriggerLLMDeploymentReconcile is called after an SLLM instance is deleted.
// If the model still wants more replicas, it starts a sync task to replace the deleted instance.
func TriggerLLMDeploymentReconcile(ctx context.Context, userCred mcclient.TokenCredential, llmDeploymentId string) {
	modelObj, err := GetLLMDeploymentManager().FetchById(llmDeploymentId)
	if err != nil {
		log.Infof("TriggerLLMDeploymentReconcile: model %s not found (may be deleted), skip", llmDeploymentId)
		return
	}
	model := modelObj.(*SLLMDeployment)

	// Refresh ReadyReplicas to reflect the just-changed instance state.
	if err := model.SyncReadyReplicas(ctx, userCred); err != nil {
		log.Errorf("TriggerLLMDeploymentReconcile: SyncReadyReplicas for %s: %s", model.Name, err)
	}

	// Don't reconcile if model is being deleted
	if model.Status == api.LLM_STATUS_DELETING || model.Replicas <= 0 {
		return
	}

	// Count current logical replicas (distributed groups count as one) and
	// detect incomplete groups whose members were partially deleted.
	var rows []distReplicaMemberRow
	if err := GetLLMManager().Query("id", "name", "status", "dist_role", "dist_rank", "dist_nnodes", "dist_leader_id", "dist_group").
		Equals("llm_deployment_id", llmDeploymentId).
		All(&rows); err != nil {
		log.Errorf("TriggerLLMDeploymentReconcile: list instances for model %s: %s", llmDeploymentId, err)
		return
	}
	groups := groupDistReplicaMembers(rows)
	// Broken distributed groups are preserved for debugging instead of being
	// recreated, so an incomplete group alone does not trigger a reconcile;
	// only a missing logical replica does.
	needReconcile := len(groups) < model.Replicas

	if needReconcile {
		log.Infof("TriggerLLMDeploymentReconcile: model %s has %d/%d logical replicas, triggering reconcile", model.Name, len(groups), model.Replicas)
		if err := model.StartSyncReplicasTask(ctx, userCred, nil); err != nil {
			log.Errorf("TriggerLLMDeploymentReconcile: start sync task for model %s: %s", model.Name, err)
		}
	}
}

// PostCreate starts a task to create SLLM instances according to Replicas.
func (model *SLLMDeployment) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	// Persist instance template (nets / auto_start / host_paths) so later scale operations
	// can rebuild instances without these fields in the scale request body.
	input := api.LLMDeploymentCreateInput{}
	_ = data.Unmarshal(&input)
	nets := api.LLMDeploymentNets(input.Nets)
	if _, err := db.Update(model, func() error {
		if len(nets) > 0 {
			model.Nets = &nets
		}
		if input.HostPaths != nil && !input.HostPaths.IsZero() {
			model.HostPaths = input.HostPaths
		}
		if len(input.PreferHosts) > 0 {
			model.PreferHosts = append([]string(nil), input.PreferHosts...)
		}
		if input.AutoRegisterAiproxy != nil {
			model.AutoRegisterAiproxy = *input.AutoRegisterAiproxy
		}
		if input.AiproxyModelPrefix != "" {
			model.AiproxyModelPrefix = strings.TrimSpace(input.AiproxyModelPrefix)
		}
		if model.AutoRegisterAiproxy {
			model.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_PENDING
		} else {
			model.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_DISABLED
		}
		return nil
	}); err != nil {
		log.Errorf("SLLMDeployment.PostCreate persist instance template: %s", err)
	}

	if err := model.StartCreateTask(ctx, userCred, data); err != nil {
		log.Errorf("SLLMDeployment.PostCreate start task failed: %s", err)
		model.SetStatus(ctx, userCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
	}
}

func (model *SLLMDeployment) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	model.SetStatus(ctx, userCred, "creating", "")
	params, _ := data.(*jsonutils.JSONDict)
	if params == nil {
		params = jsonutils.NewDict()
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMDeploymentCreateTask", model, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMDeploymentCreateTask")
	}
	return task.ScheduleRun(nil)
}

func (model *SLLMDeployment) StartSyncReplicasTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	model.SetStatus(ctx, userCred, api.LLM_DEPLOYMENT_STATUS_SYNCING, "")
	params, _ := data.(*jsonutils.JSONDict)
	if params == nil {
		params = jsonutils.NewDict()
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMDeploymentSyncReplicasTask", model, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMDeploymentSyncReplicasTask")
	}
	return task.ScheduleRun(nil)
}

// PostUpdate triggers reconcile when replicas changes.
func (model *SLLMDeployment) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)

	// Any placement-affecting change is an explicit retry: clear the paused
	// reconcile state. Reconcile only (re)creates missing groups; existing
	// healthy groups keep their layout.
	if data.Contains("replicas") || data.Contains("distributed_inference") || data.Contains("nnodes") ||
		data.Contains("parallel_mode") || data.Contains("pipeline_parallel_size") {
		model.ResetReconcileFailures()
		if err := model.StartSyncReplicasTask(ctx, userCred, data); err != nil {
			log.Errorf("SLLMDeployment.PostUpdate start sync replicas task failed: %s", err)
		}
	}
	if model.AutoRegisterAiproxy && (data.Contains("auto_register_aiproxy") || data.Contains("aiproxy_model_prefix")) {
		if err := model.StartAiproxySyncTask(ctx, userCred, "", ""); err != nil {
			log.Errorf("SLLMDeployment.PostUpdate start aiproxy sync task failed: %s", err)
		}
	}
}

// CustomizeDelete starts a cascade delete task that:
//  1. Deletes all child SLLM instances (each via its own LLMDeleteTask, with this task as parent)
//  2. After all instance delete tasks complete, deletes the deployment record itself
func (model *SLLMDeployment) PerformRegisterAiproxy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if _, err := db.Update(model, func() error {
		model.AutoRegisterAiproxy = true
		model.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_PENDING
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "enable auto_register_aiproxy")
	}
	if err := model.StartAiproxySyncTask(ctx, userCred, "", ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (model *SLLMDeployment) PerformUnregisterAiproxy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("unregister", jsonutils.JSONTrue)
	if err := model.startAiproxySyncTaskWithParams(ctx, userCred, params, ""); err != nil {
		return nil, err
	}
	return nil, nil
}

// canRestartDeploymentStatus reports whether a deployment may be restarted.
// force relaxes the allowlist so failed/deploying deployments can still apply SKU updates.
func canRestartDeploymentStatus(status string, force bool) bool {
	switch status {
	case api.LLM_STATUS_DELETING,
		api.LLM_STATUS_START_DELETE,
		api.LLM_STATUS_DELETED,
		api.LLM_STATUS_DELETE_FAILED,
		"creating",
		api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL,
		api.LLM_DEPLOYMENT_STATUS_CREATING_SKU,
		api.LLM_DEPLOYMENT_STATUS_SYNCING,
		api.LLM_DEPLOYMENT_STATUS_RESTARTING:
		return false
	}
	if force {
		return true
	}
	switch status {
	case api.STATUS_READY,
		api.LLM_DEPLOYMENT_STATUS_PARTIAL,
		api.LLM_STATUS_RUNNING:
		return true
	}
	return false
}

func (model *SLLMDeployment) ValidateRestartInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMDeploymentRestartInput,
) error {
	if input == nil {
		input = &api.LLMDeploymentRestartInput{}
	}
	if !canRestartDeploymentStatus(model.Status, input.Force) {
		return httperrors.NewInvalidStatusError("invalid deployment status %s", model.Status)
	}
	var rows []struct {
		Id string `json:"id"`
	}
	err := GetLLMManager().Query("id").Equals("llm_deployment_id", model.Id).All(&rows)
	if err != nil {
		return errors.Wrap(err, "fetch deployment instances")
	}
	if len(rows) == 0 {
		return httperrors.NewInvalidStatusError("no instances under deployment")
	}
	restartable := 0
	instInput := &api.LLMRestartInput{Force: input.Force}
	for i := range rows {
		llmObj, err := GetLLMManager().FetchById(rows[i].Id)
		if err != nil {
			continue
		}
		llm := llmObj.(*SLLM)
		if _, err := llm.ValidateRestartInput(ctx, userCred, instInput); err == nil {
			restartable++
		}
	}
	if restartable == 0 {
		return httperrors.NewInvalidStatusError("no restartable instances")
	}
	return nil
}

func (model *SLLMDeployment) PerformRestart(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.LLMDeploymentRestartInput,
) (jsonutils.JSONObject, error) {
	if input == nil {
		input = &api.LLMDeploymentRestartInput{}
	}
	if err := model.ValidateRestartInput(ctx, userCred, input); err != nil {
		return nil, err
	}
	if err := model.StartRestartTask(ctx, userCred, input.Force); err != nil {
		return nil, errors.Wrap(err, "StartRestartTask")
	}
	return nil, nil
}

func (model *SLLMDeployment) StartRestartTask(ctx context.Context, userCred mcclient.TokenCredential, force bool) error {
	model.SetStatus(ctx, userCred, api.LLM_DEPLOYMENT_STATUS_RESTARTING, "")
	params := jsonutils.NewDict()
	if force {
		params.Set("force", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMDeploymentRestartTask", model, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMDeploymentRestartTask")
	}
	return task.ScheduleRun(nil)
}

// canScaleDeploymentReplicasStatus reports whether start/stop (replica scale) is allowed.
func canScaleDeploymentReplicasStatus(status string) bool {
	switch status {
	case api.LLM_STATUS_DELETING,
		api.LLM_STATUS_START_DELETE,
		api.LLM_STATUS_DELETED,
		api.LLM_STATUS_DELETE_FAILED,
		"creating",
		api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL,
		api.LLM_DEPLOYMENT_STATUS_CREATING_SKU,
		api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED,
		api.LLM_DEPLOYMENT_STATUS_CREATE_SKU_FAILED,
		api.LLM_DEPLOYMENT_STATUS_SYNCING,
		api.LLM_DEPLOYMENT_STATUS_RESTARTING:
		return false
	}
	return true
}

func (model *SLLMDeployment) ValidateStopInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMDeploymentStopInput,
) error {
	_ = ctx
	_ = userCred
	_ = input
	if !canScaleDeploymentReplicasStatus(model.Status) {
		return httperrors.NewInvalidStatusError("invalid deployment status %s", model.Status)
	}
	if model.Replicas <= 0 {
		return httperrors.NewInvalidStatusError("deployment already stopped")
	}
	return nil
}

// PerformStop scales the deployment to zero replicas (GPUStack-style stop).
func (model *SLLMDeployment) PerformStop(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.LLMDeploymentStopInput,
) (jsonutils.JSONObject, error) {
	_ = query
	if input == nil {
		input = &api.LLMDeploymentStopInput{}
	}
	if err := model.ValidateStopInput(ctx, userCred, input); err != nil {
		return nil, err
	}
	if _, err := db.Update(model, func() error {
		model.Replicas = 0
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "set replicas to 0")
	}
	model.ResetReconcileFailures()
	if err := model.StartSyncReplicasTask(ctx, userCred, nil); err != nil {
		return nil, errors.Wrap(err, "StartSyncReplicasTask")
	}
	return nil, nil
}

func (model *SLLMDeployment) ValidateStartInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMDeploymentStartInput,
) error {
	_ = ctx
	_ = userCred
	if !canScaleDeploymentReplicasStatus(model.Status) {
		return httperrors.NewInvalidStatusError("invalid deployment status %s", model.Status)
	}
	if model.Replicas > 0 {
		return httperrors.NewInvalidStatusError("deployment already started")
	}
	if input != nil && input.Replicas != nil && *input.Replicas < 0 {
		return httperrors.NewInputParameterError("replicas must be >= 0")
	}
	return nil
}

// PerformStart scales a stopped deployment up (default 1 replica).
func (model *SLLMDeployment) PerformStart(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.LLMDeploymentStartInput,
) (jsonutils.JSONObject, error) {
	_ = query
	if input == nil {
		input = &api.LLMDeploymentStartInput{}
	}
	if err := model.ValidateStartInput(ctx, userCred, input); err != nil {
		return nil, err
	}
	target := 1
	if input.Replicas != nil && *input.Replicas > 0 {
		target = *input.Replicas
	}
	if _, err := db.Update(model, func() error {
		model.Replicas = target
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "set replicas to %d", target)
	}
	model.ResetReconcileFailures()
	if err := model.StartSyncReplicasTask(ctx, userCred, nil); err != nil {
		return nil, errors.Wrap(err, "StartSyncReplicasTask")
	}
	return nil, nil
}

// canSyncDeploymentStatus reports whether a deployment may sync replica statuses.
func canSyncDeploymentStatus(status string) bool {
	switch status {
	case api.LLM_STATUS_DELETING,
		api.LLM_STATUS_START_DELETE,
		api.LLM_STATUS_DELETED,
		api.LLM_STATUS_DELETE_FAILED,
		"creating",
		api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL,
		api.LLM_DEPLOYMENT_STATUS_CREATING_SKU,
		api.LLM_DEPLOYMENT_STATUS_SYNCING:
		return false
	}
	return true
}

func (model *SLLMDeployment) ValidateSyncstatusInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMDeploymentSyncstatusInput,
) error {
	_ = ctx
	_ = userCred
	_ = input
	if !canSyncDeploymentStatus(model.Status) {
		return httperrors.NewInvalidStatusError("invalid deployment status %s", model.Status)
	}
	var rows []struct {
		Id    string `json:"id"`
		CmpId string `json:"cmp_id"`
	}
	err := GetLLMManager().Query("id", "cmp_id").Equals("llm_deployment_id", model.Id).All(&rows)
	if err != nil {
		return errors.Wrap(err, "fetch deployment instances")
	}
	// No instances: deployment-level sync only (e.g. create_fail / stopped).
	if len(rows) == 0 {
		return nil
	}
	syncable := 0
	for i := range rows {
		if rows[i].CmpId != "" {
			syncable++
		}
	}
	if syncable == 0 {
		return httperrors.NewInvalidStatusError("no syncable instances")
	}
	return nil
}

func (model *SLLMDeployment) countDeploymentInstances() (int, error) {
	return GetLLMManager().Query().Equals("llm_deployment_id", model.Id).CountWithError()
}

func (model *SLLMDeployment) PerformSyncstatus(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.LLMDeploymentSyncstatusInput,
) (jsonutils.JSONObject, error) {
	_ = query
	if input == nil {
		input = &api.LLMDeploymentSyncstatusInput{}
	}
	if err := model.ValidateSyncstatusInput(ctx, userCred, input); err != nil {
		return nil, err
	}
	n, err := model.countDeploymentInstances()
	if err != nil {
		return nil, errors.Wrap(err, "count deployment instances")
	}
	if n == 0 {
		// Recover stuck create_fail / start_fail with no instances: clear the
		// reconcile pause, force status from replica health, and recreate if
		// the deployment still wants replicas.
		model.ResetReconcileFailures()
		if err := model.SyncReadyReplicas(ctx, userCred, SyncReadyReplicasOptions{
			ForceHealthStatus: true,
		}); err != nil {
			return nil, errors.Wrap(err, "SyncReadyReplicas")
		}
		if model.Replicas > 0 {
			if err := model.StartSyncReplicasTask(ctx, userCred, nil); err != nil {
				return nil, errors.Wrap(err, "StartSyncReplicasTask")
			}
		}
		return nil, nil
	}
	if err := model.StartSyncStatusTask(ctx, userCred); err != nil {
		return nil, errors.Wrap(err, "StartSyncStatusTask")
	}
	return nil, nil
}

func (model *SLLMDeployment) StartSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	model.SetStatus(ctx, userCred, api.LLM_DEPLOYMENT_STATUS_SYNCING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMDeploymentSyncStatusTask", model, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMDeploymentSyncStatusTask")
	}
	return task.ScheduleRun(nil)
}

func (model *SLLMDeployment) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := DeleteDeploymentAiproxyResources(ctx, model.Id); err != nil {
		log.Warningf("CustomizeDelete: delete aiproxy resources for %s: %v", model.Name, err)
	}
	// Set replicas to 0 to disable self-healing reconcile during teardown
	if _, err := db.Update(model, func() error {
		model.Replicas = 0
		return nil
	}); err != nil {
		return errors.Wrap(err, "set replicas to 0")
	}

	model.SetStatus(ctx, userCred, api.LLM_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMDeploymentDeleteTask", model, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMDeploymentDeleteTask")
	}
	return task.ScheduleRun(nil)
}

// skuDeviceCount returns the number of GPUs/devices a SKU requests.
func skuDeviceCount(sku *SLLMSku) int {
	if sku == nil || sku.Devices == nil {
		return 0
	}
	return len(*sku.Devices)
}

// validateDeploymentNnodes checks that a pinned node count can be realized for
// a SKU with gpuCount devices. singleGPUPerNode is true for device types that
// allow at most one GPU per pod (Moore Threads HAMI).
func validateDeploymentNnodes(nnodes, gpuCount int, singleGPUPerNode bool) error {
	if nnodes < 2 {
		return nil
	}
	if gpuCount <= 0 {
		return errors.Wrap(httperrors.ErrInputParameter,
			"nnodes >= 2 requires a SKU with GPUs")
	}
	if singleGPUPerNode && nnodes != gpuCount {
		return errors.Wrapf(httperrors.ErrInputParameter,
			"this device type allows one GPU per pod; %d GPUs require nnodes=%d, got %d",
			gpuCount, gpuCount, nnodes)
	}
	if gpuCount%nnodes != 0 {
		return errors.Wrapf(httperrors.ErrInputParameter,
			"cannot split %d GPUs evenly into %d nodes", gpuCount, nnodes)
	}
	return nil
}

// validateDeploymentPipelineParallel checks a deployment-level pipeline pin
// against the SKU. Single-node pipeline parallelism is valid, so there is no
// distributed_inference requirement. vLLM homogeneous layouts always need
// gpuCount % pp == 0; SGLang multi-node layouts only need pp <= nnodes, so
// SGLang divisibility is validated by the planner for the chosen layout
// instead.
func validateDeploymentPipelineParallel(llmType string, pp, gpuCount int) error {
	if pp < 2 {
		return nil
	}
	switch api.LLMContainerType(llmType) {
	case api.LLM_CONTAINER_VLLM, api.LLM_CONTAINER_SGLANG:
	default:
		return errors.Wrapf(httperrors.ErrInputParameter,
			"pipeline_parallel_size is only supported for vllm and sglang, got %q", llmType)
	}
	if gpuCount <= 0 {
		return errors.Wrap(httperrors.ErrInputParameter,
			"pipeline_parallel_size >= 2 requires a SKU with GPUs")
	}
	if pp > gpuCount {
		return errors.Wrapf(httperrors.ErrInputParameter,
			"pipeline_parallel_size %d exceeds the SKU's %d GPUs", pp, gpuCount)
	}
	if api.LLMContainerType(llmType) == api.LLM_CONTAINER_VLLM && gpuCount%pp != 0 {
		return errors.Wrapf(httperrors.ErrInputParameter,
			"cannot split %d GPUs evenly into %d pipeline stages", gpuCount, pp)
	}
	return nil
}

// validateDeploymentParallelMode checks the deployment-level parallel mode and
// its cross-field constraints. It must be called after the SKU is resolved so
// backend parameters and the inference backend are known.
func validateDeploymentParallelMode(
	mode string,
	llmType string,
	backendParams []string,
	deploymentPP int,
	netType string,
	distributed bool,
	nnodes int,
) error {
	mode = api.NormalizeLLMDeploymentParallelMode(mode)

	switch api.LLMContainerType(llmType) {
	case api.LLM_CONTAINER_VLLM, api.LLM_CONTAINER_SGLANG:
	default:
		// Non-distributed backends ignore parallel_mode.
		return nil
	}

	if api.LLMContainerType(llmType) == api.LLM_CONTAINER_SGLANG && mode == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP {
		return errors.Wrap(httperrors.ErrInputParameter,
			"SGLang 不支持数据并行(DP)，请使用 parallel_mode=pp")
	}

	// hostlocal only supports DP across hosts; PP/TP need arbitrary NCCL ports.
	if distributed && mode == api.LLM_DEPLOYMENT_PARALLEL_MODE_PP &&
		netType == string(computeapi.NETWORK_TYPE_HOSTLOCAL) {
		return errors.Wrap(httperrors.ErrInputParameter,
			"hostlocal 网络不支持跨机流水线并行(PP 需要跨机 NCCL)，请设置 parallel_mode=dp 或改用 VPC 网络")
	}

	user := scheduler.ParseUserParallelism(backendParams)

	if mode == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP {
		if deploymentPP >= 2 {
			return errors.Wrapf(httperrors.ErrInputParameter,
				"parallel_mode=dp 与 pipeline_parallel_size=%d 冲突，请移除其一", deploymentPP)
		}
		if user.PP != nil && *user.PP >= 2 {
			return errors.Wrapf(httperrors.ErrInputParameter,
				"parallel_mode=dp 与 SKU backend_parameters 中的 --pipeline-parallel-size %d 冲突，请移除其一", *user.PP)
		}
	}

	if mode == api.LLM_DEPLOYMENT_PARALLEL_MODE_PP {
		if user.DP != nil && *user.DP >= 2 {
			return errors.Wrapf(httperrors.ErrInputParameter,
				"parallel_mode=pp 与 SKU backend_parameters 中的 --data-parallel-size %d 冲突，请移除其一", *user.DP)
		}
	}

	_ = nnodes
	return nil
}

// deploymentNetType returns the effective network type for distributed topology
// validation. An explicit hostlocal net is rejected for PP; anything else
// (guest network or unspecified) is treated as VPC-capable.
func deploymentNetType(nets []*computeapi.NetworkConfig) string {
	if len(nets) == 0 {
		return ""
	}
	return string(nets[0].NetType)
}

// deploymentParallelModeForLLM returns the parallel_mode of an SLLM's
// deployment, normalizing empty values to "pp". It is safe for pod create-time
// checks where the deployment may not exist.
func deploymentParallelModeForLLM(llm *SLLM) string {
	if llm == nil || llm.LLMDeploymentId == "" {
		return api.LLM_DEPLOYMENT_PARALLEL_MODE_PP
	}
	obj, err := GetLLMDeploymentManager().FetchById(llm.LLMDeploymentId)
	if err != nil {
		return api.LLM_DEPLOYMENT_PARALLEL_MODE_PP
	}
	if deploy, ok := obj.(*SLLMDeployment); ok {
		return api.NormalizeLLMDeploymentParallelMode(deploy.ParallelMode)
	}
	return api.LLM_DEPLOYMENT_PARALLEL_MODE_PP
}

// fillDeploymentDerivedParallelism computes the derived TP/PP/DP layout from
// the deployment fields and SKU. It updates the details fields in place.
func fillDeploymentDerivedParallelism(out *api.LLMDeploymentDetails, deployment SLLMDeployment, sku SLLMSku) {
	out.ParallelMode = api.NormalizeLLMDeploymentParallelMode(deployment.ParallelMode)
	totalGPUs := 0
	if sku.Devices != nil {
		totalGPUs = len(*sku.Devices)
	}
	if totalGPUs <= 0 {
		return
	}

	// Use the actual node count from runtime groups when available, otherwise
	// the pinned nnodes. Single-node deployments show no derived layout unless
	// pipeline parallelism is pinned.
	nnodes := deployment.Nnodes
	if out.DistributedNnodes > 0 {
		nnodes = out.DistributedNnodes
	}

	if nnodes >= 2 {
		if totalGPUs%nnodes != 0 {
			return
		}
		gpusPerNode := totalGPUs / nnodes
		gpuPerNode := make([]int, nnodes)
		for i := range gpuPerNode {
			gpuPerNode[i] = gpusPerNode
		}
		user := scheduler.ParseUserParallelism(sku.BackendParameters)
		if deployment.PipelineParallelSize > 0 {
			pp := deployment.PipelineParallelSize
			user.PP = &pp
		}
		if out.ParallelMode == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP && user.DP == nil {
			dp := nnodes
			user.DP = &dp
		}
		scheduler.ApplyDefaultMultinodeParallelism(&user, gpuPerNode)
		if topo, err := scheduler.ValidateMultinodeTopology(gpuPerNode, user); err == nil {
			out.TensorParallelSize = topo.TP
			out.PipelineParallelSize = topo.PP
			out.DataParallelSize = topo.DP
			out.DistShape = string(topo.Shape)
		}
		return
	}

	if deployment.PipelineParallelSize >= 2 {
		pp := deployment.PipelineParallelSize
		if totalGPUs%pp == 0 {
			out.PipelineParallelSize = pp
			out.TensorParallelSize = totalGPUs / pp
			out.DataParallelSize = 1
		}
		return
	}

	// SKU backend parameter pins also produce a derived layout for single-node
	// deployments.
	user := scheduler.ParseUserParallelism(sku.BackendParameters)
	if user.PP != nil && *user.PP >= 2 && totalGPUs%*user.PP == 0 {
		out.PipelineParallelSize = *user.PP
		out.TensorParallelSize = totalGPUs / *user.PP
		out.DataParallelSize = 1
	}
}

// MaxDeploymentReconcileFailures is the number of consecutive distributed
// replica group failures after which automatic reconcile pauses. Without it a
// runtime that always exits would recreate replicas forever.
const MaxDeploymentReconcileFailures = 3

// RecordDeploymentGroupFailure increments the consecutive group failure counter
// and marks the deployment failed once the retry budget is spent.
func RecordDeploymentGroupFailure(ctx context.Context, userCred mcclient.TokenCredential, deploy *SLLMDeployment, cause error) {
	if deploy == nil {
		return
	}
	failures := deploy.ReconcileFailures
	if _, err := db.Update(deploy, func() error {
		deploy.ReconcileFailures++
		failures = deploy.ReconcileFailures
		return nil
	}); err != nil {
		log.Errorf("RecordDeploymentGroupFailure: update %s: %s", deploy.Name, err)
	}
	// Failed members are preserved for debugging, so there is no automatic
	// retry: surface the failure immediately.
	reason := fmt.Sprintf("distributed replica group failed (attempt %d); members kept for debugging", failures)
	if cause != nil {
		reason = fmt.Sprintf("%s: %s", reason, cause.Error())
	}
	log.Warningf("deployment %s group failure #%d: %v", deploy.Name, failures, cause)
	if err := deploy.SetStatus(ctx, userCred, api.LLM_STATUS_CREATE_FAIL, reason); err != nil {
		log.Warningf("RecordDeploymentGroupFailure: set status %s: %s", deploy.Name, err)
	}
}

// ResetReconcileFailures clears the consecutive failure counter. Called after a
// group becomes ready and by user actions that explicitly retry.
func (deploy *SLLMDeployment) ResetReconcileFailures() {
	if deploy == nil || deploy.ReconcileFailures == 0 {
		return
	}
	if _, err := db.Update(deploy, func() error {
		deploy.ReconcileFailures = 0
		return nil
	}); err != nil {
		log.Errorf("ResetReconcileFailures: update %s: %s", deploy.Name, err)
	}
}

// CheckReconcileBudget refuses to create more distributed groups once the
// retry budget is spent.
func (deploy *SLLMDeployment) CheckReconcileBudget() error {
	if deploy != nil && deploy.ReconcileFailures >= MaxDeploymentReconcileFailures {
		return errors.Errorf(
			"automatic reconcile paused after %d consecutive distributed group failures; fix the runtime issue, then restart or scale the deployment",
			deploy.ReconcileFailures)
	}
	return nil
}
