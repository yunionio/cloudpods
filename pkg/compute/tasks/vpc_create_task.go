package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type VpcCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcCreateTask{})
}

func (self *VpcCreateTask) TaskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	log.Errorf("vpc create task fail: %s", err)
	vpc.SetStatus(self.UserCred, api.VPC_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_ALLOCATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *VpcCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)
	vpc.SetStatus(self.UserCred, api.VPC_STATUS_PENDING, "")

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
	vpc.SetExternalId(self.UserCred, ivpc.GetGlobalId())

	err = cloudprovider.WaitStatus(ivpc, api.VPC_STATUS_AVAILABLE, 10*time.Second, 300*time.Second)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}

	err = vpc.SyncWithCloudVpc(ctx, self.UserCred, ivpc)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}

	err = vpc.SyncRemoteWires(ctx, self.UserCred)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}

	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
