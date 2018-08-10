package tasks

import (
	"context"
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type VpcDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcDeleteTask{})
}

func (self *VpcDeleteTask) taskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	log.Errorf("vpc delete task fail: %s", err)
	vpc.SetStatus(self.UserCred, models.VPC_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *VpcDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)

	vpc.SetStatus(self.UserCred, models.VPC_STATUS_DELETING, "")
	db.OpsLog.LogEvent(vpc, db.ACT_DELOCATING, vpc.GetShortDesc(), self.UserCred)

	region, err := vpc.GetIRegion()
	if err != nil {
		self.taskFailed(ctx, vpc, err)
		return
	}
	ivpc, err := region.GetIVpcById(vpc.GetExternalId())
	if ivpc != nil {
		err = ivpc.Delete()
		if err != nil {
			self.taskFailed(ctx, vpc, err)
			return
		}
		err = cloudprovider.WaitDeleted(ivpc, 10*time.Second, 300*time.Second)
		if err != nil {
			self.taskFailed(ctx, vpc, err)
			return
		}
	} else if err == cloudprovider.ErrNotFound {
		// already deleted, do nothing
	} else {
		self.taskFailed(ctx, vpc, err)
		return
	}

	wires := vpc.GetWires()
	if wires != nil {
		for i := 0; i < len(wires); i += 1 {
			hws, _ := wires[i].GetHostwires()
			for j := 0; hws != nil && j < len(hws); j += 1 {
				hws[j].Detach(ctx, self.UserCred)
			}
			wires[i].Delete(ctx, self.UserCred)
		}
	}

	vpc.RealDelete(ctx, self.UserCred)

	self.SetStageComplete(ctx, nil)
}
