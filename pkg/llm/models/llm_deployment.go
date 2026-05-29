package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	// new SLLM replicas with the same network / start settings as the originals.
	// Scale request bodies don't carry these fields.
	Nets      *api.LLMDeploymentNets `charset:"utf8" length:"medium" nullable:"true" list:"user"`
	AutoStart bool                   `nullable:"false" default:"false" list:"user"`
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
		input.LLMSkuId = lSku.GetId()
		// ModelSpec is meaningless here
		input.ModelSpec = nil

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
		if err := validateDeploymentGpuMemoryUtilization(input.GpuMemoryUtilization, input.AutoGpuMemoryUtilization, input.SkuSpec.LLMType); err != nil {
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
		res[i].GpuMemoryUtilization = models[i].GpuMemoryUtilization
		res[i].AutoGpuMemoryUtilization = models[i].AutoGpuMemoryUtilization
		res[i].RestartOnError = models[i].RestartOnError
		res[i].AccessPolicy = models[i].AccessPolicy
	}

	// Batch fetch SKU data for source/backend/categories info
	skuIds := make([]string, len(models))
	for i, m := range models {
		skuIds[i] = m.LLMSkuId
	}
	skuMap := make(map[string]SLLMSku)
	if err := db.FetchModelObjectsByIds(GetLLMSkuManager(), "id", skuIds, &skuMap); err == nil {
		for i, m := range models {
			if sku, ok := skuMap[m.LLMSkuId]; ok {
				res[i].Source = sku.Source
				res[i].HuggingfaceRepoId = sku.HuggingfaceRepoId
				res[i].HuggingfaceFilename = sku.HuggingfaceFilename
				res[i].ModelScopeModelId = sku.ModelScopeModelId
				res[i].ModelScopeFilePath = sku.ModelScopeFilePath
				res[i].LocalPath = sku.LocalPath
				res[i].Categories = sku.Categories
				res[i].Backend = sku.LLMType
				res[i].BackendVersion = sku.BackendVersion
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
					res[i].RunningInstances = cnt
					res[i].ReadyReplicas = cnt
				}
			}
		}
	}

	return res
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

// SyncReadyReplicas recomputes ReadyReplicas from running SLLM instances,
// persists it to the deployment row, and transitions the deployment status
// based on how many replicas are up:
//
//	ready_replicas == 0 && desired > 0  → deploying
//	0 < ready_replicas < desired        → partial
//	ready_replicas == desired           → ready
//
// Failure / lifecycle statuses (create_fail, deleting, importing_model, etc.)
// are not overridden — the running-state status only takes effect once the
// deployment has cleared the early lifecycle and has at least one instance.
//
// Call after create/scale tasks finish and on every instance status change.
func (model *SLLMDeployment) SyncReadyReplicas(ctx context.Context, userCred mcclient.TokenCredential) error {
	cnt, err := GetLLMManager().Query().
		Equals("llm_deployment_id", model.Id).
		Equals("status", api.LLM_STATUS_RUNNING).
		CountWithError()
	if err != nil {
		return errors.Wrap(err, "count running instances")
	}
	if model.ReadyReplicas != cnt {
		if _, err = db.Update(model, func() error {
			model.ReadyReplicas = cnt
			return nil
		}); err != nil {
			return errors.Wrap(err, "update ready_replicas")
		}
	}

	// Map (ready_replicas, replicas) → desired deployment status.
	desired := computeRunningStatus(cnt, model.Replicas)
	if desired == "" {
		return nil
	}
	if !canFlipToRunningStatus(model.Status) {
		return nil
	}
	if model.Status == desired {
		return nil
	}
	return model.SetStatus(ctx, userCred, desired, fmt.Sprintf("ready_replicas=%d/%d", cnt, model.Replicas))
}

// computeRunningStatus returns the status that reflects the running replica
// count. Returns "" when there's nothing to set (e.g., desired == 0).
func computeRunningStatus(ready, desired int) string {
	if desired <= 0 {
		return ""
	}
	switch {
	case ready == 0:
		return api.LLM_DEPLOYMENT_STATUS_DEPLOYING
	case ready < desired:
		return api.LLM_DEPLOYMENT_STATUS_PARTIAL
	default:
		return api.STATUS_READY
	}
}

// canFlipToRunningStatus reports whether the deployment is in a phase where
// the running-replica-driven status can override the current value. Early
// lifecycle (importing_model, creating_sku) and terminal failure / delete
// states must not be clobbered.
func canFlipToRunningStatus(current string) bool {
	switch current {
	case api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL,
		api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED,
		api.LLM_DEPLOYMENT_STATUS_CREATING_SKU,
		api.LLM_DEPLOYMENT_STATUS_CREATE_SKU_FAILED,
		api.LLM_STATUS_CREATE_FAIL,
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

	// Count current instances
	cnt, err := GetLLMManager().Query().Equals("llm_deployment_id", llmDeploymentId).CountWithError()
	if err != nil {
		log.Errorf("TriggerLLMDeploymentReconcile: count instances for model %s: %s", llmDeploymentId, err)
		return
	}

	if cnt < model.Replicas {
		log.Infof("TriggerLLMDeploymentReconcile: model %s has %d/%d instances, triggering reconcile", model.Name, cnt, model.Replicas)
		if err := model.StartSyncReplicasTask(ctx, userCred, nil); err != nil {
			log.Errorf("TriggerLLMDeploymentReconcile: start sync task for model %s: %s", model.Name, err)
		}
	}
}

// PostCreate starts a task to create SLLM instances according to Replicas.
func (model *SLLMDeployment) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	model.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	// Persist instance template (nets / auto_start) so later scale operations
	// can rebuild instances without these fields in the scale request body.
	input := api.LLMDeploymentCreateInput{}
	_ = data.Unmarshal(&input)
	nets := api.LLMDeploymentNets(input.Nets)
	if _, err := db.Update(model, func() error {
		if len(nets) > 0 {
			model.Nets = &nets
		}
		model.AutoStart = input.AutoStart
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
	model.SetStatus(ctx, userCred, "syncing", "")
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

	if data.Contains("replicas") {
		if err := model.StartSyncReplicasTask(ctx, userCred, data); err != nil {
			log.Errorf("SLLMDeployment.PostUpdate start sync replicas task failed: %s", err)
		}
	}
}

// CustomizeDelete starts a cascade delete task that:
//  1. Deletes all child SLLM instances (each via its own LLMDeleteTask, with this task as parent)
//  2. After all instance delete tasks complete, deletes the deployment record itself
func (model *SLLMDeployment) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
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
