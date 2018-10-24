package tasks

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type BaremetalDeleteTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalDeleteTask{})
}

func (self *BaremetalDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_DELETE, "")
	if !baremetal.IsBaremetalAgentReady() {
		self.OnDeleteBaremetalComplete(ctx, baremetal, nil)
		return
	}
	url := fmt.Sprintf("/baremetals/%s/delete", baremetal.Id)
	headers := http.Header{}
	headers.Set("X-Auth-Token", self.UserCred.GetTokenString())
	headers.Set("X-Task-Id", self.GetTaskId())
	self.SetStage("OnDeleteBaremetalComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		if errVal, ok := err.(*httputils.JSONClientError); ok && errVal.Code == 404 {
			self.OnDeleteBaremetalComplete(ctx, baremetal, nil)
			return
		}
		log.Errorln(err.Error())
		self.OnFailure(ctx, baremetal, nil)
	}
}

func (self *BaremetalDeleteTask) OnDeleteBaremetalComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	baremetal.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalDeleteTask) OnDeleteBaremetalCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.OnFailure(ctx, baremetal, body)
}

func (self *BaremetalDeleteTask) OnFailure(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_DELETE_FAIL, "")
	self.SetStageFailed(ctx, "")
}
