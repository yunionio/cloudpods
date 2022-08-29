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

func (self *ModelartsPoolCreateTask) taskFailed(ctx context.Context, fs *models.SModelartsPool, err error) {
	fs.SetStatus(self.UserCred, api.NAS_STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, fs, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ModelartsPoolCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)

	opts := &cloudprovider.ModelartsPoolCreateOption{
		Name:           pool.Name,
		ResourceFlavor: pool.InstanceType,
	}
	iProvider, err := pool.GetDriver(ctx)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "fs.GetIRegion"))
		return
	}
	log.Infof("nas create params: %s", jsonutils.Marshal(opts).String())

	ipool, err := iProvider.CreateIModelartsPool(opts)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iRegion.CreaetICloudFileSystem"))
		return
	}
	db.SetExternalId(pool, self.GetUserCred(), "")

	cloudprovider.WaitMultiStatus(ipool, []string{api.NAS_STATUS_AVAILABLE, api.NAS_STATUS_CREATE_FAILED}, time.Second*5, time.Minute*10)

	// tags, _ := pool.GetAllUserMetadata()
	// if len(tags) > 0 {
	// 	err = ipool.SetTags(tags, true)
	// 	if err != nil {
	// 		logclient.AddActionLogWithStartable(self, pool, logclient.ACT_UPDATE, errors.Wrapf(err, "SetTags"), self.UserCred, false)
	// 	}
	// }

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionCreate,
	})

	self.SetStage("OnSyncstatusComplete", nil)
	pool.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}
