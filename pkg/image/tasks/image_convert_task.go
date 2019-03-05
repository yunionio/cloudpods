package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
)

type ImageConvertTask struct {
	taskman.STask
}

func init() {
	convertWorker := appsrv.NewWorkerManager("ImageConvertTaskWorkerManager", 2, 512, true)
	taskman.RegisterTaskAndWorker(ImageConvertTask{}, convertWorker)
}

func (self *ImageConvertTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)

	self.SetStage("OnConvertComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		// image.SetStatus(self.UserCred, models.IMAGE_STATUS_CONVERTING, "start convert")
		// defer image.SetStatus(self.UserCred, models.IMAGE_STATUS_ACTIVE, "convert failed")
		err := image.ConvertAllSubformats()
		return nil, err
	})
}

func (self *ImageConvertTask) OnConvertComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ImageConvertTask) OnConvertCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}
