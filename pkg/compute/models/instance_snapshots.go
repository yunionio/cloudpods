// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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

	// 云主机Id
	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	// 云主机配置
	ServerConfig jsonutils.JSONObject `nullable:"true" list:"user"`
	// 云主机标签
	ServerMetadata jsonutils.JSONObject `nullable:"true" list:"user"`
	// 是否自动删除
	AutoDelete bool `default:"false" update:"user" list:"user"`
	// 引用次数
	RefCount int `default:"0" list:"user"`
	// 安全组
	SecGroups jsonutils.JSONObject `nullable:"true" list:"user"`
	// 秘钥Id
	KeypairId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	// 操作系统类型
	OsType string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	// CPU架构
	OsArch string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// 套餐名称
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`
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

// 主机快照列表
func (manager *SInstanceSnapshotManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InstanceSnapshotListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	guestStr := query.ServerId
	if len(guestStr) > 0 {
		guestObj, err := GuestManager.FetchByIdOrName(userCred, guestStr)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("guests", guestStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("guest_id", guestObj.GetId())
	}

	if len(query.OsType) > 0 {
		q = q.In("os_type", query.OsType)
	}

	return q, nil
}

func (manager *SInstanceSnapshotManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InstanceSnapshotListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SInstanceSnapshotManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SInstanceSnapshot) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SInstanceSnapshot) getMoreDetails(userCred mcclient.TokenCredential, out api.InstanceSnapshotDetails) api.InstanceSnapshotDetails {
	if guest := GuestManager.FetchGuestById(self.GuestId); guest != nil {
		out.Guest = guest.Name
		out.GuestStatus = guest.Status
	}
	var osType string
	snapshots, _ := self.GetSnapshots()
	out.Snapshots = []api.SimpleSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		if snapshots[i].DiskType == api.DISK_TYPE_SYS {
			osType = snapshots[i].OsType
		}
		out.Snapshots = append(out.Snapshots, api.SimpleSnapshot{
			Id:            snapshots[i].Id,
			Name:          snapshots[i].Name,
			StorageId:     snapshots[i].StorageId,
			DiskType:      snapshots[i].DiskType,
			CloudregionId: snapshots[i].CloudregionId,
			Size:          snapshots[i].Size,
			Status:        snapshots[i].Status,
			StorageType:   snapshots[i].GetStorageType(),
		})
		out.Size += snapshots[i].Size

		if len(snapshots[i].StorageId) > 0 && out.StorageType == "" {
			out.StorageType = snapshots[i].GetStorageType()
		}
	}
	if len(osType) > 0 {
		out.Properties = map[string]string{"os_type": osType}
	}
	return out
}

func (self *SInstanceSnapshot) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.InstanceSnapshotDetails, error) {
	return api.InstanceSnapshotDetails{}, nil
}

func (manager *SInstanceSnapshotManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.InstanceSnapshotDetails {
	rows := make([]api.InstanceSnapshotDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.InstanceSnapshotDetails{
			VirtualResourceDetails: virtRows[i],
		}
		rows[i] = objs[i].(*SInstanceSnapshot).getMoreDetails(userCred, rows[i])
	}

	return rows
}

func (self *SInstanceSnapshot) StartCreateInstanceSnapshotTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	pendingUsage quotas.IQuota,
	parentTaskId string,
) error {
	if task, err := taskman.TaskManager.NewTask(
		ctx, "InstanceSnapshotCreateTask", self, userCred, nil, parentTaskId, "", pendingUsage); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (manager *SInstanceSnapshotManager) CreateInstanceSnapshot(
	ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, name string, autoDelete bool,
) (*SInstanceSnapshot, error) {
	instanceSnapshot := &SInstanceSnapshot{}
	instanceSnapshot.SetModelManager(manager, instanceSnapshot)
	instanceSnapshot.Name = name
	instanceSnapshot.ProjectId = userCred.GetProjectId()
	instanceSnapshot.DomainId = userCred.GetProjectDomainId()
	instanceSnapshot.GuestId = guest.Id
	instanceSnapshot.AutoDelete = autoDelete
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
	if len(guest.KeypairId) > 0 {
		instanceSnapshot.KeypairId = guest.KeypairId
	}
	serverMetadata := jsonutils.NewDict()
	if loginAccount := guest.GetMetadata("login_account", nil); len(loginAccount) > 0 {
		loginKey := guest.GetMetadata("login_key", nil)
		if len(guest.KeypairId) == 0 && len(loginKey) > 0 {
			passwd, e := utils.DescryptAESBase64(guest.Id, loginKey)
			if e == nil {
				serverMetadata.Set("login_account", jsonutils.NewString(loginAccount))
				serverMetadata.Set("passwd", jsonutils.NewString(passwd))
			}
		} else {
			serverMetadata.Set("login_key", jsonutils.NewString(loginKey))
			serverMetadata.Set("login_account", jsonutils.NewString(loginAccount))
		}
	}
	if osArch := guest.GetMetadata("os_arch", nil); len(osArch) > 0 {
		serverMetadata.Set("os_arch", jsonutils.NewString(osArch))
	}
	if osDist := guest.GetMetadata("os_distribution", nil); len(osDist) > 0 {
		serverMetadata.Set("os_distribution", jsonutils.NewString(osDist))
	}
	if osName := guest.GetMetadata("os_name", nil); len(osName) > 0 {
		serverMetadata.Set("os_name", jsonutils.NewString(osName))
	}
	if osVersion := guest.GetMetadata("os_version", nil); len(osVersion) > 0 {
		serverMetadata.Set("os_version", jsonutils.NewString(osVersion))
	}
	secs, _ := guest.GetSecgroups()
	if len(secs) > 0 {
		secIds := make([]string, len(secs))
		for i := 0; i < len(secs); i++ {
			secIds[i] = secs[i].Id
		}
		instanceSnapshot.SecGroups = jsonutils.Marshal(secIds)
	}
	instanceSnapshot.OsType = guest.OsType
	instanceSnapshot.OsArch = guest.OsArch
	instanceSnapshot.ServerMetadata = serverMetadata
	instanceSnapshot.InstanceType = guest.InstanceType
	err := manager.TableSpec().Insert(ctx, instanceSnapshot)
	if err != nil {
		return nil, err
	}
	return instanceSnapshot, nil
}

func (self *SInstanceSnapshot) ToInstanceCreateInput(
	sourceInput *api.ServerCreateInput) (*api.ServerCreateInput, error) {

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
	if sourceInput.VmemSize == 0 {
		sourceInput.VmemSize = serverConfig.Memory
	}
	if sourceInput.VcpuCount == 0 {
		sourceInput.VcpuCount = serverConfig.Ncpu
	}
	if len(self.KeypairId) > 0 {
		sourceInput.KeypairId = self.KeypairId
	}
	if self.SecGroups != nil {
		secGroups := make([]string, 0)
		inputSecgs := make([]string, 0)
		self.SecGroups.Unmarshal(&secGroups)
		for i := 0; i < len(secGroups); i++ {
			if secGrp := SecurityGroupManager.FetchSecgroupById(secGroups[i]); secGrp != nil {
				inputSecgs = append(inputSecgs, secGroups[i])
			}
		}
		sourceInput.Secgroups = inputSecgs
	}
	sourceInput.OsType = self.OsType
	sourceInput.InstanceType = self.InstanceType
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
	if self.Status == api.INSTANCE_SNAPSHOT_START_DELETE {
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
	self.SetStatus(userCred, api.INSTANCE_SNAPSHOT_START_DELETE, "InstanceSnapshotDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInstanceSnapshot) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SInstanceSnapshot) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SInstanceSnapshot) AddRefCount(ctx context.Context) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)
	_, err := db.Update(self, func() error {
		self.RefCount += 1
		return nil
	})
	return err
}

func (self *SInstanceSnapshot) DecRefCount(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)
	_, err := db.Update(self, func() error {
		self.RefCount -= 1
		return nil
	})
	if err == nil && self.RefCount == 0 && self.AutoDelete {
		self.StartInstanceSnapshotDeleteTask(ctx, userCred, "")
	}
	return err
}
