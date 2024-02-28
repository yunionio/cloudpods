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
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/pinyinutils"
	"yunion.io/x/pkg/util/rbacscope"
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
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDiskManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SStorageResourceBaseManager
	SBillingResourceBaseManager
	db.SMultiArchResourceBaseManager
	db.SAutoDeleteResourceBaseManager
	db.SEncryptedResourceManager
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
	DiskManager.TableSpec().AddIndex(false, "deleted", "disk_size", "status", "storage_id")
}

type SDisk struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SBillingResourceBase
	SStorageResourceBase `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"optional"`
	db.SMultiArchResourceBase
	db.SAutoDeleteResourceBase

	db.SEncryptedResource

	// 磁盘存储类型
	// example: qcow2
	DiskFormat string `width:"32" charset:"ascii" nullable:"false" default:"qcow2" list:"user" json:"disk_format"`
	// 磁盘大小, 单位Mb
	// example: 10240
	DiskSize int `nullable:"false" list:"user" json:"disk_size"`
	// 磁盘路径
	AccessPath string `width:"256" charset:"utf8" nullable:"true" get:"user" json:"access_path"`

	// 存储Id
	// StorageId       string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"optional"`

	// 备份磁盘实例的存储ID
	BackupStorageId string `width:"128" charset:"ascii" nullable:"true" list:"admin" json:"backup_storage_id"`

	// 镜像Id
	TemplateId string `width:"256" charset:"ascii" nullable:"true" list:"user" json:"template_id"`
	// 快照Id
	SnapshotId string `width:"256" charset:"ascii" nullable:"true" list:"user" json:"snapshot_id"`
	// 备份Id
	BackupId string `width:"256" charset:"ascii" nullable:"true" list:"user" json:"backup_id"`

	// 文件系统
	FsFormat string `width:"32" charset:"ascii" nullable:"true" list:"user" json:"fs_format"`

	// 磁盘类型
	// sys: 系统盘
	// data: 数据盘
	// swap: 交换盘
	// example: sys
	DiskType string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"admin" json:"disk_type"`

	// 预分配策略
	// off: 关闭预分配，默认关闭
	// metadata: 精简制备
	// falloc: 厚制制备延迟置零
	// full: 厚制备快速置零
	Preallocation string `width:"12" default:"off" charset:"ascii" nullable:"true" list:"user" update:"admin" json:"preallocation"`
	// # is persistent
	Nonpersistent bool `default:"false" list:"user" json:"nonpersistent"`

	// 是否标记为SSD磁盘
	IsSsd bool `nullable:"false" default:"false" list:"user" update:"user" create:"optional"`

	// 最大连接数
	Iops int `nullable:"true" list:"user" create:"optional"`

	// 磁盘吞吐量
	Throughput int `nullable:"true" list:"user" create:"optional"`
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
		server, err := validators.ValidateModel(ctx, userCred, GuestManager, &guestId)
		if err != nil {
			return nil, err
		}
		guest := server.(*SGuest)
		guestDisks := GuestdiskManager.Query().SubQuery()
		q = q.Join(guestDisks, sqlchemy.AND(
			sqlchemy.Equals(guestDisks.Field("disk_id"), q.Field("id")),
			sqlchemy.Equals(guestDisks.Field("guest_id"), guest.Id),
		)).Asc(guestDisks.Field("index"))
	}

	if diskType := query.DiskType; diskType != "" {
		q = q.Filter(sqlchemy.Equals(q.Field("disk_type"), diskType))
	}

	if len(query.SnapshotpolicyId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, SnapshotPolicyManager, &query.SnapshotpolicyId)
		if err != nil {
			return nil, err
		}
		sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", query.SnapshotpolicyId)
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
		_, err := validators.ValidateModel(ctx, userCred, SnapshotManager, &query.SnapshotId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("snapshot_id", query.SnapshotId)
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
	if db.NeedOrderQuery([]string{query.OrderByServer}) {
		guestDiskQuery := GuestdiskManager.Query("disk_id", "guest_id").SubQuery()
		q = q.LeftJoin(guestDiskQuery, sqlchemy.Equals(q.Field("id"), guestDiskQuery.Field("disk_id")))
		guestQuery := GuestManager.Query().SubQuery()
		q.AppendField(q.QueryFields()...)
		q.AppendField(guestQuery.Field("name", "guest_name"))
		q.Join(guestQuery, sqlchemy.Equals(guestQuery.Field("id"), guestDiskQuery.Field("guest_id")))
		db.OrderByFields(q, []string{query.OrderByServer}, []sqlchemy.IQueryField{guestQuery.Field("name")})
	}
	if db.NeedOrderQuery([]string{query.OrderByGuestCount}) {
		guestdisks := GuestdiskManager.Query().SubQuery()
		disks := DiskManager.Query().SubQuery()
		guestdiskQ := guestdisks.Query(
			guestdisks.Field("guest_id"),
			guestdisks.Field("disk_id"),
			sqlchemy.COUNT("guest_count", guestdisks.Field("guest_id")),
		)

		guestdiskQ = guestdiskQ.LeftJoin(disks, sqlchemy.Equals(guestdiskQ.Field("disk_id"), disks.Field("id")))
		guestdiskSQ := guestdiskQ.GroupBy(guestdiskQ.Field("disk_id")).SubQuery()
		q.AppendField(q.QueryFields()...)
		q.AppendField(guestdiskSQ.Field("guest_count"))
		q = q.LeftJoin(guestdiskSQ, sqlchemy.Equals(q.Field("id"), guestdiskSQ.Field("disk_id")))
		db.OrderByFields(q, []string{query.OrderByGuestCount}, []sqlchemy.IQueryField{guestdiskQ.Field("guest_count")})
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
	if field == "guest_status" {
		guestDiskQuery := GuestdiskManager.Query("disk_id", "guest_id").SubQuery()
		q = q.LeftJoin(guestDiskQuery, sqlchemy.Equals(q.Field("id"), guestDiskQuery.Field("disk_id")))
		guestQuery := GuestManager.Query().SubQuery()
		q.AppendField(guestQuery.Field("status", field)).Distinct()
		q.Join(guestQuery, sqlchemy.Equals(guestQuery.Field("id"), guestDiskQuery.Field("guest_id")))
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
	guests := GuestManager.Query().SubQuery()
	guestdisks = guestdisks.Join(guests, sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")))
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

func (self *SDisk) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	input := new(api.DiskCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "Unmarshal json")
	}
	self.fetchDiskInfo(input.DiskConfig)
	err := self.SEncryptedResource.CustomizeCreate(ctx, userCred, ownerId, data, "disk-"+pinyinutils.Text2Pinyin(self.Name))
	if err != nil {
		return errors.Wrap(err, "SEncryptedResource.CustomizeCreate")
	}
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SDisk) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskUpdateInput) (api.DiskUpdateInput, error) {
	var err error

	if input.DiskType != "" {
		if !utils.IsInStringArray(input.DiskType, []string{api.DISK_TYPE_DATA, api.DISK_TYPE_VOLUME}) {
			return input, httperrors.NewInputParameterError("not support update disk_type %s", input.DiskType)
		}
	}

	storage, _ := self.GetStorage()
	if storage == nil {
		return input, httperrors.NewNotFoundError("failed to find storage for disk %s", self.Name)
	}

	host, _ := storage.GetMasterHost()
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

func (man *SDiskManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DiskCreateInput) (*jsonutils.JSONDict, error) {
	input, err := man.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

func diskCreateInput2ComputeQuotaKeys(input api.DiskCreateInput, ownerId mcclient.IIdentityProvider) SComputeResourceKeys {
	// input.Hypervisor must be set
	brand := guessBrandForHypervisor(input.Hypervisor)
	keys := GetDriver(input.Hypervisor).GetComputeQuotaKeys(
		rbacscope.ScopeProject,
		ownerId,
		brand,
	)
	if len(input.PreferHost) > 0 {
		hostObj, _ := HostManager.FetchById(input.PreferHost)
		host := hostObj.(*SHost)
		zone, _ := host.GetZone()
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
		zone, _ := wire.GetZone()
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
	if input.ExistingPath != "" && input.Storage == "" {
		return input, httperrors.NewInputParameterError("disk create from existing disk must give storage")
	}

	input.ProjectId = ownerId.GetProjectId()
	input.ProjectDomainId = ownerId.GetProjectDomainId()

	var quotaKey quotas.IQuotaKeys

	storageID := input.Storage

	if storageID != "" {
		storageObj, err := StorageManager.FetchByIdOrName(ctx, nil, storageID)
		if err != nil {
			return input, httperrors.NewResourceNotFoundError("Storage %s not found", storageID)
		}
		storage := storageObj.(*SStorage)

		provider := storage.GetCloudprovider()
		if provider != nil && !provider.IsAvailable() {
			return input, httperrors.NewResourceNotReadyError("cloudprovider %s not available", provider.Name)
		}

		host, _ := storage.GetMasterHost()
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

		zone, _ := storage.getZone()
		quotaKey = fetchComputeQuotaKeys(
			rbacscope.ScopeProject,
			ownerId,
			zone,
			provider,
			input.Hypervisor,
		)

	} else {
		if len(diskConfig.Backend) == 0 {
			diskConfig.Backend = api.STORAGE_LOCAL
		}
		if len(input.PreferManager) > 0 {
			_manager, err := CloudproviderManager.FetchByIdOrName(ctx, userCred, input.PreferManager)
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
		// preserve encrypt info
		encInput := input.EncryptedResourceCreateInput
		input = *serverInput.ToDiskCreateInput()
		input.EncryptedResourceCreateInput = encInput
		quotaKey = diskCreateInput2ComputeQuotaKeys(input, ownerId)
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}
	input.EncryptedResourceCreateInput, err = manager.SEncryptedResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EncryptedResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEncryptedResourceManager.ValidateCreateData")
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
	if diskConfig.ExistingPath != "" {
		if !utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
			return httperrors.NewInputParameterError(
				"Disk create from existing path, unsupport storage type %s", storage.StorageType)
		}
	}

	var guestdriver IGuestDriver = nil
	if host, _ := storage.GetMasterHost(); host != nil {
		//公有云磁盘大小检查。
		if err := host.GetHostDriver().ValidateDiskSize(storage, diskConfig.SizeMb>>10); err != nil {
			return httperrors.NewInputParameterError("%v", err)
		}
		guestdriver = GetDriver(api.HOSTTYPE_HYPERVISOR[host.HostType])
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	hoststorage := make([]SHoststorage, 0)
	if err := hoststorages.Query().Equals("storage_id", storage.Id).All(&hoststorage); err != nil {
		return err
	}
	if len(hoststorage) == 0 {
		return httperrors.NewInputParameterError("Storage[%s] must attach to a host", storage.Name)
	}
	if guestdriver == nil || guestdriver.DoScheduleStorageFilter() {
		if int64(diskConfig.SizeMb) > storage.GetFreeCapacity() && !storage.IsEmulated {
			return httperrors.NewInputParameterError("Not enough free space")
		}
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
		disk.IsSsd = (storage.MediumType == api.DISK_TYPE_SSD)
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
		zone, _ := storage.getZone()
		quotaKey = fetchComputeQuotaKeys(
			rbacscope.ScopeProject,
			ownerId,
			zone,
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

func (disk *SDisk) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := api.DiskCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("!!!data.Unmarshal api.DiskCreateInput fail %s", err)
	}
	if input.ExistingPath != "" {
		disk.SetMetadata(ctx, api.DISK_META_EXISTING_PATH, input.ExistingPath, userCred)
	}
}

func (manager *SDiskManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	input := api.DiskCreateInput{}
	err := data[0].Unmarshal(&input)
	if err != nil {
		log.Errorf("!!!data.Unmarshal api.DiskCreateInput fail %s", err)
	}

	pendingUsage := getDiskResourceRequirements(ctx, userCred, ownerId, input, len(items))
	parentTaskId, _ := data[0].GetString("parent_task_id")
	RunBatchCreateTask(ctx, items, userCred, data, pendingUsage, SRegionQuota{}, "DiskBatchCreateTask", parentTaskId)
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

func (self *SDisk) GetSnapshotFuseUrl() (string, error) {
	snapObj, err := SnapshotManager.FetchById(self.SnapshotId)
	if err != nil {
		return "", errors.Wrapf(err, "SnapshotManager.FetchById(%s)", self.SnapshotId)
	}
	snapshot := snapObj.(*SSnapshot)
	return snapshot.GetFuseUrl()
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

func (self *SDisk) getDiskAllocateFromBackupInput(ctx context.Context, backupId string) (*api.DiskAllocateFromBackupInput, error) {
	ibackup, err := DiskBackupManager.FetchById(backupId)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get backup %s", backupId)
	}
	backup := ibackup.(*SDiskBackup)
	bs, err := backup.GetBackupStorage()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get backupstorage of backup %s", backupId)
	}
	accessInfo, err := bs.GetAccessInfo()
	if err != nil {
		return nil, errors.Wrap(err, "backupStorage.GetAccessInfo")
	}
	return &api.DiskAllocateFromBackupInput{
		BackupId:                backupId,
		BackupStorageId:         bs.GetId(),
		BackupStorageAccessInfo: jsonutils.Marshal(accessInfo).(*jsonutils.JSONDict),
	}, nil
}

func (self *SDisk) StartAllocate(ctx context.Context, host *SHost, storage *SStorage, taskId string, userCred mcclient.TokenCredential, rebuild bool, snapshot string, task taskman.ITask) error {
	log.Infof("Allocating disk on host %s ...", host.GetName())

	templateId := self.GetTemplateId()
	fsFormat := self.GetFsFormat()

	input := api.DiskAllocateInput{
		Format:     self.DiskFormat,
		DiskSizeMb: self.DiskSize,
		SnapshotId: snapshot,
	}
	if self.BackupId != "" {
		allocateInput, err := self.getDiskAllocateFromBackupInput(ctx, self.BackupId)
		if err != nil {
			return errors.Wrap(err, "unable to getDiskAllocateFromBackupInput")
		}
		input.Backup = allocateInput
	}
	if len(snapshot) > 0 {
		if utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
			SnapshotManager.AddRefCount(self.SnapshotId, 1)
			self.SetMetadata(ctx, "merge_snapshot", jsonutils.JSONTrue, userCred)
		}
	} else if len(templateId) > 0 {
		input.ImageId = templateId
	}
	if len(fsFormat) > 0 {
		input.FsFormat = fsFormat
	}
	if self.IsEncrypted() {
		var err error
		input.Encryption = true
		input.EncryptInfo, err = self.GetEncryptInfo(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "GetEncryptInfo")
		}
	}
	if ePath := self.GetMetadata(ctx, api.DISK_META_EXISTING_PATH, userCred); ePath != "" {
		input.ExistingPath = ePath
	}

	if rebuild {
		return host.GetHostDriver().RequestRebuildDiskOnStorage(ctx, host, storage, self, task, input)
	} else {
		return host.GetHostDriver().RequestAllocateDiskOnStorage(ctx, userCred, host, storage, self, task, input)
	}
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

func (self *SDisk) PerformDiskReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.DiskResetInput) (jsonutils.JSONObject, error) {
	err := self.ValidateEncryption(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateEncryption")
	}
	if !utils.IsInStringArray(self.Status, []string{api.DISK_READY}) {
		return nil, httperrors.NewInputParameterError("Cannot reset disk in status %s", self.Status)
	}
	storage, err := self.GetStorage()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetStorage"))
	}

	host, err := storage.GetMasterHost()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetMasterHost"))
	}

	snapshotObj, err := validators.ValidateModel(ctx, userCred, SnapshotManager, &input.SnapshotId)
	if err != nil {
		return nil, err
	}
	snapshot := snapshotObj.(*SSnapshot)
	if snapshot.Status != api.SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Cannot reset disk with snapshot in status %s", snapshot.Status)
	}

	if snapshot.DiskId != self.Id {
		return nil, httperrors.NewBadRequestError("Cannot reset disk %s(%s),Snapshot is belong to disk %s", self.Name, self.Id, snapshot.DiskId)
	}

	guests := self.GetGuests()
	input, err = host.GetHostDriver().ValidateResetDisk(ctx, userCred, self, snapshot, guests, input)
	if err != nil {
		return nil, err
	}

	var guest *SGuest = nil
	if len(guests) > 0 {
		guest = &guests[0]
	}
	return nil, self.StartResetDisk(ctx, userCred, snapshot.Id, input.AutoStart, guest, "")
}

func (self *SDisk) StartResetDisk(
	ctx context.Context, userCred mcclient.TokenCredential,
	snapshotId string, autoStart bool, guest *SGuest, parentTaskId string,
) error {
	self.SetStatus(ctx, userCred, api.DISK_RESET, "")
	if guest != nil {
		guest.SetStatus(ctx, userCred, api.VM_DISK_RESET, "disk reset")
	}
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshotId))
	params.Set("auto_start", jsonutils.NewBool(autoStart))
	task, err := taskman.TaskManager.NewTask(ctx, "DiskResetTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (disk *SDisk) PerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskResizeInput) (jsonutils.JSONObject, error) {
	guest := disk.GetGuest()
	sizeMb, err := input.SizeMb()
	if err != nil {
		return nil, err
	}
	err = disk.doResize(ctx, userCred, sizeMb, guest)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (disk *SDisk) getHypervisor() string {
	storage, _ := disk.GetStorage()
	if storage != nil {
		host, _ := storage.GetMasterHost()
		if host != nil {
			return host.GetHostDriver().GetHypervisor()
		}
	}
	hypervisor := disk.GetMetadata(context.Background(), "hypervisor", nil)
	return hypervisor
}

func (disk *SDisk) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	storage, _ := disk.GetStorage()
	if storage == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid storage")
	}
	provider := storage.GetCloudprovider()
	if provider == nil && len(storage.ManagerId) > 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid manager")
	}
	zone, _ := storage.getZone()
	if zone == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid zone")
	}
	return fetchComputeQuotaKeys(
		rbacscope.ScopeProject,
		disk.GetOwnerId(),
		zone,
		provider,
		disk.getHypervisor(),
	), nil
}

func (disk *SDisk) doResize(ctx context.Context, userCred mcclient.TokenCredential, sizeMb int, guest *SGuest) error {
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
	storage, _ := disk.GetStorage()
	if storage == nil {
		return httperrors.NewInternalServerError("disk has no valid storage")
	}

	var guestdriver IGuestDriver
	if host, _ := storage.GetMasterHost(); host != nil {
		if err := host.GetHostDriver().ValidateDiskSize(storage, sizeMb>>10); err != nil {
			return httperrors.NewInputParameterError("%v", err)
		}
		guestdriver = GetDriver(api.HOSTTYPE_HYPERVISOR[host.HostType])
	}
	if guestdriver == nil || guestdriver.DoScheduleStorageFilter() {
		if int64(addDisk) > storage.GetFreeCapacity() && !storage.IsEmulated {
			return httperrors.NewOutOfResourceError("Not enough free space")
		}
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

func (self *SDisk) GetIStorage(ctx context.Context) (cloudprovider.ICloudStorage, error) {
	storage, err := self.GetStorage()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage")
	}
	istorage, err := storage.GetIStorage(ctx)
	if err != nil {
		return nil, err
	}
	return istorage, nil
}

func (self *SDisk) GetIDisk(ctx context.Context) (cloudprovider.ICloudDisk, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iStorage, err := self.GetIStorage(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIStorage")
	}
	return iStorage.GetIDiskById(self.GetExternalId())
}

func (self *SDisk) GetZone() (*SZone, error) {
	storage, err := self.GetStorage()
	if err != nil {
		return nil, err
	}
	return storage.getZone()
}

func (self *SDisk) PrepareSaveImage(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerSaveImageInput) (string, error) {
	zone, _ := self.GetZone()
	if zone == nil {
		return "", httperrors.NewResourceNotFoundError("No zone for this disk")
	}
	if len(input.GenerateName) == 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		imageList, err := image.Images.List(s, jsonutils.Marshal(map[string]string{"name": input.Name, "admin": "true"}))
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
		OsArch       string
		Properties   map[string]string

		ProjectId string

		EncryptKeyId string
	}{
		Name:         input.Name,
		GenerateName: input.GenerateName,
		VirtualSize:  self.DiskSize,
		DiskFormat:   self.DiskFormat,
		OsArch:       input.OsArch,
		Properties: map[string]string{
			"notes":   input.Notes,
			"os_type": input.OsType,
			"os_arch": input.OsArch,
		},

		// inherit the ownership of disk
		ProjectId: self.ProjectId,
	}

	if self.IsEncrypted() {
		encKey, err := self.GetEncryptInfo(ctx, userCred)
		if err != nil {
			return "", errors.Wrap(err, "GetEncryptInfo")
		}
		opts.EncryptKeyId = encKey.Id
	}

	/*
		no need to check quota anymore
		session := auth.GetSession(userCred, options.Options.Region, "v2")
		quota := image_models.SQuota{Image: 1}
		if _, err := image.ImageQuotas.DoQuotaCheck(session, jsonutils.Marshal(&quota)); err != nil {
			return "", err
		}*/
	us := auth.GetSession(ctx, userCred, options.Options.Region)
	result, err := image.Images.Create(us, jsonutils.Marshal(opts))
	if err != nil {
		return "", err
	}
	imageId, err := result.GetString("id")
	if err != nil {
		return "", err
	}
	// check class metadata
	cm, err := self.GetAllClassMetadata()
	if err != nil {
		return "", errors.Wrap(err, "unable to GetAllClassMetadata")
	}
	if len(cm) > 0 {
		_, err = image.Images.PerformAction(us, imageId, "set-class-metadata", jsonutils.Marshal(cm))
		if err != nil {
			return "", errors.Wrapf(err, "unable to SetClassMetadata for image %s", imageId)
		}
	}
	return imageId, nil
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
	self.SetStatus(ctx, userCred, api.DISK_START_SAVE, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDisk) ValidateDeleteCondition(ctx context.Context, info api.DiskDetails) error {
	if len(info.Guests) > 0 {
		return httperrors.NewNotEmptyError("Virtual disk %s(%s) used by virtual servers", self.Name, self.Id)
	}
	if self.IsNotDeletablePrePaid() {
		return httperrors.NewForbiddenError("not allow to delete prepaid disk in valid status")
	}
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SDisk) validateDeleteCondition(ctx context.Context, isPurge bool) error {
	if !isPurge {
		storage, _ := self.GetStorage()
		if storage == nil {
			// storage is empty, a dirty data, allow delete
			return nil
		}
		host, _ := storage.GetMasterHost()
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
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SDisk) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	provider := self.GetCloudprovider()
	if provider != nil {
		if !provider.IsAvailable() {
			return false
		}

		account, _ := provider.GetCloudaccount()
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
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(ctx, userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(ctx, userCred, self)
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
	storage, _ := self.GetStorage()
	if storage != nil {
		return storage.IsLocal()
	}
	return false
}

func (self *SDisk) GetCloudproviderId() string {
	storage, _ := self.GetStorage()
	if storage != nil {
		return storage.GetCloudproviderId()
	}
	return ""
}

func (self *SDisk) GetStorage() (*SStorage, error) {
	store, err := StorageManager.FetchById(self.StorageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage(%s)", self.StorageId)
	}
	return store.(*SStorage), nil
}

func (self *SDisk) GetRegionDriver() (IRegionDriver, error) {
	storage, _ := self.GetStorage()
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
	if storage, _ := self.GetStorage(); storage != nil {
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

func (self *SDisk) GetMasterHost(storage *SStorage) (*SHost, error) {
	if storage.MasterHost != "" {
		return storage.GetMasterHost()
	}

	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query().Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.StorageId))
	q = q.IsTrue("enabled")
	q = q.Equals("host_status", api.HOST_ONLINE).Asc("id")
	guest := self.GetGuest()
	if guest != nil && len(guest.OsArch) > 0 {
		switch guest.OsArch {
		case apis.OS_ARCH_X86:
			q = q.In("cpu_architecture", apis.ARCH_X86)
		case apis.OS_ARCH_ARM:
			q = q.In("cpu_architecture", apis.ARCH_ARM)
		}
	}
	host := SHost{}
	host.SetModelManager(HostManager, &host)
	err := q.First(&host)
	if err != nil {
		return nil, errors.Wrapf(err, "q.First")
	}
	return &host, nil
}

func (self *SDisk) GetFetchUrl() (string, error) {
	storage, err := self.GetStorage()
	if err != nil {
		return "", errors.Wrapf(err, "self.GetStorage")
	}
	host, err := storage.GetMasterHost()
	if err != nil {
		return "", errors.Wrapf(err, "storage.GetMasterHost")
	}
	return fmt.Sprintf("%s/disks/%s", host.GetFetchUrl(true), self.Id), nil
}

func (self *SDisk) GetFsFormat() string {
	return self.FsFormat
}

func (self *SDisk) GetCacheImageFormat() string {
	if self.DiskFormat == "raw" {
		return "qcow2"
	}
	return self.DiskFormat
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

func (manager *SDiskManager) findOrCreateDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, vdisk cloudprovider.ICloudDisk, index int, syncOwnerId mcclient.IIdentityProvider, managerId string) (*SDisk, error) {
	diskId := vdisk.GetGlobalId()
	diskObj, err := db.FetchByExternalIdAndManagerId(manager, diskId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := StorageManager.Query().SubQuery()
		return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("storage_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), managerId))
	})
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, errors.Wrapf(err, "db.FetchByExternalIdAndManagerId %s", diskId)
		}
		vstorage, err := vdisk.GetIStorage()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to GetIStorage of vdisk %q", vdisk.GetName())
		}

		storageObj, err := db.FetchByExternalIdAndManagerId(StorageManager, vstorage.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", managerId)
		})
		if err != nil {
			return nil, errors.Wrapf(err, "cannot find storage of vdisk %s", vdisk.GetName())
		}
		storage := storageObj.(*SStorage)
		return manager.newFromCloudDisk(ctx, userCred, provider, vdisk, storage, -1, syncOwnerId)
	}
	return diskObj.(*SDisk), nil
}

func (manager *SDiskManager) SyncDisks(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, storage *SStorage, disks []cloudprovider.ICloudDisk, syncOwnerId mcclient.IIdentityProvider, xor bool) ([]SDisk, []cloudprovider.ICloudDisk, compare.SyncResult) {
	lockman.LockRawObject(ctx, manager.Keyword(), storage.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), storage.Id)

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
			// vm not sync, so skip disk used by vm error
			if errors.Cause(err) == httperrors.ErrNotEmpty {
				continue
			}
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		skip, key := IsNeedSkipSync(commonext[i])
		if skip {
			log.Infof("delete disk %s(%s) with tag key: %s", commonext[i].GetName(), commonext[i].GetGlobalId(), key)
			err := commondb[i].RealDelete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
				continue
			}
			syncResult.Delete()
			continue
		}
		if !xor {
			err = commondb[i].syncWithCloudDisk(ctx, userCred, provider, commonext[i], -1, syncOwnerId, storage.ManagerId)
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
		}
		localDisks = append(localDisks, commondb[i])
		remoteDisks = append(remoteDisks, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		skip, key := IsNeedSkipSync(added[i])
		if skip {
			log.Infof("skip disk %s(%s) sync with tag key: %s", added[i].GetName(), added[i].GetGlobalId(), key)
			continue
		}
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
			localDisks = append(localDisks, *new)
			remoteDisks = append(remoteDisks, added[i])
			syncResult.Add()
		}
	}

	return localDisks, remoteDisks, syncResult
}

func (self *SDisk) syncDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, idisk cloudprovider.ICloudDisk, managerId string) error {
	istorage, err := idisk.GetIStorage()
	if err != nil {
		return errors.Wrapf(err, "idisk.GetIStorage %s", idisk.GetGlobalId())
	}
	storageExtId := istorage.GetGlobalId()
	storage, err := db.FetchByExternalIdAndManagerId(StorageManager, storageExtId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", managerId)
	})
	if err != nil {
		return errors.Wrapf(err, "storage db.FetchByExternalIdAndManagerId(%s)", storageExtId)
	}
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.StorageId = storage.GetId()
		self.Status = idisk.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.UpdateWithLock")
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SDisk) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	storage, _ := self.GetStorage()
	if storage == nil {
		return nil, fmt.Errorf("failed to get storage for disk %s(%s)", self.Name, self.Id)
	}

	provider, err := storage.GetDriver(ctx)
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for storage %s(%s) error: %v", storage.Name, storage.Id, err)
	}

	if provider.GetFactory().IsOnPremise() {
		return provider.GetOnPremiseIRegion()
	}
	region, err := storage.GetRegion()
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SDisk) syncRemoveCloudDisk(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	iregion, err := self.GetIRegion(ctx)
	if err != nil {
		return err
	}
	iDisk, err := iregion.GetIDiskById(self.ExternalId)
	if err == nil {
		if storageId := iDisk.GetIStorageId(); len(storageId) > 0 {
			storage, err := db.FetchByExternalIdAndManagerId(StorageManager, storageId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				if s, _ := self.GetStorage(); s != nil {
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

	err = self.validateDeleteCondition(ctx, true)
	if err != nil {
		self.SetStatus(ctx, userCred, api.DISK_UNKNOWN, "missing original disk after sync")
		return err
	}
	err = self.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (self *SDisk) syncWithCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, index int, syncOwnerId mcclient.IIdentityProvider, managerId string) error {
	recycle := false
	guests := self.GetGuests()
	if provider.GetFactory().IsSupportPrepaidResources() && len(guests) == 1 && guests[0].IsPrepaidRecycle() {
		recycle = true
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, extDisk.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.Status = extDisk.GetStatus()
		self.DiskFormat = extDisk.GetDiskFormat()
		self.DiskSize = extDisk.GetDiskSizeMB()
		self.AccessPath = extDisk.GetAccessPath()
		self.Preallocation = extDisk.GetPreallocation()
		if iops := extDisk.GetIops(); iops > 0 {
			self.Iops = iops
		}
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
		return errors.Wrapf(err, "db.UpdateWithLock")
	}
	storage, err := self.GetStorage()
	if err != nil {
		return errors.Wrapf(err, "GetStorage")
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	if account := storage.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, extDisk, account.ReadOnly)
	}

	if len(guests) == 0 {
		if provider := storage.GetCloudprovider(); provider != nil {
			SyncCloudProject(ctx, userCred, self, syncOwnerId, extDisk, provider)
		}
	} else {
		self.SyncCloudProjectId(userCred, guests[0].GetOwnerId())
	}

	return nil
}

func (manager *SDiskManager) newFromCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, storage *SStorage, index int, syncOwnerId mcclient.IIdentityProvider) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(manager, &disk)

	disk.Status = extDisk.GetStatus()
	disk.ExternalId = extDisk.GetGlobalId()
	disk.StorageId = storage.Id

	disk.Iops = extDisk.GetIops()
	disk.DiskFormat = extDisk.GetDiskFormat()
	disk.DiskSize = extDisk.GetDiskSizeMB()
	disk.AutoDelete = extDisk.GetIsAutoDelete()
	disk.Preallocation = extDisk.GetPreallocation()
	disk.DiskType = extDisk.GetDiskType()
	if index == 0 {
		disk.DiskType = api.DISK_TYPE_SYS
	}
	disk.Nonpersistent = extDisk.GetIsNonPersistent()

	disk.IsEmulated = extDisk.IsEmulated()

	if templateId := extDisk.GetTemplateId(); len(templateId) > 0 {
		cachedImage, err := db.FetchByExternalId(CachedimageManager, templateId)
		if err == nil && cachedImage != nil {
			disk.TemplateId = cachedImage.GetId()
		}
	}

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

	syncVirtualResourceMetadata(ctx, userCred, &disk, extDisk, false)

	if provider := storage.GetCloudprovider(); provider != nil {
		SyncCloudProject(ctx, userCred, &disk, syncOwnerId, extDisk, provider)
	}

	db.OpsLog.LogEvent(&disk, db.ACT_CREATE, disk.GetShortDesc(ctx), userCred)

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &disk,
		Action: notifyclient.ActionSyncCreate,
	})

	return &disk, nil
}

func totalDiskSize(
	scope rbacscope.TRbacScope,
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
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		q = q.Filter(sqlchemy.Equals(disks.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacscope.ScopeProject:
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
	if info.Storage != "" {
		if err := fillDiskConfigByStorage(ctx, userCred, info, info.Storage); err != nil {
			return nil, errors.Wrap(err, "fillDiskConfigByStorage")
		}
	}
	if info.DiskId != "" {
		if err := fillDiskConfigByDisk(ctx, userCred, info, info.DiskId); err != nil {
			return nil, errors.Wrap(err, "fillDiskConfigByDisk")
		}
	}
	if info.SnapshotId != "" {
		if err := fillDiskConfigBySnapshot(ctx, userCred, info, info.SnapshotId); err != nil {
			return nil, errors.Wrap(err, "fillDiskConfigBySnapshot")
		}
	}
	if info.BackupId != "" {
		if err := fillDiskConfigByBackup(ctx, userCred, info, info.BackupId); err != nil {
			return nil, errors.Wrap(err, "fillDiskConfigByBackup")
		}
	}
	if info.ImageId != "" {
		if err := fillDiskConfigByImage(ctx, userCred, info, info.ImageId); err != nil {
			if len(info.SnapshotId) == 0 && len(info.BackupId) == 0 {
				// return error only if no valid snapshotId and backId
				// otherwise, the disk was crated by snapshot or backup, not depends on vald image info
				return nil, errors.Wrap(err, "fillDiskConfigByImage")
			}
		}
	}
	if info.ExistingPath != "" {
		info.ExistingPath = strings.TrimSpace(info.ExistingPath)
		_, err := filepath.Rel("/", info.ExistingPath)
		if err != nil {
			return nil, errors.Wrap(err, "invaild existing path")
		}
	}

	// XXX: do not set default disk size here, set it by each hypervisor driver
	// if len(diskConfig.ImageId) > 0 && diskConfig.SizeMb == 0 {
	// 	diskConfig.SizeMb = options.Options.DefaultDiskSize // MB
	// else
	if len(info.ImageId) == 0 && info.SizeMb == 0 && info.ExistingPath == "" && info.NVMEDevice == nil {
		return nil, httperrors.NewInputParameterError("Diskinfo index %d: both imageID and size are absent", info.Index)
	}
	return info, nil
}

func fillDiskConfigBySnapshot(ctx context.Context, userCred mcclient.TokenCredential, diskConfig *api.DiskConfig, snapshotId string) error {
	iSnapshot, err := SnapshotManager.FetchByIdOrName(ctx, userCred, snapshotId)
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

func fillDiskConfigByBackup(ctx context.Context, userCred mcclient.TokenCredential, diskConfig *api.DiskConfig, backupId string) error {
	iBakcup, err := DiskBackupManager.FetchByIdOrName(ctx, userCred, backupId)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewNotFoundError("Backup %s not found", backupId)
		}
		return err
	}
	backup := iBakcup.(*SDiskBackup)
	if diskConfig.DiskType == "" {
		diskConfig.DiskType = backup.DiskType
	}
	diskConfig.BackupId = backup.GetId()
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
		diskConfig.ImageEncryptKeyId = image.EncryptKeyId
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

func fillDiskConfigByDisk(ctx context.Context, userCred mcclient.TokenCredential,
	diskConfig *api.DiskConfig, diskId string) error {
	diskObj, err := DiskManager.FetchByIdOrName(ctx, userCred, diskId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError2("disk", diskId)
		} else {
			return errors.Wrapf(err, "DiskManager.FetchByIdOrName %s", diskId)
		}
	}
	disk := diskObj.(*SDisk)
	if disk.Status != api.DISK_READY {
		return errors.Wrapf(httperrors.ErrInvalidStatus, "disk status %s not ready", disk.Status)
	}
	guests := disk.GetGuests()
	if len(guests) > 0 {
		return errors.Wrapf(httperrors.ErrInvalidStatus, "disk %s has been used", diskId)
	}

	diskConfig.DiskId = disk.Id
	diskConfig.SizeMb = disk.DiskSize
	if disk.SnapshotId != "" {
		diskConfig.SnapshotId = disk.SnapshotId
	}
	if disk.TemplateId != "" {
		diskConfig.ImageId = disk.TemplateId
	}
	if disk.OsArch != "" {
		diskConfig.OsArch = disk.OsArch
	}
	storage, err := disk.GetStorage()
	if err != nil {
		return errors.Wrap(err, "disk.GetStorage")
	}
	if !storage.Enabled.IsTrue() {
		return errors.Wrap(httperrors.ErrInvalidStatus, "storage not enabled")
	}
	if storage.Status != api.STORAGE_ONLINE {
		return errors.Wrap(httperrors.ErrInvalidStatus, "storage not online")
	}
	diskConfig.Storage = disk.StorageId
	return nil
}

func fillDiskConfigByStorage(ctx context.Context, userCred mcclient.TokenCredential,
	diskConfig *api.DiskConfig, storageId string) error {
	storageObj, err := StorageManager.FetchByIdOrName(ctx, userCred, storageId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError2("storage", storageId)
		} else {
			return errors.Wrapf(err, "StorageManager.FetchByIdOrName %s", storageId)
		}
	}
	storage := storageObj.(*SStorage)
	if !storage.Enabled.IsTrue() {
		return errors.Wrap(httperrors.ErrInvalidStatus, "storage not enabled")
	}
	if storage.Status != api.STORAGE_ONLINE {
		return errors.Wrap(httperrors.ErrInvalidStatus, "storage not online")
	}
	if storage.StorageType == api.STORAGE_NVME_PT {
		return httperrors.NewBadRequestError("storage type %s require assign isolated device", api.STORAGE_NVME_PT)
	}
	diskConfig.Storage = storage.Id
	diskConfig.Backend = storage.StorageType
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
	if len(diskConfig.SnapshotId) > 0 {
		self.SnapshotId = diskConfig.SnapshotId
		self.DiskType = diskConfig.DiskType
	}
	if len(diskConfig.BackupId) > 0 {
		self.BackupId = diskConfig.BackupId
		self.DiskType = diskConfig.DiskType
	}
	if len(diskConfig.ImageId) > 0 {
		self.TemplateId = diskConfig.ImageId
		// support for create vm from guest image
		if len(diskConfig.DiskType) == 0 {
			self.DiskType = api.DISK_TYPE_SYS
		} else {
			self.DiskType = diskConfig.DiskType
		}
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
	storage, _ := self.GetStorage()
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
	storage, _ := self.GetStorage()
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
	// diskbackups := DiskBackupManager.Query("id").Equals("disk_id", self.Id)
	guestdisks := GuestdiskManager.Query("row_id").Equals("disk_id", self.Id)
	diskpolicies := SnapshotPolicyDiskManager.Query("row_id").Equals("disk_id", self.Id)
	pairs := []purgePair{
		// {manager: DiskBackupManager, key: "id", q: diskbackups},
		{manager: GuestdiskManager, key: "row_id", q: guestdisks},
		{manager: SnapshotPolicyDiskManager, key: "row_id", q: diskpolicies},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
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

func (self *SDisk) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.validateDeleteCondition(ctx, true)
	if err != nil {
		return nil, err
	}

	provider := self.GetCloudprovider()
	if provider != nil && utils.IsInStringArray(provider.Provider, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
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
		if provider := self.GetCloudprovider(); provider != nil && utils.IsInStringArray(provider.Provider, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS}) {
			cnt, err := self.GetSnapshotCount()
			if err != nil {
				return httperrors.NewInternalServerError("GetSnapshotCount fail %s", err)
			}
			if cnt > 0 {
				return httperrors.NewForbiddenError("not allow to delete. Virtual disk must not have snapshots")
			}
		} else if storage, _ := self.GetStorage(); storage != nil && storage.StorageType == api.STORAGE_RBD {
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
	encRows := manager.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	diskIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.DiskDetails{
			VirtualResourceDetails: virtRows[i],
			StorageResourceInfo:    storeRows[i],

			EncryptedResourceDetails: encRows[i],
		}

		disk := objs[i].(*SDisk)
		if disk.PendingDeleted {
			pendingDeletedAt := disk.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
			rows[i].AutoDeleteAt = pendingDeletedAt
		}
		diskIds[i] = disk.Id
	}

	guestSQ := GuestManager.Query().SubQuery()
	gds := GuestdiskManager.Query().SubQuery()
	q := guestSQ.Query(
		guestSQ.Field("id"),
		guestSQ.Field("name"),
		guestSQ.Field("status"),
		gds.Field("disk_id"),
		gds.Field("index"),
		gds.Field("driver"),
		gds.Field("cache_mode"),
	).
		Join(gds, sqlchemy.Equals(gds.Field("guest_id"), guestSQ.Field("id"))).
		Filter(sqlchemy.In(gds.Field("disk_id"), diskIds))

	guestInfo := []struct {
		Id     string
		Name   string
		Status string
		DiskId string

		Index     int
		Driver    string
		CacheMode string
	}{}
	err := q.All(&guestInfo)
	if err != nil {
		log.Errorf("query disk guest info error: %v", err)
		return rows
	}

	guests := map[string][]api.SimpleGuest{}
	for _, guest := range guestInfo {
		_, ok := guests[guest.DiskId]
		if !ok {
			guests[guest.DiskId] = []api.SimpleGuest{}
		}
		guests[guest.DiskId] = append(guests[guest.DiskId], api.SimpleGuest{
			Id:     guest.Id,
			Name:   guest.Name,
			Status: guest.Status,

			Index:     guest.Index,
			Driver:    guest.Driver,
			CacheMode: guest.CacheMode,
		})
	}

	policySQ := SnapshotPolicyManager.Query().SubQuery()
	dps := SnapshotPolicyDiskManager.Query().SubQuery()

	q = policySQ.Query(
		policySQ.Field("id"),
		policySQ.Field("name"),
		policySQ.Field("time_points"),
		policySQ.Field("repeat_weekdays"),
		dps.Field("disk_id"),
	).Join(dps, sqlchemy.Equals(dps.Field("snapshotpolicy_id"), policySQ.Field("id"))).
		Filter(sqlchemy.In(dps.Field("disk_id"), diskIds))

	policyInfo := []struct {
		Id             string
		Name           string
		Status         string
		TimePoints     []int
		RepeatWeekdays []int
		DiskId         string
	}{}
	err = q.All(&policyInfo)
	if err != nil {
		log.Errorf("query disk snapshot policy info error: %v", err)
		return rows
	}

	policies := map[string][]api.SimpleSnapshotPolicy{}
	for _, policy := range policyInfo {
		_, ok := policies[policy.DiskId]
		if !ok {
			policies[policy.DiskId] = []api.SimpleSnapshotPolicy{}
		}
		policies[policy.DiskId] = append(policies[policy.DiskId], api.SimpleSnapshotPolicy{
			Id:             policy.Id,
			Name:           policy.Name,
			RepeatWeekdays: policy.RepeatWeekdays,
			TimePoints:     policy.TimePoints,
		})
	}

	storageSQ := StorageManager.Query().SubQuery()
	diskSQ := DiskManager.Query().SubQuery()
	q = storageSQ.Query(
		storageSQ.Field("storage_type"),
		diskSQ.Field("id").Label("disk_id"),
	).Join(diskSQ, sqlchemy.Equals(diskSQ.Field("storage_id"), storageSQ.Field("id"))).
		Filter(sqlchemy.In(diskSQ.Field("id"), diskIds))

	storageInfo := []struct {
		StorageType string
		DiskId      string
	}{}
	err = q.All(&storageInfo)
	if err != nil {
		log.Errorf("query disk storage info error: %v", err)
		return rows
	}

	storages := map[string]string{}
	for _, storage := range storageInfo {
		storages[storage.DiskId] = storage.StorageType
	}

	snapshotSQ := SnapshotManager.Query().SubQuery()
	q = snapshotSQ.Query(
		snapshotSQ.Field("id"),
		diskSQ.Field("id").Label("disk_id"),
	).Join(diskSQ, sqlchemy.Equals(diskSQ.Field("id"), snapshotSQ.Field("disk_id"))).
		Filter(
			sqlchemy.AND(
				sqlchemy.In(diskSQ.Field("id"), diskIds),
				sqlchemy.Equals(snapshotSQ.Field("created_by"), api.SNAPSHOT_MANUAL),
				sqlchemy.Equals(snapshotSQ.Field("fake_deleted"), false),
			),
		)

	snapshotInfo := []struct {
		Id     string
		DiskId string
	}{}
	err = q.All(&snapshotInfo)
	if err != nil {
		log.Errorf("query disk snapshot info error: %v", err)
		return rows
	}
	snapshots := map[string][]string{}
	for _, snapshot := range snapshotInfo {
		_, ok := snapshots[snapshot.DiskId]
		if !ok {
			snapshots[snapshot.DiskId] = []string{}
		}
		snapshots[snapshot.DiskId] = append(snapshots[snapshot.DiskId], snapshot.Id)
	}

	for i := range rows {
		rows[i].Guests, _ = guests[diskIds[i]]
		names, status := []string{}, []string{}
		for _, guest := range rows[i].Guests {
			names = append(names, guest.Name)
			status = append(status, guest.Status)
		}
		rows[i].GuestCount = len(rows[i].Guests)
		rows[i].Guest = strings.Join(names, ",")
		rows[i].GuestStatus = strings.Join(status, ",")

		rows[i].Snapshotpolicies, _ = policies[diskIds[i]]

		storageType, ok := storages[diskIds[i]]
		if ok && utils.IsInStringArray(storageType, append(api.SHARED_FILE_STORAGE, api.STORAGE_LOCAL)) {
			rows[i].MaxManualSnapshotCount = options.Options.DefaultMaxManualSnapshotCount
			snps, _ := snapshots[diskIds[i]]
			rows[i].ManualSnapshotCount = len(snps)
		}
		disk := objs[i].(*SDisk)
		if len(disk.StorageId) == 0 && disk.Status == api.VM_SCHEDULE_FAILED {
			rows[i].Brand = "Unknown"
			rows[i].Provider = "Unknown"
		}
	}

	return rows
}

func (self *SDisk) StartDiskResizeTask(ctx context.Context, userCred mcclient.TokenCredential, sizeMb int64, parentTaskId string, pendingUsage quotas.IQuota) error {
	self.SetStatus(ctx, userCred, api.DISK_START_RESIZE, "StartDiskResizeTask")
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
	self.SetStatus(ctx, userCred, api.DISK_READY, reason)
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
	storage, _ := self.GetStorage()
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
	desc.Add(jsonutils.NewString(self.DiskType), "disk_type")
	storage, _ := self.GetStorage()
	if storage != nil {
		desc.Add(jsonutils.NewString(storage.StorageType), "storage_type")
		desc.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	}

	if hypervisor := self.GetMetadata(ctx, "hypervisor", nil); len(hypervisor) > 0 {
		desc.Add(jsonutils.NewString(hypervisor), "hypervisor")
	}

	if len(self.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(self.ExternalId), "externalId")
	}

	if self.IsSsd {
		desc.Add(jsonutils.JSONTrue, "is_ssd")
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

	if priceKey := self.GetMetadata(ctx, "ext:price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (self *SDisk) getDev() string {
	return self.GetMetadata(context.Background(), "dev", nil)
}

func (self *SDisk) GetMountPoint() string {
	return self.GetMetadata(context.Background(), "mountpoint", nil)
}

func (self *SDisk) isReady() bool {
	return self.Status == api.DISK_READY
}

func (self *SDisk) isInit() bool {
	return self.Status == api.DISK_INIT
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
		disks[i].StartDiskDeleteTask(ctx, userCred, "", false, true, false)
	}
}

func (manager *SDiskManager) GetNeedAutoSnapshotDisks() ([]SSnapshotPolicyDisk, error) {
	tz, _ := time.LoadLocation(options.Options.TimeZone)
	t := time.Now().In(tz)
	week := t.Weekday()
	if week == 0 { // sunday is zero
		week += 7
	}
	timePoint := t.Hour()

	policy := SnapshotPolicyManager.Query().Equals("cloudregion_id", api.DEFAULT_REGION_ID)
	policy = policy.Filter(sqlchemy.Contains(policy.Field("repeat_weekdays"), fmt.Sprintf("%d", week)))
	sq := policy.Filter(
		sqlchemy.OR(
			sqlchemy.Contains(policy.Field("time_points"), fmt.Sprintf(",%d,", timePoint)),
			sqlchemy.Startswith(policy.Field("time_points"), fmt.Sprintf("[%d,", timePoint)),
			sqlchemy.Endswith(policy.Field("time_points"), fmt.Sprintf(",%d]", timePoint)),
			sqlchemy.Equals(policy.Field("time_points"), fmt.Sprintf("[%d]", timePoint)),
		),
	).SubQuery()
	disks := DiskManager.Query().SubQuery()
	q := SnapshotPolicyDiskManager.Query()
	q = q.Join(sq, sqlchemy.Equals(q.Field("snapshotpolicy_id"), sq.Field("id")))
	q = q.Join(disks, sqlchemy.Equals(q.Field("disk_id"), disks.Field("id")))
	ret := []SSnapshotPolicyDisk{}
	err := db.FetchModelObjects(SnapshotPolicyDiskManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (disk *SDisk) validateDiskAutoCreateSnapshot() error {
	guests := disk.GetGuests()
	if len(guests) == 0 {
		return fmt.Errorf("Disks %s not attach guest, can't create snapshot", disk.GetName())
	}
	storage, err := disk.GetStorage()
	if err != nil {
		return errors.Wrapf(err, "GetStorage")
	}
	if len(guests) == 1 && utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
		if !utils.IsInStringArray(guests[0].Status, []string{api.VM_RUNNING, api.VM_READY}) {
			return fmt.Errorf("Guest(%s) in status(%s) cannot do disk snapshot", guests[0].Id, guests[0].Status)
		}
	}
	return nil
}

func (manager *SDiskManager) AutoDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	disks, err := manager.GetNeedAutoSnapshotDisks()
	if err != nil {
		log.Errorf("Get auto snapshot disks id failed: %s", err)
		return
	}
	log.Debugf("auto snapshot %d disks", len(disks))
	now := time.Now()
	for i := 0; i < len(disks); i++ {
		disk, err := disks[i].GetDisk()
		if err != nil {
			log.Errorf("get disk error: %v", err)
			continue
		}
		autoSnapshotCount := options.Options.DefaultMaxSnapshotCount - options.Options.DefaultMaxManualSnapshotCount

		err = func() error {
			policy, err := disks[i].GetSnapshotPolicy()
			if err != nil {
				return errors.Wrapf(err, "GetSnapshotPolicy")
			}

			if len(disk.ExternalId) == 0 {
				err = disk.validateDiskAutoCreateSnapshot()
				if err != nil {
					return errors.Wrapf(err, "validateDiskAutoCreateSnapshot")
				}
			}

			snapshot, err := disk.CreateSnapshotAuto(ctx, userCred, policy)
			if err != nil {
				return errors.Wrapf(err, "CreateSnapshotAuto")
			}

			snapCount, err := SnapshotManager.Query().Equals("fake_deleted", false).
				Equals("disk_id", disk.Id).Equals("created_by", api.SNAPSHOT_AUTO).
				CountWithError()
			if err != nil {
				return errors.Wrap(err, "get snapshot count")
			}
			// if auto snapshot count gt max auto snapshot count, do clean overdued snapshots
			cleanOverdueSnapshots := snapCount > autoSnapshotCount
			if cleanOverdueSnapshots {
				disk.CleanOverdueSnapshots(ctx, userCred, policy, now)
			}
			db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT, snapshot.Name, userCred)
			policy.ExecuteNotify(ctx, userCred, disk.GetName())
			return nil
		}()
		if err != nil {
			log.Errorf("auto snapshot %s error: %v", disk.Name, err)
			db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, err.Error(), userCred)
			notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, errors.Wrapf(err, "Disk auto create snapshot").Error())
		}
	}

}

func (self *SDisk) CreateSnapshotAuto(
	ctx context.Context, userCred mcclient.TokenCredential,
	policy *SSnapshotPolicy,
) (*SSnapshot, error) {
	storage, err := self.GetStorage()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage")
	}
	snapshot := &SSnapshot{}
	snapshot.SetModelManager(SnapshotManager, snapshot)
	snapshot.ProjectId = self.ProjectId
	snapshot.DomainId = self.DomainId
	snapshot.DiskId = self.Id
	if len(self.ExternalId) == 0 {
		snapshot.StorageId = self.StorageId
	}

	// inherit encrypt_key_id
	snapshot.EncryptKeyId = self.EncryptKeyId

	driver, err := storage.GetRegionDriver()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegionDriver")
	}
	snapshot.OutOfChain = driver.SnapshotIsOutOfChain(self)
	snapshot.Size = self.DiskSize
	snapshot.DiskType = self.DiskType
	snapshot.Location = ""
	snapshot.CreatedBy = api.SNAPSHOT_AUTO
	snapshot.ManagerId = storage.ManagerId
	if cloudregion, _ := storage.GetRegion(); cloudregion != nil {
		snapshot.CloudregionId = cloudregion.GetId()
	}
	snapshot.Name = fmt.Sprintf("%s-auto-snapshot-%d", self.Name, time.Now().Unix())
	snapshot.Status = api.SNAPSHOT_CREATING
	if policy.RetentionDays > 0 {
		snapshot.ExpiredAt = time.Now().AddDate(0, 0, policy.RetentionDays)
	}
	snapshot.IsSystem = self.IsSystem
	err = SnapshotManager.TableSpec().Insert(ctx, snapshot)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	db.OpsLog.LogEvent(snapshot, db.ACT_CREATE, "disk create snapshot auto", userCred)
	err = snapshot.StartSnapshotCreateTask(ctx, userCred, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "disk auto snapshot start snapshot task")
	}
	return snapshot, nil
}

func (self *SDisk) CleanOverdueSnapshots(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, now time.Time) error {
	kwargs := jsonutils.NewDict()
	kwargs.Set("retention_day", jsonutils.NewInt(int64(sp.RetentionDays)))
	kwargs.Set("start_time", jsonutils.NewTimeString(now))
	task, err := taskman.TaskManager.NewTask(ctx, "DiskCleanOverduedSnapshots", self, userCred, kwargs, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
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
		return errors.Wrapf(err, "SaveRenewInfo.Update")
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
	storage, _ := self.GetStorage()
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
	storage, _ := self.GetStorage()
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

func (disk *SDisk) PerformBindSnapshotpolicy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.DiskSnapshotpolicyInput,
) (jsonutils.JSONObject, error) {
	spObj, err := validators.ValidateModel(ctx, userCred, SnapshotPolicyManager, &input.SnapshotpolicyId)
	if err != nil {
		return nil, err
	}
	sp := spObj.(*SSnapshotPolicy)
	if len(sp.ManagerId) > 0 {
		storage, err := disk.GetStorage()
		if err != nil {
			return nil, errors.Wrapf(err, "GetStorage")
		}
		if storage.ManagerId != sp.ManagerId {
			return nil, httperrors.NewConflictError("The snapshot policy %s and disk account are different", sp.Name)
		}
		zone, err := storage.GetZone()
		if err != nil {
			return nil, errors.Wrapf(err, "GetZone")
		}
		if sp.CloudregionId != zone.CloudregionId {
			return nil, httperrors.NewConflictError("The snapshot policy %s and the disk are in different region", sp.Name)
		}
	}
	return nil, sp.StartBindDisksTask(ctx, userCred, []string{disk.Id})
}

func (disk *SDisk) PerformUnbindSnapshotpolicy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.DiskSnapshotpolicyInput,
) (jsonutils.JSONObject, error) {
	spObj, err := validators.ValidateModel(ctx, userCred, SnapshotPolicyManager, &input.SnapshotpolicyId)
	if err != nil {
		return nil, err
	}
	sp := spObj.(*SSnapshotPolicy)
	return nil, sp.StartUnbindDisksTask(ctx, userCred, []string{disk.Id})
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
