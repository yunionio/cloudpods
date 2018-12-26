package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
)

type ImageCheckTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ImageCheckTask{})
}

func (self *ImageCheckTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)

	image.DoCheckStatus(ctx, self.UserCred, true)

	self.SetStageComplete(ctx, nil)
}
