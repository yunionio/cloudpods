package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalUnmaintenanceTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalUnmaintenanceTask{})
}

func (self *BaremetalUnmaintenanceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	url := fmt.Sprintf("/baremetals/%s/unmaintenance", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnUnmaintenantComplete", nil)
	action := self.Action()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, self.Params)
	if err != nil {
		if len(action) > 0 {
			msg := fmt.Sprintf("unmaintenance error %s", err.Error())
			logclient.AddActionLogWithStartable(self, baremetal, action, msg, self.UserCred, false)
		}
		self.SetStageFailed(ctx, err.Error())
	} else {
		if len(action) > 0 {
			logclient.AddActionLogWithStartable(self, baremetal, action, "", self.UserCred, true)
		}
	}
}

func (self *BaremetalUnmaintenanceTask) OnUnmaintenantComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	baremetal.GetModelManager().TableSpec().Update(baremetal, func() error {
		baremetal.IsMaintenance = false
		return nil
	})
	metadatas := map[string]interface{}{
		"__maint_username": "None",
		"__maint_password": "None",
		"__maint_ip":       "None",
	}
	baremetal.SetAllMetadata(ctx, metadatas, self.UserCred)
	self.SetStageComplete(ctx, nil)
	guest := baremetal.GetBaremetalServer()
	if guest != nil {
		guest.StartSyncstatus(ctx, self.UserCred, "")
	}
}
