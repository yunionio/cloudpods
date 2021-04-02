package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type HostSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostSyncTask{})
}

func (self *HostSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	err := host.GetHostDriver().RequestSyncOnHost(ctx, host, self)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		log.Errorf("syncHost:%s err:%v", host.GetId(), err)
	}
}
