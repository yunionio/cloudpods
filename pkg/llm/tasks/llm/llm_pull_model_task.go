package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

type LLMBaseTask struct {
	taskman.STask
}

func (t *LLMBaseTask) GetLLM() *models.SOllama {
	return t.GetObject().(*models.SOllama)
}

type LLMPullModelTask struct {
	LLMBaseTask
}

func init() {
	taskman.RegisterTask(LLMPullModelTask{})
}

func (t *LLMPullModelTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	if err := obj.(*models.SOllama).ConfirmContainerId(ctx, t.GetUserCred()); err != nil {
		t.OnGetManifestsFailed(ctx, obj.(*models.SOllama), jsonutils.NewString(err.Error()))
	}
	t.requestGetManifests(ctx, obj.(*models.SOllama))
}

func (t *LLMPullModelTask) requestGetManifests(ctx context.Context, llm *models.SOllama) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLING_MODEL, "")
	t.SetStage("OnGetManifests", nil)

	if err := llm.DownloadManifests(ctx, t.GetUserCred(), t.GetId()); err != nil {
		t.OnGetManifestsFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *LLMPullModelTask) OnGetManifestsFailed(ctx context.Context, llm *models.SOllama, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_GET_MANIFESTS_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMPullModelTask) OnGetManifests(ctx context.Context, llm *models.SOllama, data jsonutils.JSONObject) {
	blobs, err := llm.FetchBlobs(ctx, t.GetUserCred())
	if err != nil {
		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
	input := &api.LLMAccessCacheInput{
		Blobs:     blobs,
		ModelName: llm.GetModelName(),
	}
	t.SetStage("OnAccessCache", jsonutils.Marshal(input).(*jsonutils.JSONDict))
	if err := llm.AccessBlobsCache(ctx, t.GetUserCred(), t); nil != err {
		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *LLMPullModelTask) OnAccessCacheFailed(ctx context.Context, llm *models.SOllama, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_DOWNLOADING_BLOBS_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMPullModelTask) OnAccessCache(ctx context.Context, llm *models.SOllama, data jsonutils.JSONObject) {
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

func (t *LLMPullModelTask) OnPulledModel(ctx context.Context, llm *models.SOllama, data jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLED_MODEL, "")
	t.SetStageComplete(ctx, nil)
}
