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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(BaremetalUploadAllGuestsStatusTask{})
}

type BaremetalUploadAllGuestsStatusTask struct {
	SBaremetalBaseTask
}

func (t *BaremetalUploadAllGuestsStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
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
		bs.SetStatus(ctx, t.UserCred, api.STORAGE_OFFLINE, "")
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
			db.OpsLog.LogEvent(guest, db.ACT_CONVERT_COMPLETE, "", t.UserCred)
			logclient.AddActionLogWithStartable(t, guest, logclient.ACT_BM_CONVERT_HYPER, "", t.UserCred, true)
		}
	}
	t.SetStage("OnGuestProbeStatusComplete", nil)
	if err := t.requestUploadGuestsStatus(ctx, baremetal); err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (t *BaremetalUploadAllGuestsStatusTask) requestUploadGuestsStatus(ctx context.Context, baremetal *models.SHost) error {
	gsts, _ := baremetal.GetGuests()
	needSyncGsts := make([]models.SGuest, 0)
	for _, gst := range gsts {
		needSyncGsts = append(needSyncGsts, gst)
	}
	drv, err := baremetal.GetHostDriver()
	if err != nil {
		return errors.Wrap(err, "GetHostDriver")
	}
	if len(needSyncGsts) > 0 {
		return drv.RequestUploadGuestsStatus(ctx, t.GetUserCred(), baremetal, needSyncGsts)
	}
	return t.ScheduleRun(nil)
}

/*func (self *BaremetalUploadAllGuestsStatusTask) OnGuestGetStatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	guests, _ := baremetal.GetGuests()
	for _, guest := range guests {
		if guest.Status == api.VM_UNKNOWN && guest.Hypervisor != api.HYPERVISOR_BAREMETAL {
			guest.StartSyncstatus(ctx, self.GetUserCred(), "")
		}
	}
	log.Infof("All unknown guests syncstatus complete")
	self.SetStageComplete(ctx, nil)
}*/

func (t *BaremetalUploadAllGuestsStatusTask) OnGuestProbeStatusComplete(ctx context.Context, host *models.SHost, body jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *BaremetalUploadAllGuestsStatusTask) OnGuestProbeStatusCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	t.SetStageFailed(ctx, body)
}
