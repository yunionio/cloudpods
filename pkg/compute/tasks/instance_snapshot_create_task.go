package tasks

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type InstanceSnapshotCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotCreateTask{})
}

func (self *InstanceSnapshotCreateTask) SetStageFailed(ctx context.Context, reason string) {
	self.finalReleasePendingUsage(ctx)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotCreateTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err == nil && !pendingUsage.IsEmpty() {
		isp := self.GetObject().(*models.SInstanceSnapshot)
		guest := models.GuestManager.FetchGuestById(isp.GuestId)
		quotaPlatform := guest.GetQuotaPlatformID()
		models.QuotaManager.CancelPendingUsage(
			ctx, self.UserCred, rbacutils.ScopeProject,
			guest.GetOwnerId(), quotaPlatform, &pendingUsage, &pendingUsage,
		)
	}
}

func (self *InstanceSnapshotCreateTask) taskFail(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, reason string) {

	if guest == nil {
		guest = models.GuestManager.FetchGuestById(isp.GuestId)
	}
	isp.SetStatus(self.UserCred, compute.INSTANCE_SNAPSHOT_FAILED, reason)
	guest.SetStatus(self.UserCred, compute.VM_INSTANCE_SNAPSHOT_FAILED, reason)

	db.OpsLog.LogEvent(isp, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, isp, logclient.ACT_CREATE, false, self.UserCred, false)
	notifyclient.NotifySystemError(isp.GetId(), isp.Name, compute.INSTANCE_SNAPSHOT_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotCreateTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, data jsonutils.JSONObject) {

	self.finalReleasePendingUsage(ctx)
	if guest == nil {
		guest = models.GuestManager.FetchGuestById(isp.GuestId)
	}
	isp.SetStatus(self.UserCred, compute.INSTANCE_SNAPSHOT_READY, "")
	guest.StartSyncstatus(ctx, self.UserCred, "")

	db.OpsLog.LogEvent(isp, db.ACT_ALLOCATE, "instance snapshot create success", self.UserCred)
	logclient.AddActionLogWithStartable(self, isp, logclient.ACT_CREATE, false, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotCreateTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	self.GuestDiskCreateSnapshot(ctx, isp, guest, 0)
}

func (self *InstanceSnapshotCreateTask) GuestDiskCreateSnapshot(
	ctx context.Context, isp *models.SInstanceSnapshot, guest *models.SGuest, diskIndex int) {

	disks := guest.GetDisks()
	if diskIndex >= len(disks) {
		self.taskComplete(ctx, isp, guest, nil)
		return
	}

	snapshot, err := models.SnapshotManager.CreateSnapshot(
		ctx, self.UserCred, compute.SNAPSHOT_MANUAL, disks[diskIndex].DiskId, guest.Id,
		"", fmt.Sprintf("%s-snapshot-%s", guest.Name, timeutils.CompactTime(time.Now())), -1)
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}

	params := jsonutils.NewDict()
	params.Set("disk_index", jsonutils.NewInt(int64(diskIndex)))
	params.Set(strconv.Itoa(diskIndex), jsonutils.NewString(snapshot.Id))
	self.SetStage("OnDiskSnapshot", params)

	if err := snapshot.StartSnapshotCreateTask(ctx, self.UserCred, nil, self.Id); err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}
}

func (self *InstanceSnapshotCreateTask) OnDiskSnapshot(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	guest := models.GuestManager.FetchGuestById(isp.GuestId)

	diskIndex, err := self.Params.Int("disk_index")
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}

	snapshotId, err := self.Params.GetString(strconv.Itoa(int(diskIndex)))
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}

	err = models.InstanceSnapshotJointManager.CreateJoint(isp.Id, snapshotId, int8(diskIndex))
	if err != nil {
		self.taskFail(ctx, isp, guest, err.Error())
		return
	}

	self.GuestDiskCreateSnapshot(ctx, isp, guest, int(diskIndex+1))
}

func (self *InstanceSnapshotCreateTask) OnDiskSnapshotFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFail(ctx, isp, nil, data.String())
}
