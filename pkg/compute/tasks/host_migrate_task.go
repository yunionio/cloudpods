package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type HostMigrateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostMigrateTask{})
}

func (self *HostMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	guests := host.GetGuests()
	preferHostId, _ := data.GetString("prefer_host_id")

	var doMigrate bool
	for i := 0; i < len(guests); i++ {
		if guests[i].Status == api.VM_READY {
			err := guests[i].StartMigrateTask(ctx, self.UserCred, false, false, guests[i].Status, preferHostId, self.Id)
			if err != nil {
				log.Errorf("Start migrate task failed %s", err)
				continue
			} else {
				log.Infof("Start migrate %s(%s)", guests[i].Name, guests[i].Id)
				doMigrate = true
				break
			}
		} else if guests[i].Status == api.VM_RUNNING {
			err := guests[i].StartGuestLiveMigrateTask(ctx, self.UserCred, guests[i].Status, preferHostId, self.Id)
			if err != nil {
				log.Errorf("Start migrate task failed %s", err)
				continue
			} else {
				log.Infof("Start migrate %s(%s)", guests[i].Name, guests[i].Id)
				doMigrate = true
				break
			}
		}
	}
	if !doMigrate {
		db.OpsLog.LogEvent(host, db.ACT_HOST_MIGRATE, "host migrate", self.UserCred)
		logclient.AddSimpleActionLog(host, logclient.ACT_HOST_MIGRATE, "host migrate succ", self.UserCred, true)
		self.SetStageComplete(ctx, nil)
		return
	}
}

// Ignore guest migrate fail
func (self *HostMigrateTask) OnInitFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.OnInit(ctx, obj, data)
}
