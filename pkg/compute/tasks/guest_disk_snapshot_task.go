package tasks

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestDiskSnapshotTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDiskSnapshotTask{})
	taskman.RegisterTask(SnapshotDeleteTask{})
	taskman.RegisterTask(BatchSnapshotsDeleteTask{})
}

func (self *GuestDiskSnapshotTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.DoDiskSnapshot(ctx, guest)
}

func (self *GuestDiskSnapshotTask) DoDiskSnapshot(ctx context.Context, guest *models.SGuest) {
	diskId, err := self.Params.GetString("disk_id")
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
		return
	}
	snapshotId, err := self.Params.GetString("snapshot_id")
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
		return
	}
	self.SetStage("OnDiskSnapshotComplete", nil)
	guest.SetStatus(self.UserCred, models.VM_SNAPSHOT, "")
	err = guest.GetDriver().RequestDiskSnapshot(ctx, guest, self, snapshotId, diskId)
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
		return
	}
}

func (self *GuestDiskSnapshotTask) OnDiskSnapshotComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	res := data.(*jsonutils.JSONDict)
	location, err := res.GetString("location")
	if err != nil {
		log.Infof("OnDiskSnapshotComplete called with data no location")
		return
	}
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	models.SnapshotManager.TableSpec().Update(snapshot, func() error {
		snapshot.Location = location
		snapshot.Status = models.SNAPSHOT_READY
		return nil
	})
	guest.SetStatus(self.UserCred, models.VM_SNAPSHOT_SUCC, "")
	self.TaskComplete(ctx, guest, nil)
}

func (self *GuestDiskSnapshotTask) OnDiskSnapshotCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, err.String())
}

func (self *GuestDiskSnapshotTask) OnAutoDeleteSnapshot(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, guest, data)
}

func (self *GuestDiskSnapshotTask) OnAutoDeleteSnapshotFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	log.Errorf("Auto Delete Snapshot Failed %s", err.String())
	self.TaskComplete(ctx, guest, err)
}

func (self *GuestDiskSnapshotTask) TaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	self.SetStage("OnSyncStatus", nil)
}

func (self *GuestDiskSnapshotTask) OnSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDiskSnapshotTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	models.SnapshotManager.TableSpec().Update(snapshot, func() error {
		snapshot.Status = models.SNAPSHOT_FAILED
		return nil
	})
	self.SetStageFailed(ctx, reason)
	guest.SetStatus(self.UserCred, models.VM_SNAPSHOT_FAILED, reason)
}

/***************************** Snapshot Delete Task *****************************/

type SnapshotDeleteTask struct {
	taskman.STask
}

func (self *SnapshotDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	guest, err := snapshot.GetGuest()
	if err != nil {
		if err != sql.ErrNoRows {
			self.SetStageFailed(ctx, err.Error())
			return
		} else {
			// if snapshot is not used
			self.DeleteStaticSnapshot(ctx, snapshot)
			return
		}
	}
	if jsonutils.QueryBoolean(self.Params, "reload_disk", false) && snapshot.OutOfChain {
		self.StartReloadDisk(ctx, snapshot, guest)
	} else {
		self.StartDeleteSnapshot(ctx, snapshot, guest)
	}
}

func (self *SnapshotDeleteTask) StartReloadDisk(ctx context.Context, snapshot *models.SSnapshot, guest *models.SGuest) {
	self.SetStage("OnReloadDiskSnapshot", nil)
	guest.SetStatus(self.UserCred, models.VM_SNAPSHOT, "Start Reload Snapshot")
	params := jsonutils.NewDict()
	params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	err := guest.GetDriver().RequestReloadDiskSnapshot(ctx, guest, self, params)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotDeleteTask) StartDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, guest *models.SGuest) {
	snapshot.SetStatus(self.UserCred, models.SNAPSHOT_DELETING, "On SnapshotDeleteTask StartDeleteSnapshot")
	convertSnapshot, err := models.SnapshotManager.GetConvertSnapshot(snapshot)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
		return
	}
	if convertSnapshot == nil {
		self.TaskFailed(ctx, snapshot, "snapshot dose not have convert snapshot")
		return
	}
	params := jsonutils.NewDict()
	params.Set("delete_snapshot", jsonutils.NewString(snapshot.Id))
	params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	if !snapshot.OutOfChain {
		params.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
		var FakeDelete = jsonutils.JSONFalse
		if snapshot.CreatedBy == models.MANUAL && snapshot.FakeDeleted == false {
			FakeDelete = jsonutils.JSONTrue
		}
		params.Set("pending_delete", FakeDelete)
	} else {
		params.Set("auto_deleted", jsonutils.JSONTrue)
	}

	self.SetStage("OnDeleteSnapshot", nil)
	guest.SetStatus(self.UserCred, models.VM_SNAPSHOT, "Start Delete Snapshot")
	err = guest.GetDriver().RequestDeleteSnapshot(ctx, guest, self, params)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotDeleteTask) DeleteStaticSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	err := snapshot.FakeDelete()
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotDeleteTask) OnDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(data, "deleted", false) {
		log.Infof("OnDeleteSnapshot with no deleted")
		return
	}
	snapshot.SetStatus(self.UserCred, models.SNAPSHOT_READY, "OnDeleteSnapshot")
	if snapshot.OutOfChain {
		snapshot.RealDelete(ctx, self.UserCred)
		self.TaskComplete(ctx, snapshot, nil)
	} else {
		guest, _ := snapshot.GetGuest()
		var FakeDelete = false
		if snapshot.CreatedBy == models.MANUAL && snapshot.FakeDeleted == false {
			FakeDelete = true
		}
		if FakeDelete {
			models.SnapshotManager.TableSpec().Update(snapshot, func() error {
				snapshot.OutOfChain = true
				return nil
			})
		} else {
			snapshot.RealDelete(ctx, self.UserCred)
		}
		self.SetStage("TaskComplete", nil)
		guest.StartSyncstatus(ctx, self.UserCred, "")
	}
}

func (self *SnapshotDeleteTask) OnDeleteSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data.String())
}

func (self *SnapshotDeleteTask) OnReloadDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(data, "reopen", false) {
		log.Infof("OnDeleteSnapshot with no reopen")
		return
	}
	if snapshot.CreatedBy == models.AUTO {
		err := snapshot.RealDelete(ctx, self.UserCred)
		if err != nil {
			self.TaskFailed(ctx, snapshot, err.Error())
			return
		}
	} else {
		err := snapshot.FakeDelete()
		if err != nil {
			self.TaskFailed(ctx, snapshot, err.Error())
			return
		}
	}
	self.SetStage("TaskComplete", nil)
	guest, err := snapshot.GetGuest()
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *SnapshotDeleteTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotDeleteTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason string) {
	if snapshot.Status == models.SNAPSHOT_DELETING {
		snapshot.SetStatus(self.UserCred, models.SNAPSHOT_READY, "On SnapshotDeleteTask TaskFailed")
	}
	self.SetStageFailed(ctx, reason)
	guest, err := snapshot.GetGuest()
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

/***************************** Batch Snapshots Delete Task *****************************/

type BatchSnapshotsDeleteTask struct {
	taskman.STask
}

func (self *BatchSnapshotsDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	self.StartStorageDeleteSnapshot(ctx, snapshot)
}

func (self *BatchSnapshotsDeleteTask) StartStorageDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	host := snapshot.GetHost()
	if host == nil {
		self.SetStageFailed(ctx, "Cannot found snapshot host")
		return
	}
	self.SetStage("OnStorageDeleteSnapshot", nil)
	err := host.GetHostDriver().RequestDeleteSnapshotsWithStorage(ctx, host, snapshot, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *BatchSnapshotsDeleteTask) OnStorageDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshots := models.SnapshotManager.GetDiskSnapshots(snapshot.DiskId)
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].RealDelete(ctx, self.UserCred)
	}
	self.SetStageComplete(ctx, nil)
}
