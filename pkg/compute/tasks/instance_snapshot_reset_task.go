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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceSnapshotResetTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotResetTask{})
}

func (self *InstanceSnapshotResetTask) taskFail(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, reason string) {

	if guest == nil {
		guest = models.GuestManager.FetchGuestById(isp.GuestId)
	}
	guest.SetStatus(self.UserCred, compute.VM_SNAPSHOT_RESET_FAILED, reason)

	db.OpsLog.LogEvent(guest, db.ACT_VM_RESET_SNAPSHOT_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_RESET, false, self.UserCred, false)
	notifyclient.NotifySystemError(guest.GetId(), isp.Name, compute.VM_SNAPSHOT_RESET_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotResetTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, data jsonutils.JSONObject) {

	if guest == nil {
		guest = models.GuestManager.FetchGuestById(isp.GuestId)
	}
	guest.StartSyncstatus(ctx, self.UserCred, "")

	db.OpsLog.LogEvent(isp, db.ACT_VM_RESET_SNAPSHOT, "instance snapshot reset success", self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_RESET, false, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotResetTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	self.GuestDiskResetTask(ctx, isp, guest, 0)
}

func (self *InstanceSnapshotResetTask) GuestDiskResetTask(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, diskIndex int) {

	disks := guest.GetDisks()
	if diskIndex >= len(disks) {
		self.taskComplete(ctx, isp, guest, nil)
		return
	}

	isj, err := isp.GetInstanceSnapshotJointAt(diskIndex)
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}

	params := jsonutils.NewDict()
	params.Set("disk_index", jsonutils.NewInt(int64(diskIndex)))
	self.SetStage("OnDiskReset", params)

	disk := disks[diskIndex].GetDisk()
	err = disk.StartResetDisk(ctx, self.UserCred, isj.SnapshotId, false, guest, self.Id)
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}
}

func (self *InstanceSnapshotResetTask) OnDiskReset(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	diskIndex, err := self.Params.Int("disk_index")
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}
	self.GuestDiskResetTask(ctx, isp, guest, int(diskIndex+1))
}

func (self *InstanceSnapshotResetTask) OnDiskResetFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFail(ctx, isp, nil, data.String())
}
