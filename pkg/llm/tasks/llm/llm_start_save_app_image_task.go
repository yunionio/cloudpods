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

type LLMStartSaveAppImageTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMStartSaveAppImageTask{})
}

func (task *LLMStartSaveAppImageTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_SAVE_APP_FAILED, err)
	db.OpsLog.LogEvent(llm, db.ACT_SAVE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_SAVE_IMAGE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMStartSaveAppImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	// first stop the desktop
	task.SetStage("OnStopLLMComplete", nil)
	err := llm.StartLLMStopTask(ctx, task.UserCred, task.GetTaskId())
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
}

func (task *LLMStartSaveAppImageTask) OnStopLLMCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMStartSaveAppImageTask) OnStopLLMComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	input := api.LLMSaveInstantAppInput{}
	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	task.SetStage("OnSaveAppImageComplete", nil)
	s := auth.GetSession(ctx, task.GetUserCred(), options.Options.Region)
	s.WithTaskCallback(task.GetId(), func() error {
		return llm.DoSaveAppImage(ctx, task.UserCred, s, input)
	})
}

func (task *LLMStartSaveAppImageTask) OnSaveAppImageComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	input := api.LLMSaveInstantAppInput{}

	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	instantAppObj, err := models.GetInstantAppManager().FetchById(input.InstantAppId)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
	instantApp := instantAppObj.(*models.SInstantApp)

	_, err = instantApp.PerformSyncstatus(ctx, task.GetUserCred(), nil, api.InstantAppSyncstatusInput{})
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
	}

	db.OpsLog.LogEvent(llm, db.ACT_SAVE, instantApp.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_SAVE_IMAGE, instantApp.GetShortDesc(ctx), task.UserCred, false)
	db.OpsLog.LogEvent(instantApp, db.ACT_SAVE, llm.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, instantApp, logclient.ACT_SAVE_IMAGE, llm.GetShortDesc(ctx), task.UserCred, true)

	task.SetStageComplete(ctx, nil)

	// if input.AutoRestart {
	// 	llm.StartRestartTask(ctx, task.UserCred, api.DesktopRestartTaskInput{
	// 		DesktopId:     llm.Id,
	// 		DesktopStatus: api.LLM_STATUS_READY,
	// 	}, "")
	// } else {
	// 	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_READY, "OnSaveAppImageComplete")
	// }
}

func (task *LLMStartSaveAppImageTask) OnSaveAppImageCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	d := obj.(*models.SLLM)
	task.taskFailed(ctx, d, err.String())
}
