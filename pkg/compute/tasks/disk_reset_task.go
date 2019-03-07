package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskResetTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskResetTask{})
	taskman.RegisterTask(DiskCleanUpSnapshotsTask{})
}

func (self *DiskResetTask) TaskFailed(ctx context.Context, disk *models.SDisk, reason string) {
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESET_DISK, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *DiskResetTask) TaskCompleted(ctx context.Context, disk *models.SDisk, data *jsonutils.JSONDict) {
	// data不能为空指针，否则会导致AddActionLog抛空指针异常
	if data == nil {
		data = jsonutils.NewDict()
	}
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESET_DISK, data, self.UserCred, true)
	self.SetStageComplete(ctx, data)
}

func (self *DiskResetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := disk.GetStorage()
	if storage == nil {
		disk.SetStatus(self.UserCred, models.DISK_READY, "")
		self.TaskFailed(ctx, disk, "Disk storage not found")
		return
	}
	host := storage.GetMasterHost()
	if host == nil {
		disk.SetStatus(self.UserCred, models.DISK_READY, "")
		self.TaskFailed(ctx, disk, "Storage master host not found")
		return
	}
	self.RequestResetDisk(ctx, disk, host)
}

func (self *DiskResetTask) RequestResetDisk(ctx context.Context, disk *models.SDisk, host *models.SHost) {
	snapshotId, err := self.Params.GetString("snapshot_id")
	if err != nil {
		disk.SetStatus(self.UserCred, models.DISK_READY, "")
		self.TaskFailed(ctx, disk, fmt.Sprintf("Get snapshotId error %s", err.Error()))
		return
	}
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	params := jsonutils.NewDict()
	if len(snapshot.ExternalId) == 0 {
		params.Set("snapshot_id", jsonutils.NewString(snapshot.Id))
		if snapshot.OutOfChain {
			params.Set("out_of_chain", jsonutils.JSONTrue)
		} else {
			params.Set("out_of_chain", jsonutils.JSONFalse)
		}
	} else {
		params.Set("snapshot_id", jsonutils.NewString(snapshot.ExternalId))
	}
	self.SetStage("OnRequestResetDisk", nil)
	err = host.GetHostDriver().RequestResetDisk(ctx, host, disk, params, self)
	if err != nil {
		disk.SetStatus(self.UserCred, models.DISK_READY, "")
		self.TaskFailed(ctx, disk, err.Error())
	}
}

func (self *DiskResetTask) OnRequestResetDisk(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)

	externalId, _ := data.GetString("exteranl_disk_id")
	if disk.DiskSize != snapshot.Size || (len(externalId) > 0 && externalId != disk.GetExternalId()) {
		_, err := db.Update(disk, func() error {
			disk.DiskSize = snapshot.Size
			disk.ExternalId = externalId
			return nil
		})
		if err != nil {
			log.Errorln(err)
		}
	}
	if len(snapshot.ExternalId) == 0 {
		err := disk.CleanUpDiskSnapshots(ctx, self.UserCred, snapshot)
		if err != nil {
			log.Errorln(err)
			self.TaskFailed(ctx, disk, fmt.Sprintf("OnRequestResetDisk %s", err.Error()))
			return
		}
	}
	if jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		guest := disk.GetGuests()[0]
		self.SetStage("OnStartGuest", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
	} else {
		disk.SetStatus(self.UserCred, models.DISK_READY, "")
		self.TaskCompleted(ctx, disk, nil)
	}
}

func (self *DiskResetTask) OnStartGuest(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, models.DISK_READY, "")
	self.TaskCompleted(ctx, disk, nil)
}

type DiskCleanUpSnapshotsTask struct {
	SDiskBaseTask
}

func (self *DiskCleanUpSnapshotsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	self.StartCleanUpSnapshots(ctx, disk)
}

func (self *DiskCleanUpSnapshotsTask) StartCleanUpSnapshots(ctx context.Context, disk *models.SDisk) {
	db.OpsLog.LogEvent(disk, db.ACT_DISK_CLEAN_UP_SNAPSHOTS,
		fmt.Sprintf("start clean up disk snapshots: %s", self.Params.String()), self.UserCred)
	host := disk.GetStorage().GetMasterHost()
	self.SetStage("OnCleanUpSnapshots", nil)
	err := host.GetHostDriver().RequestCleanUpDiskSnapshots(ctx, host, disk, self.Params, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *DiskCleanUpSnapshotsTask) OnCleanUpSnapshots(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	convertSnapshots, _ := self.Params.GetArray("convert_snapshots")
	for i := 0; i < len(convertSnapshots); i++ {
		snapshot_id, _ := convertSnapshots[i].GetString()
		iSnapshot, err := models.SnapshotManager.FetchById(snapshot_id)
		if err != nil {
			log.Errorf("OnCleanUpSnapshots Fetch snapshot by id(%s) error:%s", snapshot_id, err.Error())
			continue
		}
		snapshot := iSnapshot.(*models.SSnapshot)
		db.Update(snapshot, func() error {
			snapshot.OutOfChain = true
			return nil
		})
	}
	deleteSnapshots, _ := self.Params.GetArray("delete_snapshots")
	for i := 0; i < len(deleteSnapshots); i++ {
		snapshot_id, _ := convertSnapshots[i].GetString()
		iSnapshot, err := models.SnapshotManager.FetchById(snapshot_id)
		if err != nil {
			log.Errorf("OnCleanUpSnapshots Fetch snapshot by id(%s) error:%s", snapshot_id, err.Error())
			continue
		}
		snapshot := iSnapshot.(*models.SSnapshot)
		snapshot.RealDelete(ctx, self.UserCred)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *DiskCleanUpSnapshotsTask) OnCleanUpSnapshotsFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(disk, db.ACT_DISK_CLEAN_UP_SNAPSHOTS_FAIL, data.String(), self.UserCred)
	self.SetStageFailed(ctx, data.String())
}
