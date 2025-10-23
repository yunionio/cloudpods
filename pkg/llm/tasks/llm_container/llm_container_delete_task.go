package llmcontainer

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMContainerDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMContainerDeleteTask{})
}

func (task *LLMContainerDeleteTask) taskFailed(ctx context.Context, lc *models.SLLMContainer, status string, err error) {
	lc.SetStatus(ctx, task.UserCred, status, err.Error())
	db.OpsLog.LogEvent(lc, db.ACT_DELETE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, lc, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMContainerDeleteTask) taskComplete(ctx context.Context, lc *models.SLLMContainer) {
	lc.RealDelete(ctx, task.GetUserCred())
	task.SetStageComplete(ctx, nil)
}

func (task *LLMContainerDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	lc := obj.(*models.SLLMContainer)
	task.taskComplete(ctx, lc)
}
