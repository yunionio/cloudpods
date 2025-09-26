package llm_base

import (
	"context"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

type BaseCreatePodTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BaseCreatePodTask{})
}

func (t *BaseCreatePodTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider, ok := obj.(models.ILLMBaseProvider)
	if !ok {
		t.SetStageFailed(ctx, jsonutils.NewString("model does not implement ILLMBaseProvider"))
		return
	}

	t.requestCreatePod(ctx, provider.GetLLMBase())
}

func (t *BaseCreatePodTask) requestCreatePod(ctx context.Context, base *models.SLLMBase) {
	input := new(computeapi.ServerCreateInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		t.OnCreatePodFailed(ctx, base, jsonutils.NewString(err.Error()))
		return
	}
	if err := base.CreatePodByPolling(ctx, t.GetUserCred(), input); err != nil {
		t.OnCreatePodFailed(ctx, base, jsonutils.NewString(err.Error()))
		return
	}
	t.OnCreatePod(ctx, base, jsonutils.NewString("pod created successfully"))
}

func (t *BaseCreatePodTask) OnCreatePodFailed(ctx context.Context, base *models.SLLMBase, reason jsonutils.JSONObject) {
	base.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_CREAT_POD_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *BaseCreatePodTask) OnCreatePod(ctx context.Context, base *models.SLLMBase, reason jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
