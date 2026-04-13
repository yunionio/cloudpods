package volume

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	llmapi "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/tasks/worker"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type VolumeResizeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VolumeResizeTask{})
}

func (task *VolumeResizeTask) taskFailed(ctx context.Context, volume *models.SVolume, err error) {
	volume.SetStatus(ctx, task.UserCred, computeapi.DISK_RESIZE_FAILED, err.Error())
	db.OpsLog.LogEvent(volume, db.ACT_RESIZE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, volume, logclient.ACT_RESIZE, err.Error(), task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *VolumeResizeTask) taskComplete(ctx context.Context, volume *models.SVolume) {
	volume.SetStatus(ctx, task.UserCred, computeapi.DISK_READY, "resize success")
	logclient.AddActionLogWithStartable(task, volume, logclient.ACT_RESIZE, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}

func (task *VolumeResizeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)
	input := llmapi.VolumeResizeTaskInput{}
	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, volume, errors.Wrap(err, "Unmarshal SVolumeResizeTaskInput"))
		return
	}

	volume.SetStatus(ctx, task.UserCred, llmapi.VOLUME_STATUS_RESIZING, "resizing")

	if int(input.SizeMB) > volume.SizeMB {
		// need to resize disk
		s := auth.GetSession(ctx, task.UserCred, "")
		params := computeapi.DiskResizeInput{}
		params.Size = fmt.Sprintf("%dM", input.SizeMB)
		_, err := compute.Disks.PerformAction(s, volume.CmpId, "resize", jsonutils.Marshal(params))
		if err != nil {
			task.taskFailed(ctx, volume, errors.Wrap(err, "disk resize"))
			return
		}

		task.SetStage("OnDiskResizeComplete", nil)
		worker.BackupTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := volume.WaitDiskStatus(ctx, task.UserCred, []string{computeapi.DISK_READY}, 3600)
			if err != nil {
				return nil, errors.Wrap(err, "WaitDiskStatus")
			}
			d := volume.GetLLM()
			if d != nil && len(input.DesktopStatus) > 0 {
				_, err = d.WaitServerStatus(ctx, task.UserCred, []string{input.DesktopStatus}, 3600)
				if err != nil {
					return nil, errors.Wrap(err, "WaitServerStatus")
				}
			}
			// update volume size
			err = volume.UpdateSize(ctx, task.UserCred, input.SizeMB)
			if err != nil {
				return nil, errors.Wrap(err, "UpdateSize")
			}
			return nil, nil
		})
	} else {
		task.OnDiskResizeComplete(ctx, volume, nil)
	}
}

func (task *VolumeResizeTask) OnDiskResizeComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)
	task.taskComplete(ctx, volume)
}

func (task *VolumeResizeTask) OnDiskResizeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)
	task.taskFailed(ctx, volume, errors.Error(err.String()))
}
