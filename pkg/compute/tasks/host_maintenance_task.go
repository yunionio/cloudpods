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

type HostMaintainTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostMaintainTask{})
}

func (self *HostMaintainTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	guests := host.GetGuests()
	preferHostId, _ := data.GetString("prefer_host_id")

	var doMigrate bool
	var hasGuestCannotMigrate bool
	for i := 0; i < len(guests); i++ {
		if guests[i].Status == api.VM_READY || guests[i].Status == api.VM_UNKNOWN {
			rescueMode := guests[i].Status == api.VM_UNKNOWN
			if rescueMode {
				guestDisks := guests[i].GetDisks()
				var allDiskIsShared = true
				for _, guestDisk := range guestDisks {
					if guestDisk.GetDisk().GetStorage().StorageType == api.STORAGE_LOCAL {
						allDiskIsShared = false
						break
					}
				}
				if !allDiskIsShared {
					hasGuestCannotMigrate = true
					continue
				}
			}
			err := guests[i].StartMigrateTask(ctx, self.UserCred, rescueMode, false, guests[i].Status, preferHostId, self.Id)
			if err != nil {
				log.Errorf("Start migrate task failed %s", err)
				hasGuestCannotMigrate = true
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
				hasGuestCannotMigrate = true
				continue
			} else {
				log.Infof("Start migrate %s(%s)", guests[i].Name, guests[i].Id)
				doMigrate = true
				break
			}
		}
	}
	if hasGuestCannotMigrate {
		kwargs := jsonutils.NewDict()
		kwargs.Set("some_guest_migrate_failed", jsonutils.JSONTrue)
		self.SaveParams(kwargs)
	}

	if !doMigrate {
		hostStatus := api.HOST_MAINTAINING
		if jsonutils.QueryBoolean(self.Params, "some_guest_migrate_failed", false) {
			hostStatus = api.HOST_MAINTAIN_FAILE
		}
		host.PerformDisable(ctx, self.UserCred, nil, nil)
		host.SetStatus(self.UserCred, hostStatus, "On host maintain task complete")
		logclient.AddSimpleActionLog(host, logclient.ACT_HOST_MAINTAINING, "host maintain", self.UserCred, hostStatus == api.HOST_MAINTAINING)
		self.SetStageComplete(ctx, nil)
		return
	}
}

// Ignore guest migrate fail
func (self *HostMaintainTask) OnInitFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	kwargs := jsonutils.NewDict()
	kwargs.Set("some_guest_migrate_failed", jsonutils.JSONTrue)
	self.SaveParams(kwargs)
	self.OnInit(ctx, obj, data)
}
