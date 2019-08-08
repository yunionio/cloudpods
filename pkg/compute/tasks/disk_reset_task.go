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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskResetTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskResetTask{})
	taskman.RegisterTask(DiskCleanUpSnapshotsTask{})
}

func (self *DiskResetTask) TaskFailed(ctx context.Context, disk *models.SDisk, reason string) {
	disk.SetStatus(self.UserCred, api.DISK_READY, "")
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESET_DISK, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
	guests := disk.GetGuests()
	if len(guests) == 1 {
		guests[0].SetStatus(self.UserCred, api.VM_DISK_RESET_FAIL, reason)
	}
}

func (self *DiskResetTask) TaskCompleted(ctx context.Context, disk *models.SDisk, data *jsonutils.JSONDict) {
	guests := disk.GetGuests()
	if jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		if len(guests) == 1 {
			self.SetStage("OnStartGuest", nil)
			guests[0].StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
		}
	} else {
		if len(guests) == 1 {
			guests[0].SetStatus(self.UserCred, api.VM_READY, "")
		}
		// data不能为空指针，否则会导致AddActionLog抛空指针异常
		if data == nil {
			data = jsonutils.NewDict()
		}
		logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESET_DISK, data, self.UserCred, true)
		self.SetStageComplete(ctx, data)
	}
}

func (self *DiskResetTask) OnStartGuest(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	if data == nil {
		data = jsonutils.NewDict()
	}
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESET_DISK, data, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskResetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := disk.GetStorage()
	if storage == nil {
		self.TaskFailed(ctx, disk, "Disk storage not found")
		return
	}
	host := storage.GetMasterHost()
	if host == nil {
		self.TaskFailed(ctx, disk, "Storage master host not found")
		return
	}
	self.RequestResetDisk(ctx, disk, host)
}

func (self *DiskResetTask) RequestResetDisk(ctx context.Context, disk *models.SDisk, host *models.SHost) {
	snapshotId, err := self.Params.GetString("snapshot_id")
	if err != nil {
		self.TaskFailed(ctx, disk, fmt.Sprintf("Get snapshotId error %s", err.Error()))
		return
	}
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	params := snapshot.GetRegionDriver().GetDiskResetParams(snapshot)

	self.SetStage("OnRequestResetDisk", nil)
	err = host.GetHostDriver().RequestResetDisk(ctx, host, disk, params, self)
	if err != nil {
		self.TaskFailed(ctx, disk, err.Error())
	}
}

func (self *DiskResetTask) OnRequestResetDiskFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, disk, data.String())
}

func (self *DiskResetTask) OnRequestResetDisk(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)

	err := snapshot.GetRegionDriver().OnDiskReset(ctx, self.UserCred, disk, snapshot, data)
	if err != nil {
		log.Errorln(err)
		self.TaskFailed(ctx, disk, fmt.Sprintf("OnRequestResetDisk %s", err.Error()))
		return
	}

	disk.SetStatus(self.UserCred, api.DISK_READY, "")
	self.TaskCompleted(ctx, disk, nil)
}

type DiskCleanUpSnapshotsTask struct {
	SDiskBaseTask
}

func (self *DiskCleanUpSnapshotsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	self.StartCleanUpSnapshots(ctx, disk)
}

func (self *DiskCleanUpSnapshotsTask) StartCleanUpSnapshots(ctx context.Context, disk *models.SDisk) {
	db.OpsLog.LogEvent(disk, db.ACT_DISK_CLEAN_UP_SNAPSHOTS,
		fmt.Sprintf("start clean up disk snapshots: %s", self.Params.String()), self.UserCred)
	host := disk.GetStorage().GetMasterHost()
	self.SetStage("OnCleanUpSnapshots", nil)
	err := host.GetHostDriver().RequestCleanUpDiskSnapshots(ctx, host, disk, self.Params, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *DiskCleanUpSnapshotsTask) OnCleanUpSnapshots(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	convertSnapshots, _ := self.Params.GetArray("convert_snapshots")
	for i := 0; i < len(convertSnapshots); i++ {
		snapshot_id, _ := convertSnapshots[i].GetString()
		iSnapshot, err := models.SnapshotManager.FetchById(snapshot_id)
		if err != nil {
			log.Errorf("OnCleanUpSnapshots Fetch snapshot by id(%s) error:%s", snapshot_id, err.Error())
			continue
		}
		snapshot := iSnapshot.(*models.SSnapshot)
		db.Update(snapshot, func() error {
			snapshot.OutOfChain = true
			return nil
		})
	}
	deleteSnapshots, _ := self.Params.GetArray("delete_snapshots")
	for i := 0; i < len(deleteSnapshots); i++ {
		snapshot_id, _ := convertSnapshots[i].GetString()
		iSnapshot, err := models.SnapshotManager.FetchById(snapshot_id)
		if err != nil {
			log.Errorf("OnCleanUpSnapshots Fetch snapshot by id(%s) error:%s", snapshot_id, err.Error())
			continue
		}
		snapshot := iSnapshot.(*models.SSnapshot)
		snapshot.RealDelete(ctx, self.UserCred)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *DiskCleanUpSnapshotsTask) OnCleanUpSnapshotsFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(disk, db.ACT_DISK_CLEAN_UP_SNAPSHOTS_FAIL, data.String(), self.UserCred)
	self.SetStageFailed(ctx, data.String())
}
