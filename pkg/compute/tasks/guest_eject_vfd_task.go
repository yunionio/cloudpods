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
)

type GuestEjectVFDTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestEjectVFDTask{})
}

func (self *GuestEjectVFDTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.startEjectVfd(ctx, obj)
}

func (self *GuestEjectVFDTask) startEjectVfd(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	floppyOrdinal, _ := self.Params.Int("floppy_ordinal")
	if guest.EjectVfd(floppyOrdinal, self.UserCred) && guest.Status == api.VM_RUNNING {
		self.SetStage("OnConfigSyncComplete", nil)
		drv, err := guest.GetDriver()
		if err != nil {
			self.SetStageFailed(ctx, jsonutils.Marshal(map[string]string{"reason": err.Error()}))
			return
		}
		drv.RequestGuestHotRemoveVfd(ctx, guest, self)
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestEjectVFDTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
