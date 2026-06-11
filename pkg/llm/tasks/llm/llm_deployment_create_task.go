package llm

import (
	"context"
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/utils/vram"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// LLMDeploymentCreateTask creates SLLM instances after an SLLMDeployment is created.
// It delegates to the shared reconcile logic.
type LLMDeploymentCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMDeploymentCreateTask{})
}

// taskFailed sets a phase-specific failure status on the deployment.
// Use the helpers below (taskFailedImporting, taskFailedCreatingSku, taskFailedGeneric)
// instead of calling this directly.
func (task *LLMDeploymentCreateTask) taskFailed(ctx context.Context, model *models.SLLMDeployment, status string, err error) {
	model.SetStatus(ctx, task.UserCred, status, err.Error())
	db.OpsLog.LogEvent(model, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_CREATE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMDeploymentCreateTask) taskFailedImporting(ctx context.Context, model *models.SLLMDeployment, err error) {
	task.taskFailed(ctx, model, api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED, err)
}

func (task *LLMDeploymentCreateTask) taskFailedCreatingSku(ctx context.Context, model *models.SLLMDeployment, err error) {
	task.taskFailed(ctx, model, api.LLM_DEPLOYMENT_STATUS_CREATE_SKU_FAILED, err)
}

func (task *LLMDeploymentCreateTask) taskFailedGeneric(ctx context.Context, model *models.SLLMDeployment, err error) {
	task.taskFailed(ctx, model, api.LLM_STATUS_CREATE_FAIL, err)
}

func (task *LLMDeploymentCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)

	input := api.LLMDeploymentCreateInput{}
	if err := task.GetParams().Unmarshal(&input); err != nil {
		task.taskFailedGeneric(ctx, model, errors.Wrap(err, "unmarshal LLMDeploymentCreateInput"))
		return
	}

	log.Infof("LLMDeploymentCreateTask.OnInit deployment=%s llm_sku_id=%q sku_spec_set=%v model_spec_set=%v",
		model.Name, model.LLMSkuId, input.SkuSpec != nil, input.ModelSpec != nil)

	switch {
	case input.ModelSpec != nil:
		// Mode C: optionally import InstantModel first, then create SKU.
		// If an enabled InstantModel with the same (llm_type, model_name,
		// model_tag) already exists, skip the import and reuse it.
		if existing, err := models.GetInstantModelManager().FindReadyInstantModel(
			string(input.ModelSpec.LlmType),
			input.ModelSpec.ModelName,
			input.ModelSpec.ModelTag,
		); err != nil {
			log.Warningf("FindReadyInstantModel: %s — proceeding with fresh import", err)
		} else if existing != nil {
			log.Infof("LLMDeploymentCreateTask: reusing existing InstantModel %s (%s) for deployment %s",
				existing.GetId(), existing.GetName(), model.Name)
			task.createSkuAndReconcileWithMountedModel(ctx, model, input.SkuSpec, existing.GetId(), body)
			return
		}
		task.startImportInstantModel(ctx, model, input)
	case input.SkuSpec != nil:
		// Mode B: create SKU then reconcile
		task.createSkuAndReconcile(ctx, model, input.SkuSpec, body)
	default:
		// Mode A: SKU already exists, just reconcile
		if model.LLMSkuId == "" {
			task.taskFailedGeneric(ctx, model, errors.Error("Mode A entered but LLMSkuId is empty (neither sku_spec/model_spec nor llm_sku_id provided in task params)"))
			return
		}
		task.reconcileAndComplete(ctx, model, body)
	}
}

// createSkuAndReconcileWithMountedModel is the Mode C reuse path: an existing
// InstantModel matches the catalog spec, so we skip the import and go straight
// to SKU creation with the mounted-models list populated by the existing id.
func (task *LLMDeploymentCreateTask) createSkuAndReconcileWithMountedModel(
	ctx context.Context,
	model *models.SLLMDeployment,
	skuSpec *api.LLMSkuCreateInput,
	instantModelId string,
	body jsonutils.JSONObject,
) {
	cloned := *skuSpec
	cloned.MountedModels = append([]string{}, cloned.MountedModels...)
	cloned.MountedModels = append(cloned.MountedModels, instantModelId)
	task.createSkuAndReconcile(ctx, model, &cloned, body)
}

func (task *LLMDeploymentCreateTask) reconcileAndComplete(ctx context.Context, model *models.SLLMDeployment, body jsonutils.JSONObject) {
	err := reconcileReplicas(ctx, task.UserCred, model, task.GetParams())
	if err != nil {
		task.taskFailedGeneric(ctx, model, err)
		return
	}
	// Move into the running-state state machine. SyncReadyReplicas picks the
	// concrete status (deploying / partial / ready) based on how many instances
	// are already running. As each SLLM reaches running, its SetStatus override
	// re-runs SyncReadyReplicas, which will eventually flip us to ready.
	model.SetStatus(ctx, task.UserCred, api.LLM_DEPLOYMENT_STATUS_DEPLOYING, "instances created, waiting for running")
	if err := model.SyncReadyReplicas(ctx, task.UserCred, models.SyncReadyReplicasOptions{SkipAiproxySync: model.AutoRegisterAiproxy}); err != nil {
		log.Warningf("LLMDeploymentCreateTask: SyncReadyReplicas for %s: %s", model.Name, err)
	}
	if model.AutoRegisterAiproxy {
		task.SetStage("OnAiproxySyncComplete", nil)
		if err := model.StartAiproxySyncTask(ctx, task.UserCred, "", task.GetTaskId()); err != nil {
			log.Warningf("LLMDeploymentCreateTask: start aiproxy sync for %s: %v", model.Name, err)
			task.SetStageComplete(ctx, nil)
		}
		return
	}
	task.SetStageComplete(ctx, nil)
}

// OnAiproxySyncComplete is called after the child LLMAiproxySyncTask completes.
func (task *LLMDeploymentCreateTask) OnAiproxySyncComplete(ctx context.Context, model *models.SLLMDeployment, body jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

// OnAiproxySyncCompleteFailed is called if the child LLMAiproxySyncTask fails.
func (task *LLMDeploymentCreateTask) OnAiproxySyncCompleteFailed(ctx context.Context, model *models.SLLMDeployment, body jsonutils.JSONObject) {
	log.Warningf("LLMDeploymentCreateTask: aiproxy sync failed for %s: %s", model.Name, body)
	task.SetStageComplete(ctx, nil)
}

// startImportInstantModel starts an async InstantModel import as a child task.
// When the child task finishes, the framework calls OnInstantModelReady.
//
// The returned InstantModel's id is persisted to task params under
// "imported_instant_model_id" so OnInstantModelReady can reference the new
// row by id (the SKU validator's FetchByIdOrName needs an id or name match,
// not the upstream `repo:tag` string).
func (task *LLMDeploymentCreateTask) startImportInstantModel(ctx context.Context, model *models.SLLMDeployment, input api.LLMDeploymentCreateInput) {
	model.SetStatus(ctx, task.UserCred, api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL, "import instant model")

	modelSpec := *input.ModelSpec
	task.SetStage("OnInstantModelReady", nil)

	instantModel, err := models.GetInstantModelManager().DoImportWithParent(ctx, task.UserCred, modelSpec, task.GetTaskId())
	if err != nil {
		task.taskFailedImporting(ctx, model, errors.Wrap(err, "DoImportWithParent"))
		return
	}
	extra := jsonutils.NewDict()
	extra.Set("imported_instant_model_id", jsonutils.NewString(instantModel.GetId()))
	if err := task.SaveParams(extra); err != nil {
		log.Warningf("LLMDeploymentCreateTask: persist imported instant model id: %s", err)
	}
	log.Infof("LLMDeploymentCreateTask: started instant model import id=%s for deployment %s", instantModel.GetId(), model.Name)
}

// OnInstantModelReady is called after the child InstantModel import task completes.
func (task *LLMDeploymentCreateTask) OnInstantModelReady(ctx context.Context, model *models.SLLMDeployment, body jsonutils.JSONObject) {
	input := api.LLMDeploymentCreateInput{}
	if err := task.GetParams().Unmarshal(&input); err != nil {
		task.taskFailedImporting(ctx, model, errors.Wrap(err, "unmarshal create input"))
		return
	}
	if input.SkuSpec == nil || input.ModelSpec == nil {
		task.taskFailedImporting(ctx, model, errors.Error("OnInstantModelReady: missing SkuSpec or ModelSpec"))
		return
	}

	// Reference the just-imported InstantModel by its id (saved by
	// startImportInstantModel). The SKU validator resolves mounted_models via
	// FetchByIdOrName, so a literal id is the most reliable form.
	instantId, _ := task.GetParams().GetString("imported_instant_model_id")
	if instantId == "" {
		task.taskFailedImporting(ctx, model, errors.Error("OnInstantModelReady: missing imported_instant_model_id in task params"))
		return
	}

	// Auto-enable the freshly-imported InstantModel — without this it stays
	// disabled and FindReadyInstantModel can't dedup it on the next catalog
	// deploy, plus the SKU/LLM mount validators depend on enabled rows.
	if err := models.EnableInstantModelForUse(ctx, task.UserCred, instantId); err != nil {
		log.Warningf("LLMDeploymentCreateTask: enable InstantModel %s: %s", instantId, err)
	}

	skuSpec := *input.SkuSpec
	skuSpec.MountedModels = append(skuSpec.MountedModels, instantId)

	task.createSkuAndReconcile(ctx, model, &skuSpec, task.GetParams())
}

// OnInstantModelReadyFailed is called if the child InstantModel import task fails.
func (task *LLMDeploymentCreateTask) OnInstantModelReadyFailed(ctx context.Context, model *models.SLLMDeployment, body jsonutils.JSONObject) {
	task.taskFailedImporting(ctx, model, fmt.Errorf("InstantModel import failed: %s", body))
}

// createSkuAndReconcile synchronously creates a SKU via the standard framework
// dispatcher, then writes its ID back to the deployment and runs reconcile.
func (task *LLMDeploymentCreateTask) createSkuAndReconcile(ctx context.Context, model *models.SLLMDeployment, skuSpec *api.LLMSkuCreateInput, body jsonutils.JSONObject) {
	model.SetStatus(ctx, task.UserCred, api.LLM_DEPLOYMENT_STATUS_CREATING_SKU, "create sku")

	// Default SKU name from deployment name if not provided
	if skuSpec.Name == "" {
		skuSpec.Name = fmt.Sprintf("%s-sku", model.Name)
	}

	// Auto-fill VramClaimMb from the largest mounted InstantModel's
	// weight_size_bytes. User-provided non-zero values are respected.
	if skuSpec.VramClaimMb == 0 {
		var maxWeight int64
		for _, id := range skuSpec.MountedModels {
			obj, err := models.GetInstantModelManager().FetchById(id)
			if err != nil {
				continue
			}
			if w := obj.(*models.SInstantModel).WeightSizeBytes; w > maxWeight {
				maxWeight = w
			}
		}
		if maxWeight > 0 {
			skuSpec.VramClaimMb = vram.EstimateClaimMb(maxWeight, skuSpec.LLMType)
			log.Infof("LLMDeploymentCreateTask: auto vram_claim_mb=%d for sku=%s (weight=%d bytes, llm_type=%s)",
				skuSpec.VramClaimMb, skuSpec.Name, maxWeight, skuSpec.LLMType)
		}
	}

	skuParams := jsonutils.Marshal(skuSpec).(*jsonutils.JSONDict)
	// __meta__ from VirtualResourceCreateInput marshals as null when Metadata is nil;
	// keep params clean to avoid surprises in the framework dispatcher.
	skuParams.Remove("__meta__")
	createCtx := context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, task.UserCred)
	handler := db.NewModelHandler(models.GetLLMSkuManager())

	log.Infof("LLMDeploymentCreateTask.createSkuAndReconcile deployment=%s skuParams=%s", model.Name, skuParams.String())
	// IMPORTANT: query must not be nil — FetchCustomizeColumns calls query.Contains() which panics on nil.
	resp, err := handler.Create(createCtx, jsonutils.NewDict(), skuParams, nil)
	if err != nil {
		task.taskFailedCreatingSku(ctx, model, errors.Wrapf(err, "create SKU (params=%s)", skuParams.String()))
		return
	}
	log.Infof("LLMDeploymentCreateTask.createSkuAndReconcile SKU create response: %s", resp.String())

	skuId, _ := resp.GetString("id")
	if skuId == "" {
		task.taskFailedCreatingSku(ctx, model, errors.Errorf("create SKU response missing id: %s", resp.String()))
		return
	}
	skuObj, err := models.GetLLMSkuManager().FetchById(skuId)
	if err != nil {
		task.taskFailedCreatingSku(ctx, model, errors.Wrapf(err, "fetch created SKU %s", skuId))
		return
	}
	if err := models.ValidateLLMSkuReadyForUse(skuObj.(*models.SLLMSku)); err != nil {
		task.taskFailedCreatingSku(ctx, model, err)
		return
	}

	if _, err := db.Update(model, func() error {
		return assignDeploymentAutoCreatedSku(model, skuId)
	}); err != nil {
		task.taskFailedCreatingSku(ctx, model, errors.Wrap(err, "write back llm_sku_id"))
		return
	}

	log.Infof("LLMDeploymentCreateTask: created SKU %s for deployment %s (deployment.LLMSkuId now=%q)", skuId, model.Name, model.LLMSkuId)
	task.reconcileAndComplete(ctx, model, body)
}

func assignDeploymentAutoCreatedSku(model *models.SLLMDeployment, skuId string) error {
	model.LLMSkuId = skuId
	model.ManagedLLMSkuId = skuId
	return nil
}

// reconcileReplicas is the core logic that ensures actual SLLM count matches desired replicas.
// - If actual < desired: create new instances (scale up)
// - If actual > desired: delete excess instances (scale down, prefer unhealthy)
// - If actual == desired: nothing to do
//
// `body` is the original task body (JSON dict) that contains the deployment create input,
// including network config. We forward it to SLLM creation as-is to preserve all fields.
func reconcileReplicas(ctx context.Context, userCred mcclient.TokenCredential, deploy *models.SLLMDeployment, params jsonutils.JSONObject) error {
	desired := deploy.Replicas
	if desired < 0 {
		desired = 0
	}

	// Fetch current instances
	instances, err := fetchModelInstances(deploy.Id)
	if err != nil {
		return fmt.Errorf("fetch instances: %w", err)
	}
	actual := len(instances)

	if actual < desired {
		// Scale up
		return scaleUp(ctx, userCred, deploy, params, instances, desired-actual)
	} else if actual > desired {
		// Scale down
		return scaleDown(ctx, userCred, deploy, instances, actual-desired)
	}

	return nil
}

type instanceInfo struct {
	Id     string
	Name   string
	Status string
}

func fetchModelInstances(modelId string) ([]instanceInfo, error) {
	q := models.GetLLMManager().Query("id", "name", "status").Equals("llm_deployment_id", modelId)
	var rows []instanceInfo
	err := q.All(&rows)
	return rows, err
}

// scaleUp creates `count` new SLLM instances via the standard DBModelDispatcher.Create flow.
func scaleUp(ctx context.Context, userCred mcclient.TokenCredential, deployment *models.SLLMDeployment, params jsonutils.JSONObject, existing []instanceInfo, count int) error {
	if deployment.LLMSkuId == "" {
		return fmt.Errorf("scaleUp: deployment %s (id=%s) has empty LLMSkuId", deployment.Name, deployment.Id)
	}
	sku, err := models.GetLLMSkuManager().FetchById(deployment.LLMSkuId)
	if err != nil {
		return fmt.Errorf("fetch SKU %s: %w", deployment.LLMSkuId, err)
	}
	llmSku := sku.(*models.SLLMSku)
	imageId := llmSku.GetLLMImageId()

	// Find the next index to use
	nextIndex := len(existing)
	for _, inst := range existing {
		var idx int
		if _, err := fmt.Sscanf(inst.Name, deployment.Name+"-%d", &idx); err == nil {
			if idx >= nextIndex {
				nextIndex = idx + 1
			}
		}
	}

	// Instance template (nets / auto_start / host_paths) lives on the deployment row,
	// captured at create time by SLLMDeployment.PostCreate. Scale request
	// bodies don't carry these fields, so the row is the source of truth.
	if deployment.Nets == nil || len(*deployment.Nets) == 0 {
		return fmt.Errorf("scaleUp: deployment %s has no stored nets template", deployment.Name)
	}
	var nets []*computeapi.NetworkConfig = *deployment.Nets

	// Inject userCred into ctx so DBModelDispatcher.Create can fetch it
	createCtx := context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
	handler := db.NewModelHandler(models.GetLLMManager())

	llmSpec, err := models.BuildDeploymentResolvedGpuMemoryLLMSpec(ctx, userCred, deployment, llmSku)
	if err != nil {
		return fmt.Errorf("resolve GPU memory utilization: %w", err)
	}

	var lastErr error
	created := 0
	for i := 0; i < count; i++ {
		instanceName := fmt.Sprintf("%s-%d", deployment.Name, nextIndex+i)

		// Build typed LLMCreateInput, then marshal to JSON for handler.Create
		llmInput := buildDeploymentLLMCreateInput(deployment, nets, imageId, llmSpec)

		llmParams := jsonutils.Marshal(llmInput).(*jsonutils.JSONDict)
		// Name lives on the embedded VirtualResourceCreateInput → set explicitly
		llmParams.Set("name", jsonutils.NewString(instanceName))
		llmParams.Remove("generate_name")

		// Use DBModelDispatcher.Create — the full framework flow:
		// ValidateCreateData → CustomizeCreate → Insert → PostCreate → OnCreateComplete
		_, err := handler.Create(createCtx, jsonutils.NewDict(), llmParams, nil)
		if err != nil {
			log.Errorf("reconcile: scale up instance %s failed: %s", instanceName, err)
			lastErr = err
			continue
		}
		created++
		log.Infof("reconcile: created LLM instance %s for model %s", instanceName, deployment.Name)
	}

	if created == 0 && lastErr != nil {
		return fmt.Errorf("failed to create any instance: %w", lastErr)
	}
	return nil
}

func buildDeploymentLLMCreateInput(deployment *models.SLLMDeployment, nets []*computeapi.NetworkConfig, imageId string, llmSpec *api.LLMSpec) api.LLMCreateInput {
	return api.LLMCreateInput{
		LLMBaseCreateInput: api.LLMBaseCreateInput{
			AutoStart: deployment.AutoStart,
			Nets:      nets,
		},
		LLMSkuId:        deployment.LLMSkuId,
		LLMImageId:      imageId,
		LLMDeploymentId: deployment.Id,
		LLMSpec:         llmSpec,
		HostPaths:       deployment.HostPaths,
	}
}

// scaleDown deletes `count` SLLM instances, preferring unhealthy ones.
func scaleDown(ctx context.Context, userCred mcclient.TokenCredential, model *models.SLLMDeployment, instances []instanceInfo, count int) error {
	// Sort: unhealthy/error/stopped first, then by name descending (newest first)
	sort.Slice(instances, func(i, j int) bool {
		pi := instancePriority(instances[i].Status)
		pj := instancePriority(instances[j].Status)
		if pi != pj {
			return pi < pj // lower priority = delete first
		}
		return instances[i].Name > instances[j].Name // newer name = delete first
	})

	deleted := 0
	for i := 0; i < count && i < len(instances); i++ {
		inst := instances[i]
		llmObj, err := models.GetLLMManager().FetchById(inst.Id)
		if err != nil {
			log.Errorf("reconcile: fetch instance %s for deletion: %s", inst.Id, err)
			continue
		}
		llm := llmObj.(*models.SLLM)
		if err := llm.StartDeleteTask(ctx, userCred, ""); err != nil {
			log.Errorf("reconcile: delete instance %s failed: %s", inst.Id, err)
			continue
		}
		deleted++
		log.Infof("reconcile: deleting LLM instance %s (%s) for model %s scale-down", inst.Name, inst.Id, model.Name)
	}

	if deleted == 0 && count > 0 {
		return fmt.Errorf("failed to delete any instance")
	}
	return nil
}

// instancePriority returns a sort priority for scale-down.
// Lower value = delete first.
func instancePriority(status string) int {
	switch status {
	case api.LLM_STATUS_CREATE_FAIL, api.LLM_STATUS_DELETE_FAILED:
		return 0 // delete failed/create-failed instances first
	case api.LLM_STATUS_UNKNOWN, api.LLM_LLM_STATUS_NO_SERVER, api.LLM_LLM_STATUS_NO_CONTAINER:
		return 1 // then broken instances
	case api.LLM_STATUS_READY:
		return 2 // then stopped instances
	case api.LLM_STATUS_RUNNING:
		return 3 // keep running instances last
	default:
		return 1
	}
}
