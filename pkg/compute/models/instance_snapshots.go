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
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	SizeMb int `nullable:"false"`
}

type SInstanceSnapshotManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	db.SMultiArchResourceBaseManager
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

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
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

func (self *SInstanceSnapshot) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
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
				Status:        snapshots[i].Status,
				StorageType:   snapshots[i].GetStorageType(),
			})
			out.Size += snapshots[i].Size

			if len(snapshots[i].StorageId) > 0 && out.StorageType == "" {
				out.StorageType = snapshots[i].GetStorageType()
			}
		}
	} else if guest != nil {
		out.Size = self.SizeMb
		disk, err := guest.GetSystemDisk()
		if err != nil {
			log.Errorf("unable to GetSystemDisk of guest %q", guest.GetId())
		} else {
			s := disk.GetStorage()
			if s != nil {
				out.StorageType = s.StorageType
			}
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
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.InstanceSnapshotDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    manRows[i],
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

func (manager *SInstanceSnapshotManager) fillInstanceSnapshot(userCred mcclient.TokenCredential, guest *SGuest, instanceSnapshot *SInstanceSnapshot) {
	instanceSnapshot.SetModelManager(manager, instanceSnapshot)
	instanceSnapshot.ProjectId = guest.ProjectId
	instanceSnapshot.DomainId = guest.DomainId
	instanceSnapshot.GuestId = guest.Id
	guestSchedInput := guest.ToSchedDesc()

	host := guest.GetHost()
	instanceSnapshot.ManagerId = host.ManagerId
	zone := host.GetZone()
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
}

func (manager *SInstanceSnapshotManager) CreateInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, name string, autoDelete bool) (*SInstanceSnapshot, error) {
	instanceSnapshot := &SInstanceSnapshot{}
	instanceSnapshot.SetModelManager(manager, instanceSnapshot)
	instanceSnapshot.Name = name
	instanceSnapshot.AutoDelete = autoDelete
	manager.fillInstanceSnapshot(userCred, guest, instanceSnapshot)
	// compute size of instanceSnapshot
	instanceSnapshot.SizeMb = guest.getDiskSize()
	err := manager.TableSpec().Insert(ctx, instanceSnapshot)
	if err != nil {
		return nil, err
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
	sourceInput.InstanceType = self.InstanceType
	if len(sourceInput.Networks) == 0 {
		sourceInput.Networks = serverConfig.Networks
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
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		self.GetOwnerId(),
		self.GetRegion(),
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

func TotalInstanceSnapshotCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) (int, error) {
	q := InstanceSnapshotManager.Query()

	switch scope {
	case rbacutils.ScopeSystem:
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}

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

func (self *SInstanceSnapshot) ValidateDeleteCondition(ctx context.Context) error {
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

func (is *SInstanceSnapshot) syncRemoveCloudInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, is)
	defer lockman.ReleaseObject(ctx, is)

	err := is.ValidateDeleteCondition(ctx)
	if err != nil {
		err = is.SetStatus(userCred, api.INSTANCE_SNAPSHOT_UNKNOWN, "sync to delete")
	} else {
		err = is.RealDelete(ctx, userCred)
	}
	return err
}

func (is *SInstanceSnapshot) SyncWithCloudInstanceSnapshot(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInstanceSnapshot, guest *SGuest) error {
	diff, err := db.UpdateWithLock(ctx, is, func() error {
		is.Status = ext.GetStatus()
		InstanceSnapshotManager.fillInstanceSnapshot(userCred, guest, is)
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
	manager.fillInstanceSnapshot(userCred, guest, &instanceSnapshot)
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
			host := guest.GetHost()
			zone := host.GetZone()
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
