package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(GuestDeleteOnHostTask{})
}

type GuestDeleteOnHostTask struct {
	SGuestBaseTask
}

func (self *GuestDeleteOnHostTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	hostId, err := self.Params.GetString("host_id")
	if err != nil {
		self.SetStageFailed(ctx, "Missing param host id")
		return
	}
	host := models.HostManager.FetchHostById(hostId)
	if host == nil {
		self.SetStageFailed(ctx, "Host is nil")
		return
	}
	guest := obj.(*models.SGuest)

	self.SetStage("OnStopGuest", nil)
	self.Params.Set("is_force", jsonutils.JSONTrue)
	if err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self); err != nil {
		log.Errorf("RequestStopGuestForDelete fail %s", err)
		self.OnStopGuest(ctx, guest, nil)
	}
}

func (self *GuestDeleteOnHostTask) OnStopGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	host := models.HostManager.FetchHostById(hostId)

	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	disks := guest.GetDisks()

	for _, guestDiks := range disks {
		disk := guestDiks.GetDisk()
		storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
		if storage != nil && !isPurge {
			if err := host.GetHostDriver().RequestDeallocateBackupDiskOnHost(ctx, host, storage, disk, self); err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
		}
		_, err := models.DiskManager.TableSpec().Update(disk, func() error {
			disk.BackupStorageId = ""
			return nil
		})
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
	}
	if !isPurge {
		self.SetStage("OnUnDeployGuest", nil)
		guest.StartUndeployGuestTask(ctx, self.GetUserCred(), self.GetTaskId(), hostId)
	} else {
		self.OnUnDeployGuest(ctx, guest, nil)
	}
}

func (self *GuestDeleteOnHostTask) OnUnDeployGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	if guest.BackupHostId == hostId {
		_, err := models.GuestManager.TableSpec().Update(guest, func() error {
			guest.BackupHostId = ""
			return nil
		})
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
	}
	self.SetStageComplete(ctx, nil)
}
