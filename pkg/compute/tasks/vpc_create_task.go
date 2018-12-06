package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type VpcCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcCreateTask{})
}

func (self *VpcCreateTask) TaskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	log.Errorf("vpc create task fail: %s", err)
	vpc.SetStatus(self.UserCred, models.VPC_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *VpcCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)
	vpc.SetStatus(self.UserCred, models.VPC_STATUS_PENDING, "")

	iregion, err := vpc.GetIRegion()
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}
	ivpc, err := iregion.CreateIVpc(vpc.Name, vpc.Description, vpc.CidrBlock)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}
	vpc.SetExternalId(ivpc.GetGlobalId())

	err = cloudprovider.WaitStatus(ivpc, models.VPC_STATUS_AVAILABLE, 10*time.Second, 300*time.Second)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}

	err = vpc.SyncWithCloudVpc(ivpc)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}

	provider := models.CloudproviderManager.FetchCloudproviderById(vpc.ManagerId)
	syncVpcWires(ctx, provider, self, vpc, ivpc, &models.SSyncRange{})

	hosts := models.HostManager.GetHostsByManagerAndRegion(provider.Id, vpc.CloudregionId)
	for i := 0; i < len(hosts); i += 1 {
		ihost, err := hosts[i].GetIHost()
		if err != nil {
			self.TaskFailed(ctx, vpc, err)
			return
		}
		syncHostWires(ctx, provider, self, &hosts[i], ihost)
	}

	self.SetStageComplete(ctx, nil)
}
