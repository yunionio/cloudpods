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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestIsolatedDeviceSyncTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestIsolatedDeviceSyncTask{})
}

func (self *GuestIsolatedDeviceSyncTask) needStart() bool {
	return jsonutils.QueryBoolean(self.Params, "auto_start", false)
}

func (self *GuestIsolatedDeviceSyncTask) onTaskFail(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	self.SetStageFailed(ctx, err)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_SYNC_ISOLATED_DEVICE_FAILED, err.String())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_SYNC_ISOLATED_DEVICE, err, self.GetUserCred(), false)
}

func (self *GuestIsolatedDeviceSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnSyncConfigComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.onTaskFail(ctx, guest, jsonErrorObj(err))
		return
	}
	err = drv.RequestSyncIsolatedDevice(ctx, guest, self)
	if err != nil {
		self.onTaskFail(ctx, guest, jsonErrorObj(err))
		return
	}
}

func (self *GuestIsolatedDeviceSyncTask) OnSyncConfigComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if self.needStart() {
		self.SetStage("OnStartComplete", nil)
		guest.StartGueststartTask(ctx, self.GetUserCred(), nil, self.GetId())
	} else {
		self.OnStartComplete(ctx, guest, data)
	}
}

func (self *GuestIsolatedDeviceSyncTask) OnSyncConfigCompleteFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	self.onTaskFail(ctx, guest, reason)
}

func (self *GuestIsolatedDeviceSyncTask) OnStartComplete(ctx context.Context, obj *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_VM_SYNC_ISOLATED_DEVICE, nil, self.GetUserCred(), true)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestIsolatedDeviceSyncTask) OnStartCompleteFailed(ctx context.Context, obj *models.SGuest, data jsonutils.JSONObject) {
	self.onTaskFail(ctx, obj, data)
}
