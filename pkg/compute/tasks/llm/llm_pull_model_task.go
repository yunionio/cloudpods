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
	t.requestGetManifests(ctx, obj.(*models.SLLM))
}

func (t *LLMPullModelTask) requestGetManifests(ctx context.Context, llm *models.SLLM) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLING_MODEL, "")

	manifests, err := llm.DownloadManifests(ctx, t.GetUserCred())
	if err != nil {
		t.OnGetManifestsFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
	t.OnGotManifests(ctx, llm, manifests)
}

func (t *LLMPullModelTask) OnGetManifestsFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_GET_MANIFESTS_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMPullModelTask) OnGotManifests(ctx context.Context, llm *models.SLLM, manifests []string) {
	input := &api.LLMAccessCacheInput{
		Blobs:     manifests,
		ModelName: llm.GetModelName(),
	}
	t.SetStage("OnAccessCache", jsonutils.Marshal(input).(*jsonutils.JSONDict))
	if err := llm.AccessBlobsCache(ctx, t.GetUserCred(), t); nil != err {
		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *LLMPullModelTask) OnAccessCacheFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_DOWNLOADING_BLOBS_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMPullModelTask) OnAccessCache(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
	// log.Infoln("try to find out blobs: ", t.GetParams().String(), data.String())
	input := new(api.LLMAccessCacheInput)
	if err := t.GetParams().Unmarshal(input); nil != err {
		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
	if err := llm.CopyBlobs(ctx, t.GetUserCred(), input.Blobs); nil != err {
		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
	t.OnPulledModel(ctx, llm, nil)
}

func (t *LLMPullModelTask) OnPulledModel(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLED_MODEL, "")
	t.SetStageComplete(ctx, nil)
}
