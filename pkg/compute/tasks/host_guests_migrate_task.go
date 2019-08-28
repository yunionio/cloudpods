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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(HostGuestsMigrateTask{})
}

type HostGuestsMigrateTask struct {
	taskman.STask
}

func (self *HostGuestsMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guests := make([]*api.GuestBatchMigrateParams, 0)
	err := self.Params.Unmarshal(&guests, "guests")
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	preferHostId, _ := self.Params.GetString("prefer_host_id")

	var guestMigrating bool
	var migrateIndex int
	for i := 0; i < len(guests); i++ {
		guest := models.GuestManager.FetchGuestById(guests[i].Id)
		if guests[i].LiveMigrate {
			err := guest.StartGuestLiveMigrateTask(
				ctx, self.UserCred, guests[i].OldStatus, preferHostId, self.Id)
			if err != nil {
				log.Errorln(err)
				continue
			} else {
				guestMigrating = true
				migrateIndex = i
				break
			}
		} else {
			err := guest.StartMigrateTask(ctx, self.UserCred, guests[i].RescueMode,
				false, guests[i].OldStatus, preferHostId, self.Id)
			if err != nil {
				log.Errorln(err)
				continue
			} else {
				guestMigrating = true
				migrateIndex = i
				break
			}
		}
	}
	if !guestMigrating {
		if jsonutils.QueryBoolean(self.Params, "some_guest_migrate_failed", false) {
			self.SetStageFailed(ctx, "some guest migrate failed")
		} else {
			self.SetStageComplete(ctx, nil)
		}
	} else {
		guests := append(guests[:migrateIndex], guests[migrateIndex+1:]...)
		params := jsonutils.NewDict()
		params.Set("guests", jsonutils.Marshal(guests))
		self.SaveParams(params)
	}
}

func (self *HostGuestsMigrateTask) OnInitFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	kwargs := jsonutils.NewDict()
	kwargs.Set("some_guest_migrate_failed", jsonutils.JSONTrue)
	self.SaveParams(kwargs)
	self.OnInit(ctx, obj, data)
}
