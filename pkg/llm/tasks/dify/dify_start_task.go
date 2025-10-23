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
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/llm/tasks/worker"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DifyStartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DifyStartTask{})
}

func (task *DifyStartTask) taskFailed(ctx context.Context, dify *models.SDify, err string) {
	dify.SetStatus(ctx, task.UserCred, api.LLM_STATUS_START_FAIL, err)
	db.OpsLog.LogEvent(dify, db.ACT_START, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, dify, logclient.ACT_START, err, task.UserCred, false)
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStart, nil, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *DifyStartTask) taskComplete(ctx context.Context, dify *models.SDify) {
	dify.SetStatus(ctx, task.GetUserCred(), api.LLM_STATUS_RUNNING, "start complete")
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStart, nil, true)
	task.SetStageComplete(ctx, nil)
}

func (t *DifyStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestStart(ctx, obj.(*models.SDify))
}

func (t *DifyStartTask) requestStart(ctx context.Context, dify *models.SDify) {
	s := auth.GetSession(ctx, t.GetUserCred(), options.Options.Region)
	_, err := compute.Servers.PerformAction(s, dify.SvrId, "start", nil)
	if err != nil {
		t.taskFailed(ctx, dify, err.Error())
		return
	}

	t.SetStage("OnStarted", nil)
	worker.StartTaskRun(t, func() (jsonutils.JSONObject, error) {
		_, err := dify.WaitServerStatus(ctx, t.GetUserCred(), []string{computeapi.VM_RUNNING}, 900)
		if err != nil {
			return nil, errors.Wrap(err, "WaitServerStatus")
		}
		// time.Sleep(time.Second)
		// _, err = d.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_RUNNING}, 900)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "WaitServerStatus")
		// }
		return nil, nil
	})
	// if err := llm.RunModel(ctx, t.GetUserCred()); nil != err {
	// 	t.OnStartedFailed(ctx, llm, jsonutils.NewString(err.Error()))
	// 	return
	// }
}

func (t *DifyStartTask) OnStartedFailed(ctx context.Context, dify *models.SDify, err jsonutils.JSONObject) {
	t.taskFailed(ctx, dify, err.String())
}

func (t *DifyStartTask) OnStarted(ctx context.Context, dify *models.SDify, reason jsonutils.JSONObject) {
	t.taskComplete(ctx, dify)
}
