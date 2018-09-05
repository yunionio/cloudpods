package tasks

import (
	"fmt"
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"

)

type EipDeallocateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipDeallocateTask{})
}

func (self *EipDeallocateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	if len(eip.ExternalId) > 0 {
		expEip, err := eip.GetIEip()
		if err != nil {
			msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
			eip.SetStatus(self.UserCred, models.EIP_STATUS_DEALLOCATE_FAIL, msg)
			self.SetStageFailed(ctx, msg)
			return
		}

		err = expEip.Delete()
		if err != nil {
			msg := fmt.Sprintf("fail to delete iEIP %s", err)
			eip.SetStatus(self.UserCred, models.EIP_STATUS_DEALLOCATE_FAIL, msg)
			self.SetStageFailed(ctx, msg)
			return
		}
	}

	err := eip.RealDelete(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to delete EIP %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_DEALLOCATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}