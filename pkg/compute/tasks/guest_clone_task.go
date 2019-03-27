package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestCloneTask struct {
	GuestBatchCreateTask
}

func init() {
	taskman.RegisterTask(GuestCloneTask{})
}

func (self *GuestCloneTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, []db.IStandaloneModel{obj})
}

func (self *GuestCloneTask) OnScheduleComplete(ctx context.Context, guest *models.SGuest, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
