package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
)

type ImageCheckTask struct {
	taskman.STask
}

func init() {
	checkWorker := appsrv.NewWorkerManager("ImageCheckTaskWorkerManager", 2, 1024, true)
	taskman.RegisterTaskAndWorker(ImageCheckTask{}, checkWorker)
}

func (self *ImageCheckTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)

	self.SetStage("OnCheckComplete", nil)

	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		image.DoCheckStatus(ctx, self.UserCred, true)
		return nil, nil
	})
}

func (self *ImageCheckTask) OnCheckComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ImageCheckTask) OnCheckCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}
