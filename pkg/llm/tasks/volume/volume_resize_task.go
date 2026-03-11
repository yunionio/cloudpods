package volume

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
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
	db.OpsLog.LogEvent(volume, "resize", err, task.UserCred)
	logclient.AddActionLogWithStartable(task, volume, "resize", err.Error(), task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *VolumeResizeTask) taskComplete(ctx context.Context, volume *models.SVolume) {
	volume.SetStatus(ctx, task.UserCred, computeapi.DISK_READY, "")
	task.SetStageComplete(ctx, nil)
}

func (task *VolumeResizeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)

	input := api.VolumeResizeTaskInput{}
	if task.GetParams() != nil {
		_ = task.GetParams().Unmarshal(&input)
	}
	if input.SizeMB <= 0 {
		task.taskFailed(ctx, volume, errors.Wrap(httperrors.ErrInputParameter, "missing size_mb"))
		return
	}
	if volume.SizeMB >= input.SizeMB {
		task.taskComplete(ctx, volume)
		return
	}
	if len(volume.CmpId) == 0 {
		task.taskFailed(ctx, volume, errors.Wrap(errors.ErrInvalidStatus, "empty cmp_id"))
		return
	}

	// Resize disk
	s := auth.GetSession(ctx, task.UserCred, "")
	params := jsonutils.NewDict()
	params.Set("size", jsonutils.NewString(fmt.Sprintf("%dM", input.SizeMB)))

	task.SetStage("OnDiskResizeComplete", nil)
	err := s.WithTaskCallback(task.GetId(), func() error {
		_, err := compute.Disks.PerformAction(s, volume.CmpId, "resize", params)
		return err
	})
	if err != nil {
		task.taskFailed(ctx, volume, errors.Wrap(err, "disk resize"))
		return
	}
}

func (task *VolumeResizeTask) OnDiskResizeComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)

	input := api.VolumeResizeTaskInput{}
	if task.GetParams() != nil {
		_ = task.GetParams().Unmarshal(&input)
	}
	if _, err := volume.WaitDiskStatus(ctx, task.UserCred, []string{computeapi.DISK_READY}, 7200); err != nil {
		task.taskFailed(ctx, volume, err)
		return
	}
	_, err := db.Update(volume, func() error {
		volume.SizeMB = input.SizeMB
		return nil
	})
	if err != nil {
		task.taskFailed(ctx, volume, errors.Wrap(err, "update volume size_mb"))
		return
	}
	task.taskComplete(ctx, volume)
}

func (task *VolumeResizeTask) OnDiskResizeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)
	task.taskFailed(ctx, volume, errors.Error(err.String()))
}
