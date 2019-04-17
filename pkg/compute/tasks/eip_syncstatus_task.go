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

type EipSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipSyncstatusTask{})
}

func (self *EipSyncstatusTask) taskFail(ctx context.Context, eip *models.SElasticip, msg string) {
	eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
	db.OpsLog.LogEvent(eip, db.ACT_SYNC_STATUS, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_SYNC_STATUS, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
}

func (self *EipSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find ieip for eip %s", err)
		self.taskFail(ctx, eip, msg)
		return
	}

	err = extEip.Refresh()
	if err != nil {
		msg := fmt.Sprintf("fail to refresh eip status %s", err)
		self.taskFail(ctx, eip, msg)
		return
	}

	err = eip.SyncWithCloudEip(ctx, self.UserCred, eip.GetCloudprovider(), extEip, "")
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		self.taskFail(ctx, eip, msg)
		return
	}

	err = eip.SyncInstanceWithCloudEip(ctx, self.UserCred, extEip)
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		self.taskFail(ctx, eip, msg)
		return
	}

	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
