package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotPolicyCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotPolicyCreateTask{})
	taskman.RegisterTask(SnapshotPolicyApplyTask{})
	taskman.RegisterTask(SnapshotPolicyCancelTask{})
}

func (self *SnapshotPolicyCreateTask) taskFail(ctx context.Context, sp *models.SSnapshotPolicy, reason string) {
	sp.SetStatus(self.UserCred, compute.SNAPSHOT_POLICY_CREATE_FAILED, "")
	db.OpsLog.LogEvent(sp, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_CREATE, false, self.UserCred, false)
	notifyclient.NotifySystemError(sp.GetId(), sp.Name, compute.SNAPSHOT_POLICY_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotPolicyCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshotPolicy := obj.(*models.SSnapshotPolicy)

	region := snapshotPolicy.GetRegion()
	if region == nil {
		self.taskFail(ctx, snapshotPolicy, fmt.Sprintf("failed to find region for snapshot policy %s", snapshotPolicy.Name))
		return
	}
	self.SetStage("OnSnapshotPolicyCreate", nil)
	if err := region.GetDriver().RequestCreateSnapshotPolicy(ctx, self.GetUserCred(), snapshotPolicy, self); err != nil {
		self.taskFail(ctx, snapshotPolicy, err.Error())
	}
}

func (self *SnapshotPolicyCreateTask) OnSnapshotPolicyCreate(
	ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject,
) {
	sp.SetStatus(self.UserCred, compute.SNAPSHOT_POLICY_READY, "")
	db.OpsLog.LogEvent(sp, db.ACT_ALLOCATE, sp.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyCreateTask) OnSnapshotPolicyCreateFailed(
	ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject,
) {
	self.taskFail(ctx, sp, data.String())
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type SnapshotPolicyApplyTask struct {
	taskman.STask
}

func (self *SnapshotPolicyApplyTask) taskFail(ctx context.Context, sp *models.SSnapshotPolicy, reason string) {
	stringIds, _ := getDiskIds(self)
	disks := make([]models.SDisk, 0)
	q := models.DiskManager.Query().In("id", stringIds)
	err := db.FetchModelObjects(models.DiskManager, q, &disks)
	if err == nil {
		for i := 0; i < len(disks); i++ {
			db.OpsLog.LogEvent(&disks[i], db.ACT_APPLY_SNAPSHOT_POLICY_FAILED, reason, self.UserCred)
			logclient.AddActionLogWithStartable(self, &disks[i], logclient.ACT_APPLY_SNAPSHOT_POLICY, reason, self.UserCred, false)
		}
	}
	self.SetStageFailed(ctx, reason)
}

func getDiskIds(task *SnapshotPolicyApplyTask) ([]string, error) {
	diskIds, err := task.Params.GetArray("disk_ids")
	if err != nil {
		return nil, fmt.Errorf("Missing parasm disk_ids")
	}
	stringIds := make([]string, len(diskIds))
	for i := 0; i < len(diskIds); i++ {
		stringIds[i], _ = diskIds[i].GetString()
	}
	return stringIds, nil
}

func (self *SnapshotPolicyApplyTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshotPolicy := obj.(*models.SSnapshotPolicy)

	region := snapshotPolicy.GetRegion()
	if region == nil {
		self.taskFail(ctx, snapshotPolicy, fmt.Sprintf("failed to find region for snapshot policy %s", snapshotPolicy.Name))
		return
	}

	stringIds, err := getDiskIds(self)
	if err != nil {
		self.taskFail(ctx, snapshotPolicy, err.Error())
		return
	}

	diskExt, err := models.DiskManager.Query("external_id").In("id", stringIds).AllStringMap()
	if err != nil {
		self.taskFail(ctx, snapshotPolicy, fmt.Sprintf("Fetch disks external_id failed %s", err))
		return
	}

	diskExtIds := make([]string, 0)
	for i := 0; i < len(diskExt); i++ {
		val, ok := diskExt[i]["external_id"]
		if ok {
			diskExtIds = append(diskExtIds, val)
		}
	}

	self.SetStage("OnSnapshotPolicyApply", nil)
	if err := region.GetDriver().RequestApplySnapshotPolicy(ctx, self.GetUserCred(), snapshotPolicy, self, diskExtIds); err != nil {
		self.taskFail(ctx, snapshotPolicy, err.Error())
	}
}

func (self *SnapshotPolicyApplyTask) OnSnapshotPolicyApply(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	stringIds, _ := getDiskIds(self)
	disks := make([]models.SDisk, 0)
	q := models.DiskManager.Query().In("id", stringIds)
	err := db.FetchModelObjects(models.DiskManager, q, &disks)
	if err != nil {
		self.taskFail(ctx, sp, fmt.Sprintf("Fetch disks failed %s", err))
		return
	}
	for i := 0; i < len(disks); i++ {
		disks[i].SetSnapshotPolicy(sp.Id)
		db.OpsLog.LogEvent(&disks[i], db.ACT_APPLY_SNAPSHOT_POLICY, nil, self.UserCred)
		logclient.AddActionLogWithStartable(self, &disks[i], logclient.ACT_APPLY_SNAPSHOT_POLICY, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyApplyTask) OnSnapshotPolicyApplyFailed(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	self.taskFail(ctx, sp, data.String())
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type SnapshotPolicyCancelTask struct {
	taskman.STask
}

func (self *SnapshotPolicyCancelTask) taskFail(ctx context.Context, disk *models.SDisk, reason string) {
	db.OpsLog.LogEvent(disk, db.ACT_CANCEL_SNAPSHOT_POLICY_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_CANCEL_SNAPSHOT_POLICY, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotPolicyCancelTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := disk.GetStorage()
	if storage == nil {
		self.taskFail(ctx, disk, fmt.Sprintf("failed to find storage for disk %s", disk.Name))
		return
	}
	region := storage.GetRegion()
	if region == nil {
		self.taskFail(ctx, disk, fmt.Sprintf("failed to find region for disk %s", disk.Name))
		return
	}

	iSnapshotPolicy, err := models.SnapshotPolicyManager.FetchById(disk.SnapshotPolicyId)
	if err != nil {
		self.taskFail(ctx, disk, fmt.Sprintf("failed to find snapshot policy for disk %s, %s", disk.Name, err))
		return
	}
	snapshotPolicy := iSnapshotPolicy.(*models.SSnapshotPolicy)
	iRegion, err := snapshotPolicy.GetIRegion()
	if err != nil {
		self.taskFail(ctx, disk, fmt.Sprintf("failed to find region for snapshot policy %s", snapshotPolicy.Name))
		return
	}
	self.SetStage("OnSnapshotPolicyCancel", nil)
	if err := region.GetDriver().RequestCancelSnapshotPolicy(
		ctx, self.GetUserCred(), iRegion, self, []string{disk.ExternalId}); err != nil {
		self.taskFail(ctx, disk, err.Error())
	}
}

func (self *SnapshotPolicyCancelTask) OnSnapshotPolicyCancel(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetSnapshotPolicy("")
	db.OpsLog.LogEvent(disk, db.ACT_CANCEL_SNAPSHOT_POLICY, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_CANCEL_SNAPSHOT_POLICY, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyCancelTask) OnSnapshotPolicyCancelFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.taskFail(ctx, disk, data.String())
}
