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

func (self *HostGuestsMigrateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, data jsonutils.JSONObject) {
	guests := make([]*api.GuestBatchMigrateParams, 0)
	err := self.Params.Unmarshal(&guests, "guests")
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	preferHostId, _ := self.Params.GetString("prefer_host_id")

	self.SetStage("OnMigrateComplete", nil)

	for i := range objs {
		guest := objs[i].(*models.SGuest)
		if guests[i].LiveMigrate {
			err := guest.StartGuestLiveMigrateTask(
				ctx, self.UserCred, guests[i].OldStatus, preferHostId, &guests[i].SkipCpuCheck, &guests[i].SkipKernelCheck, guests[i].EnableTLS, self.Id)
			if err != nil {
				log.Errorln(err)
			}
		} else {
			err := guest.StartMigrateTask(ctx, self.UserCred, guests[i].RescueMode,
				false, guests[i].OldStatus, preferHostId, self.Id)
			if err != nil {
				log.Errorln(err)
			}
		}
	}
}

func (self *HostGuestsMigrateTask) OnMigrateComplete(ctx context.Context, objs []db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *HostGuestsMigrateTask) OnMigrateCompleteFailed(ctx context.Context, objs []db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
