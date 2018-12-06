package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
	url := fmt.Sprintf("/baremetals/%s/syncstatus", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err == nil {
		self.SetStageComplete(ctx, nil)
	} else {
		self.SetStageFailed(ctx, err.Error())
	}
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
		guest.GetModelManager().TableSpec().Update(guest, func() error {
			guest.IsSystem = true
			guest.VmemSize = 0
			guest.VcpuCount = 0
			return nil
		})
		bs := baremetal.GetBaremetalstorage().GetStorage()
		bs.SetStatus(self.UserCred, models.STORAGE_OFFLINE, "")
		if first && baremetal.Name != guest.Name {
			baremetal.GetModelManager().TableSpec().Update(baremetal, func() error {
				if models.HostManager.IsNewNameUnique(guest.Name, self.UserCred, nil) {
					baremetal.Name = guest.Name
				} else {
					baremetal.Name = db.GenerateName(baremetal.GetModelManager(),
						self.UserCred.GetTokenString(), guest.Name)
				}
				return nil
			})
		}
		if first {
			db.OpsLog.LogEvent(guest, db.ACT_CONVERT_COMPLETE, "", self.UserCred)
			logclient.AddActionLog(guest, logclient.ACT_BM_CONVERT_HYPER, "", self.UserCred, true)
		}
	}
	self.SetStage("OnGuestSyncStatusComplete", nil)
	self.OnGuestSyncStatusComplete(ctx, baremetal, nil)
}

func (self *BaremetalSyncAllGuestsStatusTask) OnGuestSyncStatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	var guests = make([]models.SGuest, 0)
	for _, guest := range baremetal.GetGuests() {
		if guest.Status == models.VM_UNKNOWN && guest.Hypervisor != models.HYPERVISOR_BAREMETAL {
			guest.SetStatus(self.UserCred, models.VM_SYNCING_STATUS, "")
			guests = append(guests, guest)
		}
	}
	for _, guest := range guests {
		guest.StartSyncstatus(ctx, self.UserCred, "")
	}
	log.Infof("All unknown guests syncstatus complete")
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(BaremetalSyncStatusTask{})
	taskman.RegisterTask(BaremetalSyncAllGuestsStatusTask{})
}
