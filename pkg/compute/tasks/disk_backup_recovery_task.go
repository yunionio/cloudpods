// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	backup.SetStatus(ctx, self.UserCred, api.BACKUP_STATUS_RECOVERY_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_RECOVERY, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *DiskBackupRecoveryTask) taskSuccess(ctx context.Context, backup *models.SDiskBackup, data *jsonutils.JSONDict) {
	backup.SetStatus(ctx, self.UserCred, api.BACKUP_STATUS_READY, "")
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
	input.Description = fmt.Sprintf("recovered from backup %s", backup.GetName())
	input.Hypervisor = api.HYPERVISOR_KVM
	if backup.DiskConfig != nil {
		if backup.DiskConfig.BackupAsTar != nil && backup.DiskConfig.BackupAsTar.ContainerId != "" {
			input.Hypervisor = api.HYPERVISOR_POD
		}
	}
	input.DiskConfig = diskConfig
	ownerId := backup.GetOwnerId()
	input.ProjectDomainId = ownerId.GetProjectDomainId()
	input.ProjectId = ownerId.GetProjectId()

	if backup.IsEncrypted() {
		input.EncryptKeyId = &backup.EncryptKeyId
	}

	params := input.JSON(input)
	diskObj, err := db.DoCreate(models.DiskManager, ctx, self.UserCred, nil, params, ownerId)
	if err != nil {
		self.taskFaild(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
	disk := diskObj.(*models.SDisk)
	err = backup.InheritTo(ctx, self.UserCred, disk)
	if err != nil {
		self.taskFaild(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}

	func() {
		lockman.LockObject(ctx, disk)
		defer lockman.ReleaseObject(ctx, disk)

		disk.PostCreate(ctx, self.UserCred, backup.GetOwnerId(), nil, params)
	}()

	params.Set("disk_id", jsonutils.NewString(disk.Id))
	self.SetStage("OnCreateDisk", params)

	params.Set("parent_task_id", jsonutils.NewString(self.GetTaskId()))
	models.DiskManager.OnCreateComplete(ctx, []db.IModel{disk}, self.UserCred, ownerId, nil, []jsonutils.JSONObject{params})
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
