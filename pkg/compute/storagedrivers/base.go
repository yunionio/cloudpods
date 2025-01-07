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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	if guest == nil {
		var host *models.SHost
		disk, err := models.DiskManager.FetchById(snapshot.DiskId)
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrap(err, "get disk by snapshot")
		}
		if disk != nil {
			sDisk := disk.(*models.SDisk)
			if hostId := sDisk.GetLastAttachedHost(ctx, task.GetUserCred()); hostId != "" {
				host = models.HostManager.FetchHostById(hostId)
			}
		} else {
			if hostId := snapshot.GetMetadata(ctx, api.DISK_META_LAST_ATTACHED_HOST, task.GetUserCred()); hostId != "" {
				host = models.HostManager.FetchHostById(hostId)
			}
		}
		if host == nil {
			storage := snapshot.GetStorage()
			host, err = storage.GetMasterHost()
			if err != nil {
				return err
			}
		}

		convertSnapshot, err := models.SnapshotManager.GetConvertSnapshot(snapshot)
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrap(err, "get convert snapshot")
		}
		params := jsonutils.NewDict()
		params.Set("delete_snapshot", jsonutils.NewString(snapshot.Id))
		params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))

		if disk != nil {
			sDisk, _ := disk.(*models.SDisk)
			if sDisk.IsEncrypted() {
				if encryptInfo, err := sDisk.GetEncryptInfo(ctx, task.GetUserCred()); err != nil {
					return errors.Wrap(err, "faild get encryptInfo")
				} else {
					params.Set("encrypt_info", jsonutils.Marshal(encryptInfo))
				}
			}
		}
		if !snapshot.OutOfChain {
			if convertSnapshot != nil {
				params.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
			} else if disk != nil {
				params.Set("block_stream", jsonutils.JSONTrue)
			} else {
				params.Set("auto_deleted", jsonutils.JSONTrue)
			}
		} else {
			params.Set("auto_deleted", jsonutils.JSONTrue)
		}

		drv, err := host.GetHostDriver()
		if err != nil {
			return err
		}

		return drv.RequestDeleteSnapshotWithoutGuest(ctx, host, snapshot, params, task)
	}

	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}

	if jsonutils.QueryBoolean(task.GetParams(), "reload_disk", false) && snapshot.OutOfChain {
		guest.SetStatus(ctx, task.GetUserCred(), api.VM_SNAPSHOT, "Start Reload Snapshot")
		params := jsonutils.NewDict()
		params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
		return drv.RequestReloadDiskSnapshot(ctx, guest, task, params)
	} else {
		convertSnapshot, err := models.SnapshotManager.GetConvertSnapshot(snapshot)
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrap(err, "get convert snapshot")
		}
		snapshot.SetStatus(ctx, task.GetUserCred(), api.SNAPSHOT_DELETING, "On SnapshotDeleteTask StartDeleteSnapshot")
		params := jsonutils.NewDict()
		params.Set("delete_snapshot", jsonutils.NewString(snapshot.Id))
		params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))

		disk, err := models.DiskManager.FetchById(snapshot.DiskId)
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrap(err, "get disk by snapshot")
		}
		sDisk, _ := disk.(*models.SDisk)
		if sDisk.IsEncrypted() {
			if encryptInfo, err := sDisk.GetEncryptInfo(ctx, task.GetUserCred()); err != nil {
				return errors.Wrap(err, "faild get encryptInfo")
			} else {
				params.Set("encrypt_info", jsonutils.Marshal(encryptInfo))
			}
		}

		if !snapshot.OutOfChain {
			if convertSnapshot != nil {
				params.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
			} else {
				params.Set("block_stream", jsonutils.JSONTrue)
			}
		} else {
			params.Set("auto_deleted", jsonutils.JSONTrue)
		}
		taskParams := task.GetParams()
		if taskParams.Contains("snapshot_total_count") {
			totalCnt, _ := taskParams.Get("snapshot_total_count")
			params.Set("snapshot_total_count", totalCnt)
			deletedCnt, _ := taskParams.Get("deleted_snapshot_count")
			params.Set("deleted_snapshot_count", deletedCnt)
		}

		guest.SetStatus(ctx, task.GetUserCred(), api.VM_SNAPSHOT_DELETE, "Start Delete Snapshot")
		return drv.RequestDeleteSnapshot(ctx, guest, task, params)
	}
}

func (self *SBaseStorageDriver) SnapshotIsOutOfChain(disk *models.SDisk) bool {
	return false
}

func (self *SBaseStorageDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, data jsonutils.JSONObject) error {
	return disk.CleanUpDiskSnapshots(ctx, userCred, snapshot)
}
