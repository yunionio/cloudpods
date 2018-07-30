package tasks

import (
	"context"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/jsonutils"
)

type GuestInsertIsoTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestInsertIsoTask{})
}

func (self *GuestInsertIsoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// TODO
}
