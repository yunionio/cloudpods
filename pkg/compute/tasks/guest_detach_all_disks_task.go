package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestDetachAllDisksTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDetachAllDisksTask{})
}

func (self *GuestDetachAllDisksTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_disk_delete_complete", nil)
	self.OnDiskDeleteComplete(ctx, obj, data)
}

func (self *GuestDetachAllDisksTask) OnDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if guest.DiskCount() == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}
	host := guest.GetHost()
	purge := false
	if (host == nil || !host.Enabled) && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	for _, guestdisk := range guest.GetDisks() {
		taskData := jsonutils.NewDict()
		taskData.Add(jsonutils.NewString(guestdisk.DiskId), "disk_id")
		if purge {
			taskData.Add(jsonutils.JSONTrue, "purge")
		}
		if jsonutils.QueryBoolean(self.Params, "override_pending_delete", false) {
			taskData.Add(jsonutils.JSONTrue, "override_pending_delete")
		}
		disk := guestdisk.GetDisk()
		storage := disk.GetStorage()
		if storage.IsLocal() {
			taskData.Add(jsonutils.JSONFalse, "keep_disk")
		} else {
			taskData.Add(jsonutils.JSONTrue, "keep_disk")
		}
		task, err := taskman.TaskManager.NewTask(ctx, "GuestDetachDiskTask", guest, self.UserCred, taskData, self.GetTaskId(), "", nil)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
		} else {
			task.ScheduleRun(nil)
		}
		break
	}
}
