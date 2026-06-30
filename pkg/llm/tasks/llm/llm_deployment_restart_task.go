package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// LLMDeploymentRestartTask restarts all restartable SLLM instances under a
// deployment in parallel, then reconciles replica health on the deployment row.
//
// Flow:
//  1. OnInit: enumerate instances, start each restartable instance's
//     LLMRestartTask with this task as parent.
//  2. Framework invokes OnInstancesRestarted after all child tasks complete.
//  3. OnInstancesRestarted calls SyncReadyReplicas and completes.
type LLMDeploymentRestartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMDeploymentRestartTask{})
}

func (task *LLMDeploymentRestartTask) taskFailed(ctx context.Context, model *models.SLLMDeployment, err error) {
	db.OpsLog.LogEvent(model, "restart", err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_VM_RESTART, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMDeploymentRestartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
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

	task.SetStage("OnInstancesRestarted", nil)

	startedCount := 0
	for _, inst := range instances {
		llmObj, err := models.GetLLMManager().FetchById(inst.Id)
		if err != nil {
			log.Errorf("LLMDeploymentRestartTask: fetch instance %s: %s", inst.Id, err)
			continue
		}
		llm := llmObj.(*models.SLLM)
		taskInput, err := llm.ValidateRestartInput(ctx, task.UserCred, &api.LLMRestartInput{})
		if err != nil {
			log.Warningf("LLMDeploymentRestartTask: skip instance %s: %s", inst.Id, err)
			continue
		}
		if _, err := llm.StartRestartTask(ctx, task.UserCred, taskInput, task.GetTaskId()); err != nil {
			log.Errorf("LLMDeploymentRestartTask: start restart for %s: %s", inst.Id, err)
			continue
		}
		startedCount++
	}

	if startedCount == 0 {
		task.taskFailed(ctx, model, errors.Error("no instance restart tasks could be started"))
	}
}

func (task *LLMDeploymentRestartTask) OnInstancesRestarted(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	if err := model.SyncReadyReplicas(ctx, task.UserCred); err != nil {
		log.Warningf("LLMDeploymentRestartTask: SyncReadyReplicas for %s: %s", model.Name, err)
	}
	db.OpsLog.LogEvent(model, "restart", nil, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_VM_RESTART, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMDeploymentRestartTask) OnInstancesRestartedFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)
	log.Warningf("LLMDeploymentRestartTask: some instances failed to restart: %s", body)
	if err := model.SyncReadyReplicas(ctx, task.UserCred); err != nil {
		log.Warningf("LLMDeploymentRestartTask: SyncReadyReplicas for %s: %s", model.Name, err)
	}
	db.OpsLog.LogEvent(model, "restart", body, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_VM_RESTART, body, task.UserCred, false)
	task.SetStageComplete(ctx, nil)
}
