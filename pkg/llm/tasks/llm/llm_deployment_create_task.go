package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
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
		// model_tag, source) already exists, skip the import and reuse it.
		if existing, err := models.GetInstantModelManager().FindReadyInstantModel(
			string(input.ModelSpec.LlmType),
			input.ModelSpec.ModelName,
			input.ModelSpec.ModelTag,
			input.ModelSpec.Source,
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

// reconcileReplicas ensures the number of logical replicas matches deploy.Replicas.
// A logical replica is either one single-pod SLLM or one distributed group.
func reconcileReplicas(ctx context.Context, userCred mcclient.TokenCredential, deploy *models.SLLMDeployment, params jsonutils.JSONObject) error {
	desired := deploy.Replicas
	if desired < 0 {
		desired = 0
	}

	instances, err := fetchModelInstances(deploy.Id)
	if err != nil {
		return fmt.Errorf("fetch instances: %w", err)
	}
	groups := groupInstanceInfos(instances)

	// Distributed groups that failed or are structurally incomplete are kept
	// as-is so the failed pods (logs, container state, placement) can be
	// inspected. They are only removed by an explicit scale-down or deployment
	// deletion. Single-pod groups keep the self-healing delete+recreate path.
	healthy := make([]replicaGroupInfo, 0, len(groups))
	preserved := make([]replicaGroupInfo, 0)
	for _, g := range groups {
		if replicaGroupHasFailure(g) || !replicaGroupComplete(g) {
			if g.Group >= 0 {
				log.Warningf("reconcile: preserve broken distributed group %s for debugging; skip automatic delete/recreate", g.Key)
				preserved = append(preserved, g)
				continue
			}
			if err := deleteReplicaGroup(ctx, userCred, deploy, g); err != nil {
				log.Errorf("reconcile: delete incomplete group %s: %s", g.Key, err)
				healthy = append(healthy, g)
			}
			continue
		}
		healthy = append(healthy, g)
	}

	// Preserved groups still occupy their replica slots: never create more
	// groups on top of a broken one, but allow explicit scale-down to clean up.
	actual := len(healthy) + len(preserved)
	if actual < desired && len(preserved) > 0 {
		log.Warningf("reconcile: %s has %d broken replicated group(s) preserved; not creating new replicas until they are cleaned up", deploy.Name, len(preserved))
		return nil
	}
	if actual < desired {
		return scaleUp(ctx, userCred, deploy, params, instances, desired-actual)
	} else if actual > desired {
		return scaleDown(ctx, userCred, deploy, append(healthy, preserved...), actual-desired)
	}

	return nil
}

func replicaGroupComplete(g replicaGroupInfo) bool {
	if g.Group < 0 {
		return len(g.Members) == 1
	}
	expected := 0
	leaders := 0
	ranks := make(map[int]struct{}, len(g.Members))
	for _, m := range g.Members {
		if m.DistNnodes > expected {
			expected = m.DistNnodes
		}
		switch m.DistRole {
		case api.LLM_DIST_ROLE_LEADER:
			leaders++
			if m.DistRank != 0 {
				return false
			}
		case api.LLM_DIST_ROLE_FOLLOWER:
			if m.DistRank <= 0 {
				return false
			}
		default:
			return false
		}
		if _, dup := ranks[m.DistRank]; dup {
			return false
		}
		ranks[m.DistRank] = struct{}{}
	}
	return expected >= 2 && len(g.Members) == expected && leaders == 1
}

func replicaGroupHasFailure(g replicaGroupInfo) bool {
	for _, m := range g.Members {
		switch m.Status {
		case api.LLM_STATUS_CREATE_FAIL, api.LLM_STATUS_START_FAIL, api.LLM_STATUS_DELETE_FAILED:
			return true
		}
	}
	return false
}

func deleteReplicaGroup(ctx context.Context, userCred mcclient.TokenCredential, deploy *models.SLLMDeployment, group replicaGroupInfo) error {
	if len(group.Members) == 0 {
		return nil
	}
	// Tear followers down first so the leader keeps serving (and reconcile does
	// not briefly observe a leader without its ranks).
	members := append([]instanceInfo(nil), group.Members...)
	sort.SliceStable(members, func(i, j int) bool {
		return distRoleDeletePriority(members[i].DistRole) < distRoleDeletePriority(members[j].DistRole)
	})
	deleted := 0
	for _, inst := range members {
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
	}
	if deleted == 0 {
		return fmt.Errorf("no member of group %s could be deleted", group.Key)
	}
	log.Infof("reconcile: deleting incomplete/failed replica group %s (%d members) for %s", group.Key, deleted, deploy.Name)
	db.OpsLog.LogEvent(deploy, db.ACT_UPDATE, jsonutils.NewString(fmt.Sprintf(
		"rebuilding replica group %s: deleted %d incomplete/failed member(s)", group.Key, deleted)), userCred)
	return nil
}

func distRoleDeletePriority(role string) int {
	switch role {
	case api.LLM_DIST_ROLE_FOLLOWER:
		return 0
	case api.LLM_DIST_ROLE_LEADER:
		return 2
	default:
		return 1
	}
}

type instanceInfo struct {
	Id           string
	Name         string
	Status       string
	DistRole     string
	DistRank     int
	DistNnodes   int
	DistLeaderId string
	DistGroup    int
}

type replicaGroupInfo struct {
	Key      string
	Group    int
	Members  []instanceInfo
	Priority int
}

func fetchModelInstances(modelId string) ([]instanceInfo, error) {
	q := models.GetLLMManager().Query("id", "name", "status", "dist_role", "dist_rank", "dist_nnodes", "dist_leader_id", "dist_group").
		Equals("llm_deployment_id", modelId)
	var rows []instanceInfo
	err := q.All(&rows)
	return rows, err
}

func groupInstanceInfos(instances []instanceInfo) []replicaGroupInfo {
	order := make([]string, 0)
	byKey := make(map[string]*replicaGroupInfo)
	for _, inst := range instances {
		key := "s:" + inst.Id
		group := -1
		if inst.DistRole == api.LLM_DIST_ROLE_LEADER || inst.DistRole == api.LLM_DIST_ROLE_FOLLOWER {
			key = fmt.Sprintf("g:%d", inst.DistGroup)
			group = inst.DistGroup
		}
		g, ok := byKey[key]
		if !ok {
			g = &replicaGroupInfo{Key: key, Group: group}
			byKey[key] = g
			order = append(order, key)
		}
		g.Members = append(g.Members, inst)
		p := instancePriority(inst.Status)
		if len(g.Members) == 1 || p < g.Priority {
			g.Priority = p
		}
	}
	out := make([]replicaGroupInfo, 0, len(order))
	for _, key := range order {
		out = append(out, *byKey[key])
	}
	return out
}

func nextReplicaGroupIndex(deploymentName string, existing []instanceInfo) int {
	nextIndex := 0
	for _, inst := range existing {
		if g, ok := models.ParseDistGroupFromInstanceName(deploymentName, inst.Name); ok {
			if g >= nextIndex {
				nextIndex = g + 1
			}
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(inst.Name, deploymentName+"-%d", &idx); err == nil {
			if idx >= nextIndex {
				nextIndex = idx + 1
			}
		}
	}
	return nextIndex
}

// scaleUp creates `count` new logical replicas.
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

	nextIndex := nextReplicaGroupIndex(deployment.Name, existing)

	if deployment.Nets == nil || len(*deployment.Nets) == 0 {
		return fmt.Errorf("scaleUp: deployment %s has no stored nets template", deployment.Name)
	}
	var nets []*computeapi.NetworkConfig = *deployment.Nets

	createCtx := context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
	handler := db.NewModelHandler(models.GetLLMManager())

	llmSpec, err := models.BuildDeploymentResolvedGpuMemoryLLMSpec(ctx, userCred, deployment, llmSku)
	if err != nil {
		return fmt.Errorf("resolve GPU memory utilization: %w", err)
	}
	preferHosts := models.GetDeploymentPreferHosts(deployment, llmSku)

	var lastErr error
	created := 0
	reserved := make(map[string]int)
	for i := 0; i < count; i++ {
		group := nextIndex + i
		plan, planErr := models.PlanDeploymentReplicaWithReserved(ctx, userCred, deployment, llmSku, reserved)
		if planErr != nil {
			lastErr = planErr
			log.Errorf("reconcile: plan replica group %d failed: %s", group, planErr)
			continue
		}
		models.ReservePlanGPUs(reserved, plan)
		if plan.Distributed {
			log.Infof("reconcile: planning distributed replica group %d for %s: nnodes=%d gpus_per_node=%d",
				group, deployment.Name, plan.NNodes, plan.GpusPerNode)
			if err := deployment.CheckReconcileBudget(); err != nil {
				// Stop recreating groups after repeated runtime failures.
				lastErr = err
				log.Warningf("reconcile: skip distributed group %d for %s: %s", group, deployment.Name, err)
				break
			}
			hostIds := make([]string, 0, len(plan.Members))
			for _, m := range plan.Members {
				hostIds = append(hostIds, m.HostId)
			}
			layout := ""
			if plan.Parallelism != nil {
				layout = fmt.Sprintf("; tp=%d pp=%d dp=%d",
					plan.Parallelism.TP, plan.Parallelism.PP, plan.Parallelism.DP)
			}
			db.OpsLog.LogEvent(deployment, db.ACT_UPDATE, jsonutils.NewString(fmt.Sprintf(
				"replica group %d planned on %d hosts (%d GPUs each): %s%s",
				group, plan.NNodes, plan.GpusPerNode, strings.Join(hostIds, ", "), layout)), userCred)
			taskParams := jsonutils.Marshal(distributedGroupCreateParams{
				DeploymentId: deployment.Id,
				Group:        group,
				Plan:         plan,
				LLMSpec:      llmSpec,
			}).(*jsonutils.JSONDict)
			task, err := taskman.TaskManager.NewTask(ctx, "LLMDistributedGroupCreateTask", deployment, userCred, taskParams, "", "", nil)
			if err != nil {
				lastErr = err
				log.Errorf("reconcile: new distributed group task: %s", err)
				continue
			}
			if err := task.ScheduleRun(nil); err != nil {
				lastErr = err
				log.Errorf("reconcile: schedule distributed group task: %s", err)
				continue
			}
			created++
			log.Infof("reconcile: started distributed replica group %d for %s", group, deployment.Name)
			continue
		}

		instanceName := models.DistLeaderInstanceName(deployment.Name, group)
		preferHost := plan.Members[0].HostId
		if preferHost == "" {
			preferHost = models.SelectPreferHostForInstanceIndex(preferHosts, group)
		}
		if deployment.DistributedInference != nil && *deployment.DistributedInference && !plan.Distributed {
			// Surface why a deployment that allows multi-host ended up on one host.
			db.OpsLog.LogEvent(deployment, db.ACT_UPDATE, jsonutils.NewString(fmt.Sprintf(
				"replica group %d planned on a single host %s: %d GPUs fit on one host (set nnodes >= 2 to force distributed inference)",
				group, preferHost, plan.GpusPerNode)), userCred)
		}
		llmInput := buildDeploymentLLMCreateInput(deployment, nets, imageId, llmSpec, preferHost)
		if len(plan.Members) > 0 && len(plan.Members[0].Devices) > 0 {
			devs := plan.Members[0].Devices
			llmInput.Devices = &devs
		}
		llmInput.DistGroup = group

		llmParams := jsonutils.Marshal(llmInput).(*jsonutils.JSONDict)
		llmParams.Set("name", jsonutils.NewString(instanceName))
		llmParams.Remove("generate_name")

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

func buildDeploymentLLMCreateInput(deployment *models.SLLMDeployment, nets []*computeapi.NetworkConfig, imageId string, llmSpec *api.LLMSpec, preferHost string) api.LLMCreateInput {
	return api.LLMCreateInput{
		LLMBaseCreateInput: api.LLMBaseCreateInput{
			// Deployments always boot their replicas; there is no deployment-level
			// auto_start switch anymore. Distributed members override this to false
			// and are started by the group task after the whole group exists.
			AutoStart:  true,
			Nets:       nets,
			PreferHost: preferHost,
		},
		LLMSkuId:        deployment.LLMSkuId,
		LLMImageId:      imageId,
		LLMDeploymentId: deployment.Id,
		LLMSpec:         llmSpec,
		HostPaths:       deployment.HostPaths,
	}
}

// scaleDown deletes `count` logical replica groups, preferring unhealthy ones.
func scaleDown(ctx context.Context, userCred mcclient.TokenCredential, model *models.SLLMDeployment, groups []replicaGroupInfo, count int) error {
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Priority != groups[j].Priority {
			return groups[i].Priority < groups[j].Priority
		}
		// Distributed groups: delete the highest group index first. Comparing
		// "g:10" vs "g:9" as strings would order them incorrectly.
		if groups[i].Group >= 0 && groups[j].Group >= 0 && groups[i].Group != groups[j].Group {
			return groups[i].Group > groups[j].Group
		}
		if groups[i].Group >= 0 && groups[j].Group < 0 {
			return true
		}
		if groups[i].Group < 0 && groups[j].Group >= 0 {
			return false
		}
		return groups[i].Key > groups[j].Key
	})

	deletedGroups := 0
	for i := 0; i < count && i < len(groups); i++ {
		group := groups[i]
		// Delete followers first, then leader / single member.
		members := append([]instanceInfo(nil), group.Members...)
		sort.SliceStable(members, func(a, b int) bool {
			if members[a].DistRole == api.LLM_DIST_ROLE_FOLLOWER && members[b].DistRole != api.LLM_DIST_ROLE_FOLLOWER {
				return true
			}
			if members[b].DistRole == api.LLM_DIST_ROLE_FOLLOWER && members[a].DistRole != api.LLM_DIST_ROLE_FOLLOWER {
				return false
			}
			return members[a].DistRank > members[b].DistRank
		})
		groupDeleted := 0
		for _, inst := range members {
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
			groupDeleted++
			log.Infof("reconcile: deleting LLM instance %s (%s) for model %s scale-down", inst.Name, inst.Id, model.Name)
		}
		if groupDeleted > 0 {
			deletedGroups++
		}
	}

	if deletedGroups == 0 && count > 0 {
		return fmt.Errorf("failed to delete any replica group")
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
