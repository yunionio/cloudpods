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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalServerRebuildRootTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerRebuildRootTask{})
}

func (self *BaremetalServerRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if jsonutils.QueryBoolean(self.Params, "need_stop", false) {
		self.SetStage("OnStopServerComplete", nil)
		guest.StartGuestStopTask(ctx, self.UserCred, false, self.GetTaskId())
		return
	}
	self.StartRebuildRootDisk(ctx, guest)
}

func (self *BaremetalServerRebuildRootTask) OnStopServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.StartRebuildRootDisk(ctx, guest)
}

func (self *BaremetalServerRebuildRootTask) StartRebuildRootDisk(ctx context.Context, guest *models.SGuest) {
	if guest.Status != api.VM_ADMIN {
		guest.SetStatus(self.UserCred, api.VM_REBUILD_ROOT, "")
	}
	imageId, _ := self.Params.GetString("image_id")
	db.OpsLog.LogEvent(guest, db.ACT_REBUILDING_ROOT, imageId, self.UserCred)
	gds := guest.CategorizeDisks()
	oldStatus := gds.Root.Status
	_, err := db.Update(gds.Root, func() error {
		gds.Root.TemplateId = imageId
		gds.Root.Status = api.DISK_REBUILD
		return nil
	})
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
		return
	} else {
		db.OpsLog.LogEvent(gds.Root, db.ACT_UPDATE_STATUS,
			fmt.Sprintf("%s=>%s", oldStatus, api.DISK_REBUILD), self.UserCred)
	}
	self.SetStage("OnRebuildRootDiskComplete", nil)

	// clear logininfo
	loginParams := make(map[string]interface{})
	loginParams["login_account"] = "none"
	loginParams["login_key"] = "none"
	loginParams["login_key_timestamp"] = "none"
	guest.SetAllMetadata(ctx, loginParams, self.UserCred)
	guest.StartGuestDeployTask(ctx, self.UserCred, self.Params, "rebuild", self.GetTaskId())
}

func (self *BaremetalServerRebuildRootTask) OnRebuildRootDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT, "", self.UserCred)
	self.SetStage("OnSyncStatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *BaremetalServerRebuildRootTask) OnRebuildRootDiskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT_FAIL, data, self.UserCred)
	if guest.Status != api.VM_ADMIN {
		guest.SetStatus(self.UserCred, api.VM_REBUILD_ROOT_FAIL, "")
	}
}

func (self *BaremetalServerRebuildRootTask) OnSyncStatusComplete(ctx context.Context, _ *models.SGuest, _ jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
