package tasks

import (
	"fmt"
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type EipAssociateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipAssociateTask{})
}

func (self *EipAssociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ASSOCIATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	instanceId, _ := self.Params.GetString("instance_id")
	server := models.GuestManager.FetchGuestById(instanceId)
	if server == nil {
		msg := fmt.Sprintf("fail to find server for instanceId %s", instanceId)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ASSOCIATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = extEip.Associate(server.ExternalId)
	if err != nil {
		msg := fmt.Sprintf("fail to remote associate EIP %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ASSOCIATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.AssociateVM(self.UserCred, server)
	if err != nil {
		msg := fmt.Sprintf("fail to local associate EIP %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ASSOCIATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	eip.SetStatus(self.UserCred, models.EIP_STATUS_READY, "associate")

	self.SetStageComplete(ctx, nil)
}