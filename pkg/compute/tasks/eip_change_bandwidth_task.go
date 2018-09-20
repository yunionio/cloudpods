package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type EipChangeBandwidthTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipChangeBandwidthTask{})
}

func (self *EipChangeBandwidthTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		eip.SetStatus(self.UserCred, models.EIP_STATUS_READY, "fail to change bandwidth")
		msg := fmt.Sprintf("fail to find iEip %s", err)
		self.SetStageFailed(ctx, msg)
		return
	}

	bandwidth, _ := self.Params.Int("bandwidth")
	if bandwidth <= 0 {
		eip.SetStatus(self.UserCred, models.EIP_STATUS_READY, "fail to change bandwidth")
		msg := fmt.Sprintf("invalid bandwidth %d", bandwidth)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = extEip.ChangeBandwidth(int(bandwidth))

	if err != nil {
		eip.SetStatus(self.UserCred, models.EIP_STATUS_READY, "fail to change bandwidth")
		msg := fmt.Sprintf("fail to find iEip %s", err)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.DoChangeBandwidth(self.UserCred, int(bandwidth))

	if err != nil {
		msg := fmt.Sprintf("fail to synchronize iEip bandwidth %s", err)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
