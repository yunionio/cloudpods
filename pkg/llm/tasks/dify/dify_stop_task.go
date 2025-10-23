package dify

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DifyStopTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DifyStopTask{})
}

func (task *DifyStopTask) taskFailed(ctx context.Context, dify *models.SDify, err string) {
	dify.SetStatus(ctx, task.UserCred, api.LLM_STATUS_STOP_FAILED, err)
	db.OpsLog.LogEvent(dify, db.ACT_STOP, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, dify, logclient.ACT_VM_STOP, err, task.UserCred, false)
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStop, nil, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *DifyStopTask) taskComplete(ctx context.Context, dify *models.SDify) {
	if !task.HasParentTask() {
		dify.SetStatus(ctx, task.GetUserCred(), api.LLM_STATUS_READY, "")
	}
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStop, nil, true)
	task.SetStageComplete(ctx, nil)
}

func (task *DifyStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dify := obj.(*models.SDify)

	srv, err := dify.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, dify, errors.Wrap(err, "GetServer").Error())
		return
	}

	if srv.Status == computeapi.VM_READY {
		task.taskComplete(ctx, dify)
		return
	}

	task.SetStage("OnStopComplete", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		s := auth.GetSession(ctx, task.UserCred, "")
		_, err = compute.Servers.PerformAction(s, dify.SvrId, "stop", nil)
		if err != nil {
			task.taskFailed(ctx, dify, err.Error())
			return nil, errors.Wrap(err, "server perform stop")
		}
		_, err := dify.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY}, 600)
		if err != nil {
			if errors.Cause(err) == errors.ErrTimeout {
				params := computeapi.ServerStopInput{
					IsForce:     true,
					TimeoutSecs: 10,
				}
				_, err = compute.Servers.PerformAction(s, dify.SvrId, "stop", jsonutils.Marshal(params))
				if err != nil {
					return nil, errors.Wrap(err, "server perform stop by force")
				}
				_, err := dify.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY}, 600)
				if err != nil {
					return nil, errors.Wrap(err, "WaitServerStatus 2")
				}
			} else {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}
		}
		return nil, nil
	})
}

func (task *DifyStopTask) OnStopComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dify := obj.(*models.SDify)
	task.taskComplete(ctx, dify)
}

func (task *DifyStopTask) OnStopCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	dify := obj.(*models.SDify)
	task.taskFailed(ctx, dify, err.String())
}
