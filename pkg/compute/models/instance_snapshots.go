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
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase
	db.SMultiArchResourceBase

	db.SEncryptedResource

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
	// 套餐名称
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`
	// 主机快照磁盘容量和
	// SizeMb int `nullable:"false" list:"user"`
	// 镜像ID
	ImageId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
	// 是否保存内存
	WithMemory bool `default:"false" get:"user" list:"user"`
	// 内存文件大小
	MemorySizeKB int `nullable:"true" get:"user" list:"user" old_name:"memory_size_mb"`
	// 内存文件所在宿主机
	MemoryFileHostId string `width:"36" charset:"ascii" nullable:"true" get:"user" list:"user"`
	// 内存文件路径
	MemoryFilePath string `width:"512" charset:"utf8" nullable:"true" get:"user" list:"user"`
	// 内存文件校验和
	MemoryFileChecksum string `width:"32" charset:"ascii" nullable:"true" get:"user" list:"user"`
}

type SInstanceSnapshotManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	db.SMultiArchResourceBaseManager
	db.SEncryptedResourceManager
}

var InstanceSnapshotManager *SInstanceSnapshotManager

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

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, query.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMultiArchResourceBaseManager.ListItemFilter")
	}

	guestStr := query.ServerId
	if len(guestStr) > 0 {
		guestObj, err := GuestManager.FetchByIdOrName(ctx, userCred, guestStr)
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

	if query.WithMemory != nil {
		if *query.WithMemory {
			q = q.IsTrue("with_memory")
		} else {
			q = q.IsFalse("with_memory")
		}
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

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	if db.NeedOrderQuery([]string{query.OrderByDiskSnapshotCount}) {
		nQ := SnapshotManager.Query()
		nQ = nQ.AppendField(nQ.Field("disk_id"), sqlchemy.COUNT("snapshot_count"))
		nQ = nQ.GroupBy("disk_id")
		nSQ := nQ.SubQuery()

		guestdiskQ := GuestdiskManager.Query()
		guestdiskQ = guestdiskQ.LeftJoin(nSQ, sqlchemy.Equals(nSQ.Field("disk_id"), guestdiskQ.Field("disk_id")))
		guestdiskSQ := guestdiskQ.AppendField(guestdiskQ.Field("guest_id"), nSQ.Field("snapshot_count")).SubQuery()

		q = q.LeftJoin(guestdiskSQ, sqlchemy.Equals(guestdiskSQ.Field("guest_id"), q.Field("guest_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(guestdiskSQ.Field("snapshot_count"))
		q = db.OrderByFields(q, []string{query.OrderByDiskSnapshotCount}, []sqlchemy.IQueryField{q.Field("snapshot_count")})
	}

	if db.NeedOrderQuery([]string{query.OrderByGuest}) {
		guestQ := GuestManager.Query()
		guestSQ := guestQ.AppendField(guestQ.Field("id"), guestQ.Field("name").Label("guest_name")).SubQuery()
		q = q.LeftJoin(guestSQ, sqlchemy.Equals(guestSQ.Field("id"), q.Field("guest_id")))
		q = q.AppendField(q.QueryFields()...)
		q = q.AppendField(guestSQ.Field("guest_name"))
		q = db.OrderByFields(q, []string{query.OrderByGuest}, []sqlchemy.IQueryField{q.Field("guest_name")})
	}
	return q, nil
}

func (manager *SInstanceSnapshotManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SInstanceSnapshotManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SInstanceSnapshotManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, resource string, fields []string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraFields(q, resource, fields)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SInstanceSnapshot) GetGuest() (*SGuest, error) {
	if len(self.GuestId) == 0 {
		return nil, errors.ErrNotFound
	}
	guest := GuestManager.FetchGuestById(self.GuestId)
	if guest == nil {
		return nil, errors.ErrNotFound
	}
	return guest, nil
}

func (self *SInstanceSnapshot) getMoreDetails(userCred mcclient.TokenCredential, out api.InstanceSnapshotDetails) api.InstanceSnapshotDetails {
	guest := GuestManager.FetchGuestById(self.GuestId)
	if guest != nil {
		out.Guest = guest.Name
		out.GuestStatus = guest.Status
	}
	var osType string
	provider := self.GetProviderName()
	if utils.IsInStringArray(provider, ProviderHasSubSnapshot) {
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
				VirtualSize:   snapshots[i].VirtualSize,
				Status:        snapshots[i].Status,
				StorageType:   snapshots[i].GetStorageType(),
				EncryptKeyId:  snapshots[i].EncryptKeyId,
				CreatedAt:     snapshots[i].CreatedAt,
			})
			out.SizeMb += snapshots[i].Size
			out.VirtualSizeMb += snapshots[i].VirtualSize
			if len(snapshots[i].StorageId) > 0 && out.StorageType == "" {
				out.StorageType = snapshots[i].GetStorageType()
			}
		}
		if out.VirtualSizeMb <= 0 && guest != nil {
			out.VirtualSizeMb = guest.getDiskSize()
		}
	} else if guest != nil {
		disk, err := guest.GetSystemDisk()
		if err != nil {
			log.Errorf("unable to GetSystemDisk of guest %q", guest.GetId())
		} else {
			s, _ := disk.GetStorage()
			if s != nil {
				out.StorageType = s.StorageType
			}
		}
		out.VirtualSizeMb = guest.getDiskSize()
		out.SizeMb = out.VirtualSizeMb
	}
	if len(osType) > 0 {
		out.Properties = map[string]string{"os_type": osType}
	}
	out.SizeMb += out.MemorySizeKB / 1024
	out.Size = out.SizeMb * 1024 * 1024
	return out
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
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	encRows := manager.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.InstanceSnapshotDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    manRows[i],

			EncryptedResourceDetails: encRows[i],
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

func (manager *SInstanceSnapshotManager) fillInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, instanceSnapshot *SInstanceSnapshot) {
	instanceSnapshot.SetModelManager(manager, instanceSnapshot)
	instanceSnapshot.ProjectId = guest.ProjectId
	instanceSnapshot.DomainId = guest.DomainId
	instanceSnapshot.GuestId = guest.Id
	instanceSnapshot.InstanceType = guest.InstanceType
	instanceSnapshot.ImageId = guest.GetTemplateId()

	// inherit encrypt_key_id from guest
	instanceSnapshot.EncryptKeyId = guest.EncryptKeyId

	guestSchedInput := guest.ToSchedDesc()

	host, _ := guest.GetHost()
	instanceSnapshot.ManagerId = host.ManagerId
	zone, _ := host.GetZone()
	instanceSnapshot.CloudregionId = zone.CloudregionId

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
		keypair, _ := KeypairManager.FetchById(guest.KeypairId)
		if keypair != nil {
			instanceSnapshot.KeypairId = guest.KeypairId
		}
	}
	serverMetadata := jsonutils.NewDict()
	if loginAccount := guest.GetMetadata(ctx, "login_account", nil); len(loginAccount) > 0 {
		loginKey := guest.GetMetadata(ctx, "login_key", nil)
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
	if osArch := guest.GetMetadata(ctx, "os_arch", nil); len(osArch) > 0 {
		serverMetadata.Set("os_arch", jsonutils.NewString(osArch))
	}
	if osDist := guest.GetMetadata(ctx, "os_distribution", nil); len(osDist) > 0 {
		serverMetadata.Set("os_distribution", jsonutils.NewString(osDist))
	}
	if osName := guest.GetMetadata(ctx, "os_name", nil); len(osName) > 0 {
		serverMetadata.Set("os_name", jsonutils.NewString(osName))
	}
	if osVersion := guest.GetMetadata(ctx, "os_version", nil); len(osVersion) > 0 {
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
}

func (manager *SInstanceSnapshotManager) CreateInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, name string, autoDelete bool, withMemory bool) (*SInstanceSnapshot, error) {
	instanceSnapshot := &SInstanceSnapshot{}
	instanceSnapshot.SetModelManager(manager, instanceSnapshot)
	instanceSnapshot.Name = name
	instanceSnapshot.AutoDelete = autoDelete
	if autoDelete {
		// hide auto-delete instance snapshots
		instanceSnapshot.IsSystem = true
	}
	manager.fillInstanceSnapshot(ctx, userCred, guest, instanceSnapshot)
	// compute size of instanceSnapshot
	// instanceSnapshot.SizeMb = guest.getDiskSize()
	instanceSnapshot.WithMemory = withMemory
	instanceSnapshot.MemoryFileHostId = guest.HostId
	err := manager.TableSpec().Insert(ctx, instanceSnapshot)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}
	err = db.InheritFromTo(ctx, userCred, guest, instanceSnapshot)
	if err != nil {
		return nil, errors.Wrap(err, "Inherit ClassMetadata")
	}
	return instanceSnapshot, nil
}

var HypervisorIndependentInstanceSnapshot = []string{
	api.HYPERVISOR_KVM,
}

var ProviderHasSubSnapshot = []string{
	api.CLOUD_PROVIDER_ONECLOUD,
}

func (self *SInstanceSnapshot) ToInstanceCreateInput(
	sourceInput *api.ServerCreateInput) (*api.ServerCreateInput, error) {

	serverConfig := new(schedapi.ServerConfig)
	if err := self.ServerConfig.Unmarshal(serverConfig); err != nil {
		return nil, errors.Wrap(err, "unmarshal sched input")
	}

	provider := self.GetProviderName()
	if utils.IsInStringArray(provider, ProviderHasSubSnapshot) {
		isjs := make([]SInstanceSnapshotJoint, 0)
		err := InstanceSnapshotJointManager.Query().Equals("instance_snapshot_id", self.Id).Asc("disk_index").All(&isjs)
		if err != nil {
			return nil, errors.Wrap(err, "fetch instance snapshots")
		}

		for i := 0; i < len(serverConfig.Disks); i++ {
			index := serverConfig.Disks[i].Index
			if index < len(isjs) {
				serverConfig.Disks[i].SnapshotId = isjs[index].SnapshotId
				if i == 0 && len(self.ImageId) > 0 {
					// system disk, save ImageId
					serverConfig.Disks[i].ImageId = self.ImageId
				}
			}
		}
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
			_, err := SecurityGroupManager.FetchSecgroupById(secGroups[i])
			if err == nil {
				inputSecgs = append(inputSecgs, secGroups[i])
			}
		}
		sourceInput.Secgroups = inputSecgs
	}
	sourceInput.OsType = self.OsType
	sourceInput.OsArch = self.OsArch
	sourceInput.InstanceType = self.InstanceType
	if len(sourceInput.Networks) == 0 {
		sourceInput.Networks = serverConfig.Networks
	}

	if self.IsEncrypted() {
		if sourceInput.EncryptKeyId != nil && *sourceInput.EncryptKeyId != self.EncryptKeyId {
			return nil, errors.Wrap(httperrors.ErrConflict, "encrypt_key_id conflict with instance_snapshot's encrypt_key_id")
		}
		sourceInput.EncryptKeyId = &self.EncryptKeyId
	}

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

func (self *SInstanceSnapshot) GetQuotaKeys() quotas.IQuotaKeys {
	region, _ := self.GetRegion()
	return fetchRegionalQuotaKeys(
		rbacscope.ScopeProject,
		self.GetOwnerId(),
		region,
		self.GetCloudprovider(),
	)
}

func (self *SInstanceSnapshot) GetUsages() []db.IUsage {
	if self.PendingDeleted || self.Deleted {
		return nil
	}
	usage := SRegionQuota{InstanceSnapshot: 1}
	keys := self.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func TotalInstanceSnapshotCount(ctx context.Context, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) (int, error) {
	q := InstanceSnapshotManager.Query()

	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

	q = db.ObjectIdQueryWithPolicyResult(ctx, q, InstanceSnapshotManager, policyResult)

	q = RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)
	q = CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	return q.CountWithError()
}

func (self *SInstanceSnapshot) GetInstanceSnapshotJointAt(diskIndex int) (*SInstanceSnapshotJoint, error) {
	ispj := new(SInstanceSnapshotJoint)
	err := InstanceSnapshotJointManager.Query().
		Equals("instance_snapshot_id", self.Id).Equals("disk_index", diskIndex).First(ispj)
	return ispj, err
}

func (self *SInstanceSnapshot) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.Status == api.INSTANCE_SNAPSHOT_START_DELETE || self.Status == api.INSTANCE_SNAPSHOT_RESET {
		return httperrors.NewForbiddenError("can't delete instance snapshot with wrong status")
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
	self.SetStatus(ctx, userCred, api.INSTANCE_SNAPSHOT_START_DELETE, "InstanceSnapshotDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (self *SInstanceSnapshot) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	snapshots, err := self.GetSnapshots()
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		snapshotId := snapshots[i].Id
		isjp := new(SInstanceSnapshotJoint)
		err = InstanceSnapshotJointManager.Query().
			Equals("instance_snapshot_id", self.Id).Equals("snapshot_id", snapshotId).First(isjp)
		err = isjp.Delete(ctx, userCred)
		if err != nil {
			return nil, errors.Wrapf(err, "delete instance snapshot joint: %s", snapshotId)
		}
		_, err = snapshots[i].PerformPurge(ctx, userCred, query, data)
		if err != nil {
			return nil, errors.Wrapf(err, "delete snapshot: %s", snapshotId)
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
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

func (is *SInstanceSnapshot) syncRemoveCloudInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, is)
	defer lockman.ReleaseObject(ctx, is)

	err := is.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		err = is.SetStatus(ctx, userCred, api.INSTANCE_SNAPSHOT_UNKNOWN, "sync to delete")
	} else {
		err = is.RealDelete(ctx, userCred)
	}
	return err
}

func (is *SInstanceSnapshot) SyncWithCloudInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInstanceSnapshot, guest *SGuest) error {
	diff, err := db.UpdateWithLock(ctx, is, func() error {
		is.Status = ext.GetStatus()
		InstanceSnapshotManager.fillInstanceSnapshot(ctx, userCred, guest, is)
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(is, diff, userCred)
	return nil
}

func (manager *SInstanceSnapshotManager) newFromCloudInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential, extSnapshot cloudprovider.ICloudInstanceSnapshot, guest *SGuest) (*SInstanceSnapshot, error) {
	instanceSnapshot := SInstanceSnapshot{}
	instanceSnapshot.SetModelManager(manager, &instanceSnapshot)

	instanceSnapshot.ExternalId = extSnapshot.GetGlobalId()
	instanceSnapshot.Status = extSnapshot.GetStatus()
	manager.fillInstanceSnapshot(ctx, userCred, guest, &instanceSnapshot)
	var err = func() error {
		lockman.LockClass(ctx, manager, "name")
		defer lockman.ReleaseClass(ctx, manager, "name")

		newName, err := db.GenerateName(ctx, manager, nil, extSnapshot.GetName())
		if err == nil {
			instanceSnapshot.Name = extSnapshot.GetName()
		} else {
			instanceSnapshot.Name = newName
		}
		return manager.TableSpec().Insert(ctx, &instanceSnapshot)
	}()
	if err != nil {
		return nil, err
	}
	db.OpsLog.LogEvent(&instanceSnapshot, db.ACT_CREATE, instanceSnapshot.GetShortDesc(ctx), userCred)
	return &instanceSnapshot, nil
}

func (self *SInstanceSnapshot) GetRegionDriver() IRegionDriver {
	provider := self.GetProviderName()
	return GetRegionDriver(provider)
}

func (ism *SInstanceSnapshotManager) InitializeData() error {
	q := ism.Query().IsNullOrEmpty("cloudregion_id")
	var isps []SInstanceSnapshot
	err := db.FetchModelObjects(ism, q, &isps)
	if err != nil {
		return errors.Wrap(err, "unable to FetchModelObjects")
	}
	var (
		cloudregionId string
	)
	for i := range isps {
		isp := &isps[i]
		guest, err := isp.GetGuest()
		if errors.Cause(err) == errors.ErrNotFound {
			cloudregionId = api.DEFAULT_REGION_ID
		}
		if err != nil {
			return errors.Wrapf(err, "unable to GetGuest for isp %q", isp.GetId())
		} else {
			host, _ := guest.GetHost()
			zone, _ := host.GetZone()
			cloudregionId = zone.CloudregionId
		}
		_, err = db.Update(isp, func() error {
			isp.CloudregionId = cloudregionId
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "unable to Update db")
		}
	}
	return nil
}

func (isp *SInstanceSnapshot) GetInstanceSnapshotJointsByOrder(guest *SGuest) ([]*SInstanceSnapshotJoint, error) {
	disks, err := guest.GetGuestDisks()
	if err != nil {
		return nil, errors.Wrap(err, "GetGuestDisks")
	}
	ss, err := isp.GetSnapshots()
	if err != nil {
		return nil, errors.Wrapf(err, "Get %s subsnapshots", isp.GetName())
	}
	jIsps := make([]*SInstanceSnapshotJoint, 0)
	for idx, gd := range disks {
		d := gd.GetDisk()
		if d == nil {
			return nil, errors.Wrapf(err, "Not get guestdisk %d related disk", idx)
		}
		if idx >= len(ss) {
			break
		}
		jIsp, err := isp.GetInstanceSnapshotJointAt(idx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetInstanceSnapshotJointAt %d", idx)
		}
		sd, err := ss[idx].GetDisk()
		if err != nil {
			return nil, errors.Wrapf(err, "Get snapshot %d disk", idx)
		}
		if ss[idx].GetId() != jIsp.SnapshotId {
			return nil, errors.Wrapf(err, "InstanceSnapshotJoint %d snapshot_id %q != %q", idx, jIsp.SnapshotId, ss[idx].GetId())
		}
		if sd.GetId() != d.GetId() {
			return nil, errors.Wrapf(err, "Disk Snapshot %d's disk id %q != current disk %q", idx, sd.GetId(), d.GetId())
		}
		jIsps = append(jIsps, jIsp)
	}
	return jIsps, nil
}

func (self *SInstanceSnapshot) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	// use disk's ownerId instead of default ownerId
	guestObj, err := GuestManager.FetchById(self.GuestId)
	if err != nil {
		return errors.Wrap(err, "GuestManager.FetchById")
	}
	ownerId = guestObj.(*SGuest).GetOwnerId()
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (manager *SInstanceSnapshotManager) GetNeedAutoSnapshotServers() ([]SSnapshotPolicyResource, error) {
	tz, _ := time.LoadLocation(options.Options.TimeZone)
	t := time.Now().In(tz)
	week := t.Weekday()
	if week == 0 { // sunday is zero
		week += 7
	}
	timePoint := t.Hour()

	policy := SnapshotPolicyManager.Query().Equals("type", api.SNAPSHOT_POLICY_TYPE_SERVER).Equals("cloudregion_id", api.DEFAULT_REGION_ID)
	policy = policy.Filter(sqlchemy.Contains(policy.Field("repeat_weekdays"), fmt.Sprintf("%d", week)))
	sq := policy.Filter(
		sqlchemy.OR(
			sqlchemy.Contains(policy.Field("time_points"), fmt.Sprintf(",%d,", timePoint)),
			sqlchemy.Startswith(policy.Field("time_points"), fmt.Sprintf("[%d,", timePoint)),
			sqlchemy.Endswith(policy.Field("time_points"), fmt.Sprintf(",%d]", timePoint)),
			sqlchemy.Equals(policy.Field("time_points"), fmt.Sprintf("[%d]", timePoint)),
		),
	).SubQuery()
	servers := GuestManager.Query().SubQuery()
	q := SnapshotPolicyResourceManager.Query().Equals("resource_type", api.SNAPSHOT_POLICY_TYPE_SERVER)
	q = q.Join(sq, sqlchemy.Equals(q.Field("snapshotpolicy_id"), sq.Field("id")))
	q = q.Join(servers, sqlchemy.Equals(q.Field("resource_id"), servers.Field("id")))
	ret := []SSnapshotPolicyResource{}
	err := db.FetchModelObjects(SnapshotPolicyResourceManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (manager *SInstanceSnapshotManager) AutoServerSnapshot(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	servers, err := manager.GetNeedAutoSnapshotServers()
	if err != nil {
		log.Errorf("Get auto snapshot servers id failed: %s", err)
		return
	}
	log.Infof("auto snapshot %d servers", len(servers))

	serverMap := map[string]*SGuest{}
	for i := range servers {
		server, err := servers[i].GetServer()
		if err != nil {
			log.Errorf("get server error: %v", err)
			continue
		}
		serverMap[server.Id] = server
	}
	for i := range serverMap {
		input := api.ServerInstanceSnapshot{}
		input.GenerateName = fmt.Sprintf("auto-%s-%d", serverMap[i].Name, time.Now().Unix())
		serverMap[i].PerformInstanceSnapshot(ctx, userCred, jsonutils.NewDict(), input)
	}
}

var instanceSnapshotCleanupTaskRunning int32 = 0

func (manager *SInstanceSnapshotManager) CleanupInstanceSnapshots(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if instanceSnapshotCleanupTaskRunning > 0 {
		log.Errorf("Previous CleanupInstanceSnapshots tasks still running !!!")
		return
	}
	instanceSnapshotCleanupTaskRunning = 1
	defer func() {
		instanceSnapshotCleanupTaskRunning = 0
	}()
	sq := manager.Query().Equals("status", api.INSTANCE_SNAPSHOT_READY).Startswith("name", "auto-").SubQuery()

	iss := []struct {
		GuestCnt int
		GuestId  string
	}{}
	q := sq.Query(
		sqlchemy.COUNT("guest_cnt", sq.Field("guest_id")),
		sq.Field("guest_id"),
	).GroupBy(sq.Field("guest_id"))
	err := q.All(&iss)
	if err != nil {
		log.Errorf("Cleanup instance snapshots job fetch instance snapshot failed %s", err)
		return
	}

	guestCount := map[string]int{}
	for i := range iss {
		guestCount[iss[i].GuestId] = iss[i].GuestCnt
	}

	// cleanup retention count instance snapshots
	{
		sq = SnapshotPolicyManager.Query().Equals("type", api.SNAPSHOT_POLICY_TYPE_SERVER).GT("retention_count", 0).SubQuery()
		spr := SnapshotPolicyResourceManager.Query().Equals("resource_type", api.SNAPSHOT_POLICY_TYPE_SERVER).SubQuery()
		q = sq.Query(
			sq.Field("retention_count"),
			spr.Field("resource_id").Label("guest_id"),
		)
		q = q.Join(spr, sqlchemy.Equals(q.Field("id"), spr.Field("snapshotpolicy_id")))

		guestRetentions := []struct {
			GuestId        string
			RetentionCount int
		}{}
		err = q.All(&guestRetentions)
		if err != nil {
			log.Errorf("Cleanup instance snapshots job fetch guest retentions failed %s", err)
			return
		}
		guestRetentionMap := map[string]int{}
		for i := range guestRetentions {
			if _, ok := guestRetentionMap[guestRetentions[i].GuestId]; !ok {
				guestRetentionMap[guestRetentions[i].GuestId] = guestRetentions[i].RetentionCount
			}
			// 取最小保留个数
			if guestRetentionMap[guestRetentions[i].GuestId] > guestRetentions[i].RetentionCount {
				guestRetentionMap[guestRetentions[i].GuestId] = guestRetentions[i].RetentionCount
			}
		}

		for guestId, retentionCnt := range guestRetentionMap {
			if cnt, ok := guestCount[guestId]; ok && cnt > retentionCnt {
				manager.startCleanupRetentionCount(ctx, userCred, guestId, cnt-retentionCnt)
			}
		}
	}

	// cleanup retention days instance snapshots
	{
		sq = SnapshotPolicyManager.Query().Equals("type", api.SNAPSHOT_POLICY_TYPE_SERVER).GT("retention_days", 0).SubQuery()
		spr := SnapshotPolicyResourceManager.Query().Equals("resource_type", api.SNAPSHOT_POLICY_TYPE_SERVER).SubQuery()
		q = sq.Query(
			sq.Field("retention_days"),
			spr.Field("resource_id").Label("guest_id"),
		)
		q = q.Join(spr, sqlchemy.Equals(q.Field("id"), spr.Field("snapshotpolicy_id")))

		guestRetentions := []struct {
			GuestId       string
			RetentionDays int
		}{}
		err = q.All(&guestRetentions)
		if err != nil {
			log.Errorf("Cleanup instance snapshots job fetch guest retentions failed %s", err)
			return
		}
		guestRetentionMap := map[string]int{}
		for i := range guestRetentions {
			if _, ok := guestRetentionMap[guestRetentions[i].GuestId]; !ok {
				guestRetentionMap[guestRetentions[i].GuestId] = guestRetentions[i].RetentionDays
			}
			// 取最小保留天数
			if guestRetentionMap[guestRetentions[i].GuestId] > guestRetentions[i].RetentionDays {
				guestRetentionMap[guestRetentions[i].GuestId] = guestRetentions[i].RetentionDays
			}
		}
		for guestId, retentionDays := range guestRetentionMap {
			manager.startCleanupRetentionDays(ctx, userCred, guestId, retentionDays)
		}
	}
}

func (manager *SInstanceSnapshotManager) startCleanupRetentionCount(ctx context.Context, userCred mcclient.TokenCredential, guestId string, cnt int) error {
	q := manager.Query().Equals("guest_id", guestId).Equals("status", api.INSTANCE_SNAPSHOT_READY).Startswith("name", "auto-").Asc("created_at").Limit(cnt)
	vms := []SInstanceSnapshot{}
	err := db.FetchModelObjects(manager, q, &vms)
	if err != nil {
		return errors.Wrapf(err, "FetchModelObjects")
	}
	for i := range vms {
		vms[i].StartInstanceSnapshotDeleteTask(ctx, userCred, "")
	}
	return nil
}

func (manager *SInstanceSnapshotManager) startCleanupRetentionDays(ctx context.Context, userCred mcclient.TokenCredential, guestId string, day int) error {
	expiredTime := time.Now().AddDate(0, 0, -day)
	q := manager.Query().Equals("guest_id", guestId).Equals("status", api.INSTANCE_SNAPSHOT_READY).Startswith("name", "auto-").LE("created_at", expiredTime)
	vms := []SInstanceSnapshot{}
	err := db.FetchModelObjects(manager, q, &vms)
	if err != nil {
		return errors.Wrapf(err, "FetchModelObjects")
	}
	for i := range vms {
		vms[i].StartInstanceSnapshotDeleteTask(ctx, userCred, "")
	}
	return nil
}
