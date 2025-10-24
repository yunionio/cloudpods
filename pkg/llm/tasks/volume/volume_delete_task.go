package volume

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type VolumeDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VolumeDeleteTask{})
}

func (volumeDeleteTask *VolumeDeleteTask) taskFailed(ctx context.Context, volume *models.SVolume, status string, err error) {
	volume.SetStatus(ctx, volumeDeleteTask.UserCred, status, err.Error())
	db.OpsLog.LogEvent(volume, db.ACT_DELETE, err, volumeDeleteTask.UserCred)
	logclient.AddActionLogWithStartable(volumeDeleteTask, volume, logclient.ACT_DELETE, err, volumeDeleteTask.UserCred, false)
	volumeDeleteTask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (volumeDeleteTask *VolumeDeleteTask) taskComplete(ctx context.Context, volume *models.SVolume) {
	volume.RealDelete(ctx, volumeDeleteTask.GetUserCred())
	volumeDeleteTask.SetStageComplete(ctx, nil)
}

func (volumeDeleteTask *VolumeDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	volume := obj.(*models.SVolume)
	if len(volume.SvrId) == 0 {
		volumeDeleteTask.taskComplete(ctx, volume)
		return
	}
	s := auth.GetSession(ctx, volumeDeleteTask.UserCred, "")
	_, err := compute.Disks.Delete(s, volume.SvrId, nil)
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			volumeDeleteTask.taskComplete(ctx, volume)
			return
		}
		volumeDeleteTask.taskFailed(ctx, volume, computeapi.DISK_DEALLOC_FAILED, errors.Wrap(err, "wait status"))
		return
	}
	for i := 0; i < 60; i++ {
		_, err := compute.Disks.GetById(s, volume.SvrId, jsonutils.Marshal(map[string]interface{}{
			"scope": "max",
		}))
		if err != nil {
			if strings.Contains(err.Error(), "ResourceNotFoundError") {
				volumeDeleteTask.taskComplete(ctx, volume)
				return
			}
		}
		time.Sleep(30 * time.Second)
	}
	volumeDeleteTask.taskFailed(ctx, volume, computeapi.MODELARTS_POOL_STATUS_TIMEOUT, errors.Wrap(err, "wait status"))
}
