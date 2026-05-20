package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

// LLMDeploymentSyncReplicasTask reconciles SLLM instances on replica changes
// (scale up / scale down). It runs after the deployment's Replicas is updated
// or after an instance is unexpectedly deleted and the count drifts below the
// desired value.
//
// Unlike LLMDeploymentCreateTask, this task's body does NOT carry the original
// create-time payload (nets / auto_start / prefer_host). Those fields are read
// from the deployment row, where PostCreate persisted them.
type LLMDeploymentSyncReplicasTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMDeploymentSyncReplicasTask{})
}

func (task *LLMDeploymentSyncReplicasTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SLLMDeployment)

	err := reconcileReplicas(ctx, task.UserCred, model, task.GetParams())
	if err != nil {
		model.SetStatus(ctx, task.UserCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	// Scale operations enter the deploying / partial / ready state machine.
	// SyncReadyReplicas computes the right one from the live instance count.
	if err := model.SyncReadyReplicas(ctx, task.UserCred); err != nil {
		log.Warningf("LLMDeploymentSyncReplicasTask: SyncReadyReplicas for %s: %s", model.Name, err)
	}
	task.SetStageComplete(ctx, nil)
}
