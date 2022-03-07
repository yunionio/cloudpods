package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BackupStorageSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BackupStorageSyncstatusTask{})
}

func (self *BackupStorageSyncstatusTask) taskFailed(ctx context.Context, bs *models.SBackupStorage, err jsonutils.JSONObject) {
	logclient.AddActionLogWithContext(ctx, bs, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *BackupStorageSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	bs := obj.(*models.SBackupStorage)

	self.SetStage("OnBackupStorageSyncStatus", nil)
	err := bs.GetRegionDriver().RequestSyncBackupStorageStatus(ctx, self.GetUserCred(), bs, self)
	if err != nil {
		self.taskFailed(ctx, bs, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *BackupStorageSyncstatusTask) OnBackupStorageSyncStatus(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *BackupStorageSyncstatusTask) OnBackupStorageSyncStatusFailed(ctx context.Context, backup *models.SBackupStorage, data jsonutils.JSONObject) {
	self.taskFailed(ctx, backup, data)
}
