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
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMStartSaveModelImageTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMStartSaveModelImageTask{})
}

func (task *LLMStartSaveModelImageTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_SAVE_MODEL_FAILED, err)
	db.OpsLog.LogEvent(llm, db.ACT_SAVE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_SAVE_IMAGE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMStartSaveModelImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	// first stop the desktop
	task.SetStage("OnStopLLMComplete", nil)
	err := llm.StartLLMStopTask(ctx, task.UserCred, task.GetTaskId())
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
}

func (task *LLMStartSaveModelImageTask) OnStopLLMCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMStartSaveModelImageTask) OnStopLLMComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	input := api.LLMSaveInstantModelInput{}
	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	task.SetStage("OnSaveModelImageComplete", nil)
	s := auth.GetSession(ctx, task.GetUserCred(), options.Options.Region)
	s.WithTaskCallback(task.GetId(), func() error {
		return llm.DoSaveModelImage(ctx, task.UserCred, s, input)
	})
}

func (task *LLMStartSaveModelImageTask) OnSaveModelImageComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	input := api.LLMSaveInstantModelInput{}

	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	instantModelObj, err := models.GetInstantModelManager().FetchById(input.InstantModelId)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
	instantModel := instantModelObj.(*models.SInstantModel)

	_, err = instantModel.PerformSyncstatus(ctx, task.GetUserCred(), nil, api.InstantModelSyncstatusInput{})
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
	}

	db.OpsLog.LogEvent(llm, db.ACT_SAVE, instantModel.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_SAVE_IMAGE, instantModel.GetShortDesc(ctx), task.UserCred, false)
	db.OpsLog.LogEvent(instantModel, db.ACT_SAVE, llm.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, instantModel, logclient.ACT_SAVE_IMAGE, llm.GetShortDesc(ctx), task.UserCred, true)

	task.SetStageComplete(ctx, nil)

	// if input.AutoRestart {
	// 	llm.StartRestartTask(ctx, task.UserCred, api.DesktopRestartTaskInput{
	// 		DesktopId:     llm.Id,
	// 		DesktopStatus: api.LLM_STATUS_READY,
	// 	}, "")
	// } else {
	// 	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_READY, "OnSaveModelImageComplete")
	// }
}

func (task *LLMStartSaveModelImageTask) OnSaveModelImageCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	d := obj.(*models.SLLM)
	task.taskFailed(ctx, d, err.String())
}
