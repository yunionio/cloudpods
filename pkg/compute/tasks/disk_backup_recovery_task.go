package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	compute_modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskBackupRecoveryTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskBackupRecoveryTask{})
}

func (self *DiskBackupRecoveryTask) taskFaild(ctx context.Context, backup *models.SDiskBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	backup.SetStatus(self.UserCred, api.BACKUP_STATUS_RECOVERY_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_RECOVERY, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *DiskBackupRecoveryTask) taskSuccess(ctx context.Context, backup *models.SDiskBackup, data *jsonutils.JSONDict) {
	backup.SetStatus(self.UserCred, api.BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_RECOVERY, nil, self.UserCred, true)
	self.SetStageComplete(ctx, data)
}

func (self *DiskBackupRecoveryTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDiskBackup)
	diskName, _ := self.Params.GetString("disk_name")
	if diskName == "" {
		diskName = backup.DiskConfig.Name
	}
	diskConfig := &backup.DiskConfig.DiskConfig
	diskConfig.ImageId = ""
	diskConfig.SnapshotId = ""
	diskConfig.BackupId = backup.GetId()
	input := api.DiskCreateInput{}
	input.GenerateName = diskName
	input.Description = fmt.Sprintf("recovery from backup %s", backup.GetName())
	input.Hypervisor = api.HYPERVISOR_KVM
	input.DiskConfig = diskConfig

	taskHeader := self.GetTaskRequestHeader()
	session := auth.GetSession(ctx, self.UserCred, "", "")
	log.Infof("task_notify_url: %s\ntask_id: %s\n", taskHeader.Get(mcclient.TASK_NOTIFY_URL), taskHeader.Get(mcclient.TASK_ID))
	session.Header.Set(mcclient.TASK_NOTIFY_URL, taskHeader.Get(mcclient.TASK_NOTIFY_URL))
	session.Header.Set(mcclient.TASK_ID, taskHeader.Get(mcclient.TASK_ID))
	diskData, err := compute_modules.Disks.Create(session, jsonutils.Marshal(input))
	if err != nil {
		self.taskFaild(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
	diskId, _ := diskData.GetString("id")
	params := jsonutils.NewDict()
	params.Set("disk_id", jsonutils.NewString(diskId))
	self.SetStage("OnCreateDisk", params)
}

func (self *DiskBackupRecoveryTask) OnCreateDisk(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	diskId, _ := self.Params.GetString("disk_id")
	disk := models.DiskManager.FetchDiskById(diskId)
	if disk == nil {
		self.taskFaild(ctx, backup, jsonutils.NewString(fmt.Sprintf("disk %s disappeared", diskId)))
		return
	}
	imageId := backup.DiskConfig.ImageId
	snapshotId := backup.DiskConfig.SnapshotId
	db.Update(disk, func() error {
		disk.TemplateId = imageId
		disk.SnapshotId = snapshotId
		return nil
	})
	self.taskSuccess(ctx, backup, nil)
}

func (self *DiskBackupRecoveryTask) OnCreateDiskFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	self.taskFaild(ctx, backup, data)
}
