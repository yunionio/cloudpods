package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LLMBaseTask struct {
	taskman.STask
}

func (t *LLMBaseTask) GetLLM() *models.SLLM {
	return t.GetObject().(*models.SLLM)
}

func (t *LLMBaseTask) GetContainer() (*models.SContainer, error) {
	return t.GetLLM().GetContainer()
}

type LLMPullModelTask struct {
	LLMBaseTask
}

func init() {
	taskman.RegisterTask(LLMPullModelTask{})
}

func (t *LLMPullModelTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestRestoreModel(ctx, obj.(*models.SLLM))
}

func (t *LLMPullModelTask) requestRestoreModel(ctx context.Context, llm *models.SLLM) {
	if err := llm.RestoreModel(t.GetUserCred()); nil != err {
		t.requestPullModel(ctx, llm)
	}
	t.OnPulledModel(ctx, llm, nil)
}

func (t *LLMPullModelTask) requestPullModel(ctx context.Context, llm *models.SLLM) {
	// t.SetStage("OnPulledModel", nil)
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLING_MODEL, "")
	if err := llm.PullModel(ctx, t.GetUserCred()); nil != err {
		t.OnPulledModelFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
	t.requestCacheModel(ctx, llm)
}

func (t *LLMPullModelTask) requestCacheModel(ctx context.Context, llm *models.SLLM) {
	if err := llm.CacheModel(t.GetUserCred()); nil != err {
		t.OnPulledModelFailed(ctx, llm, jsonutils.NewString(err.Error()))
	}
	t.OnPulledModel(ctx, llm, nil)
}

func (t *LLMPullModelTask) OnPulledModelFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULL_MODEL_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMPullModelTask) OnPulledModel(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLED_MODEL, "")
	t.SetStageComplete(ctx, nil)
}
