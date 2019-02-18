package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalUnconvertHypervisorTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalUnconvertHypervisorTask{})
}

func (self *BaremetalUnconvertHypervisorTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_CONVERTING, "")
	guests := baremetal.GetGuests()
	if len(guests) > 1 {
		self.SetStageFailed(ctx, "Host guest conut > 1")
	}
	if len(guests) == 1 {
		guest := guests[0]
		self.SetStage("OnGuestDeleteComplete", nil)
		guest.StartDeleteGuestTask(ctx, self.UserCred, self.GetTaskId(), false, true)
	} else {
		self.OnGuestDeleteComplete(ctx, baremetal, nil)
	}
}

func (self *BaremetalUnconvertHypervisorTask) OnGuestDeleteComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	db.OpsLog.LogEvent(baremetal, db.ACT_UNCONVERT_COMPLETE, "", self.UserCred)
	driver := baremetal.GetDriverWithDefault()
	err := driver.FinishUnconvert(ctx, self.UserCred, baremetal)
	if err != nil {
		log.Errorf("Fail to exec finish_unconvert: %s", err.Error())
	}
	self.SetStage("OnPrepareComplete", nil)
	baremetal.StartPrepareTask(ctx, self.UserCred, "", self.GetTaskId())
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_UNCONVERT_HYPER, nil, self.UserCred, true)
}

func (self *BaremetalUnconvertHypervisorTask) OnGuestDeleteCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	db.OpsLog.LogEvent(baremetal, db.ACT_UNCONVERT_FAIL, body, self.UserCred)
	self.SetStage("OnFailSyncstatusComplete", nil)
	baremetal.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_UNCONVERT_HYPER, nil, self.UserCred, false)
}

func (self *BaremetalUnconvertHypervisorTask) OnPrepareComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
func (self *BaremetalUnconvertHypervisorTask) OnFailSyncstatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageFailed(ctx, "Delete server failed")
}
