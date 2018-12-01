package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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

	}
	self.SetStage("OnGuestSyncStatusComplete", nil)
	self.OnGuestSyncStatusComplete(ctx, baremetal, nil)
}

func (self *BaremetalSyncAllGuestsStatusTask) OnGuestSyncStatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	guests := baremetal.GetGuests()
	for _, guest := range guests {
		if guest.Status == models.VM_UNKNOWN && guest.Hypervisor == models.HYPERVISOR_BAREMETAL {
			guest.SetStatus(self.UserCred, models.VM_SYNCING_STATUS, "")
			// GuestBatchSyncStatusTask
			guest.StartSyncstatus(ctx, self.UserCred, "")
		}
	}
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(BaremetalSyncStatusTask{})
	taskman.RegisterTask(BaremetalSyncAllGuestsStatusTask{})
}
