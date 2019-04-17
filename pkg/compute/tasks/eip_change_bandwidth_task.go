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

type EipChangeBandwidthTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipChangeBandwidthTask{})
}

func (self *EipChangeBandwidthTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg string) {
	eip.SetStatus(self.UserCred, models.EIP_STATUS_READY, msg)
	db.OpsLog.LogEvent(eip, db.ACT_CHANGE_BANDWIDTH, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_CHANGE_BANDWIDTH, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
}

func (self *EipChangeBandwidthTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find iEip %s", err)
		self.TaskFail(ctx, eip, msg)
		return
	}

	bandwidth, _ := self.Params.Int("bandwidth")
	if bandwidth <= 0 {
		msg := fmt.Sprintf("invalid bandwidth %d", bandwidth)
		self.TaskFail(ctx, eip, msg)
		return
	}

	err = extEip.ChangeBandwidth(int(bandwidth))

	if err != nil {
		msg := fmt.Sprintf("fail to find iEip %s", err)
		self.TaskFail(ctx, eip, msg)
		return
	}

	err = eip.DoChangeBandwidth(self.UserCred, int(bandwidth))

	if err != nil {
		msg := fmt.Sprintf("fail to synchronize iEip bandwidth %s", err)
		self.TaskFail(ctx, eip, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
