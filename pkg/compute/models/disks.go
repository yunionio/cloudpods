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
	"path"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDiskManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SStorageResourceBaseManager
	SBillingResourceBaseManager
	db.SMultiArchResourceBaseManager
	db.SAutoDeleteResourceBaseManager
}

var DiskManager *SDiskManager

func init() {
	DiskManager = &SDiskManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDisk{},
			"disks_tbl",
			"disk",
			"disks",
		),
	}
	DiskManager.SetVirtualObject(DiskManager)
}

type SDisk struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SBillingResourceBase
	SStorageResourceBase `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	db.SMultiArchResourceBase
	db.SAutoDeleteResourceBase

	// 磁盘存储类型
	// example: qcow2
	DiskFormat string `width:"32" charset:"ascii" nullable:"false" default:"qcow2" list:"user" json:"disk_format"`
	// 磁盘大小, 单位Mb
	// example: 10240
	DiskSize int `nullable:"false" list:"user" json:"disk_size"`
	// 磁盘路径
	AccessPath string `width:"256" charset:"ascii" nullable:"true" get:"user" json:"access_path"`

	// 存储Id
	// StorageId       string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"optional"`

	// 备份磁盘实例的存储ID
	BackupStorageId string `width:"128" charset:"ascii" nullable:"true" list:"admin" json:"backup_storage_id"`

	// 镜像Id
	TemplateId string `width:"256" charset:"ascii" nullable:"true" list:"user" json:"template_id"`
	// 快照Id
	SnapshotId string `width:"256" charset:"ascii" nullable:"true" list:"user" json:"snapshot_id"`

	// 文件系统
	FsFormat string `width:"32" charset:"ascii" nullable:"true" list:"user" json:"fs_format"`

	// 磁盘类型
	// sys: 系统盘
	// data: 数据盘
	// swap: 交换盘
	// example: sys
	DiskType string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"admin" json:"disk_type"`

	// # is persistent
	Nonpersistent bool `default:"false" list:"user" json:"nonpersistent"`

	// 是否标记为SSD磁盘
	IsSsd bool `nullable:"false" default:"false" list:"user" update:"user" create:"optional"`
}

func (manager *SDiskManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{StorageManager},
	}
}

func (manager *SDiskManager) FetchDiskById(diskId string) *SDisk {
	disk, err := manager.FetchById(diskId)
	if err != nil {
		log.Errorf("FetchById fail %s", err)
		return nil
	}
	return disk.(*SDisk)
}

// 磁盘列表
func (manager *SDiskManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DiskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStorageResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SBillingResourceBaseManager.ListItemFilter(ctx, q, userCred, query.BillingResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SBillingResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, query.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMultiArchResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SAutoDeleteResourceBaseManager.ListItemFilter(ctx, q, userCred, query.AutoDeleteResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SAutoDeleteResourceBaseManager.ListItemFilter")
	}

	if query.Unused != nil {
		guestdisks := GuestdiskManager.Query().SubQuery()
		sq := guestdisks.Query(guestdisks.Field("disk_id"))
		if *query.Unused {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		}
	}

	guestId := query.ServerId
	if len(guestId) > 0 {
		iGuest, err := GuestManager.FetchByIdOrName(userCred, guestId)
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("guest %q not found", guestId)
		} else if err != nil {
			return nil, err
		}
		guest := iGuest.(*SGuest)
		guestDisks := GuestdiskManager.Query().SubQuery()
		q = q.Join(guestDisks, sqlchemy.AND(
			sqlchemy.Equals(guestDisks.Field("disk_id"), q.Field("id")),
			sqlchemy.Equals(guestDisks.Field("guest_id"), guest.Id),
		))
	}

	if diskType := query.DiskType; diskType != "" {
		q = q.Filter(sqlchemy.Equals(q.Field("disk_type"), diskType))
	}

	// for snapshotpolicy_id
	snapshotpolicyStr := query.SnapshotpolicyId
	if len(snapshotpolicyStr) > 0 {
		snapshotpolicyObj, err := SnapshotPolicyManager.FetchByIdOrName(userCred, snapshotpolicyStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("snapshotpolicy %s not found: %s", snapshotpolicyStr, err)
		}
		snapshotpolicyId := snapshotpolicyObj.GetId()
		sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", snapshotpolicyId)
		q = q.In("id", sq)
	}

	if len(query.DiskFormat) > 0 {
		q = q.Equals("disk_format", query.DiskFormat)
	}

	if query.DiskSize > 0 {
		q = q.Equals("disk_size", query.DiskSize)
	}

	if len(query.FsFormat) > 0 {
		q = q.Equals("fs_format", query.FsFormat)
	}

	if len(query.ImageId) > 0 {
		img, err := CachedimageManager.getImageInfo(ctx, userCred, query.ImageId, false)
		if err != nil {
			return nil, errors.Wrap(err, "CachedimageManager.getImageInfo")
		}
		q = q.Equals("template_id", img.Id)
	}

	if len(query.SnapshotId) > 0 {
		snapObj, err := SnapshotManager.FetchByIdOrName(userCred, query.SnapshotId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(SnapshotManager.Keyword(), query.SnapshotId)
			} else {
				return nil, errors.Wrap(err, "SnapshotManager.FetchByIdOrName")
			}
		}
		q = q.Equals("snapshot_id", snapObj.GetId())
	}

	return q, nil
}

func (manager *SDiskManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DiskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStorageResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SBillingResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.BillingResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SBillingResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SDiskManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SStorageResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SDisk) GetGuestDiskCount() (int, error) {
	guestdisks := GuestdiskManager.Query()
	return guestdisks.Equals("disk_id", self.Id).CountWithError()
}

func (self *SDisk) isAttached() (bool, error) {
	cnt, err := self.GetGuestDiskCount()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (self *SDisk) GetGuestdisks() []SGuestdisk {
	guestdisks := make([]SGuestdisk, 0)
	q := GuestdiskManager.Query().Equals("disk_id", self.Id)
	err := db.FetchModelObjects(GuestdiskManager, q, &guestdisks)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return guestdisks
}

func (self *SDisk) GetGuests() []SGuest {
	result := make([]SGuest, 0)
	query := GuestManager.Query()
	guestdisks := GuestdiskManager.Query().SubQuery()
	q := query.Join(guestdisks, sqlchemy.AND(
		sqlchemy.Equals(guestdisks.Field("guest_id"), query.Field("id")))).
		Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id))
	// q.DebugQuery()
	err := db.FetchModelObjects(GuestManager, q, &result)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	return result
}

func (self *SDisk) GetGuest() *SGuest {
	guests := self.GetGuests()
	if len(guests) > 0 {
		return &guests[0]
	}
	return nil
}

func (self *SDisk) GetGuestsCount() (int, error) {
	guests := GuestManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()
	return guests.Query().Join(guestdisks, sqlchemy.AND(
		sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")))).
		Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id)).CountWithError()
}

func (self *SDisk) GetRuningGuestCount() (int, error) {
	guests := GuestManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()
	return guests.Query().Join(guestdisks, sqlchemy.AND(
		sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")))).
		Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id)).
		Filter(sqlchemy.Equals(guests.Field("status"), api.VM_RUNNING)).CountWithError()
}

func (self *SDisk) getSnapshotpoliciesCount() (int, error) {
	q := SnapshotPolicyDiskManager.Query().Equals("disk_id", self.Id)
	return q.CountWithError()
}

func (self *SDisk) DetachAllSnapshotpolicies(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := SnapshotPolicyDiskManager.SyncDetachByDisk(ctx, userCred, nil, self)
	if err != nil {
		return errors.Wrap(err, "detach after delete failed")
	}
	return nil
}

func (self *SDisk) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	input := new(api.DiskCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}
	self.fetchDiskInfo(input.DiskConfig)
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SDisk) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskUpdateInput) (api.DiskUpdateInput, error) {
	var err error

	if input.DiskType != "" {
		if !utils.IsInStringArray(input.DiskType, []string{api.DISK_TYPE_DATA, api.DISK_TYPE_VOLUME}) {
			return input, httperrors.NewInputParameterError("not support update disk_type %s", input.DiskType)
		}
	}

	storage := self.GetStorage()
	if storage == nil {
		return input, httperrors.NewNotFoundError("failed to find storage for disk %s", self.Name)
	}

	host := storage.GetMasterHost()
	if host == nil {
		return input, httperrors.NewNotFoundError("failed to find host for storage %s with disk %s", storage.Name, self.Name)
	}

	input, err = host.GetHostDriver().ValidateUpdateDisk(ctx, userCred, input)
	if err != nil {
		return input, errors.Wrap(err, "GetHostDriver().ValidateUpdateDisk")
	}

	input.VirtualResourceBaseUpdateInput, err = self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func diskCreateInput2ComputeQuotaKeys(input api.DiskCreateInput, ownerId mcclient.IIdentityProvider) SComputeResourceKeys {
	// input.Hypervisor must be set
	brand := guessBrandForHypervisor(input.Hypervisor)
	keys := GetDriver(input.Hypervisor).GetComputeQuotaKeys(
		rbacutils.ScopeProject,
		ownerId,
		brand,
	)
	if len(input.PreferHost) > 0 {
		hostObj, _ := HostManager.FetchById(input.PreferHost)
		host := hostObj.(*SHost)
		zone := host.GetZone()
		keys.ZoneId = zone.Id
		keys.RegionId = zone.CloudregionId
	} else if len(input.PreferZone) > 0 {
		zoneObj, _ := ZoneManager.FetchById(input.PreferZone)
		zone := zoneObj.(*SZone)
		keys.ZoneId = zone.Id
		keys.RegionId = zone.CloudregionId
	} else if len(input.PreferWire) > 0 {
		wireObj, _ := WireManager.FetchById(input.PreferWire)
		wire := wireObj.(*SWire)
		zone := wire.GetZone()
		keys.ZoneId = zone.Id
		keys.RegionId = zone.CloudregionId
	} else if len(input.PreferRegion) > 0 {
		regionObj, _ := CloudregionManager.FetchById(input.PreferRegion)
		keys.RegionId = regionObj.GetId()
	}
	return keys
}

func (manager *SDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DiskCreateInput) (api.DiskCreateInput, error) {
	diskConfig := input.DiskConfig
	diskConfig, err := parseDiskInfo(ctx, userCred, diskConfig)
	if err != nil {
		return input, err
	}
	input.ProjectId = ownerId.GetProjectId()
	input.ProjectDomainId = ownerId.GetProjectDomainId()

	var quotaKey quotas.IQuotaKeys

	storageID := input.Storage
	if storageID != "" {
		storageObj, err := StorageManager.FetchByIdOrName(nil, storageID)
		if err != nil {
			return input, httperrors.NewResourceNotFoundError("Storage %s not found", storageID)
		}
		storage := storageObj.(*SStorage)

		provider := storage.GetCloudprovider()
		if provider != nil && !provider.IsAvailable() {
			return input, httperrors.NewResourceNotReadyError("cloudprovider %s not available", provider.Name)
		}

		host := storage.GetMasterHost()
		if host == nil {
			return input, httperrors.NewResourceNotFoundError("storage %s(%s) need online and attach host for create disk", storage.Name, storage.Id)
		}
		input.Hypervisor = host.GetHostDriver().GetHypervisor()
		if len(diskConfig.Backend) == 0 {
			diskConfig.Backend = storage.StorageType
		}
		err = manager.validateDiskOnStorage(diskConfig, storage)
		if err != nil {
			return input, err
		}
		input.Storage = storage.Id

		quotaKey = fetchComputeQuotaKeys(
			rbacutils.ScopeProject,
			ownerId,
			storage.getZone(),
			provider,
			input.Hypervisor,
		)
	} else {
		if len(diskConfig.Backend) == 0 {
			diskConfig.Backend = api.STORAGE_LOCAL
		}
		if len(input.PreferManager) > 0 {
			_manager, err := CloudproviderManager.FetchByIdOrName(userCred, input.PreferManager)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return input, httperrors.NewResourceNotFoundError2("cloudprovider", input.PreferManager)
				}
				return input, httperrors.NewGeneralError(err)
			}
			manager := _manager.(*SCloudprovider)
			if !manager.IsAvailable() {
				return input, httperrors.NewResourceNotReadyError("cloudprovider %s not available", manager.Name)
			}
			input.PreferManager = manager.Id
		}
		serverInput, err := ValidateScheduleCreateData(ctx, userCred, input.ToServerCreateInput(), input.Hypervisor)
		if err != nil {
			return input, err
		}
		input = *serverInput.ToDiskCreateInput()
		quotaKey = diskCreateInput2ComputeQuotaKeys(input, ownerId)
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	pendingUsage := SQuota{Storage: diskConfig.SizeMb}
	pendingUsage.SetKeys(quotaKey)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return input, httperrors.NewOutOfQuotaError("%s", err)
	}
	return input, nil
}

func (manager *SDiskManager) validateDiskOnStorage(diskConfig *api.DiskConfig, storage *SStorage) error {
	if storage.Enabled.IsFalse() {
		return httperrors.NewInputParameterError("Cannot create disk with disabled storage[%s]", storage.Name)
	}
	if !utils.IsInStringArray(storage.Status, []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}) {
		return httperrors.NewInputParameterError("Cannot create disk with offline storage[%s]", storage.Name)
	}
	if storage.StorageType != diskConfig.Backend {
		return httperrors.NewInputParameterError("Storage type[%s] not match backend %s", storage.StorageType, diskConfig.Backend)
	}
	if host := storage.GetMasterHost(); host != nil {
		//公有云磁盘大小检查。
		if err := host.GetHostDriver().ValidateDiskSize(storage, diskConfig.SizeMb>>10); err != nil {
			return httperrors.NewInputParameterError("%v", err)
		}
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	hoststorage := make([]SHoststorage, 0)
	if err := hoststorages.Query().Equals("storage_id", storage.Id).All(&hoststorage); err != nil {
		return err
	}
	if len(hoststorage) == 0 {
		return httperrors.NewInputParameterError("Storage[%s] must attach to a host", storage.Name)
	}
	if int64(diskConfig.SizeMb) > storage.GetFreeCapacity() && !storage.IsEmulated {
		return httperrors.NewInputParameterError("Not enough free space")
	}
	return nil
}

func (disk *SDisk) SetStorage(storageId string, diskConfig *api.DiskConfig) error {
	backend := diskConfig.Backend
	if backend == "" {
		return fmt.Errorf("Backend is empty")
	}
	storage := StorageManager.FetchStorageById(storageId)
	if storage == nil {
		return fmt.Errorf("Not found backend %s storage %s", backend, storageId)
	}
	err := DiskManager.validateDiskOnStorage(diskConfig, storage)
	if err != nil {
		return err
	}
	_, err = db.Update(disk, func() error {
		disk.StorageId = storage.Id
		return nil
	})
	return err
}

func (disk *SDisk) SetStorageByHost(hostId string, diskConfig *api.DiskConfig, storageIds []string) error {
	host := HostManager.FetchHostById(hostId)
	backend := diskConfig.Backend
	if backend == "" {
		return fmt.Errorf("Backend is empty")
	}
	var storage *SStorage
	if len(storageIds) != 0 {
		storage = StorageManager.FetchStorageById(storageIds[0])
	} else if utils.IsInStringArray(backend, api.STORAGE_LIMITED_TYPES) {
		storage = host.GetLeastUsedStorage(backend)
	} else {
		// unlimited pulic cloud storages
		storages := host.GetAttachedEnabledHostStorages(nil)
		for _, s := range storages {
			if s.StorageType == backend {
				tmpS := s
				storage = &tmpS
			}
		}
	}
	if storage == nil {
		return fmt.Errorf("Not found host %s backend %s storage", host.Name, backend)
	}
	err := DiskManager.validateDiskOnStorage(diskConfig, storage)
	if err != nil {
		return err
	}
	_, err = db.Update(disk, func() error {
		disk.StorageId = storage.Id
		return nil
	})
	return err
}

func getDiskResourceRequirements(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DiskCreateInput, count int) SQuota {
	req := SQuota{
		Storage: input.SizeMb * count,
	}
	var quotaKey SComputeResourceKeys
	if len(input.Storage) > 0 {
		storageObj, _ := StorageManager.FetchById(input.Storage)
		storage := storageObj.(*SStorage)
		quotaKey = fetchComputeQuotaKeys(
			rbacutils.ScopeProject,
			ownerId,
			storage.getZone(),
			storage.GetCloudprovider(),
			input.Hypervisor,
		)
	} else {
		quotaKey = diskCreateInput2ComputeQuotaKeys(input, ownerId)
	}
	req.SetKeys(quotaKey)
	return req
}

/*func (manager *SDiskManager) convertToBatchCreateData(data jsonutils.JSONObject) *jsonutils.JSONDict {
	diskConfig, _ := data.Get("disk")
	newData := data.(*jsonutils.JSONDict).CopyExcludes("disk")
	newData.Add(diskConfig, "disk.0")
	return newData
}*/

func (manager *SDiskManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.DiskCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("!!!data.Unmarshal api.DiskCreateInput fail %s", err)
	}
	pendingUsage := getDiskResourceRequirements(ctx, userCred, ownerId, input, len(items))
	RunBatchCreateTask(ctx, items, userCred, data, pendingUsage, SRegionQuota{}, "DiskBatchCreateTask", "")
}

func (self *SDisk) StartDiskCreateTask(ctx context.Context, userCred mcclient.TokenCredential, rebuild bool, snapshot string, parentTaskId string) error {
	kwargs := jsonutils.NewDict()
	if rebuild {
		kwargs.Add(jsonutils.JSONTrue, "rebuild")
	}
	if len(snapshot) > 0 {
		kwargs.Add(jsonutils.NewString(snapshot), "snapshot")
	}

	taskName := "DiskCreateTask"
	if self.BackupStorageId != "" {
		taskName = "HADiskCreateTask"
	}
	if task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, kwargs, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) GetSnapshotCount() (int, error) {
	q := SnapshotManager.Query()
	return q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), self.Id),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).CountWithError()
}

func (self *SDisk) GetManualSnapshotCount() (int, error) {
	return SnapshotManager.Query().
		Equals("disk_id", self.Id).Equals("fake_deleted", false).
		Equals("created_by", api.SNAPSHOT_MANUAL).CountWithError()
}

func (self *SDisk) StartAllocate(ctx context.Context, host *SHost, storage *SStorage, taskId string, userCred mcclient.TokenCredential, rebuild bool, snapshot string, task taskman.ITask) error {
	log.Infof("Allocating disk on host %s ...", host.GetName())

	templateId := self.GetTemplateId()
	fsFormat := self.GetFsFormat()

	content := jsonutils.NewDict()
	content.Add(jsonutils.NewString(self.DiskFormat), "format")
	content.Add(jsonutils.NewInt(int64(self.DiskSize)), "size")
	if len(snapshot) > 0 {
		content.Add(jsonutils.NewString(snapshot), "snapshot")
		if utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
			SnapshotManager.AddRefCount(self.SnapshotId, 1)
			self.SetMetadata(ctx, "merge_snapshot", jsonutils.JSONTrue, userCred)
		}
	} else if len(templateId) > 0 {
		content.Add(jsonutils.NewString(templateId), "image_id")
	}
	if len(fsFormat) > 0 {
		content.Add(jsonutils.NewString(fsFormat), "fs_format")
		if fsFormat == "ext4" {
			name := strings.ToLower(self.GetName())
			for _, key := range []string{"encrypt", "secret", "cipher", "private"} {
				if strings.Index(key, name) > 0 {
					content.Add(jsonutils.JSONTrue, "encryption")
					break
				}
			}
		}
	}
	if rebuild {
		return host.GetHostDriver().RequestRebuildDiskOnStorage(ctx, host, storage, self, task, content)
	} else {
		return host.GetHostDriver().RequestAllocateDiskOnStorage(ctx, userCred, host, storage, self, task, content)
	}
}

func (self *SDisk) AllowGetDetailsConvertSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "convert-snapshot")
}

func (self *SDisk) GetDetailsConvertSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	needs, err := SnapshotManager.IsDiskSnapshotsNeedConvert(self.Id)
	if err != nil {
		return nil, httperrors.NewInternalServerError("Fetch snapshot count failed %s", err)
	}
	if !needs {
		return nil, httperrors.NewBadRequestError("Disk %s don't need convert snapshots", self.Id)
	}

	deleteSnapshot := SnapshotManager.GetDiskFirstSnapshot(self.Id)
	if deleteSnapshot == nil {
		return nil, httperrors.NewNotFoundError("Can not get disk snapshot")
	}
	convertSnapshot, err := SnapshotManager.GetConvertSnapshot(deleteSnapshot)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Get convert snapshot failed: %s", err.Error())
	}
	if convertSnapshot == nil {
		return nil, httperrors.NewBadRequestError("Snapshot %s dose not have convert snapshot", deleteSnapshot.Id)
	}
	var FakeDelete bool
	if deleteSnapshot.CreatedBy == api.SNAPSHOT_MANUAL && !deleteSnapshot.FakeDeleted {
		FakeDelete = true
	}
	ret := jsonutils.NewDict()
	ret.Set("delete_snapshot", jsonutils.NewString(deleteSnapshot.Id))
	ret.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
	ret.Set("pending_delete", jsonutils.NewBool(FakeDelete))
	return ret, nil
}

// make snapshot after reset out of chain
func (self *SDisk) CleanUpDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, snapshot *SSnapshot) error {
	dest := make([]SSnapshot, 0)
	query := SnapshotManager.Query()
	query.Filter(sqlchemy.Equals(query.Field("disk_id"), self.Id)).
		GT("created_at", snapshot.CreatedAt).Asc("created_at").All(&dest)
	if len(dest) == 0 {
		return nil
	}
	convertSnapshots := jsonutils.NewArray()
	deleteSnapshots := jsonutils.NewArray()
	for i := 0; i < len(dest); i++ {
		if !dest[i].FakeDeleted && !dest[i].OutOfChain {
			convertSnapshots.Add(jsonutils.NewString(dest[i].Id))
		} else if dest[i].FakeDeleted {
			deleteSnapshots.Add(jsonutils.NewString(dest[i].Id))
		}
	}
	params := jsonutils.NewDict()
	params.Set("convert_snapshots", convertSnapshots)
	params.Set("delete_snapshots", deleteSnapshots)
	task, err := taskman.TaskManager.NewTask(ctx, "DiskCleanUpSnapshotsTask", self, userCred, params, "", "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) AllowPerformDiskReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "disk-reset")
}

func (self *SDisk) PerformDiskReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.DISK_READY}) {
		return nil, httperrors.NewInputParameterError("Cannot reset disk in status %s", self.Status)
	}
	storage := self.GetStorage()
	if storage == nil {
		return nil, httperrors.NewNotFoundError("failed to find storage for disk %s", self.Name)
	}

	host := storage.GetMasterHost()
	if host == nil {
		return nil, httperrors.NewNotFoundError("failed to find host for storage %s with disk %s", storage.Name, self.Name)
	}

	snapshotV := validators.NewModelIdOrNameValidator("snapshot", "snapshot", userCred)
	err := snapshotV.Validate(data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	snapshot := snapshotV.Model.(*SSnapshot)
	if snapshot.Status != api.SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Cannot reset disk with snapshot in status %s", snapshot.Status)
	}

	if snapshot.DiskId != self.Id {
		return nil, httperrors.NewBadRequestError("Cannot reset disk %s(%s),Snapshot is belong to disk %s", self.Name, self.Id, snapshot.DiskId)
	}

	guests := self.GetGuests()
	data, err = host.GetHostDriver().ValidateResetDisk(ctx, userCred, self, snapshot, guests, data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}

	autoStart := jsonutils.QueryBoolean(data, "auto_start", false)
	var guest *SGuest = nil
	if len(guests) > 0 {
		guest = &guests[0]
	}
	return nil, self.StartResetDisk(ctx, userCred, snapshot.Id, autoStart, guest, "")
}

func (self *SDisk) StartResetDisk(
	ctx context.Context, userCred mcclient.TokenCredential,
	snapshotId string, autoStart bool, guest *SGuest, parentTaskId string,
) error {
	self.SetStatus(userCred, api.DISK_RESET, "")
	if guest != nil {
		guest.SetStatus(userCred, api.VM_DISK_RESET, "disk reset")
	}
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshotId))
	params.Set("auto_start", jsonutils.NewBool(autoStart))
	task, err := taskman.TaskManager.NewTask(ctx, "DiskResetTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) AllowPerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "resize")
}

func (disk *SDisk) PerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guest := disk.GetGuest()
	err := disk.doResize(ctx, userCred, data, guest)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (disk *SDisk) getHypervisor() string {
	storage := disk.GetStorage()
	if storage != nil {
		host := storage.GetMasterHost()
		if host != nil {
			return host.GetHostDriver().GetHypervisor()
		}
	}
	hypervisor := disk.GetMetadata("hypervisor", nil)
	return hypervisor
}

func (disk *SDisk) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	storage := disk.GetStorage()
	if storage == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid storage")
	}
	provider := storage.GetCloudprovider()
	if provider == nil && len(storage.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	zone := storage.getZone()
	if zone == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid zone")
	}
	return fetchComputeQuotaKeys(
		rbacutils.ScopeProject,
		disk.GetOwnerId(),
		zone,
		provider,
		disk.getHypervisor(),
	), nil
}

func (disk *SDisk) doResize(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, guest *SGuest) error {
	sizeStr, err := data.GetString("size")
	if err != nil {
		return httperrors.NewMissingParameterError("size")
	}
	sizeMb, err := fileutils.GetSizeMb(sizeStr, 'M', 1024)
	if err != nil {
		return err
	}
	if disk.Status != api.DISK_READY {
		return httperrors.NewResourceNotReadyError("Resize disk when disk is READY")
	}
	if sizeMb < disk.DiskSize {
		return httperrors.NewUnsupportOperationError("Disk cannot be thrink")
	}
	if sizeMb == disk.DiskSize {
		return nil
	}
	addDisk := sizeMb - disk.DiskSize
	storage := disk.GetStorage()
	if storage == nil {
		return httperrors.NewInternalServerError("disk has no valid storage")
	}
	if host := storage.GetMasterHost(); host != nil {
		if err := host.GetHostDriver().ValidateDiskSize(storage, sizeMb>>10); err != nil {
			return httperrors.NewInputParameterError("%v", err)
		}
	}
	if int64(addDisk) > storage.GetFreeCapacity() && !storage.IsEmulated {
		return httperrors.NewOutOfResourceError("Not enough free space")
	}
	if guest != nil {
		if err := guest.ValidateResizeDisk(disk, storage); err != nil {
			return httperrors.NewInputParameterError("%v", err)
		}
	}
	pendingUsage := SQuota{Storage: int(addDisk)}
	keys, err := disk.GetQuotaKeys()
	if err != nil {
		return httperrors.NewInternalServerError("disk.GetQuotaKeys fail %s", err)
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	if guest != nil {
		return guest.StartGuestDiskResizeTask(ctx, userCred, disk.Id, int64(sizeMb), "", &pendingUsage)
	} else {
		return disk.StartDiskResizeTask(ctx, userCred, int64(sizeMb), "", &pendingUsage)
	}
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	storage := self.GetStorage()
	if storage == nil {
		return nil, httperrors.NewResourceNotFoundError("fail to find storage for disk %s", self.GetName())
	}
	istorage, err := storage.GetIStorage()
	if err != nil {
		return nil, err
	}
	return istorage, nil
}

func (self *SDisk) GetIDisk() (cloudprovider.ICloudDisk, error) {
	iStorage, err := self.GetIStorage()
	if err != nil {
		log.Errorf("fail to find iStorage: %v", err)
		return nil, err
	}
	return iStorage.GetIDiskById(self.GetExternalId())
}

func (self *SDisk) GetZone() *SZone {
	if storage := self.GetStorage(); storage != nil {
		return storage.getZone()
	}
	return nil
}

func (self *SDisk) PrepareSaveImage(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerSaveImageInput) (string, error) {
	zone := self.GetZone()
	if zone == nil {
		return "", httperrors.NewResourceNotFoundError("No zone for this disk")
	}
	if len(input.GenerateName) == 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region, "")
		imageList, err := modules.Images.List(s, jsonutils.Marshal(map[string]string{"name": input.Name, "admin": "true"}))
		if err != nil {
			return "", err
		}
		if imageList.Total > 0 {
			return "", httperrors.NewConflictError("Duplicate image name %s", input.Name)
		}
	}

	opts := struct {
		Name         string
		GenerateName string
		VirtualSize  int
		DiskFormat   string
		Properties   map[string]string
	}{
		Name:         input.Name,
		GenerateName: input.GenerateName,
		VirtualSize:  self.DiskSize,
		DiskFormat:   self.DiskFormat,
		Properties: map[string]string{
			"notes":   input.Notes,
			"os_type": input.OsType,
			"os_arch": input.OsArch,
		},
	}

	/*
		no need to check quota anymore
		session := auth.GetSession(userCred, options.Options.Region, "v2")
		quota := image_models.SQuota{Image: 1}
		if _, err := modules.ImageQuotas.DoQuotaCheck(session, jsonutils.Marshal(&quota)); err != nil {
			return "", err
		}*/
	us := auth.GetSession(ctx, userCred, options.Options.Region, "")
	result, err := modules.Images.Create(us, jsonutils.Marshal(opts))
	if err != nil {
		return "", err
	}
	imageId, err := result.GetString("id")
	if err != nil {
		return "", err
	}
	return imageId, nil
}

func (self *SDisk) AllowPerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "save")
}

func (self *SDisk) PerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskSaveInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DISK_READY {
		return nil, httperrors.NewResourceNotReadyError("Save disk when disk is READY")

	}
	cnt, err := self.GetRuningGuestCount()
	if err != nil {
		return nil, httperrors.NewInternalServerError("GetRuningGuestCount fail %s", err)
	}
	if cnt > 0 {
		return nil, httperrors.NewResourceNotReadyError("Save disk when not being USED")
	}

	if len(input.Name) == 0 {
		return nil, httperrors.NewInputParameterError("Image name is required")
	}
	opts := api.ServerSaveImageInput{
		Name: input.Name,
	}
	input.ImageId, err = self.PrepareSaveImage(ctx, userCred, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "PrepareSaveImage")
	}
	return nil, self.StartDiskSaveTask(ctx, userCred, input, "")
}

func (self *SDisk) StartDiskSaveTask(ctx context.Context, userCred mcclient.TokenCredential, input api.DiskSaveInput, parentTaskId string) error {
	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "DiskSaveTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.DISK_START_SAVE, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDisk) ValidateDeleteCondition(ctx context.Context) error {
	provider := self.GetCloudprovider()
	if provider != nil {
		if !provider.IsAvailable() {
			return httperrors.NewNotSufficientPrivilegeError("cloud provider %s is not available", provider.GetName())
		}

		account := provider.GetCloudaccount()
		if account != nil && !account.IsAvailable() {
			return httperrors.NewNotSufficientPrivilegeError("cloud account %s is not available", account.GetName())
		}
	}

	return self.validateDeleteCondition(ctx, false)
}

func (self *SDisk) ValidatePurgeCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, true)
}

func (self *SDisk) validateDeleteCondition(ctx context.Context, isPurge bool) error {
	if !isPurge {
		storage := self.GetStorage()
		if storage == nil {
			// storage is empty, a dirty data, allow delete
			return nil
		}
		host := storage.GetMasterHost()
		if host == nil {
			return httperrors.NewBadRequestError("storage of disk %s no valid host", self.Id)
		}
	}
	cnt, err := self.GetGuestDiskCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestDiskCount for disk %s fail %s", self.Id, err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Virtual disk %s(%s) used by virtual servers", self.Name, self.Id)
	}
	if !isPurge && self.IsNotDeletablePrePaid() {
		return httperrors.NewForbiddenError("not allow to delete prepaid disk in valid status")
	}
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SDisk) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	provider := self.GetCloudprovider()
	if provider != nil {
		if !provider.IsAvailable() {
			return false
		}

		account := provider.GetCloudaccount()
		if account != nil && !account.IsAvailable() {
			return false
		}
	}

	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
}

func (self *SDisk) GetTemplateId() string {
	if len(self.TemplateId) == 0 {
		return ""
	}
	imageObj, err := CachedimageManager.FetchById(self.TemplateId)
	if err != nil || imageObj == nil {
		log.Errorf("failed to found disk %s(%s) templateId %s: %s", self.Name, self.Id, self.TemplateId, err)
		return ""
	}
	return self.TemplateId
}

func (self *SDisk) IsLocal() bool {
	storage := self.GetStorage()
	if storage != nil {
		return storage.IsLocal()
	}
	return false
}

func (self *SDisk) GetCloudproviderId() string {
	storage := self.GetStorage()
	if storage != nil {
		return storage.GetCloudproviderId()
	}
	return ""
}

func (self *SDisk) GetStorage() *SStorage {
	store, _ := StorageManager.FetchById(self.StorageId)
	if store != nil {
		return store.(*SStorage)
	}
	return nil
}

func (self *SDisk) GetRegionDriver() (IRegionDriver, error) {
	storage := self.GetStorage()
	if storage == nil {
		return nil, fmt.Errorf("failed to found storage for disk %s(%s)", self.Name, self.Id)
	}
	return storage.GetRegionDriver()
}

func (self *SDisk) GetBackupStorage() *SStorage {
	if len(self.BackupStorageId) == 0 {
		return nil
	}
	store, _ := StorageManager.FetchById(self.BackupStorageId)
	if store != nil {
		return store.(*SStorage)
	}
	return nil
}

func (self *SDisk) GetCloudprovider() *SCloudprovider {
	if storage := self.GetStorage(); storage != nil {
		return storage.GetCloudprovider()
	}
	return nil
}

func (self *SDisk) GetPathAtHost(host *SHost) string {
	hostStorage := host.GetHoststorageOfId(self.StorageId)
	if hostStorage != nil {
		return path.Join(hostStorage.MountPoint, self.Id)
	} else if len(self.BackupStorageId) > 0 {
		hostStorage = host.GetHoststorageOfId(self.BackupStorageId)
		if hostStorage != nil {
			return path.Join(hostStorage.MountPoint, self.Id)
		}
	}
	return ""
}

func (self *SDisk) GetFetchUrl() string {
	storage := self.GetStorage()
	if storage == nil {
		return ""
	}
	host := storage.GetMasterHost()
	return fmt.Sprintf("%s/disks/%s", host.GetFetchUrl(true), self.Id)
}

func (self *SDisk) GetFsFormat() string {
	return self.FsFormat
}

func (manager *SDiskManager) getDisksByStorage(storage *SStorage) ([]SDisk, error) {
	disks := make([]SDisk, 0)
	q := manager.Query().Equals("storage_id", storage.Id)
	err := db.FetchModelObjects(manager, q, &disks)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return disks, nil
}

func (manager *SDiskManager) syncCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, vdisk cloudprovider.ICloudDisk, index int, syncOwnerId mcclient.IIdentityProvider, managerId string) (*SDisk, error) {
	// ownerProjId := projectId

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	diskObj, err := db.FetchByExternalIdAndManagerId(manager, vdisk.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := StorageManager.Query().SubQuery()
		return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("storage_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), managerId))
	})
	if err != nil {
		if err == sql.ErrNoRows {
			vstorage, err := vdisk.GetIStorage()
			if err != nil {
				return nil, errors.Wrapf(err, "unable to GetIStorage of vdisk %q", vdisk.GetName())
			}

			storageObj, err := db.FetchByExternalIdAndManagerId(StorageManager, vstorage.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", managerId)
			})
			if err != nil {
				log.Errorf("cannot find storage of vdisk %s", err)
				return nil, err
			}
			storage := storageObj.(*SStorage)
			return manager.newFromCloudDisk(ctx, userCred, provider, vdisk, storage, -1, syncOwnerId)
		} else {
			return nil, err
		}
	} else {
		disk := diskObj.(*SDisk)
		err = disk.syncWithCloudDisk(ctx, userCred, provider, vdisk, index, syncOwnerId, managerId)
		if err != nil {
			return nil, err
		}
		return disk, nil
	}
}

func (manager *SDiskManager) SyncDisks(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, storage *SStorage, disks []cloudprovider.ICloudDisk, syncOwnerId mcclient.IIdentityProvider) ([]SDisk, []cloudprovider.ICloudDisk, compare.SyncResult) {
	// syncOwnerId := projectId

	lockman.LockRawObject(ctx, "disks", storage.Id)
	defer lockman.ReleaseRawObject(ctx, "disks", storage.Id)

	localDisks := make([]SDisk, 0)
	remoteDisks := make([]cloudprovider.ICloudDisk, 0)
	syncResult := compare.SyncResult{}

	dbDisks, err := manager.getDisksByStorage(storage)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SDisk, 0)
	commondb := make([]SDisk, 0)
	commonext := make([]cloudprovider.ICloudDisk, 0)
	added := make([]cloudprovider.ICloudDisk, 0)

	err = compare.CompareSets(dbDisks, disks, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudDisk(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudDisk(ctx, userCred, provider, commonext[i], -1, syncOwnerId, storage.ManagerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncVirtualResourceMetadata(ctx, userCred, &commondb[i], commonext[i])
			localDisks = append(localDisks, commondb[i])
			remoteDisks = append(remoteDisks, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		extId := added[i].GetGlobalId()
		_disk, err := db.FetchByExternalIdAndManagerId(manager, extId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := StorageManager.Query().SubQuery()
			return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("storage_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), storage.ManagerId))
		})
		if err != nil && err != sql.ErrNoRows {
			//主要是显示duplicate err及 general err,方便排错
			msg := fmt.Errorf("failed to found disk by external Id %s error: %v", extId, err)
			syncResult.Error(msg)
			continue
		}
		if _disk != nil {
			disk := _disk.(*SDisk)
			err = disk.syncDiskStorage(ctx, userCred, added[i], storage.ManagerId)
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
			continue
		}
		new, err := manager.newFromCloudDisk(ctx, userCred, provider, added[i], storage, -1, syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncVirtualResourceMetadata(ctx, userCred, new, added[i])
			localDisks = append(localDisks, *new)
			remoteDisks = append(remoteDisks, added[i])
			syncResult.Add()
		}
	}

	return localDisks, remoteDisks, syncResult
}

func (self *SDisk) syncDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, idisk cloudprovider.ICloudDisk, managerId string) error {
	extId := idisk.GetGlobalId()
	istorage, err := idisk.GetIStorage()
	if err != nil {
		log.Errorf("failed to get istorage for disk %s error: %v", extId, err)
		return err
	}
	storageExtId := istorage.GetGlobalId()
	storage, err := db.FetchByExternalIdAndManagerId(StorageManager, storageExtId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", managerId)
	})
	if err != nil {
		log.Errorf("failed to found storage by istorage %s error: %v", storageExtId, err)
		return err
	}
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.StorageId = storage.GetId()
		self.Status = idisk.GetStatus()
		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudDisk error %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SDisk) GetIRegion() (cloudprovider.ICloudRegion, error) {
	storage := self.GetStorage()
	if storage == nil {
		return nil, fmt.Errorf("failed to get storage for disk %s(%s)", self.Name, self.Id)
	}

	provider, err := storage.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for storage %s(%s) error: %v", storage.Name, storage.Id, err)
	}

	if provider.GetFactory().IsOnPremise() {
		return provider.GetOnPremiseIRegion()
	}
	region := storage.GetRegion()
	if region == nil {
		msg := "fail to find region of storage???"
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SDisk) syncRemoveCloudDisk(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	iregion, err := self.GetIRegion()
	if err != nil {
		return err
	}
	iDisk, err := iregion.GetIDiskById(self.ExternalId)
	if err == nil {
		if storageId := iDisk.GetIStorageId(); len(storageId) > 0 {
			storage, err := db.FetchByExternalIdAndManagerId(StorageManager, storageId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				if s := self.GetStorage(); s != nil {
					return q.Equals("manager_id", s.ManagerId)
				}
				return q
			})
			if err == nil {
				_, err = db.Update(self, func() error {
					self.StorageId = storage.GetId()
					return nil
				})
				return err
			}
		}
	} else if errors.Cause(err) != cloudprovider.ErrNotFound {
		return err
	}

	err = self.ValidatePurgeCondition(ctx)
	if err != nil {
		self.SetStatus(userCred, api.DISK_UNKNOWN, "missing original disk after sync")
		return err
	}
	// detach joint modle aboutsnapshotpolicy and disk
	err = SnapshotPolicyDiskManager.SyncDetachByDisk(ctx, userCred, nil, self)
	if err != nil {
		return err
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SDisk) syncWithCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, index int, syncOwnerId mcclient.IIdentityProvider, managerId string) error {
	recycle := false
	guests := self.GetGuests()
	if provider.GetFactory().IsSupportPrepaidResources() && len(guests) == 1 && guests[0].IsPrepaidRecycle() {
		recycle = true
	}
	extDisk.Refresh()

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = extDisk.GetName()
		self.Status = extDisk.GetStatus()
		self.DiskFormat = extDisk.GetDiskFormat()
		self.DiskSize = extDisk.GetDiskSizeMB()
		self.AccessPath = extDisk.GetAccessPath()
		if extDisk.GetIsAutoDelete() {
			self.AutoDelete = true
		}
		// self.TemplateId = extDisk.GetTemplateId() no sync template ID
		if templateId := extDisk.GetTemplateId(); len(templateId) > 0 {
			cachedImage, err := db.FetchByExternalId(CachedimageManager, templateId)
			if err == nil && cachedImage != nil {
				self.TemplateId = cachedImage.GetId()
			}
		}
		self.DiskType = extDisk.GetDiskType()
		if index == 0 {
			self.DiskType = api.DISK_TYPE_SYS
		}
		// self.FsFormat = extDisk.GetFsFormat()
		self.Nonpersistent = extDisk.GetIsNonPersistent()

		self.IsEmulated = extDisk.IsEmulated()

		if provider.GetFactory().IsSupportPrepaidResources() && !recycle {
			if billintType := extDisk.GetBillingType(); len(billintType) > 0 {
				self.BillingType = extDisk.GetBillingType()
				if self.BillingType == billing_api.BILLING_TYPE_PREPAID {
					self.AutoRenew = extDisk.IsAutoRenew()
				}
			}
			if expiredAt := extDisk.GetExpiredAt(); !expiredAt.IsZero() {
				self.ExpiredAt = extDisk.GetExpiredAt()
			}
		}

		if createdAt := extDisk.GetCreatedAt(); !createdAt.IsZero() {
			self.CreatedAt = createdAt
		}

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudDisk error %s", err)
		return err
	}

	// sync disk's snapshotpolicy
	snapshotpolicies, err := extDisk.GetExtSnapshotPolicyIds()
	if err != nil {
		return errors.Wrapf(err, "Get snapshot policies of ICloudDisk %s.", extDisk.GetId())
	}
	storage := self.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	err = SnapshotPolicyDiskManager.SyncByDisk(ctx, userCred, snapshotpolicies, syncOwnerId, self, storage)
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, syncOwnerId, extDisk, storage.ManagerId)

	return nil
}

func (manager *SDiskManager) newFromCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, storage *SStorage, index int, syncOwnerId mcclient.IIdentityProvider) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(manager, &disk)

	disk.Status = extDisk.GetStatus()
	disk.ExternalId = extDisk.GetGlobalId()
	disk.StorageId = storage.Id

	disk.DiskFormat = extDisk.GetDiskFormat()
	disk.DiskSize = extDisk.GetDiskSizeMB()
	disk.AutoDelete = extDisk.GetIsAutoDelete()
	disk.DiskType = extDisk.GetDiskType()
	if index == 0 {
		disk.DiskType = api.DISK_TYPE_SYS
	}
	disk.Nonpersistent = extDisk.GetIsNonPersistent()

	disk.IsEmulated = extDisk.IsEmulated()

	if provider.GetFactory().IsSupportPrepaidResources() {
		disk.BillingType = extDisk.GetBillingType()
		if expired := extDisk.GetExpiredAt(); !expired.IsZero() {
			disk.ExpiredAt = expired
		}
		disk.AutoRenew = extDisk.IsAutoRenew()
	}

	if createAt := extDisk.GetCreatedAt(); !createAt.IsZero() {
		disk.CreatedAt = createAt
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, syncOwnerId, extDisk.GetName())
		if err != nil {
			return err
		}
		disk.Name = newName

		return manager.TableSpec().Insert(ctx, &disk)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudDisk")
	}

	// create new joint model aboutsnapshotpolicy and disk
	snapshotpolicies, err := extDisk.GetExtSnapshotPolicyIds()
	if err != nil {
		return nil, errors.Wrapf(err, "Get snapshot policies of ICloudDisk %s.", extDisk.GetId())
	}
	err = SnapshotPolicyDiskManager.SyncAttachDiskExt(ctx, userCred, snapshotpolicies, syncOwnerId, &disk, storage)
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, &disk, syncOwnerId, extDisk, storage.ManagerId)

	db.OpsLog.LogEvent(&disk, db.ACT_CREATE, disk.GetShortDesc(ctx), userCred)

	return &disk, nil
}

func totalDiskSize(
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	active tristate.TriState,
	ready tristate.TriState,
	includeSystem bool,
	pendingDelete bool,
	rangeObjs []db.IStandaloneModel,
	providers []string,
	brands []string,
	cloudEnv string,
	hypervisors []string,
) int {
	disks := DiskManager.Query().SubQuery()
	q := disks.Query(sqlchemy.SUM("total", disks.Field("disk_size")))
	storages := StorageManager.Query().SubQuery()
	q = q.Join(storages, sqlchemy.Equals(storages.Field("id"), disks.Field("storage_id")))
	q = CloudProviderFilter(q, storages.Field("manager_id"), providers, brands, cloudEnv)
	q = RangeObjectsFilter(q, rangeObjs, nil, storages.Field("zone_id"), storages.Field("manager_id"), nil, storages.Field("id"))
	if len(hypervisors) > 0 {
		hoststorages := HoststorageManager.Query().SubQuery()
		hosts := HostManager.Query().SubQuery()
		q = q.Join(hoststorages, sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")))
		q = q.Join(hosts, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.In(hosts.Field("host_type"), api.Hypervisors2HostTypes(hypervisors)))
	}
	if !active.IsNone() {
		if active.IsTrue() {
			q = q.Filter(sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}))
		} else {
			q = q.Filter(sqlchemy.NotIn(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}))
		}
	}

	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(disks.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacutils.ScopeProject:
		q = q.Filter(sqlchemy.Equals(disks.Field("tenant_id"), ownerId.GetProjectId()))
	}

	if !ready.IsNone() {
		if ready.IsTrue() {
			q = q.Filter(sqlchemy.Equals(disks.Field("status"), api.DISK_READY))
		} else {
			q = q.Filter(sqlchemy.NotEquals(disks.Field("status"), api.DISK_READY))
		}
	}
	if !includeSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(disks.Field("is_system")),
			sqlchemy.IsFalse(disks.Field("is_system"))))
	}
	if pendingDelete {
		q = q.Filter(sqlchemy.IsTrue(disks.Field("pending_deleted")))
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(disks.Field("pending_deleted")), sqlchemy.IsFalse(disks.Field("pending_deleted"))))
	}

	row := q.Row()
	size := sql.NullInt64{}
	err := row.Scan(&size)
	if err != nil {
		log.Errorf("totalDiskSize error %s: %s", err, q.String())
		return 0
	}
	if size.Valid {
		return int(size.Int64)
	} else {
		return 0
	}
}

func parseDiskInfo(ctx context.Context, userCred mcclient.TokenCredential, info *api.DiskConfig) (*api.DiskConfig, error) {
	if info.SnapshotId != "" {
		if err := fillDiskConfigBySnapshot(userCred, info, info.SnapshotId); err != nil {
			return nil, err
		}
	}
	if info.ImageId != "" {
		if err := fillDiskConfigByImage(ctx, userCred, info, info.ImageId); err != nil {
			return nil, err
		}
	}
	// XXX: do not set default disk size here, set it by each hypervisor driver
	// if len(diskConfig.ImageId) > 0 && diskConfig.SizeMb == 0 {
	// 	diskConfig.SizeMb = options.Options.DefaultDiskSize // MB
	// else
	if len(info.ImageId) == 0 && info.SizeMb == 0 {
		return nil, httperrors.NewInputParameterError("Diskinfo index %d: both imageID and size are absent", info.Index)
	}
	return info, nil
}

func fillDiskConfigBySnapshot(userCred mcclient.TokenCredential, diskConfig *api.DiskConfig, snapshotId string) error {
	iSnapshot, err := SnapshotManager.FetchByIdOrName(userCred, snapshotId)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewNotFoundError("Snapshot %s not found", snapshotId)
		}
		return err
	}
	var snapshot = iSnapshot.(*SSnapshot)
	if storage := StorageManager.FetchStorageById(snapshot.StorageId); storage == nil {
		return httperrors.NewBadRequestError("Snapshot %s storage %s not found, is public cloud?",
			snapshotId, snapshot.StorageId)
	} else {
		if disk := DiskManager.FetchDiskById(snapshot.DiskId); disk != nil {
			diskConfig.Fs = disk.FsFormat
			if len(diskConfig.Format) == 0 {
				diskConfig.Format = disk.DiskFormat
			}
		}
		diskConfig.SnapshotId = snapshot.Id
		diskConfig.DiskType = snapshot.DiskType
		diskConfig.SizeMb = snapshot.Size
		diskConfig.Backend = storage.StorageType
		diskConfig.Fs = ""
		diskConfig.Mountpoint = ""
		diskConfig.OsArch = snapshot.OsArch
	}
	return nil
}

func fillDiskConfigByImage(ctx context.Context, userCred mcclient.TokenCredential,
	diskConfig *api.DiskConfig, imageId string) error {
	if userCred == nil {
		diskConfig.ImageId = imageId
	} else {
		image, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
		if err != nil {
			log.Errorf("getImageInfo %s fail %s", imageId, err)
			return err
		}
		if image.Status != cloudprovider.IMAGE_STATUS_ACTIVE {
			return httperrors.NewInvalidStatusError("Image status is not active")
		}
		diskConfig.ImageId = image.Id
		diskConfig.ImageProperties = image.Properties
		diskConfig.ImageProperties[imageapi.IMAGE_DISK_FORMAT] = image.DiskFormat
		// if len(diskConfig.Format) == 0 {
		// 	diskConfig.Format = image.DiskFormat
		// }
		// diskConfig.ImageDiskFormat = image.DiskFormat
		CachedimageManager.ImageAddRefCount(image.Id)
		if diskConfig.SizeMb != api.DISK_SIZE_AUTOEXTEND && diskConfig.SizeMb < image.MinDiskMB {
			diskConfig.SizeMb = image.MinDiskMB // MB
		}
		if strings.Contains(image.Properties["os_arch"], "aarch") {
			diskConfig.OsArch = apis.OS_ARCH_AARCH64
		} else {
			diskConfig.OsArch = image.Properties["os_arch"]
		}
	}
	return nil
}

func parseIsoInfo(ctx context.Context, userCred mcclient.TokenCredential, imageId string) (*cloudprovider.SImage, error) {
	image, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
	if err != nil {
		log.Errorf("getImageInfo fail %s", err)
		return nil, err
	}
	if image.Status != cloudprovider.IMAGE_STATUS_ACTIVE {
		return nil, httperrors.NewInvalidStatusError("Image status is not active")
	}
	return image, nil
}

func (self *SDisk) fetchDiskInfo(diskConfig *api.DiskConfig) {
	if len(diskConfig.ImageId) > 0 {
		self.TemplateId = diskConfig.ImageId
		// support for create vm from guest image
		if len(diskConfig.DiskType) == 0 {
			self.DiskType = api.DISK_TYPE_SYS
		} else {
			self.DiskType = diskConfig.DiskType
		}
	} else if len(diskConfig.SnapshotId) > 0 {
		self.SnapshotId = diskConfig.SnapshotId
		self.DiskType = diskConfig.DiskType
	}
	if len(diskConfig.Fs) > 0 {
		self.FsFormat = diskConfig.Fs
	}
	if self.FsFormat == "swap" {
		self.DiskType = api.DISK_TYPE_SWAP
		self.Nonpersistent = true
	} else {
		if len(self.DiskType) == 0 {
			diskType := api.DISK_TYPE_DATA
			if diskConfig.DiskType == api.DISK_TYPE_VOLUME {
				diskType = api.DISK_TYPE_VOLUME
			}
			self.DiskType = diskType
		}
		self.Nonpersistent = false
	}
	if len(diskConfig.DiskId) > 0 && utils.IsMatchUUID(diskConfig.DiskId) {
		self.Id = diskConfig.DiskId
	}
	self.DiskFormat = diskConfig.Format
	self.DiskSize = diskConfig.SizeMb
	self.OsArch = diskConfig.OsArch
}

type DiskInfo struct {
	ImageId    string
	Fs         string
	MountPoint string
	Format     string
	Size       int64
	Storage    string
	Backend    string
	MediumType string
	Driver     string
	Cache      string
	DiskType   string
}

// DEPRECATE: will be remove in future, use ToDiskConfig
func (self *SDisk) ToDiskInfo() DiskInfo {
	ret := DiskInfo{
		ImageId:    self.GetTemplateId(),
		Fs:         self.GetFsFormat(),
		MountPoint: self.GetMountPoint(),
		Format:     self.DiskFormat,
		Size:       int64(self.DiskSize),
		DiskType:   self.DiskType,
	}
	storage := self.GetStorage()
	if storage == nil {
		return ret
	}
	ret.Storage = storage.Id
	ret.Backend = storage.StorageType
	ret.MediumType = storage.MediumType
	return ret
}

func (self *SDisk) ToDiskConfig() *api.DiskConfig {
	ret := &api.DiskConfig{
		Index:      -1,
		ImageId:    self.GetTemplateId(),
		Fs:         self.GetFsFormat(),
		Mountpoint: self.GetMountPoint(),
		Format:     self.DiskFormat,
		SizeMb:     self.DiskSize,
		DiskType:   self.DiskType,
	}
	storage := self.GetStorage()
	if storage == nil {
		return ret
	}
	ret.Storage = storage.Id
	ret.Backend = storage.StorageType
	ret.Medium = storage.MediumType
	return ret
}

func (self *SDisk) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("disk delete do nothing")
	return nil
}

func (self *SDisk) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.DetachAllSnapshotpolicies(ctx, userCred)
	if err != nil {
		log.Errorf("unable to DetachAllSnapshotpolicies: %v", err)
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SDisk) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

// 同步磁盘状态
func (self *SDisk) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Disk has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DiskSyncstatusTask", "")
}

func (self *SDisk) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SDisk) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidatePurgeCondition(ctx)
	if err != nil {
		return nil, err
	}

	provider := self.GetCloudprovider()
	if provider != nil && provider.Provider == api.CLOUD_PROVIDER_HUAWEI {
		cnt, err := self.GetSnapshotCount()
		if err != nil {
			return nil, httperrors.NewInternalServerError("GetSnapshotCount fail %s", err)
		}
		if cnt > 0 {
			return nil, httperrors.NewForbiddenError("not allow to purge. Virtual disk must not have snapshots")
		}
	}

	return nil, self.StartDiskDeleteTask(ctx, userCred, "", true, false, false)
}

func (self *SDisk) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !jsonutils.QueryBoolean(query, "delete_snapshots", false) {
		if provider := self.GetCloudprovider(); provider != nil && provider.Provider == api.CLOUD_PROVIDER_HUAWEI {
			cnt, err := self.GetSnapshotCount()
			if err != nil {
				return httperrors.NewInternalServerError("GetSnapshotCount fail %s", err)
			}
			if cnt > 0 {
				return httperrors.NewForbiddenError("not allow to delete. Virtual disk must not have snapshots")
			}
		} else if storage := self.GetStorage(); storage != nil && storage.StorageType == api.STORAGE_RBD {
			scnt, err := self.GetSnapshotCount()
			if err != nil {
				return err
			}
			if scnt > 0 {
				return httperrors.NewBadRequestError("not allow to delete %s disk with snapshots", storage.StorageType)
			}
		}
	}

	return self.StartDiskDeleteTask(ctx, userCred, "", false,
		jsonutils.QueryBoolean(query, "override_pending_delete", false),
		jsonutils.QueryBoolean(query, "delete_snapshots", false))
}

func (self *SDisk) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, out api.DiskDetails) api.DiskDetails {
	out.Guests = []api.SimpleGuest{}
	guests, guestStatus := []string{}, []string{}
	for _, guest := range self.GetGuests() {
		guests = append(guests, guest.Name)
		guestStatus = append(guestStatus, guest.Status)
		out.Guests = append(out.Guests, api.SimpleGuest{
			Name:   guest.Name,
			Id:     guest.Id,
			Status: guest.Status,
		})
	}
	out.Guest = strings.Join(guests, ",")
	out.GuestCount = len(guests)
	out.GuestStatus = strings.Join(guestStatus, ",")

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		out.AutoDeleteAt = pendingDeletedAt
	}
	// the binded snapshot policy list
	sds, err := SnapshotPolicyDiskManager.FetchAllByDiskID(ctx, userCred, self.Id)
	if err != nil {
		return out
	}
	spIds := make([]string, len(sds))
	for i := range sds {
		spIds[i] = sds[i].SnapshotpolicyId
	}
	sps, err := SnapshotPolicyManager.FetchAllByIds(spIds)
	if err != nil {
		return out
	}
	if len(sps) > 0 {
		out.SnapshotpolicyStatus = sds[0].Status
	}

	// check status
	// construction for snapshotpolicies attached to disk
	out.Snapshotpolicies = []api.SimpleSnapshotPolicy{}
	for i := range sps {
		policy := api.SimpleSnapshotPolicy{}
		policy.RepeatWeekdays = SnapshotPolicyManager.RepeatWeekdaysToIntArray(sps[i].RepeatWeekdays)
		policy.TimePoints = SnapshotPolicyManager.TimePointsToIntArray(sps[i].TimePoints)
		policy.Id = sps[i].Id
		policy.Name = sps[i].Name
		out.Snapshotpolicies = append(out.Snapshotpolicies, policy)
	}
	storage := self.GetStorage()
	if storage != nil {
		manualSnapshotCount, _ := self.GetManualSnapshotCount()
		if utils.IsInStringArray(storage.StorageType, append(api.SHARED_FILE_STORAGE, api.STORAGE_LOCAL)) {
			out.ManualSnapshotCount = manualSnapshotCount
			out.MaxManualSnapshotCount = options.Options.DefaultMaxManualSnapshotCount
		}
	}

	return out
}

func (self *SDisk) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DiskDetails, error) {
	return api.DiskDetails{}, nil
}

func (manager *SDiskManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DiskDetails {
	rows := make([]api.DiskDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	storeRows := manager.SStorageResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DiskDetails{
			VirtualResourceDetails: virtRows[i],
			StorageResourceInfo:    storeRows[i],
		}
		rows[i] = objs[i].(*SDisk).getMoreDetails(ctx, userCred, rows[i])
	}
	return rows
}

func (self *SDisk) StartDiskResizeTask(ctx context.Context, userCred mcclient.TokenCredential, sizeMb int64, parentTaskId string, pendingUsage quotas.IQuota) error {
	self.SetStatus(userCred, api.DISK_START_RESIZE, "StartDiskResizeTask")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(sizeMb), "size")
	task, err := taskman.TaskManager.NewTask(ctx, "DiskResizeTask", self, userCred, params, parentTaskId, "", pendingUsage)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDisk) StartDiskDeleteTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string,
	isPurge, overridePendingDelete, deleteSnapshots bool,
) error {
	params := jsonutils.NewDict()
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	if deleteSnapshots {
		params.Add(jsonutils.JSONTrue, "delete_snapshots")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "DiskDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDisk) GetAttachedGuests() []SGuest {
	guests := GuestManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()

	q := guests.Query()
	q = q.Join(guestdisks, sqlchemy.AND(sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")),
		sqlchemy.IsFalse(guestdisks.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id))

	ret := make([]SGuest, 0)
	if err := db.FetchModelObjects(GuestManager, q, &ret); err != nil {
		log.Errorf("Fetch Geusts Objects %v", err)
		return nil
	}
	return ret
}

func (self *SDisk) SetDiskReady(ctx context.Context, userCred mcclient.TokenCredential, reason string) {
	self.SetStatus(userCred, api.DISK_READY, reason)
	guests := self.GetAttachedGuests()
	if guests != nil {
		for _, guest := range guests {
			guest.StartSyncstatus(ctx, userCred, "")
		}
	}
}

func (self *SDisk) SwitchToBackup(userCred mcclient.TokenCredential) error {
	diff, err := db.Update(self, func() error {
		self.StorageId, self.BackupStorageId = self.BackupStorageId, self.StorageId
		return nil
	})
	if err != nil {
		log.Errorf("SwitchToBackup fail %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (self *SDisk) ClearHostSchedCache() error {
	storage := self.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	hosts := storage.GetAllAttachingHosts()
	if hosts == nil {
		return fmt.Errorf("get attaching host error")
	}
	for _, h := range hosts {
		err := h.ClearSchedDescCache()
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SDisk) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewInt(int64(self.DiskSize)), "size")
	storage := self.GetStorage()
	if storage != nil {
		desc.Add(jsonutils.NewString(storage.StorageType), "storage_type")
		desc.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	}

	if hypervisor := self.GetMetadata("hypervisor", nil); len(hypervisor) > 0 {
		desc.Add(jsonutils.NewString(hypervisor), "hypervisor")
	}

	if len(self.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(self.ExternalId), "externalId")
	}

	fs := self.GetFsFormat()
	if len(fs) > 0 {
		desc.Add(jsonutils.NewString(fs), "fs_format")
	}
	tid := self.GetTemplateId()
	if len(tid) > 0 {
		desc.Add(jsonutils.NewString(tid), "template_id")
	}

	var billingInfo SCloudBillingInfo

	if storage != nil {
		billingInfo.SCloudProviderInfo = storage.getCloudProviderInfo()
	}

	if priceKey := self.GetMetadata("ext:price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (self *SDisk) getDev() string {
	return self.GetMetadata("dev", nil)
}

func (self *SDisk) GetMountPoint() string {
	return self.GetMetadata("mountpoint", nil)
}

func (self *SDisk) isReady() bool {
	return self.Status == api.DISK_READY
}

func (self *SDisk) isInit() bool {
	return self.Status == api.DISK_INIT
}

func (self *SDisk) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cancel-delete")
}

func (self *SDisk) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted && !self.Deleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		if err != nil {
			return nil, err
		}
		self.RecoverUsages(ctx, userCred)
	}
	return nil, nil
}

func (manager *SDiskManager) getExpiredPendingDeleteDisks() []SDisk {
	deadline := time.Now().Add(time.Duration(options.Options.PendingDeleteExpireSeconds*-1) * time.Second)

	q := manager.Query()
	q = q.IsTrue("pending_deleted").LT("pending_deleted_at", deadline).Limit(options.Options.PendingDeleteMaxCleanBatchSize)

	disks := make([]SDisk, 0)
	err := db.FetchModelObjects(DiskManager, q, &disks)
	if err != nil {
		log.Errorf("fetch disks error %s", err)
		return nil
	}

	return disks
}

func (manager *SDiskManager) CleanPendingDeleteDisks(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	disks := manager.getExpiredPendingDeleteDisks()
	if disks == nil {
		return
	}
	for i := 0; i < len(disks); i += 1 {
		disks[i].StartDiskDeleteTask(ctx, userCred, "", false, false, false)
	}
}

func (manager *SDiskManager) getAutoSnapshotDisksId(isExternal bool) ([]SSnapshotPolicyDisk, error) {

	t := time.Now()
	week := t.Weekday()
	if week == 0 { // sunday is zero
		week += 7
	}
	timePoint := t.Hour()

	sps, err := SnapshotPolicyManager.GetSnapshotPoliciesAt(uint32(week), uint32(timePoint))
	if err != nil {
		return nil, err
	}
	if len(sps) == 0 {
		return nil, nil
	}

	spds := make([]SSnapshotPolicyDisk, 0)
	spdq := SnapshotPolicyDiskManager.Query()
	spdq.NotEquals("status", api.SNAPSHOT_POLICY_DISK_INIT)
	spdq.Filter(sqlchemy.In(spdq.Field("snapshotpolicy_id"), sps))

	diskQ := DiskManager.Query().SubQuery()
	spdq.Join(diskQ, sqlchemy.Equals(spdq.Field("disk_id"), diskQ.Field("id")))
	if !isExternal {
		spdq.Filter(sqlchemy.IsNullOrEmpty(diskQ.Field("external_id")))
	} else {
		spdq.Filter(sqlchemy.IsNotEmpty(diskQ.Field("external_id")))
	}
	err = spdq.All(&spds)
	if err != nil {
		return nil, err
	}
	return spds, nil
}

func generateAutoSnapshotName() string {
	name := "Auto-" + rand.String(8)
	for SnapshotManager.Query().Equals("name", name).Count() > 0 {
		name = "Auto-" + rand.String(8)
	}
	return name
}

func (disk *SDisk) validateDiskAutoCreateSnapshot() error {
	guests := disk.GetGuests()
	if len(guests) == 0 {
		return fmt.Errorf("Disks %s not attach guest, can't create snapshot", disk.GetName())
	}
	storage := disk.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	if len(guests) == 1 && utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
		if !utils.IsInStringArray(guests[0].Status, []string{api.VM_RUNNING, api.VM_READY}) {
			return fmt.Errorf("Guest(%s) in status(%s) cannot do disk snapshot", guests[0].Id, guests[0].Status)
		}
	}
	return nil
}

func (manager *SDiskManager) AutoDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	spds, err := manager.getAutoSnapshotDisksId(false)
	if err != nil {
		log.Errorf("Get auto snapshot disks id failed: %s", err)
		return
	}
	if len(spds) == 0 {
		log.Infof("CronJob AutoDiskSnapshot: No disk need create snapshot")
		return
	}
	now := time.Now()
	for i := 0; i < len(spds); i++ {
		var (
			disk                  = manager.FetchDiskById(spds[i].DiskId)
			snapshotPolicy, _     = SnapshotPolicyManager.FetchSnapshotPolicyById(spds[i].SnapshotpolicyId)
			snapshotName          = generateAutoSnapshotName()
			autoSnapshotCount     = options.Options.DefaultMaxSnapshotCount - options.Options.DefaultMaxManualSnapshotCount
			err                   error
			snapCount             int
			cleanOverdueSnapshots bool
		)

		if err = disk.validateDiskAutoCreateSnapshot(); err != nil {
			goto onFail
		}

		if err = disk.CreateSnapshotAuto(ctx, userCred, snapshotName, snapshotPolicy); err != nil {
			goto onFail
		}

		snapCount, err = SnapshotManager.Query().Equals("fake_deleted", false).
			Equals("disk_id", disk.Id).Equals("created_by", api.SNAPSHOT_AUTO).
			CountWithError()
		if err != nil {
			err = errors.Wrap(err, "get snapshot count")
			goto onFail
		}
		// if auto snapshot count gt max auto snapshot count, do clean overdued snapshots
		cleanOverdueSnapshots = snapCount > autoSnapshotCount
		if cleanOverdueSnapshots {
			disk.CleanOverdueSnapshots(ctx, userCred, snapshotPolicy, now)
		}
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT, "disk auto snapshot "+snapshotName, userCred)
		snapshotPolicy.ExecuteNotify(ctx, userCred, disk.GetName())
		continue
	onFail:
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, err.Error(), userCred)
		reason := fmt.Sprintf("Disk auto create snapshot failed: %s", err.Error())
		notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, reason)
	}
}

func (self *SDisk) CreateSnapshotAuto(
	ctx context.Context, userCred mcclient.TokenCredential,
	snapshotName string, snapshotPolicy *SSnapshotPolicy,
) error {
	snap, err := SnapshotManager.CreateSnapshot(ctx, self.GetOwnerId(), api.SNAPSHOT_AUTO,
		self.Id, "", "", snapshotName, snapshotPolicy.RetentionDays)
	if err != nil {
		return errors.Wrap(err, "disk create snapshot auto")
	}

	db.OpsLog.LogEvent(snap, db.ACT_CREATE, "disk create snapshot auto", userCred)
	err = snap.StartSnapshotCreateTask(ctx, userCred, nil, "")
	if err != nil {
		return errors.Wrap(err, "disk auto snapshot start snapshot task")
	}
	return nil
}

func (self *SDisk) CleanOverdueSnapshots(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, now time.Time) error {
	kwargs := jsonutils.NewDict()
	kwargs.Set("snapshotpolicy_id", jsonutils.NewString(sp.Id))
	kwargs.Set("start_time", jsonutils.NewTimeString(now))
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskCleanOverduedSnapshots", self, userCred, kwargs, "", "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) StartCreateBackupTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskCreateBackupTask", self, userCred, nil, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) DeleteSnapshots(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskDeleteSnapshotsTask", self, userCred, nil, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) SaveRenewInfo(
	ctx context.Context, userCred mcclient.TokenCredential,
	bc *billing.SBillingCycle, expireAt *time.Time, billingType string,
) error {
	_, err := db.Update(self, func() error {
		if billingType == "" {
			billingType = billing_api.BILLING_TYPE_PREPAID
		}
		if self.BillingType == "" {
			self.BillingType = billingType
		}
		if expireAt != nil && !expireAt.IsZero() {
			self.ExpiredAt = *expireAt
		} else if bc != nil {
			self.BillingCycle = bc.String()
			self.ExpiredAt = bc.EndAt(self.ExpiredAt)
		}
		return nil
	})
	if err != nil {
		log.Errorf("Update error %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SDisk) CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential) error {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return fmt.Errorf("billing type %s not support cancel expire", self.BillingType)
	}
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set expired_at = NULL and billing_cycle = NULL where id = ?",
			DiskManager.TableSpec().Name(),
		), self.Id,
	)
	if err != nil {
		return errors.Wrap(err, "disk cancel expire time")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, "disk cancel expire time", userCred)
	return nil
}

func (self *SDisk) IsDetachable() bool {
	storage := self.GetStorage()
	if storage == nil {
		return true
	}
	if storage.IsLocal() {
		return false
	}
	if self.BillingType == billing_api.BILLING_TYPE_PREPAID {
		return false
	}
	if utils.IsInStringArray(self.DiskType, []string{api.DISK_TYPE_SYS, api.DISK_TYPE_SWAP}) {
		return false
	}
	if self.AutoDelete {
		return false
	}
	return true
}

func (self *SDisk) GetDynamicConditionInput() *jsonutils.JSONDict {
	conf := self.ToDiskConfig()
	return conf.JSON(conf)
}

func (self *SDisk) IsNeedWaitSnapshotsDeleted() (bool, error) {
	storage := self.GetStorage()
	if storage == nil {
		return false, fmt.Errorf("no valid storage")
	}
	if storage.StorageType == api.STORAGE_RBD {
		scnt, err := self.GetSnapshotCount()
		if err != nil {
			return false, err
		}
		if scnt > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (self *SDisk) UpdataSnapshotsBackingDisk(backingDiskId string) error {
	snapshots := make([]SSnapshot, 0)
	err := SnapshotManager.Query().Equals("disk_id", self.Id).IsNullOrEmpty("backing_disk_id").All(&snapshots)
	if err != nil {
		return err
	}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].SetModelManager(SnapshotManager, &snapshots[i])
		_, err := db.Update(&snapshots[i], func() error {
			snapshots[i].BackingDiskId = backingDiskId
			return nil
		})
		if err != nil {
			log.Errorln(err)
		}
	}
	return nil
}

func (manager *SDiskManager) AutoSyncExtDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {

	now := time.Now()
	log.Infof("AutoSyncExtDiskSnapshot starts: %s", now)

	week := now.Weekday()
	if week == 0 {
		week += 7
	}
	timePoint := now.Hour()

	q := SnapshotPolicyDiskManager.Query().LE("next_sync_time", now)
	spds := make([]SSnapshotPolicyDisk, 0)
	err := db.FetchModelObjects(SnapshotPolicyDiskManager, q, &spds)
	if err != nil {
		log.Errorf("unable to FetchModelObjects: %v", err)
	}
	// fetch all snapshotpolicy
	spIdSet := sets.NewString()
	for i := range spds {
		spIdSet.Insert(spds[i].SnapshotpolicyId)
	}
	sps, err := SnapshotPolicyManager.FetchAllByIds(spIdSet.UnsortedList())
	if err != nil {
		log.Errorf("unable to FetchAllByIds: %v", err)
	}
	spMap := make(map[string]*SSnapshotPolicy, len(sps))
	for i := range sps {
		spMap[sps[i].GetId()] = &sps[i]
	}

	for i := 0; i < len(spds); i++ {
		spd := &spds[i]
		obj, err := manager.FetchById(spd.DiskId)
		if errors.Cause(err) == sql.ErrNoRows || errors.Cause(err) == errors.ErrNotFound {
			err := spd.RealDetach(ctx, userCred)
			if err != nil {
				log.Errorf("unable to detach s %q, d %q: %v", spd.SnapshotpolicyId, spd.DiskId, err)
			}
			continue
		}
		disk := obj.(*SDisk)
		syncResult := disk.syncSnapshots(ctx, userCred)
		if syncResult.IsError() {
			db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SYNC_SNAPSHOT_FAIL, syncResult.Result(), userCred)
			continue
		}
		sp := spMap[spd.SnapshotpolicyId]
		repeatWeekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(sp.RepeatWeekdays)
		timePoints := SnapshotPolicyManager.TimePointsToIntArray(sp.TimePoints)
		if isInInts(int(week), repeatWeekdays) && isInInts(timePoint, timePoints) && syncResult.AddCnt == 0 {
			// should add one
			continue
		}
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SYNC_SNAPSHOT, "disk auto sync snapshot successfully", userCred)
		_, err = db.Update(spd, func() error {
			newNextSyncTime := spMap[spd.SnapshotpolicyId].ComputeNextSyncTime(now)
			spd.NextSyncTime = newNextSyncTime
			return nil
		})
		if err != nil {
			log.Errorf("unable to update NextSyncTime for snapshotpolicydisk %q %q", spd.SnapshotpolicyId, spd.DiskId)
		}
	}
	log.Infof("AutoSyncExtDiskSnapshot ends: %s", time.Now())
}

func isInInts(a int, array []int) bool {
	for _, i := range array {
		if i == a {
			return true
		}
	}
	return false
}

func (self *SDisk) syncSnapshots(ctx context.Context, userCred mcclient.TokenCredential) compare.SyncResult {
	syncResult := compare.SyncResult{}

	extDisk, err := self.GetIDisk()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	provider := self.GetCloudprovider()
	syncOwnerId := provider.GetOwnerId()
	storage := self.GetStorage()
	if storage == nil {
		syncResult.Error(fmt.Errorf("no valid storage"))
		return syncResult
	}
	region := storage.GetRegion()

	extSnapshots, err := extDisk.GetISnapshots()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	localSnapshots := SnapshotManager.GetDiskSnapshots(self.Id)

	lockman.LockRawObject(ctx, "snapshots", self.Id)
	defer lockman.ReleaseRawObject(ctx, "snapshots", self.Id)

	removed := make([]SSnapshot, 0)
	commondb := make([]SSnapshot, 0)
	commonext := make([]cloudprovider.ICloudSnapshot, 0)
	added := make([]cloudprovider.ICloudSnapshot, 0)

	err = compare.CompareSets(localSnapshots, extSnapshots, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudSnapshot(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudSnapshot(ctx, userCred, commonext[i], syncOwnerId, region)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		local, err := SnapshotManager.newFromCloudSnapshot(ctx, userCred, added[i], region, syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SDisk) GetSnapshotsNotInInstanceSnapshot() ([]SSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	sq := InstanceSnapshotJointManager.Query("snapshot_id").SubQuery()
	q := SnapshotManager.Query().IsFalse("fake_deleted").Equals("disk_id", self.Id)
	q = q.LeftJoin(sq, sqlchemy.Equals(q.Field("id"), sq.Field("snapshot_id"))).
		Filter(sqlchemy.IsNull(sq.Field("snapshot_id")))
	err := db.FetchModelObjects(SnapshotManager, q, &snapshots)
	if err != nil {
		log.Errorf("Fetch db snapshots failed %s", err)
		return nil, err
	}
	return snapshots, nil
}

func (self *SDisk) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {

	_, err := self.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	snapshotQuery := SnapshotManager.Query().Equals("disk_id", self.Id)
	snapshots := make([]SSnapshot, 0, 1)
	err = db.FetchModelObjects(SnapshotManager, snapshotQuery, &snapshots)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to fetch snapshots of disk %s", self.Id)
	}
	for i := range snapshots {
		snapshot := snapshots[i]
		err := func() error {
			lockman.LockObject(ctx, &snapshot)
			defer lockman.ReleaseObject(ctx, &snapshot)
			_, err := snapshot.PerformChangeOwner(ctx, userCred, query, input)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return nil, errors.Wrapf(err, "fail to change owner of this disk(%s)'s snapshot %s", self.Id, snapshot.Id)
		}
	}
	return nil, nil
}

func (disk *SDisk) GetUsages() []db.IUsage {
	if disk.PendingDeleted || disk.Deleted {
		return nil
	}
	usage := SQuota{Storage: disk.DiskSize}
	keys, err := disk.GetQuotaKeys()
	if err != nil {
		log.Errorf("disk.GetQuotaKeys fail %s", err)
		return nil
	}
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (disk *SDisk) AllowPerformBindSnapshotpolicy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return disk.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, disk, "bind-snapshotpolicy")
}

func (disk *SDisk) PerformBindSnapshotpolicy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	spIden, err := data.GetString("snapshotpolicy")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("miss snapshotpolicy")
	}
	// check snapshotpolicy
	imodel, err := db.FetchByIdOrName(SnapshotPolicyManager, userCred, spIden)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, httperrors.NewInputParameterError("no such snapshotpolicy %s", spIden)
	}
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchByIdOrName")
	}
	snapshotpolicy := imodel.(*SSnapshotPolicy)

	// try to bind
	spd, err := SnapshotPolicyDiskManager.newSnapshotpolicyDisk(ctx, userCred, snapshotpolicy, disk)

	if errors.Cause(err) == ErrExistSD {
		if spd.Status != api.SNAPSHOT_POLICY_DISK_INIT {
			return nil, nil
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "SnapshotPolicyDiskManager.newSnapshotpolicyDisk")
	}

	// start up SnapshotPolicyApplyTask
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.Marshal(spd), "snapshotPolicyDisk")
	taskData.Add(jsonutils.Marshal(snapshotpolicy), "snapshotPolicy")
	if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyApplyTask", disk, userCred, nil, "", "",
		nil); err != nil {
		return nil, errors.Wrap(err, "fail to start up SnapshotPolicyApplyTask")
	} else {
		task.ScheduleRun(taskData)
	}
	return nil, nil
}

func (disk *SDisk) AllowPerformUnbindSnapshotpolicy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return disk.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, disk, "unbind-snapshotpolicy")
}

func (disk *SDisk) PerformUnbindSnapshotpolicy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	spIden, err := data.GetString("snapshotpolicy")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("miss snapshotpolicy")
	}
	// check snapshotpolicy
	imodel, err := db.FetchByIdOrName(SnapshotPolicyManager, userCred, spIden)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, httperrors.NewInputParameterError("no such snapshotpolicy %s", spIden)
	}
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchByIdOrName")
	}
	snapshotpolicy := imodel.(*SSnapshotPolicy)

	spd, err := SnapshotPolicyDiskManager.FetchBySnapshotPolicyDisk(snapshotpolicy.GetId(), disk.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SnapshotPolicyDiskManager.FetchBySnapshotPolicyDisk")
	}
	if spd == nil {
		// has been detach
		return nil, nil
	}

	// start up SnapshotPolicyCancelTask
	taskdata := jsonutils.NewDict()
	taskdata.Add(jsonutils.NewString(snapshotpolicy.Id), "snapshot_policy_id")
	taskdata.Add(jsonutils.Marshal(spd), "snapshotPolicyDisk")
	if task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCancelTask", disk, userCred, nil, "", "",
		nil); err != nil {
		return nil, errors.Wrap(err, "fail to start up SnapshotPolicyCancelTask")
	} else {
		spd.SetStatus(userCred, api.SNAPSHOT_POLICY_DISK_DELETING, "")
		task.ScheduleRun(taskdata)
	}
	return nil, nil
}

func (manager *SDiskManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SStorageResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SStorageResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
