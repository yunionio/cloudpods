package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMStopTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMStopTask{})
}

func (task *LLMStopTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_STOP_FAILED, err)
	db.OpsLog.LogEvent(llm, db.ACT_STOP, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_VM_STOP, err, task.UserCred, false)
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStop, nil, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMStopTask) taskComplete(ctx context.Context, llm *models.SLLM) {
	if !task.HasParentTask() {
		llm.SetStatus(ctx, task.GetUserCred(), api.LLM_STATUS_READY, "")
	}
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStop, nil, true)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	srv, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetServer").Error())
		return
	}

	if srv.Status == computeapi.VM_READY {
		task.taskComplete(ctx, llm)
		return
	}

	task.SetStage("OnStopComplete", nil)
	s := auth.GetSession(ctx, task.UserCred, "")
	err = s.WithTaskCallback(task.GetId(), func() error {
		_, err = compute.Servers.PerformAction(s, llm.CmpId, "stop", nil)
		return err
	})
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
}

func (task *LLMStopTask) OnStopComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskComplete(ctx, llm)
}

func (task *LLMStopTask) OnStopCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}
