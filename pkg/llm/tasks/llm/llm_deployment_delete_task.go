package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// LLMDeploymentDeleteTask cascades deletion of all SLLM instances under the
// deployment, then deletes the deployment record itself.
//
// Flow:
//  1. OnInit: enumerate all SLLM instances of this deployment.
//  2. Set stage to "OnInstancesDeleted" and start each instance's delete task
//     with this task as parent.
//  3. Framework invokes OnInstancesDeleted after all child tasks complete.
//  4. OnInstancesDeleted calls RealDelete on the deployment record.
type LLMDeploymentDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMDeploymentDeleteTask{})
}

func (task *LLMDeploymentDeleteTask) taskFailed(ctx context.Context, model *models.SLLMDeployment, err error) {
	model.SetStatus(ctx, task.UserCred, api.LLM_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(model, db.ACT_DELETE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMDeploymentDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	model.SetStatus(ctx, task.UserCred, api.LLM_STATUS_DELETING, "deployment delete task started")

	// List all child SLLM instances
	instances, err := fetchModelInstances(model.Id)
	if err != nil {
		task.taskFailed(ctx, model, err)
		return
	}

	// No instances → delete deployment immediately
	if len(instances) == 0 {
		task.deleteDeployment(ctx, model)
		return
	}

	// Set stage; framework invokes OnInstancesDeleted after all children complete
	task.SetStage("OnInstancesDeleted", nil)

	startedCount := 0
	for _, inst := range instances {
		llmObj, err := models.GetLLMManager().FetchById(inst.Id)
		if err != nil {
			log.Errorf("LLMDeploymentDeleteTask: fetch instance %s: %s", inst.Id, err)
			continue
		}
		llm := llmObj.(*models.SLLM)
		// Pass this task's ID as parent so we get notified when delete completes
		if err := llm.StartDeleteTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
			log.Errorf("LLMDeploymentDeleteTask: start delete for %s: %s", inst.Id, err)
			continue
		}
		startedCount++
	}

	// If we couldn't start any child task, finish immediately to delete the deployment
	if startedCount == 0 {
		log.Warningf("LLMDeploymentDeleteTask: no instance delete tasks could be started, proceeding to delete deployment")
		task.deleteDeployment(ctx, model)
	}
}

// OnInstancesDeleted is called after all child SLLM delete tasks complete.
func (task *LLMDeploymentDeleteTask) OnInstancesDeleted(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	task.deleteDeployment(ctx, model)
}

// OnInstancesDeletedFailed is called if any child task fails.
func (task *LLMDeploymentDeleteTask) OnInstancesDeletedFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	// Even if some instances failed to delete, we still try to delete the deployment.
	// Failed instances will remain visible and can be cleaned up manually.
	log.Warningf("LLMDeploymentDeleteTask: some instances failed to delete: %s", body)
	task.deleteDeployment(ctx, model)
}

func (task *LLMDeploymentDeleteTask) deleteDeployment(ctx context.Context, model *models.SLLMDeployment) {
	// Capture managed SKU id BEFORE deleting the deployment so we can cascade
	// only SKUs automatically created for this deployment.
	skuId := deploymentManagedSkuIdForCascade(model)

	if err := model.RealDelete(ctx, task.UserCred); err != nil {
		log.Errorf("LLMDeploymentDeleteTask: RealDelete deployment %s: %s", model.Id, err)
		task.taskFailed(ctx, model, err)
		return
	}
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_DELETE, nil, task.UserCred, true)

	// Best-effort cascade: delete the SKU. InstantModels (秒装模型) are kept since
	// they can be reused by other deployments later.
	task.cascadeDeleteSku(ctx, skuId)

	task.SetStageComplete(ctx, nil)
}

func deploymentManagedSkuIdForCascade(model *models.SLLMDeployment) string {
	if model == nil {
		return ""
	}
	return model.ManagedLLMSkuId
}

// cascadeDeleteSku tries to delete the SKU created for this deployment. If the
// SKU is still referenced (e.g., by another deployment's instances), the
// framework's ValidateDeleteCondition rejects it and we log + continue.
func (task *LLMDeploymentDeleteTask) cascadeDeleteSku(ctx context.Context, skuId string) {
	if skuId == "" {
		return
	}

	skuObj, err := models.GetLLMSkuManager().FetchById(skuId)
	if err != nil {
		log.Infof("LLMDeploymentDeleteTask: SKU %s not found (already deleted?), skip cascade", skuId)
		return
	}
	sku := skuObj.(*models.SLLMSku)

	if err := sku.ValidateDeleteCondition(ctx, nil); err != nil {
		log.Infof("LLMDeploymentDeleteTask: skip SKU %s delete: %s", skuId, err)
		return
	}
	if err := sku.Delete(ctx, task.UserCred); err != nil {
		log.Errorf("LLMDeploymentDeleteTask: delete SKU %s: %s", skuId, err)
		return
	}
	log.Infof("LLMDeploymentDeleteTask: cascade-deleted SKU %s", skuId)
}
