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

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceSnapshotAndCloneTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotAndCloneTask{})
}

func (self *InstanceSnapshotAndCloneTask) taskFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, reason jsonutils.JSONObject) {
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	guest.SetStatus(ctx, self.UserCred, compute.VM_SNAPSHOT_AND_CLONE_FAILED, reason.String())
	logclient.AddActionLogWithContext(
		ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, reason, self.UserCred, false,
	)
	db.OpsLog.LogEvent(guest, db.ACT_VM_SNAPSHOT_AND_CLONE_FAILED, reason, self.UserCred)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotAndCloneTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.finalReleasePendingUsage(ctx, true)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	guest.StartSyncstatus(ctx, self.UserCred, "")
	db.OpsLog.LogEvent(guest, db.ACT_VM_SNAPSHOT_AND_CLONE, "", self.UserCred)
	logclient.AddActionLogWithContext(
		ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, self.Params, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotAndCloneTask) SetStageFailed(ctx context.Context, reason jsonutils.JSONObject) {
	self.finalReleasePendingUsage(ctx, false)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotAndCloneTask) finalReleasePendingUsage(ctx context.Context, success bool) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage, 0)
	if err == nil && !pendingUsage.IsEmpty() {
		quotas.CancelPendingUsage(ctx, self.UserCred, &pendingUsage, &pendingUsage, success)
	}

	pendingRegionUsage := models.SRegionQuota{}
	err = self.GetPendingUsage(&pendingRegionUsage, 1)
	if err == nil && !pendingRegionUsage.IsEmpty() {
		quotas.CancelPendingUsage(ctx, self.UserCred, &pendingRegionUsage, &pendingRegionUsage, success)
	}
}

func (self *InstanceSnapshotAndCloneTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	self.SetStage("OnCreateInstanceSnapshot", nil)
	err := isp.StartCreateInstanceSnapshotTask(ctx, self.UserCred, nil, self.Id)
	if err != nil {
		self.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceSnapshotAndCloneTask) OnCreateInstanceSnapshot(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	// start create server
	var input compute.ServerSnapshotAndCloneInput
	err := self.Params.Unmarshal(&input, "guest_params")
	if err != nil {
		self.taskFailed(ctx, isp, jsonutils.NewString("Failed get new guest params"))
		return
	}
	count := 0
	if input.Count != nil {
		count = *input.Count
	}
	if count == 0 {
		count = 1
	}
	err = self.doGuestCreate(ctx, isp, input, count)
	if err != nil {
		self.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	self.taskComplete(ctx, isp, nil)
}

func (self *InstanceSnapshotAndCloneTask) doGuestCreate(
	ctx context.Context, isp *models.SInstanceSnapshot, cloneInput compute.ServerSnapshotAndCloneInput, count int) error {

	var (
		errStr string
	)
	for i := 0; i < count; i++ {
		newGuest, params, err := models.GuestManager.CreateGuestFromInstanceSnapshot(
			ctx, self.UserCred, cloneInput, isp)
		if err != nil {
			log.Errorln(err)
			errStr += err.Error() + "\n"
			continue
		}
		isp.AddRefCount(ctx)
		models.GuestManager.OnCreateComplete(ctx, []db.IModel{newGuest}, self.UserCred, self.UserCred, nil, []jsonutils.JSONObject{params})
	}
	if len(errStr) > 0 {
		return fmt.Errorf(errStr)
	}
	return nil
}

func (self *InstanceSnapshotAndCloneTask) OnCreateInstanceSnapshotFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFailed(ctx, isp, data)
}
