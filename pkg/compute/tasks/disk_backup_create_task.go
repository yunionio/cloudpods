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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rand"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskBackupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskBackupCreateTask{})
}

func (self *DiskBackupCreateTask) taskFailed(ctx context.Context, backup *models.SDiskBackup, reason jsonutils.JSONObject, status string) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	if len(snapshotId) > 0 {
		snapshotModel, err := models.SnapshotManager.FetchById(snapshotId)
		if err != nil {
			log.Errorf("unable to get snapshot %s: %s", snapshotId, err.Error())
		} else {
			err := snapshotModel.(*models.SSnapshot).RealDelete(ctx, self.UserCred)
			if err != nil {
				log.Errorf("unable to delete snapshot %s: %s", snapshotId, err.Error())
			}
		}
	}
	reasonStr, _ := reason.GetString()
	backup.SetStatus(ctx, self.UserCred, status, reasonStr)
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_CREATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *DiskBackupCreateTask) taksSuccess(ctx context.Context, backup *models.SDiskBackup, data *jsonutils.JSONDict) {
	backup.SetStatus(ctx, self.UserCred, api.BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_CREATE, backup.GetShortDesc(ctx), self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    backup,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, data)
}

func (self *DiskBackupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDiskBackup)
	if self.Params.Contains("snapshot_id") {
		self.OnSnapshot(ctx, backup, nil)
		return
	}
	backup.SetStatus(ctx, self.UserCred, api.BACKUP_STATUS_SNAPSHOT, "")
	snapshot, err := self.CreateSnapshot(ctx, backup)
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_SNAPSHOT_FAILED)
		return
	}
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.GetId()))
	self.SetStage("OnSnapshot", params)
	err = snapshot.StartSnapshotCreateTask(ctx, self.UserCred, nil, self.GetId())
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_SNAPSHOT_FAILED)
		return
	}
}

func (self *DiskBackupCreateTask) OnSnapshot(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	if self.Params.Contains("only_snapshot") {
		p := jsonutils.NewDict()
		p.Set("snapshot_id", jsonutils.NewString(snapshotId))
		p.Set("disk_backup_id", jsonutils.NewString(backup.GetId()))
		self.taksSuccess(ctx, backup, p)
		return
	}
	backup.SetStatus(ctx, self.UserCred, api.BACKUP_STATUS_SAVING, "")
	self.SetStage("OnSave", nil)
	rd, err := backup.GetRegionDriver()
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_SAVE_FAILED)
		return
	}
	if err := rd.RequestCreateBackup(ctx, backup, snapshotId, self); err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_SAVE_FAILED)
		return
	}
}

func (self *DiskBackupCreateTask) OnSnapshotFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	// remove snapshot
	self.taskFailed(ctx, backup, data, api.BACKUP_STATUS_SNAPSHOT_FAILED)
}

func (self *DiskBackupCreateTask) OnSave(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	// cleanup snapshot
	snapshotId, _ := self.Params.GetString("snapshot_id")
	self.SetStage("OnCleanupSnapshot", nil)
	snapshotModel, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_CLEANUP_SNAPSHOT_FAILED)
		return
	}
	log.Infof("data from RequestCreateBackup: %s", data)
	sizeMb, _ := data.Int("size_mb")
	db.Update(backup, func() error {
		backup.SizeMb = int(sizeMb)
		return nil
	})
	snapshot := snapshotModel.(*models.SSnapshot)
	err = snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.GetId(), 0, 0)
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_CLEANUP_SNAPSHOT_FAILED)
		return
	}
}

func (self *DiskBackupCreateTask) OnSaveFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	snapshotModel, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		log.Errorf("unable to cleanup snapshot: %s", err.Error())
		self.taskFailed(ctx, backup, data, api.BACKUP_STATUS_SAVE_FAILED)
		return
	}
	snapshot := snapshotModel.(*models.SSnapshot)
	self.taskFailed(ctx, backup, data, api.BACKUP_STATUS_SAVE_FAILED)
	err = snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.GetId(), 0, 0)
	if err != nil {
		log.Errorf("unable to cleanup snapshot: %s", err.Error())
		self.taskFailed(ctx, backup, data, api.BACKUP_STATUS_SAVE_FAILED)
		return
	}
	self.taskFailed(ctx, backup, data, api.BACKUP_STATUS_SAVE_FAILED)
}

func (self *DiskBackupCreateTask) OnCleanupSnapshot(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	snapshotModel, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		log.Errorf("unable to get snapshot %s: %s", snapshotId, err.Error())
	} else {
		err := snapshotModel.(*models.SSnapshot).RealDelete(ctx, self.UserCred)
		if err != nil {
			log.Errorf("unable to delete snapshot %s: %s", snapshotId, err.Error())
		}
	}
	self.taksSuccess(ctx, backup, nil)
}

func (self *DiskBackupCreateTask) OnCleanupSnapshotFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, backup, data, api.BACKUP_STATUS_CLEANUP_SNAPSHOT_FAILED)
}

func (self *DiskBackupCreateTask) CreateSnapshot(ctx context.Context, diskBackup *models.SDiskBackup) (*models.SSnapshot, error) {
	disk, err := diskBackup.GetDisk()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to getdisk of disk backup %s", disk.GetId())
	}
	guest := disk.GetGuest()
	snapshot, err := func() (*models.SSnapshot, error) {
		lockman.LockClass(ctx, models.SnapshotManager, "name")
		defer lockman.ReleaseClass(ctx, models.SnapshotManager, "name")

		snapshotName, err := db.GenerateName(ctx, models.SnapshotManager, self.GetUserCred(),
			fmt.Sprintf("%s-%s", diskBackup.Name, rand.String(8)))
		if err != nil {
			return nil, errors.Wrap(err, "Generate snapshot name")
		}

		return models.SnapshotManager.CreateSnapshot(
			ctx, self.GetUserCred(), api.SNAPSHOT_MANUAL, disk.GetId(),
			guest.Id, "", snapshotName, -1, true, diskBackup.GetId())
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create snapshot of disk %s", disk.GetId())
	}
	return snapshot, nil
}
