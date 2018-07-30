package tasks

import (
	"context"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/jsonutils"
)

type GuestDetachDiskTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDetachDiskTask{})
}

func (self *GuestDetachDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

}
