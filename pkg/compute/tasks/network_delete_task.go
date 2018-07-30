package tasks

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type NetworkDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkDeleteTask{})
}

func (self *NetworkDeleteTask) taskFailed(ctx context.Context, network *models.SNetwork, err error) {
	log.Errorf("network create task fail: %s", err)
	network.SetStatus(self.UserCred, models.NETWORK_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *NetworkDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	network := obj.(*models.SNetwork)

	network.SetStatus(self.UserCred, models.NETWORK_STATUS_DELETING, "")
	db.OpsLog.LogEvent(network, db.ACT_DELOCATING, network.GetShortDesc(), self.UserCred)

	inet, err := network.GetINetwork()
	if inet != nil {
		err = inet.Delete()
		if err != nil {
			self.taskFailed(ctx, network, err)
			return
		}
	} else if err == cloudprovider.ErrNotFound {
		// already deleted, do nothing
	} else {
		self.taskFailed(ctx, network, err)
		return
	}

	network.RealDelete(ctx, self.UserCred)

	self.SetStageComplete(ctx, nil)
}
