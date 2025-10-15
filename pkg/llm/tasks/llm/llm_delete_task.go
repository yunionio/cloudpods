package llm

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/errors"
)

type LLMDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMDeleteTask{})
}

func (task *LLMDeleteTask) taskFailed(ctx context.Context, llm *models.SLLM, err error) {
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(llm, db.ACT_DELETE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_DELETING, "start delete")

	if len(llm.SvrId) == 0 {
		task.OnLLMRefreshStatusComplete(ctx, llm, nil)
		return
	}

	err := llm.ServerDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, llm, err)
		return
	}
	task.SetStage("OnLLMRefreshStatusComplete", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err = llm.WaitDelete(ctx, task.UserCred, 1800)
		if err != nil {
			return nil, errors.Wrap(err, "llm.WaitDelete")
		}

		return nil, nil
	})
}

func (task *LLMDeleteTask) OnLLMRefreshStatusCompleteFailed(ctx context.Context, llm *models.SLLM, err jsonutils.JSONObject) {
	task.taskFailed(ctx, llm, errors.Error(err.String()))
}

func (task *LLMDeleteTask) OnLLMRefreshStatusComplete(ctx context.Context, llm *models.SLLM, body jsonutils.JSONObject) {
	volume, err := llm.GetVolume()
	if err != nil {
		if errors.Cause(err) != errors.ErrNotFound {
			task.taskFailed(ctx, llm, err)
			return
		}
	}
	if volume != nil {
		task.SetStage("OnLLMVolumeDeleteComplete", nil)
		volume.StartDeleteTask(ctx, task.UserCred, task.GetTaskId())
	} else {
		task.OnLLMVolumeDeleteComplete(ctx, llm, nil)
	}
}

func (task *LLMDeleteTask) OnLLMVolumeDeleteCompleteFailed(ctx context.Context, llm *models.SLLM, err jsonutils.JSONObject) {
	task.taskFailed(ctx, llm, errors.Error(err.String()))
}

func (task *LLMDeleteTask) OnLLMVolumeDeleteComplete(ctx context.Context, llm *models.SLLM, body jsonutils.JSONObject) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		if errors.Cause(err) != errors.ErrNotFound && errors.Cause(err) != sql.ErrNoRows {
			task.taskFailed(ctx, llm, err)
			return
		}
	}
	if lc != nil {
		task.SetStage("OnLLMContainerDeleteComplete", nil)
		lc.StartDeleteTask(ctx, task.UserCred, task.GetTaskId())
	} else {
		task.OnLLMContainerDeleteComplete(ctx, llm, nil)
	}
}

func (task *LLMDeleteTask) OnLLMContainerDeleteCompleteFailed(ctx context.Context, llm *models.SLLM, err jsonutils.JSONObject) {
	task.taskFailed(ctx, llm, errors.Error(err.String()))
}

func (task *LLMDeleteTask) OnLLMContainerDeleteComplete(ctx context.Context, llm *models.SLLM, body jsonutils.JSONObject) {
	err := llm.RealDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, llm, err)
		return
	}
	task.SetStageComplete(ctx, nil)
}
