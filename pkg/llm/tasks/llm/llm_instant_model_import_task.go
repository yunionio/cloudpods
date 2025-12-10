package llm

import (
	"context"

	"yunion.io/x/jsonutils"

	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMInstantModelImportTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMInstantModelImportTask{})
}

func (task *LLMInstantModelImportTask) taskFailed(ctx context.Context, model *models.SInstantModel, err string) {
	model.SetStatus(ctx, task.UserCred, imageapi.IMAGE_STATUS_KILLED, err)
	db.OpsLog.LogEvent(model, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_CREATE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMInstantModelImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)

	input := apis.InstantModelImportInput{}
	err := task.Params.Unmarshal(&input, "import_input")
	if err != nil {
		task.taskFailed(ctx, model, err.Error())
		return
	}

	task.SetStage("OnImportComplete", nil)
	s := auth.GetSession(ctx, task.GetUserCred(), options.Options.Region)
	err = s.WithTaskCallback(task.GetId(), func() error {
		return model.DoImport(ctx, task.UserCred, s, input)
	})
	if err != nil {
		task.OnImportCompleteFailed(ctx, model, jsonutils.NewString(err.Error()))
	}
}

func (task *LLMInstantModelImportTask) OnImportComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)

	db.OpsLog.LogEvent(model, db.ACT_CREATE, model.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_CREATE, model.GetShortDesc(ctx), task.UserCred, true)

	task.SetStageComplete(ctx, nil)
}

func (task *LLMInstantModelImportTask) OnImportCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)
	task.taskFailed(ctx, model, err.String())
}
