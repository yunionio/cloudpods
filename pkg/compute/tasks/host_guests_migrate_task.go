package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(HostGuestsMigrateTask{})
}

type HostGuestsMigrateTask struct {
	taskman.STask
}

func (self *HostGuestsMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guests := make([]*api.GuestBatchMigrateParams, 0)
	err := self.Params.Unmarshal(&guests, "guests")
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	preferHostId, _ := self.Params.GetString("prefer_host_id")

	var guestMigrating bool
	var migrateIndex int
	for i := 0; i < len(guests); i++ {
		guest := models.GuestManager.FetchGuestById(guests[i].Id)
		if guests[i].LiveMigrate {
			err := guest.StartGuestLiveMigrateTask(
				ctx, self.UserCred, guests[i].OldStatus, preferHostId, self.Id)
			if err != nil {
				log.Errorln(err)
				continue
			} else {
				guestMigrating = true
				migrateIndex = i
				break
			}
		} else {
			err := guest.StartMigrateTask(ctx, self.UserCred, guests[i].RescueMode,
				false, guests[i].OldStatus, preferHostId, self.Id)
			if err != nil {
				log.Errorln(err)
				continue
			} else {
				guestMigrating = true
				migrateIndex = i
				break
			}
		}
	}
	if !guestMigrating {
		self.SetStageComplete(ctx, nil)
	} else {
		guests := append(guests[:migrateIndex], guests[migrateIndex+1:]...)
		params := jsonutils.NewDict()
		params.Set("guests", jsonutils.Marshal(guests))
		self.SaveParams(params)
	}
}

func (self *HostGuestsMigrateTask) OnInitFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("HostGuestsMigrateTask on failed %s", data)
	self.OnInit(ctx, obj, data)
}
