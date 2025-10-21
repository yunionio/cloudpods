package dify

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/errors"
)

type DifyDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DifyDeleteTask{})
}

func (task *DifyDeleteTask) taskFailed(ctx context.Context, dify *models.SDify, err error) {
	dify.SetStatus(ctx, task.UserCred, api.LLM_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(dify, db.ACT_DELETE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, dify, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *DifyDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dify := obj.(*models.SDify)
	dify.SetStatus(ctx, task.UserCred, api.LLM_STATUS_DELETING, "start delete")

	if len(dify.SvrId) == 0 {
		task.OnDifyRefreshStatusComplete(ctx, dify, nil)
		return
	}

	err := dify.ServerDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, dify, err)
		return
	}
	task.SetStage("OnDifyRefreshStatusComplete", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err = dify.WaitDelete(ctx, task.UserCred, 1800)
		if err != nil {
			return nil, errors.Wrap(err, "llm.WaitDelete")
		}

		return nil, nil
	})
}

func (task *DifyDeleteTask) OnDifyRefreshStatusCompleteFailed(ctx context.Context, dify *models.SDify, err jsonutils.JSONObject) {
	task.taskFailed(ctx, dify, errors.Error(err.String()))
}

func (task *DifyDeleteTask) OnDifyRefreshStatusComplete(ctx context.Context, dify *models.SDify, body jsonutils.JSONObject) {
	volume, err := dify.GetVolume()
	if err != nil {
		if errors.Cause(err) != errors.ErrNotFound {
			task.taskFailed(ctx, dify, err)
			return
		}
	}
	if volume != nil {
		task.SetStage("OnDifyVolumeDeleteComplete", nil)
		volume.StartDeleteTask(ctx, task.UserCred, task.GetTaskId())
	} else {
		task.OnDifyVolumeDeleteComplete(ctx, dify, nil)
	}
}

func (task *DifyDeleteTask) OnDifyVolumeDeleteCompleteFailed(ctx context.Context, dify *models.SDify, err jsonutils.JSONObject) {
	task.taskFailed(ctx, dify, errors.Error(err.String()))
}

func (task *DifyDeleteTask) OnDifyVolumeDeleteComplete(ctx context.Context, dify *models.SDify, body jsonutils.JSONObject) {
	err := dify.RealDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, dify, err)
		return
	}
	task.SetStageComplete(ctx, nil)
}
