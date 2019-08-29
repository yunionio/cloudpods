package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	InstanceSnapshotManager = &SInstanceSnapshotManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SInstanceSnapshot{},
			"instance_snapshots_tbl",
			"instance_snapshot",
			"instance_snapshots",
		),
	}
	InstanceSnapshotManager.SetVirtualObject(InstanceSnapshotManager)
}

type SInstanceSnapshot struct {
	db.SVirtualResourceBase

	GuestId      string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	ServerConfig jsonutils.JSONObject `nullable:"true" list:"user"`
}

type SInstanceSnapshotManager struct {
	db.SVirtualResourceBaseManager
}

var InstanceSnapshotManager *SInstanceSnapshotManager

func (manager *SInstanceSnapshotManager) AllowCreateItem(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) bool {
	return false
}

func (manager *SInstanceSnapshotManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid querystring format")
	}
	if guestId, _ := queryDict.GetString("guest_id"); len(guestId) > 0 {
		q = q.Equals("guest_id", guestId)
	}
	return q, nil
}

func (self *SInstanceSnapshot) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SInstanceSnapshot) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if guest := GuestManager.FetchGuestById(self.GuestId); guest != nil {
		extra.Set("guest_status", jsonutils.NewString(guest.Status))
		extra.Set("guest_name", jsonutils.NewString(guest.Name))
	}
	return extra
}

// func (self *SInstanceSnapshot) getMoreDetails()

func (self *SInstanceSnapshot) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	if guest := GuestManager.FetchGuestById(self.GuestId); guest != nil {
		extra.Set("guest_status", jsonutils.NewString(guest.Status))
		extra.Set("guest_name", jsonutils.NewString(guest.Name))
	}
	return extra, nil
}
func (self *SInstanceSnapshot) StartCreateInstanceSnapshotTask(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	pendingUsage quotas.IQuota, parentTaskId string) error {

	if task, err := taskman.TaskManager.NewTask(
		ctx, "InstanceSnapshotCreateTask", self, userCred, nil, parentTaskId, "", pendingUsage); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (manager *SInstanceSnapshotManager) CreateInstanceSnapshot(
	ctx context.Context, ownerId mcclient.IIdentityProvider, guest *SGuest, name string,
) (*SInstanceSnapshot, error) {
	instanceSnapshot := &SInstanceSnapshot{}
	instanceSnapshot.SetModelManager(manager, instanceSnapshot)
	instanceSnapshot.Name = name
	instanceSnapshot.ProjectId = ownerId.GetProjectId()
	instanceSnapshot.DomainId = ownerId.GetProjectDomainId()
	instanceSnapshot.GuestId = guest.Id
	guestSchedInput := guest.ToSchedDesc()

	for i := 0; i < len(guestSchedInput.Disks); i++ {
		guestSchedInput.Disks[i].ImageId = ""
	}
	guestSchedInput.Name = ""
	guestSchedInput.HostId = ""
	guestSchedInput.Project = ""
	guestSchedInput.Domain = ""
	for i := 0; i < len(guestSchedInput.Networks); i++ {
		guestSchedInput.Networks[i].Mac = ""
		guestSchedInput.Networks[i].Address = ""
		guestSchedInput.Networks[i].Address6 = ""
	}
	instanceSnapshot.ServerConfig = jsonutils.Marshal(guestSchedInput.ServerConfig)

	err := manager.TableSpec().Insert(instanceSnapshot)
	if err != nil {
		return nil, err
	}
	return instanceSnapshot, nil
}

func (self *SInstanceSnapshot) ToInstanceCreateInput(
	sourceInput *compute.ServerCreateInput) (*compute.ServerCreateInput, error) {

	serverConfig := new(schedapi.ServerConfig)
	if err := self.ServerConfig.Unmarshal(serverConfig); err != nil {
		return nil, errors.Wrap(err, "unmarshal sched input")
	}

	isjs := make([]SInstanceSnapshotJoint, 0)
	err := InstanceSnapshotJointManager.Query().Equals("instance_snapshot_id", self.Id).Asc("disk_index").All(&isjs)
	if err != nil {
		return nil, errors.Wrap(err, "fetch instance snapshots")
	}
	for i := 0; i < len(serverConfig.Disks); i++ {
		serverConfig.Disks[i].SnapshotId = isjs[serverConfig.Disks[i].Index].SnapshotId
	}
	sourceInput.Disks = serverConfig.Disks
	sourceInput.VmemSize = serverConfig.Memory
	sourceInput.VcpuCount = serverConfig.Ncpu
	sourceInput.Networks = serverConfig.Networks
	return sourceInput, nil
}

func (self *SInstanceSnapshot) GetSnapshots() ([]SSnapshot, error) {
	isjq := InstanceSnapshotJointManager.Query("snapshot_id").Equals("instance_snapshot_id", self.Id)
	snapshots := make([]SSnapshot, 0)
	err := SnapshotManager.Query().In("id", isjq).All(&snapshots)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err != nil && err == sql.ErrNoRows {
		return nil, nil
	} else {
		for i := 0; i < len(snapshots); i++ {
			snapshots[i].SetModelManager(SnapshotManager, &snapshots[i])
		}
		return snapshots, nil
	}
}

func (self *SInstanceSnapshot) GetInstanceSnapshotJointAt(diskIndex int) (*SInstanceSnapshotJoint, error) {
	ispj := new(SInstanceSnapshotJoint)
	err := InstanceSnapshotJointManager.Query().
		Equals("instance_snapshot_id", self.Id).Equals("disk_index", diskIndex).First(ispj)
	return ispj, err
}

func (self *SInstanceSnapshot) ValidateDeleteCondition(ctx context.Context) error {
	if self.Status == compute.INSTANCE_SNAPSHOT_START_DELETE {
		return httperrors.NewBadRequestError("can't delete snapshot in deleting")
	}
	return nil
}

func (self *SInstanceSnapshot) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	return self.StartInstanceSnapshotDeleteTask(ctx, userCred, "")
}

func (self *SInstanceSnapshot) StartInstanceSnapshotDeleteTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {

	task, err := taskman.TaskManager.NewTask(
		ctx, "InstanceSnapshotDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	self.SetStatus(userCred, compute.INSTANCE_SNAPSHOT_START_DELETE, "InstanceSnapshotDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInstanceSnapshot) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}
