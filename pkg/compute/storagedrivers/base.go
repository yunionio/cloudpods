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

package storagedrivers

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseStorageDriver struct {
}

func (self *SBaseStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	return fmt.Errorf("Not Implement ValidateCreateData")
}

func (self *SBaseStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {

}

func (self *SBaseStorageDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, input api.StorageUpdateInput) (api.StorageUpdateInput, error) {
	return input, nil
}

func (self *SBaseStorageDriver) DoStorageUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaseStorageDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	if snapshot.RefCount > 0 {
		return httperrors.NewBadRequestError("Snapshot reference(by disk) count > 0, can not delete")
	}

	if !snapshot.OutOfChain && snapshot.FakeDeleted {
		_, err := models.SnapshotManager.GetConvertSnapshot(snapshot)
		if err != nil {
			return httperrors.NewBadRequestError("disk need at least one of snapshot as backing file")
		}
	}
	return nil
}

func (self *SBaseStorageDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, input *api.SnapshotCreateInput) error {
	guests := disk.GetGuests()
	if len(guests) != 1 {
		return httperrors.NewBadRequestError("Disk %s dosen't attach guest ?", disk.Id)
	}
	guest := guests[0]
	if len(guest.BackupHostId) > 0 {
		return httperrors.NewBadRequestError(
			"Disk attached Guest has backup, Can't create snapshot")
	}
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return httperrors.NewInvalidStatusError("Cannot do snapshot when VM in status %s", guest.Status)
	}
	q := models.SnapshotManager.Query()
	cnt, err := q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), disk.Id),
		sqlchemy.Equals(q.Field("created_by"), api.SNAPSHOT_MANUAL),
		sqlchemy.IsFalse(q.Field("fake_deleted")))).CountWithError()
	if err != nil {
		return httperrors.NewInternalServerError("check disk snapshot count fail %s", err)
	}
	if cnt >= options.Options.DefaultMaxManualSnapshotCount {
		return httperrors.NewBadRequestError("Disk %s snapshot full, cannot take any more", disk.Id)
	}
	return nil
}

func (self *SBaseStorageDriver) RequestCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	guest, err := snapshot.GetGuest()
	if err != nil {
		return err
	}
	var params = jsonutils.NewDict()
	params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
	nt, err := taskman.TaskManager.NewTask(ctx, "GuestDiskSnapshotTask", guest, task.GetUserCred(), params, task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	nt.ScheduleRun(nil)
	return nil
}

func (self *SBaseStorageDriver) RequestDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	guest, err := snapshot.GetGuest()
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	}

	if jsonutils.QueryBoolean(task.GetParams(), "reload_disk", false) && snapshot.OutOfChain {
		guest.SetStatus(task.GetUserCred(), api.VM_SNAPSHOT, "Start Reload Snapshot")
		params := jsonutils.NewDict()
		params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
		return guest.GetDriver().RequestReloadDiskSnapshot(ctx, guest, task, params)
	} else {
		if !snapshot.FakeDeleted {
			snapshot.SetStatus(task.GetUserCred(), compute.SNAPSHOT_READY, "snapshot fake_delete")
			task.SetStageComplete(ctx, nil)
			return snapshot.FakeDelete(task.GetUserCred())
		}

		convertSnapshot, _ := models.SnapshotManager.GetConvertSnapshot(snapshot)
		if convertSnapshot == nil {
			return fmt.Errorf("snapshot dose not have convert snapshot")
		}
		snapshot.SetStatus(task.GetUserCred(), api.SNAPSHOT_DELETING, "On SnapshotDeleteTask StartDeleteSnapshot")
		params := jsonutils.NewDict()
		params.Set("delete_snapshot", jsonutils.NewString(snapshot.Id))
		params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
		if !snapshot.OutOfChain {
			params.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
			var FakeDelete = jsonutils.JSONFalse
			if snapshot.CreatedBy == api.SNAPSHOT_MANUAL && snapshot.FakeDeleted == false {
				FakeDelete = jsonutils.JSONTrue
			}
			params.Set("pending_delete", FakeDelete)
		} else {
			params.Set("auto_deleted", jsonutils.JSONTrue)
		}
		guest.SetStatus(task.GetUserCred(), api.VM_SNAPSHOT_DELETE, "Start Delete Snapshot")
		return guest.GetDriver().RequestDeleteSnapshot(ctx, guest, task, params)
	}
}

func (self *SBaseStorageDriver) SnapshotIsOutOfChain(disk *models.SDisk) bool {
	return false
}

func (self *SBaseStorageDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, data jsonutils.JSONObject) error {
	return disk.CleanUpDiskSnapshots(ctx, userCred, snapshot)
}
