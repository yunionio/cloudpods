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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type InstanceSnapshotAndCloneTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotAndCloneTask{})
}

func (self *InstanceSnapshotAndCloneTask) taskFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, reason string) {
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	guest.SetStatus(self.UserCred, compute.VM_SNAPSHOT_AND_CLONE_FAILED, reason)
	logclient.AddActionLogWithContext(
		ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, reason, self.UserCred, false,
	)
	db.OpsLog.LogEvent(guest, db.ACT_VM_SNAPSHOT_AND_CLONE_FAILED, reason, self.UserCred)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotAndCloneTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.finalReleasePendingUsage(ctx)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)
	guest.StartSyncstatus(ctx, self.UserCred, "")
	db.OpsLog.LogEvent(guest, db.ACT_VM_SNAPSHOT_AND_CLONE, "", self.UserCred)
	logclient.AddActionLogWithContext(
		ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotAndCloneTask) SetStageFailed(ctx context.Context, reason string) {
	self.finalReleasePendingUsage(ctx)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotAndCloneTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err == nil && !pendingUsage.IsEmpty() {
		isp := self.GetObject().(*models.SInstanceSnapshot)
		guest := models.GuestManager.FetchGuestById(isp.GuestId)
		quotaPlatform := guest.GetQuotaPlatformID()
		models.QuotaManager.CancelPendingUsage(
			ctx, self.UserCred, rbacutils.ScopeProject,
			guest.GetOwnerId(), quotaPlatform, &pendingUsage, &pendingUsage,
		)
	}
}

func (self *InstanceSnapshotAndCloneTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	self.SetStage("OnCreateInstanceSnapshot", nil)
	err := isp.StartCreateInstanceSnapshotTask(ctx, self.UserCred, guest.GetOwnerId(), nil, self.Id)
	if err != nil {
		self.taskFailed(ctx, isp, err.Error())
		return
	}
}

func (self *InstanceSnapshotAndCloneTask) OnCreateInstanceSnapshot(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	// start create server
	params, err := self.Params.Get("guest_params")
	if err != nil {
		self.taskFailed(ctx, isp, "Failed get new guest params")
		return
	}
	count, _ := params.Int("count")
	if count == 0 {
		count = 1
	}
	err = self.doGuestCreate(ctx, isp, params, int(count))
	if err != nil {
		self.taskFailed(ctx, isp, err.Error())
		return
	}
	self.taskComplete(ctx, isp, nil)
}

func (self *InstanceSnapshotAndCloneTask) doGuestCreate(
	ctx context.Context, isp *models.SInstanceSnapshot, params jsonutils.JSONObject, count int) error {

	var (
		dictParmas = params.(*jsonutils.JSONDict)
		errStr     string
	)
	for i := 0; i < count; i++ {
		newGuest, input, err := models.GuestManager.CreateGuestFromInstanceSnapshot(
			ctx, self.UserCred, dictParmas.DeepCopy().(*jsonutils.JSONDict), isp, i+1)
		if err != nil {
			log.Errorln(err)
			errStr += err.Error() + "\n"
			continue
		}
		isp.AddRefCount(ctx)
		models.GuestManager.OnCreateComplete(ctx, []db.IModel{newGuest}, self.UserCred, nil, input)
	}
	if len(errStr) > 0 {
		return fmt.Errorf(errStr)
	}
	return nil
}

func (self *InstanceSnapshotAndCloneTask) OnCreateInstanceSnapshotFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFailed(ctx, isp, data.String())
}
