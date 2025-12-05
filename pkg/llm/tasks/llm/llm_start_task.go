package llm

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMStartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMStartTask{})
}

func (task *LLMStartTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_START_FAIL, err)
	db.OpsLog.LogEvent(llm, db.ACT_START, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_START, err, task.UserCred, false)
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStart, nil, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMStartTask) taskComplete(ctx context.Context, llm *models.SLLM) {
	llm.SetStatus(ctx, task.GetUserCred(), api.LLM_STATUS_RUNNING, "start complete")
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStart, nil, true)
	task.SetStageComplete(ctx, nil)
}

func (t *LLMStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestStart(ctx, obj.(*models.SLLM))
}

func (t *LLMStartTask) requestStart(ctx context.Context, llm *models.SLLM) {
	t.SetStage("OnStarted", nil)
	s := auth.GetSession(ctx, t.GetUserCred(), options.Options.Region)
	err := s.WithTaskCallback(t.GetId(), func() error {
		_, err := compute.Servers.PerformAction(s, llm.SvrId, "start", nil)
		return err
	})
	if err != nil {
		t.taskFailed(ctx, llm, err.Error())
		return
	}
}

func (t *LLMStartTask) OnStartedFailed(ctx context.Context, llm *models.SLLM, err jsonutils.JSONObject) {
	t.taskFailed(ctx, llm, err.String())
}

func (t *LLMStartTask) OnStarted(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	t.taskComplete(ctx, llm)
}
