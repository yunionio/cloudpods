package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipDeallocateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipDeallocateTask{})
}

func (self *EipDeallocateTask) taskFail(ctx context.Context, eip *models.SElasticip, msg string) {
	eip.SetStatus(self.UserCred, models.EIP_STATUS_DEALLOCATE_FAIL, msg)
	db.OpsLog.LogEvent(eip, db.ACT_DELOCATE, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_DELETE, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
	return
}

func (self *EipDeallocateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	if len(eip.ExternalId) > 0 {
		expEip, err := eip.GetIEip()
		if err != nil {
			if err != cloudprovider.ErrNotFound && err != cloudprovider.ErrInvalidProvider {
				msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
				self.taskFail(ctx, eip, msg)
				return
			}
		} else {
			err = expEip.Delete()
			if err != nil {
				msg := fmt.Sprintf("fail to delete iEIP %s", err)
				self.taskFail(ctx, eip, msg)
				return
			}
		}
	}

	err := eip.RealDelete(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to delete EIP %s", err)
		self.taskFail(ctx, eip, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
