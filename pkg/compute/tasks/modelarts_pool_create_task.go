package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/errors"
)

type ModelartsPoolCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ModelartsPoolCreateTask{})
}

func (self *ModelartsPoolCreateTask) taskFailed(ctx context.Context, pool *models.SModelartsPool, err error) {
	pool.SetStatus(self.UserCred, api.MODELARTS_POOL_STATUS_CREATING, err.Error())
	logclient.AddActionLogWithStartable(self, pool, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ModelartsPoolCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)

	opts := &cloudprovider.ModelartsPoolCreateOption{
		Name:         pool.Name,
		InstanceType: pool.InstanceType,
		WorkType:     pool.WorkType,
	}
	iProvider, err := pool.GetDriver(ctx)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "fs.GetIRegion"))
		return
	}

	ipool, err := iProvider.CreateIModelartsPool(opts)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iRegion.CreateIModelartsPool"))
		return
	}
	err = db.SetExternalId(pool, self.GetUserCred(), ipool.GetGlobalId())
	if err != nil {
		log.Errorln(err, "SetExternalId")
		return
	}
	err = cloudprovider.WaitStatusWithDelay(ipool, api.MODELARTS_POOL_STATUS_RUNNING, 30*time.Second, 15*time.Second, 600*time.Second)
	// err = cloudprovider.WaitMultiStatus(ipool, []string{api.MODELARTS_POOL_STATUS_RUNNING, api.MODELARTS_POOL_STATUS_ERROR}, time.Second*5, time.Minute*10)
	if err != nil {
		log.Errorln(err, "WaitMultiStatus")
		return
	}
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionCreate,
	})

	self.SetStage("OnSyncstatusComplete", nil)
	pool.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}
