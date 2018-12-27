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

	image.DoCheckStatus(ctx, self.UserCred, true)

	self.SetStageComplete(ctx, nil)
}
