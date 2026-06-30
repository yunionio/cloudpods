package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// LLMDeploymentSyncStatusTask syncs all SLLM instances under a deployment in
// parallel, then reconciles replica health on the deployment row.
//
// Flow:
//  1. OnInit: enumerate instances, start each syncable instance's
//     LLMSyncStatusTask with this task as parent.
//  2. Framework invokes OnInstancesSyncStatusComplete after all child tasks complete.
//  3. OnInstancesSyncStatusComplete calls SyncReadyReplicas and completes.
type LLMDeploymentSyncStatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMDeploymentSyncStatusTask{})
}

func (task *LLMDeploymentSyncStatusTask) taskFailed(ctx context.Context, model *models.SLLMDeployment, err error) {
	db.OpsLog.LogEvent(model, db.ACT_SYNC_STATUS, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_SYNC_STATUS, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMDeploymentSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)

	instances, err := fetchModelInstances(model.Id)
	if err != nil {
		task.taskFailed(ctx, model, err)
		return
	}
	if len(instances) == 0 {
		task.taskFailed(ctx, model, errors.Error("no instances under deployment"))
		return
	}

	task.SetStage("OnInstancesSyncStatusComplete", nil)

	startedCount := 0
	for _, inst := range instances {
		llmObj, err := models.GetLLMManager().FetchById(inst.Id)
		if err != nil {
			log.Errorf("LLMDeploymentSyncStatusTask: fetch instance %s: %s", inst.Id, err)
			continue
		}
		llm := llmObj.(*models.SLLM)
		if llm.CmpId == "" {
			log.Warningf("LLMDeploymentSyncStatusTask: skip instance %s: no cmp_id", inst.Id)
			continue
		}
		if err := llm.StartSyncStatusTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
			log.Errorf("LLMDeploymentSyncStatusTask: start syncstatus for %s: %s", inst.Id, err)
			continue
		}
		startedCount++
	}

	if startedCount == 0 {
		task.taskFailed(ctx, model, errors.Error("no instance syncstatus tasks could be started"))
	}
}

func (task *LLMDeploymentSyncStatusTask) OnInstancesSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	if err := model.SyncReadyReplicas(ctx, task.UserCred); err != nil {
		log.Warningf("LLMDeploymentSyncStatusTask: SyncReadyReplicas for %s: %s", model.Name, err)
	}
	db.OpsLog.LogEvent(model, db.ACT_SYNC_STATUS, nil, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_SYNC_STATUS, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMDeploymentSyncStatusTask) OnInstancesSyncStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	log.Warningf("LLMDeploymentSyncStatusTask: some instances failed to sync status: %s", body)
	if err := model.SyncReadyReplicas(ctx, task.UserCred); err != nil {
		log.Warningf("LLMDeploymentSyncStatusTask: SyncReadyReplicas for %s: %s", model.Name, err)
	}
	db.OpsLog.LogEvent(model, db.ACT_SYNC_STATUS, body, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_SYNC_STATUS, body, task.UserCred, false)
	task.SetStageComplete(ctx, nil)
}
