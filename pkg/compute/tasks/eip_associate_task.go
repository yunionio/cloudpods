package tasks

import (
	"context"
	"fmt"

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

func (self *EipAssociateTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg string, vm *models.SGuest) {
	eip.SetStatus(self.UserCred, models.EIP_STATUS_ASSOCIATE_FAIL, msg)
	self.SetStageFailed(ctx, msg)
	if vm != nil {
		vm.StartSyncstatus(ctx, self.UserCred, "")
	}
}

func (self *EipAssociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
		self.TaskFail(ctx, eip, msg, nil)
		return
	}

	instanceId, _ := self.Params.GetString("instance_id")
	server := models.GuestManager.FetchGuestById(instanceId)

	if server.Status != models.VM_ASSOCIATE_EIP {
		server.SetStatus(self.UserCred, models.VM_ASSOCIATE_EIP, "associate eip")
	}

	if server == nil {
		msg := fmt.Sprintf("fail to find server for instanceId %s", instanceId)
		self.TaskFail(ctx, eip, msg, nil)
		return
	}

	err = extEip.Associate(server.ExternalId)
	if err != nil {
		msg := fmt.Sprintf("fail to remote associate EIP %s", err)
		self.TaskFail(ctx, eip, msg, server)
		return
	}

	err = eip.AssociateVM(self.UserCred, server)
	if err != nil {
		msg := fmt.Sprintf("fail to local associate EIP %s", err)
		self.TaskFail(ctx, eip, msg, server)
		return
	}

	eip.SetStatus(self.UserCred, models.EIP_STATUS_READY, "associate")

	server.StartSyncstatus(ctx, self.UserCred, "")

	self.SetStageComplete(ctx, nil)
}
