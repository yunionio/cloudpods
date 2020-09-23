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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalSyncStatusTask struct {
	SBaremetalBaseTask
}

func (self *BaremetalSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	if baremetal.IsBaremetal {
		self.DoSyncStatus(ctx, baremetal)
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *BaremetalSyncStatusTask) DoSyncStatus(ctx context.Context, baremetal *models.SHost) {
	self.SetStage("OnSyncstatusComplete", nil)
	url := fmt.Sprintf("/baremetals/%s/syncstatus", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (self *BaremetalSyncStatusTask) OnSyncstatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

type BaremetalSyncAllGuestsStatusTask struct {
	SBaremetalBaseTask
}

func (self *BaremetalSyncAllGuestsStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	guest := baremetal.GetBaremetalServer()
	if guest != nil {
		var first bool
		if !guest.IsSystem {
			first = true
		}
		db.Update(guest, func() error {
			guest.IsSystem = true
			guest.VmemSize = 0
			guest.VcpuCount = 0
			return nil
		})
		bs := baremetal.GetBaremetalstorage().GetStorage()
		bs.SetStatus(self.UserCred, api.STORAGE_OFFLINE, "")
		if first && baremetal.Name != guest.Name {

			func() {
				lockman.LockRawObject(ctx, models.HostManager.Keyword(), "name")
				defer lockman.ReleaseRawObject(ctx, models.HostManager.Keyword(), "name")

				db.Update(baremetal, func() error {
					newName, err := db.GenerateName(ctx, baremetal.GetModelManager(), nil, guest.Name)
					if err != nil {
						return err
					}
					baremetal.Name = newName
					return nil
				})
			}()
		}
		if first {
			db.OpsLog.LogEvent(guest, db.ACT_CONVERT_COMPLETE, "", self.UserCred)
			logclient.AddActionLogWithStartable(self, guest, logclient.ACT_BM_CONVERT_HYPER, "", self.UserCred, true)
		}
	}
	self.SetStage("OnGuestSyncStatusComplete", nil)
	self.OnGuestSyncStatusComplete(ctx, baremetal, nil)
}

func (self *BaremetalSyncAllGuestsStatusTask) OnGuestSyncStatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	guests, _ := baremetal.GetGuests()
	for _, guest := range guests {
		if guest.Status == api.VM_UNKNOWN && guest.Hypervisor != api.HYPERVISOR_BAREMETAL {
			guest.StartSyncstatus(ctx, self.GetUserCred(), "")
		}
	}
	log.Infof("All unknown guests syncstatus complete")
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(BaremetalSyncStatusTask{})
	taskman.RegisterTask(BaremetalSyncAllGuestsStatusTask{})
}
