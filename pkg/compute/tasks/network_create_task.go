package tasks

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type NetworkCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkCreateTask{})
}

func (self *NetworkCreateTask) taskFailed(ctx context.Context, network *models.SNetwork, event string, err error) {
	log.Errorf("network create task fail on %s: %s", event, err)
	network.SetStatus(self.UserCred, models.NETWORK_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(network, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *NetworkCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	network := obj.(*models.SNetwork)

	network.SetStatus(self.UserCred, models.NETWORK_STATUS_PENDING, "")

	wire := network.GetWire()
	if wire == nil {
		self.taskFailed(ctx, network, "getwire", fmt.Errorf("no vpc"))
		return
	}

	iwire, err := wire.GetIWire()
	if err != nil {
		self.taskFailed(ctx, network, "getiwire", err)
		return
	}

	prefix, err := network.GetPrefix()
	if err != nil {
		self.taskFailed(ctx, network, "getprefix", err)
		return
	}

	inet, err := iwire.CreateINetwork(network.Name, prefix.String(), network.Description)
	if err != nil {
		self.taskFailed(ctx, network, "createinetwork", err)
		return
	}
	network.SetExternalId(inet.GetGlobalId())

	err = cloudprovider.WaitStatus(inet, models.NETWORK_STATUS_AVAILABLE, 10*time.Second, 300*time.Second)
	if err != nil {
		self.taskFailed(ctx, network, "waitstatu", err)
		return
	}

	err = network.SyncWithCloudNetwork(self.UserCred, inet, "", false)

	if err != nil {
		self.taskFailed(ctx, network, "SyncWithCloudNetwork", err)
		return
	}

	self.SetStageComplete(ctx, nil)
}
