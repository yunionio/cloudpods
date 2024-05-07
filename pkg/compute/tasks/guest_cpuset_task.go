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

type GuestCPUSetTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestCPUSetTask{})
}

func (self *GuestCPUSetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnSyncComplete", nil)
	db.OpsLog.LogEvent(guest, db.ACT_GUEST_CPUSET, self.GetParams(), self.UserCred)
	if err := guest.StartSyncTask(ctx, self.GetUserCred(), true, self.GetTaskId()); err != nil {
		self.setStageFailed(ctx, guest, data)
	}
}

func (self *GuestCPUSetTask) setStageFailed(ctx context.Context, obj *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_VM_CPUSET, data, self.UserCred, false)
	db.OpsLog.LogEvent(obj, db.ACT_GUEST_CPUSET_FAIL, data, self.UserCred)
	self.SetStageFailed(ctx, data)
}

func (self *GuestCPUSetTask) OnSyncComplete(ctx context.Context, obj *models.SGuest, data jsonutils.JSONObject) {
	host, _ := obj.GetHost()
	input := new(api.ServerCPUSetInput)
	self.GetParams().Unmarshal(input)
	drv, err := obj.GetDriver()
	if err != nil {
		self.setStageFailed(ctx, obj, jsonutils.NewString(err.Error()))
		return
	}
	_, err = drv.RequestCPUSet(ctx, self.GetUserCred(), host, obj, input)
	if err != nil {
		self.setStageFailed(ctx, obj, jsonutils.NewString(err.Error()))
		return
	}
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_VM_CPUSET, input, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestCPUSetTask) OnSyncCompleteFailed(ctx context.Context, obj *models.SGuest, data jsonutils.JSONObject) {
	self.setStageFailed(ctx, obj, data)
}
