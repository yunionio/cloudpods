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
	t.requestPullModel(ctx, obj.(*models.SLLM))
}

func (t *LLMPullModelTask) requestPullModel(ctx context.Context, llm *models.SLLM) {
	// t.SetStage("OnPulledModel", nil)
	// log.Infoln("set stage success", t, llm.Id, llm.ContainerId)
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLING_MODEL, "")
	// log.Infoln("request LLM Pull Model", t, llm.Id, llm.ContainerId)
	if err := llm.PullModel(ctx, t.GetUserCred()); err != nil {
		t.OnPulledModelFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
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
