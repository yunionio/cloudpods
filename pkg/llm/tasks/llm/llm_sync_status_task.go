package llm

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMSyncStatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMSyncStatusTask{})
}

func (task *LLMSyncStatusTask) setLLMStatus(ctx context.Context, llm *models.SLLM, status string, reason string) {
	if !task.HasParentTask() {
		llm.SetStatus(ctx, task.UserCred, status, reason)
	}
}

func (task *LLMSyncStatusTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	task.setLLMStatus(ctx, llm, computeapi.VM_SYNC_FAIL, err)
	db.OpsLog.LogEvent(llm, db.ACT_SYNC_STATUS, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_SYNC_STATUS, err, task.UserCred, false)
	// llm.NotifyRequest(ctx, task.GetUserCred(), notify.ActionStart, nil, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMSyncStatusTask) taskComplete(ctx context.Context, _ *models.SLLM) {
	// phone.SyncStatus(ctx, task.UserCred)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	task.setLLMStatus(ctx, llm, apis.LLM_STATUS_SYNCSTATUS, "LLMSyncStatusTask.OnInit")

	s := auth.GetSession(ctx, task.UserCred, "")
	_, err := compute.Servers.PerformAction(s, llm.CmpId, "syncstatus", nil)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
	task.SetStage("OnSyncStatusComplete", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_, err := llm.WaitServerStatus(ctx, task.UserCred, []string{
			computeapi.VM_RUNNING,
			computeapi.VM_READY,
			computeapi.VM_UNKNOWN,
			computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
			computeapi.POD_STATUS_CONTAINER_EXITED,
			computeapi.POD_STATUS_UPLOADING_STATUS_FAILED,
		}, 1800)
		if err != nil {
			return nil, errors.Wrap(err, "WaitServerStatus")
		}

		time.Sleep(1 * time.Second)

		// 有可能 server 变为 ready 之后 又变为 sync_container_status
		srv, err := llm.WaitServerStatus(ctx, task.UserCred, []string{
			computeapi.VM_RUNNING,
			computeapi.VM_READY,
			computeapi.VM_UNKNOWN,
			computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
			computeapi.POD_STATUS_CONTAINER_EXITED,
			computeapi.POD_STATUS_UPLOADING_STATUS_FAILED,
		}, 1800)
		if err != nil {
			return nil, errors.Wrap(err, "WaitServerStatus")
		}

		if utils.IsInArray(srv.Status, []string{
			computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
			computeapi.POD_STATUS_CONTAINER_EXITED,
		}) {
			params := computeapi.ServerStopInput{
				IsForce:     true,
				TimeoutSecs: 10,
			}
			_, err := compute.Servers.PerformAction(s, llm.CmpId, "stop", jsonutils.Marshal(params))
			if err != nil {
				return nil, errors.Wrap(err, "ServerStop")
			}
			srv, err := llm.WaitServerStatus(ctx, task.UserCred, []string{
				computeapi.VM_READY,
			}, 1800)
			if err != nil {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}
			task.setLLMStatus(ctx, llm, srv.Status, "stop server")
		} else {
			task.setLLMStatus(ctx, llm, srv.Status, "WaitServerStatus")
		}

		volume, _ := llm.GetVolume()
		if volume != nil {
			disk, err := volume.GetDisk(ctx)
			if err != nil {
				task.setLLMStatus(ctx, llm, computeapi.VM_DISK_RESET_FAIL, errors.Wrap(err, "GetDisk").Error())
				return nil, errors.Wrap(err, "GetDisk")
			}
			if disk.Status != computeapi.DISK_READY {
				// do disk syncstatus
				_, err := compute.Disks.PerformAction(s, disk.Id, "syncstatus", nil)
				if err != nil {
					return nil, errors.Wrap(err, "disk perform action syncstatus")
				}
				_, err = volume.WaitDiskStatus(ctx, task.UserCred, []string{computeapi.DISK_READY}, 1800)
				if err != nil {
					return nil, errors.Wrap(err, "volume.WaitDiskStatus")
				}
			}
		}

		return nil, nil
	})
}

func (task *LLMSyncStatusTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	d := obj.(*models.SLLM)
	task.taskComplete(ctx, d)
}

func (task *LLMSyncStatusTask) OnSyncStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	d := obj.(*models.SLLM)
	task.taskFailed(ctx, d, err.String())
}
