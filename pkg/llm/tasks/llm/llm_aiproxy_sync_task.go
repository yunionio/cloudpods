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

type LLMAiproxySyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMAiproxySyncTask{})
}

func (task *LLMAiproxySyncTask) aiproxyLogAction() string {
	unregister, _ := task.GetParams().Bool("unregister")
	if unregister {
		return logclient.ACT_UNREGISTER_AIPROXY
	}
	return logclient.ACT_REGISTER_AIPROXY
}

func (task *LLMAiproxySyncTask) logAiproxySync(ctx context.Context, dep *models.SLLMDeployment, err error) {
	action := task.aiproxyLogAction()
	if err != nil {
		db.OpsLog.LogEvent(dep, "aiproxy_sync", err.Error(), task.UserCred)
		logclient.AddActionLogWithStartable(task, dep, action, err, task.UserCred, false)
		return
	}
	depObj, fetchErr := models.GetLLMDeploymentManager().FetchById(dep.Id)
	if fetchErr == nil {
		dep = depObj.(*models.SLLMDeployment)
	}
	if dep.AiproxySyncStatus == api.AIPROXY_SYNC_STATUS_FAILED {
		msg := models.AiproxySyncFailureReason(dep)
		if msg == "" {
			msg = "aiproxy sync failed"
		}
		db.OpsLog.LogEvent(dep, "aiproxy_sync", msg, task.UserCred)
		logclient.AddActionLogWithStartable(task, dep, action, msg, task.UserCred, false)
		return
	}
	logclient.AddActionLogWithStartable(task, dep, action, nil, task.UserCred, true)
}

func (task *LLMAiproxySyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dep := obj.(*models.SLLMDeployment)
	if !dep.AutoRegisterAiproxy {
		unregister, _ := task.GetParams().Bool("unregister")
		if !unregister {
			task.SetStageComplete(ctx, nil)
			return
		}
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
		_ = dep.SetAiproxySyncStatus(ctx, task.UserCred, api.AIPROXY_SYNC_STATUS_FAILED, err.Error())
		task.logAiproxySync(ctx, dep, err)
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	task.logAiproxySync(ctx, dep, nil)
	task.SetStageComplete(ctx, nil)
}
