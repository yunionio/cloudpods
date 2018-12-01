package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalServerResetTask struct {
	SGuestBaseTask
}

func (self *BaremetalServerResetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	baremetal := guest.GetHost()
	if baremetal == nil {
		self.SetStageFailed(ctx, "Baremetal is None")
		return
	}
	url := fmt.Sprintf("/baremetals/%s/servers/%s/reset", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		log.Errorf(err.Error())
		self.SetStageFailed(ctx, err.Error())
	} else {
		self.SetStageComplete(ctx, nil)
	}
}
