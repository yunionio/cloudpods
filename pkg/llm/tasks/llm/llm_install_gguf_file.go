package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

type LLMInstallGgufTask struct {
	LLMBaseTask
}

func init() {
	taskman.RegisterTask(LLMInstallGgufTask{})
}

func (t *LLMInstallGgufTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestGetGgufFile(ctx, obj.(*models.SLLM))
}

func (t *LLMInstallGgufTask) requestGetGgufFile(ctx context.Context, llm *models.SLLM) {
	// Update status
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_FETCHING_GGUF_FILE, "")

	// Distingush whether access gguf file from host or download from web
	input := new(api.LLMGgufSpec)
	if err := t.GetParams().Unmarshal(input); nil != err {
		t.OnGetGgufFileFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}

	switch input.Source {
	case api.LLM_OLLAMA_GGUF_SOURCE_WEB:
		if err := llm.DownloadGgufFile(ctx, t.GetUserCred(), t); nil != err {
			t.OnGetGgufFileFailed(ctx, llm, jsonutils.NewString(err.Error()))
			return
		}
		// t.OnGetGgufFile(ctx, llm)
	default:
		// t.SetStage("OnGetGgufFile", nil)
		if err := llm.AccessGgufFile(ctx, t.GetUserCred(), t); nil != err {
			t.OnGetGgufFileFailed(ctx, llm, jsonutils.NewString(err.Error()))
			return
		}
		t.OnGetGgufFile(ctx, llm)
	}
}

func (t *LLMInstallGgufTask) OnGetGgufFileFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_FETCH_GGUF_FILE_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMInstallGgufTask) OnGetGgufFile(ctx context.Context, llm *models.SLLM) {
	input := new(api.LLMGgufSpec)
	if err := t.GetParams().Unmarshal(input); nil != err {
		t.OnCreateModelFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}

	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_CREATING_GGUF_MODEL, "")

	if err := llm.InstallGgufModel(ctx, t.GetUserCred(), input.ModelFile); nil != err {
		t.OnCreateModelFailed(ctx, llm, jsonutils.NewString(err.Error()))
		return
	}
	t.OnPulledModel(ctx, llm, nil)
}

func (t *LLMInstallGgufTask) OnCreateModelFailed(ctx context.Context, llm *models.SLLM, reason jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_CREATE_GGUF_MODEL_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *LLMInstallGgufTask) OnPulledModel(ctx context.Context, llm *models.SLLM, data jsonutils.JSONObject) {
	llm.SetStatus(ctx, t.GetUserCred(), api.LLM_STATUS_PULLED_MODEL, "")
	t.SetStageComplete(ctx, nil)
}
