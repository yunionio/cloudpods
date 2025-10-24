package llm

// import (
// 	"context"

// 	"yunion.io/x/jsonutils"

// 	api "yunion.io/x/onecloud/pkg/apis/llm"
// 	"yunion.io/x/onecloud/pkg/cloudcommon/db"
// 	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
// 	"yunion.io/x/onecloud/pkg/llm/models"
// )

// type LLMPullModelTask struct {
// 	taskman.STask
// }

// func (t *LLMPullModelTask) GetLLM() *models.SLLM {
// 	return t.GetObject().(*models.SLLM)
// }

// func (t *LLMPullModelTask) GetLLMContainerDriver() models.ILLMContainerDriver {
// 	return t.GetLLM().GetLLMContainerDriver()
// }

// func init() {
// 	taskman.RegisterTask(LLMPullModelTask{})
// }

// func (t *LLMPullModelTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
// 	t.requestGetManifests(ctx, obj.(*models.SLLM))
// }

// func (t *LLMPullModelTask) requestGetManifests(ctx context.Context, llm *models.SLLM) {
// 	// llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLING_MODEL, "")
// 	t.SetStage("OnGetManifests", nil)

// 	if err := t.GetLLMContainerDriver().GetManifests(ctx, t.GetUserCred(), llm, t.GetId()); err != nil {
// 		t.OnGetManifestsFailed(ctx, llm, jsonutils.NewString(err.Error()))
// 		return
// 	}
// }

// func (t *LLMPullModelTask) OnGetManifestsFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
// 	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_GET_MANIFESTS_FAILED, reason.String())
// 	t.SetStageFailed(ctx, reason)
// }

// func (t *LLMPullModelTask) OnGetManifests(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
// 	t.SetStage("OnAccessCache", nil)
// 	if err := t.GetLLMContainerDriver().AccessBlobsCache(ctx, t.GetUserCred(), llm, t.GetId()); nil != err {
// 		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
// 		return
// 	}
// }

// func (t *LLMPullModelTask) OnAccessCacheFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
// 	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_DOWNLOADING_BLOBS_FAILED, reason.String())
// 	t.SetStageFailed(ctx, reason)
// }

// func (t *LLMPullModelTask) OnAccessCache(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
// 	// log.Infoln("try to find out blobs: ", t.GetParams().String(), data.String())
// 	if err := t.GetLLMContainerDriver().CopyBlobs(ctx, t.GetUserCred(), llm); nil != err {
// 		t.OnAccessCacheFailed(ctx, llm, jsonutils.NewString(err.Error()))
// 		return
// 	}
// 	t.OnPulledModel(ctx, llm, nil)
// }

// func (t *LLMPullModelTask) OnPulledModel(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
// 	// llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLED_MODEL, "")
// 	t.SetStageComplete(ctx, nil)
// }
