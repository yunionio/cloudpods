package tasks

import (
	"context"

	"time"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
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
		log.Errorf("SyncWithCloudVpc fail: %s", err)
		self.TaskFailed(ctx, vpc, err)
		return
	}

	provider := models.CloudproviderManager.FetchCloudproviderById(vpc.ManagerId)
	syncVpcWires(ctx, provider, self, vpc, ivpc)

	hosts := models.HostManager.GetHostsByManagerAndRegion(provider.Id, vpc.CloudregionId)
	if hosts != nil {
		for i := 0; i < len(hosts); i += 1 {
			ihost, err := hosts[i].GetIHost()
			if err != nil {
				log.Errorf("getiHost fail %s", err)
				self.TaskFailed(ctx, vpc, err)
				return
			}
			syncHostWires(ctx, provider, self, &hosts[i], ihost)
		}
	}

	self.SetStageComplete(ctx, nil)
}
