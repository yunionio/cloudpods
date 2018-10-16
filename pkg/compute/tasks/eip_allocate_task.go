package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type EipAllocateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipAllocateTask{})
}

func (self *EipAllocateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	iregion, err := eip.GetIRegion()
	if err != nil {
		msg := fmt.Sprintf("fail to find iregion for eip %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ALLOCATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	extEip, err := iregion.CreateEIP(eip.Name, eip.Bandwidth, eip.ChargeType)
	if err != nil {
		msg := fmt.Sprintf("create eip fail %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ALLOCATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.SyncWithCloudEip(self.UserCred, extEip, "", false)

	if err != nil {
		msg := fmt.Sprintf("sync eip fail %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_ALLOCATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	eipPendingUsage := &models.SQuota{Eip: 1}
	err = models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, self.UserCred.GetProjectId(), eipPendingUsage, eipPendingUsage)

	if err != nil {
		log.Errorf("CancelPendingUsage fail %s", err)
	}

	if self.Params != nil && self.Params.Contains("instance_id") {
		self.SetStage("on_eip_associate_complete", nil)
		err = eip.StartEipAssociateTask(ctx, self.UserCred, self.Params)
		if err != nil {
			msg := fmt.Sprintf("start associate task fail %s", err)
			self.SetStageFailed(ctx, msg)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *EipAllocateTask) OnEipAssociateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
