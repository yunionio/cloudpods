package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LLMStartTask struct {
	LLMBaseTask
}

func init() {
	taskman.RegisterTask(LLMStartTask{})
}

func (t *LLMStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestStart(ctx, obj.(*models.SLLM))
}

func (t *LLMStartTask) requestStart(ctx context.Context, llm *models.SLLM) {
	t.SetStage("OnStarted", nil)
	if err := llm.RunModel(ctx, t.GetUserCred()); nil != err {
		t.OnStartedFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *LLMStartTask) OnStartedFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_START_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMStartTask) OnStarted(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
