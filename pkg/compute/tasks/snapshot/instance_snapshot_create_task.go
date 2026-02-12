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

package snapshot

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceSnapshotCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotCreateTask{})
}

func (self *InstanceSnapshotCreateTask) SetStageFailed(ctx context.Context, reason jsonutils.JSONObject) {
	self.finalReleasePendingUsage(ctx)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotCreateTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SRegionQuota{}
	err := self.GetPendingUsage(&pendingUsage, 0)
	if err == nil && !pendingUsage.IsEmpty() {
		quotas.CancelPendingUsage(ctx, self.UserCred, &pendingUsage, &pendingUsage, true) // final cleanup
	}
}

func (self *InstanceSnapshotCreateTask) taskFail(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, reason jsonutils.JSONObject) {

	if guest == nil {
		guest = models.GuestManager.FetchGuestById(isp.GuestId)
	}
	isp.SetStatus(ctx, self.UserCred, compute.INSTANCE_SNAPSHOT_FAILED, reason.String())
	guest.SetStatus(ctx, self.UserCred, compute.VM_INSTANCE_SNAPSHOT_FAILED, reason.String())

	db.OpsLog.LogEvent(isp, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, isp, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, isp.GetId(), isp.Name, compute.INSTANCE_SNAPSHOT_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotCreateTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, data jsonutils.JSONObject) {

	self.finalReleasePendingUsage(ctx)
	if guest == nil {
		guest = models.GuestManager.FetchGuestById(isp.GuestId)
	}
	isp.SetStatus(ctx, self.UserCred, compute.INSTANCE_SNAPSHOT_READY, "")
	guest.StartSyncstatus(ctx, self.UserCred, "")

	db.OpsLog.LogEvent(isp, db.ACT_ALLOCATE, "instance snapshot create success", self.UserCred)
	logclient.AddActionLogWithStartable(self, isp, logclient.ACT_CREATE, "", self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    isp,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotCreateTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	self.SetStage("OnInstanceSnapshot", nil)
	params := jsonutils.NewDict()
	params.Set("disk_index", jsonutils.NewInt(0))
	if err := isp.GetRegionDriver().RequestCreateInstanceSnapshot(ctx, guest, isp, self, params); err != nil {
		self.taskFail(ctx, isp, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceSnapshotCreateTask) OnKvmDiskSnapshot(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	diskIndex, err := self.Params.Int("disk_index")
	if err != nil {
		self.taskFail(ctx, isp, guest, jsonutils.NewString(err.Error()))
		return
	}

	params := jsonutils.NewDict()
	params.Set("disk_index", jsonutils.NewInt(diskIndex+1))
	if err := isp.GetRegionDriver().RequestCreateInstanceSnapshot(ctx, guest, isp, self, params); err != nil {
		self.taskFail(ctx, isp, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceSnapshotCreateTask) OnKvmDiskSnapshotFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFail(ctx, isp, nil, data)
}

func (self *InstanceSnapshotCreateTask) OnInstanceSnapshot(ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	guest, _ := isp.GetGuest()
	if isp.WithMemory {
		resp := new(hostapi.GuestMemorySnapshotResponse)
		if err := data.Unmarshal(resp); err != nil {
			self.taskFail(ctx, isp, guest, jsonutils.NewString(err.Error()))
			return
		}
		if _, err := db.Update(isp, func() error {
			isp.MemorySizeKB = int(resp.SizeKB)
			isp.MemoryFilePath = resp.MemorySnapshotPath
			isp.MemoryFileChecksum = resp.Checksum
			return nil
		}); err != nil {
			self.taskFail(ctx, isp, guest, jsonutils.NewString(err.Error()))
			return
		}
	}
	self.taskComplete(ctx, isp, guest, data)
}

func (self *InstanceSnapshotCreateTask) OnInstanceSnapshotFailed(ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFail(ctx, isp, nil, data)
}
