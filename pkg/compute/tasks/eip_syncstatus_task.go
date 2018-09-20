package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type EipSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipSyncstatusTask{})
}

func (self *EipSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find ieip for eip %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = extEip.Refresh()
	if err != nil {
		msg := fmt.Sprintf("fail to refresh eip status %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.SyncWithCloudEip(self.UserCred, extEip)
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.SyncInstanceWithCloudEip(ctx, self.UserCred, extEip)
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
