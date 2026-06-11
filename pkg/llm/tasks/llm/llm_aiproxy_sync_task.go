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

type LLMAiproxySyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMAiproxySyncTask{})
}

func (task *LLMAiproxySyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dep := obj.(*models.SLLMDeployment)
	if !dep.AutoRegisterAiproxy {
		task.SetStageComplete(ctx, nil)
		return
	}

	llmId, _ := task.GetParams().GetString("llm_id")
	unregister, _ := task.GetParams().Bool("unregister")

	var err error
	if unregister {
		err = models.UnregisterDeploymentAiproxy(ctx, task.UserCred, dep)
	} else if llmId != "" {
		llmObj, fetchErr := models.GetLLMManager().FetchById(llmId)
		if fetchErr != nil {
			err = fetchErr
		} else {
			llm := llmObj.(*models.SLLM)
			if _, syncErr := models.SyncLlmInstance(ctx, task.UserCred, dep, llm); syncErr != nil {
				err = syncErr
			} else {
				err = models.ReconcileDeploymentAiproxy(ctx, task.UserCred, dep)
			}
		}
	} else {
		err = models.ReconcileDeploymentAiproxy(ctx, task.UserCred, dep)
	}

	if err != nil {
		log.Errorf("LLMAiproxySyncTask deployment=%s: %v", dep.Name, err)
		_, _ = db.Update(dep, func() error {
			if dep.AiproxySyncStatus != api.AIPROXY_SYNC_STATUS_SYNCED {
				dep.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_FAILED
			}
			return nil
		})
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	task.SetStageComplete(ctx, nil)
}
