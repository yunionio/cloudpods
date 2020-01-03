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
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
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
)

type SDiskManager struct {
	db.SVirtualResourceBaseManager
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

	DiskFormat string `width:"32" charset:"ascii" nullable:"false" default:"qcow2" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=False, default='qcow2')
	DiskSize   int    `nullable:"false" list:"user"`                                            // Column(Integer, nullable=False) # in MB
	AccessPath string `width:"256" charset:"ascii" nullable:"true" get:"user"`                  // = Column(VARCHAR(256, charset='ascii'), nullable=True)

	AutoDelete bool `nullable:"false" default:"false" get:"user" update:"user"` // Column(Boolean, nullable=False, default=False)

	StorageId       string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"optional"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	BackupStorageId string `width:"128" charset:"ascii" nullable:"true" list:"admin"`

	// # backing template id and type
	TemplateId string `width:"256" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=True)
	// backing snapshot id
	SnapshotId string `width:"256" charset:"ascii" nullable:"true" list:"user"`

	// # file system
	FsFormat string `width:"32" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// # disk type, OS, SWAP, DAT, VOLUME
	DiskType string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// # is persistent
	Nonpersistent bool `default:"false" list:"user"` // Column(Boolean, default=False)
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

func (manager *SDiskManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid querystring format")
	}

	var err error
	storages := StorageManager.Query().SubQuery()
	q, err = managedResourceFilterByAccount(q, query, "storage_id", func() *sqlchemy.SQuery {
		return storages.Query(storages.Field("id"))
	})
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "storage_id", func() *sqlchemy.SQuery {
		return storages.Query(storages.Field("id"))
	})

	billingTypeStr, _ := queryDict.GetString("billing_type")
	if len(billingTypeStr) > 0 {
		if billingTypeStr == billing_api.BILLING_TYPE_POSTPAID {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(q.Field("billing_type")),
					sqlchemy.Equals(q.Field("billing_type"), billingTypeStr),
				),
			)
		} else {
			q = q.Equals("billing_type", billingTypeStr)
		}
		queryDict.Remove("billing_type")
	}

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	if query.Contains("unused") {
		guestdisks := GuestdiskManager.Query().SubQuery()
		sq := guestdisks.Query(guestdisks.Field("disk_id"))
		if jsonutils.QueryBoolean(query, "unused", false) {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		}
	}

	if jsonutils.QueryBoolean(query, "share", false) {
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.NotIn(storages.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	/*if jsonutils.QueryBoolean(query, "public_cloud", false) {
		sq :=
		sq = sq.Filter(sqlchemy.In(storages.Field("manager_id"), CloudproviderManager.GetPublicProviderIdsQuery()))

		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	if jsonutils.QueryBoolean(query, "private_cloud", false) {
		sq := storages.Query(storages.Field("id"))
		sq = sq.Filter(
			sqlchemy.OR(
				sqlchemy.In(storages.Field("manager_id"), CloudproviderManager.GetPrivateProviderIdsQuery()),
				sqlchemy.IsNullOrEmpty(storages.Field("manager_id")),
			),
		)
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	if jsonutils.QueryBoolean(query, "is_on_premise", false) {
		sq := storages.Query(storages.Field("id"))
		sq = sq.Filter(
			sqlchemy.OR(
				sqlchemy.In(storages.Field("manager_id"), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
				sqlchemy.IsNullOrEmpty(storages.Field("manager_id")),
			),
		)
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	if jsonutils.QueryBoolean(query, "is_managed", false) {
		sq := storages.Query(storages.Field("id"))
		sq = sq.Filter(sqlchemy.IsNotEmpty(storages.Field("manager_id")))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}*/

	if jsonutils.QueryBoolean(query, "local", false) {
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.In(storages.Field("storage_type"), api.STORAGE_LOCAL_TYPES))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	guestId, _ := queryDict.GetString("guest")
	if len(guestId) != 0 {
		iGuest, err := GuestManager.FetchByIdOrName(userCred, guestId)
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("guest %q not found", guestId)
		} else if err != nil {
			return nil, err
		}
		guest := iGuest.(*SGuest)
		hoststorages := HoststorageManager.Query().SubQuery()
		q = q.Join(hoststorages, sqlchemy.AND(
			sqlchemy.Equals(hoststorages.Field("host_id"), guest.HostId),
			sqlchemy.IsFalse(hoststorages.Field("deleted")))).
			Join(storages, sqlchemy.AND(
				sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
				sqlchemy.IsFalse(storages.Field("deleted")))).
			Filter(sqlchemy.Equals(storages.Field("id"), q.Field("storage_id")))
	}

	storageStr := jsonutils.GetAnyString(queryDict, []string{"storage", "storage_id"})
	if len(storageStr) > 0 {
		storageObj, err := StorageManager.FetchByIdOrName(userCred, storageStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("storage %s not found: %s", storageStr, err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("storage_id"), storageObj.GetId()))
	}

	/* managerStr := jsonutils.GetAnyString(query, []string{"manager", "cloudprovider", "cloudprovider_id", "manager_id"})
	if len(managerStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := storages.Query(storages.Field("id")).Equals("manager_id", provider.GetId())
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), subq.SubQuery()))
	}

	accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id", "cloudaccount", "cloudaccount_id"})
	if len(accountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}

		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := storages.Query(storages.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), storages.Field("manager_id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), account.GetId()))

		q = q.Filter(sqlchemy.In(q.Field("storage_id"), subq.SubQuery()))
	}

	if provier, _ := queryDict.GetString("provider"); len(provier) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id"))
		sq = sq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), storages.Field("manager_id")))
		sq = sq.Filter(sqlchemy.Equals(cloudproviders.Field("provider"), provier))

		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq.SubQuery()))
	}*/

	if diskType := jsonutils.GetAnyString(query, []string{"type", "disk_type"}); diskType != "" {
		q = q.Filter(sqlchemy.Equals(q.Field("disk_type"), diskType))
	}

	// for snapshotpolicy_id
	snapshotpolicyStr := jsonutils.GetAnyString(queryDict, []string{"snapshotpolicy", "snapshotpolicy_id"})
	if len(snapshotpolicyStr) > 0 {
		snapshotpolicyObj, err := SnapshotPolicyManager.FetchByIdOrName(userCred, snapshotpolicyStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("snapshotpolicy %s not found: %s", snapshotpolicyStr, err)
		}
		snapshotpolicyId := snapshotpolicyObj.GetId()
		sq := SnapshotPolicyDiskManager.Query("disk_id").Equals("snapshotpolicy_id", snapshotpolicyId)
		q = q.In("id", sq)
	}
	return q, nil
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

func (self *SDisk) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	storage := self.GetStorage()
	if storage == nil {
		return nil, httperrors.NewNotFoundError("failed to find storage for disk %s", self.Name)
	}

	host := storage.GetMasterHost()
	if host == nil {
		return nil, httperrors.NewNotFoundError("failed to find host for storage %s with disk %s", storage.Name, self.Name)
	}

	if diskType, _ := data.GetString("disk_type"); diskType != "" {
		if !utils.IsInStringArray(diskType, []string{api.DISK_TYPE_DATA, api.DISK_TYPE_VOLUME}) {
			return nil, httperrors.NewInputParameterError("not support update disk_type %s", diskType)
		}
	}

	data, err := host.GetHostDriver().ValidateUpdateDisk(ctx, userCred, data)
	if err != nil {
		return nil, err
	}

	return self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func diskCreateInput2ComputeQuotaKeys(input api.DiskCreateInput, ownerId mcclient.IIdentityProvider) SComputeResourceKeys {
	// input.Hypervisor must be set
	keys := GetDriver(input.Hypervisor).GetComputeQuotaKeys(
		rbacutils.ScopeProject,
		ownerId,
		"",
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

func (manager *SDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input, err := cmdline.FetchDiskCreateInputByJSON(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError("parse disk input: %v", err)
	}
	diskConfig := input.DiskConfig
	diskConfig, err = parseDiskInfo(ctx, userCred, diskConfig)
	if err != nil {
		return nil, err
	}
	input.Project = ownerId.GetProjectId()
	input.Domain = ownerId.GetProjectDomainId()

	var quotaKey quotas.IQuotaKeys

	storageID := input.Storage
	if storageID != "" {
		storageObj, err := StorageManager.FetchByIdOrName(nil, storageID)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Storage %s not found", storageID)
		}
		storage := storageObj.(*SStorage)

		provider := storage.GetCloudprovider()
		if provider != nil {
			if !provider.Enabled {
				return nil, httperrors.NewInputParameterError("provider %s(%s) is disabled, you need enable provider first", provider.Name, provider.Id)
			}
			if !utils.IsInStringArray(provider.Status, api.CLOUD_PROVIDER_VALID_STATUS) {
				return nil, httperrors.NewInputParameterError("invalid provider %s(%s) status %s, require status is %s", provider.Name, provider.Id, provider.Status, api.CLOUD_PROVIDER_VALID_STATUS)
			}
			if !utils.IsInStringArray(provider.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS) {
				return nil, httperrors.NewInputParameterError("invalid provider %s(%s) health status %s, require status is %s", provider.Name, provider.Id, provider.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS)
			}
		}

		host := storage.GetMasterHost()
		if host == nil {
			return nil, httperrors.NewResourceNotFoundError("storage %s(%s) need online and attach host for create disk", storage.Name, storage.Id)
		}
		input.Hypervisor = host.GetHostDriver().GetHypervisor()
		if len(diskConfig.Backend) == 0 {
			diskConfig.Backend = storage.StorageType
		}
		err = manager.validateDiskOnStorage(diskConfig, storage)
		if err != nil {
			return nil, err
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
		diskConfig.Backend = api.STORAGE_LOCAL
		serverInput, err := ValidateScheduleCreateData(ctx, userCred, input.ToServerCreateInput(), input.Hypervisor)
		if err != nil {
			return nil, err
		}
		input = serverInput.ToDiskCreateInput()
		quotaKey = diskCreateInput2ComputeQuotaKeys(*input, ownerId)
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}

	pendingUsage := SQuota{Storage: diskConfig.SizeMb}
	pendingUsage.SetKeys(quotaKey)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}
	return input.JSON(input), nil
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
			return httperrors.NewInputParameterError(err.Error())
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
		storages := host.GetAttachedStorages("")
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
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskCreateTask", self, userCred, kwargs, parentTaskId, "", nil); err != nil {
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
		return host.GetHostDriver().RequestAllocateDiskOnStorage(ctx, host, storage, self, task, content)
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
	if host := storage.GetMasterHost(); host != nil {
		if err := host.GetHostDriver().ValidateDiskSize(storage, sizeMb>>10); err != nil {
			return httperrors.NewInputParameterError(err.Error())
		}
	}
	if int64(addDisk) > storage.GetFreeCapacity() && !storage.IsEmulated {
		return httperrors.NewOutOfResourceError("Not enough free space")
	}
	if guest != nil {
		if err := guest.ValidateResizeDisk(disk, storage); err != nil {
			return httperrors.NewInputParameterError(err.Error())
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

func (self *SDisk) PrepareSaveImage(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (string, error) {
	if zone := self.GetZone(); zone == nil {
		return "", httperrors.NewResourceNotFoundError("No zone for this disk")
	}
	data.Add(jsonutils.NewString(self.DiskFormat), "disk_format")
	name, _ := data.GetString("name")
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	imageList, err := modules.Images.List(s, jsonutils.Marshal(map[string]string{"name": name, "admin": "true"}))
	if err != nil {
		return "", err
	}
	if imageList.Total > 0 {
		return "", httperrors.NewConflictError("Duplicate image name %s", name)
	}
	/*
		no need to check quota anymore
		session := auth.GetSession(userCred, options.Options.Region, "v2")
		quota := image_models.SQuota{Image: 1}
		if _, err := modules.ImageQuotas.DoQuotaCheck(session, jsonutils.Marshal(&quota)); err != nil {
			return "", err
		}*/
	us := auth.GetSession(ctx, userCred, options.Options.Region, "")
	data.Add(jsonutils.NewInt(int64(self.DiskSize)), "virtual_size")
	result, err := modules.Images.Create(us, data)
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

func (self *SDisk) PerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

	if name, err := data.GetString("name"); err != nil || len(name) == 0 {
		return nil, httperrors.NewInputParameterError("Image name is required")
	}
	kwargs := data.(*jsonutils.JSONDict)
	if imageId, err := self.PrepareSaveImage(ctx, userCred, kwargs); err != nil {
		return nil, err
	} else {
		kwargs.Add(jsonutils.NewString(imageId), "image_id")
		return nil, self.StartDiskSaveTask(ctx, userCred, kwargs, "")
	}
}

func (self *SDisk) StartDiskSaveTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DISK_START_SAVE, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskSaveTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorf("Start DiskSaveTask failed:%v", err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) ValidateDeleteCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, false)
}

func (self *SDisk) ValidatePurgeCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, true)
}

func (self *SDisk) validateDeleteCondition(ctx context.Context, isPurge bool) error {
	cnt, err := self.GetGuestDiskCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestDiskCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Virtual disk used by virtual servers")
	}
	if !isPurge && self.IsValidPrePaid() {
		return httperrors.NewForbiddenError("not allow to delete prepaid disk in valid status")
	}
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SDisk) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	provider := self.GetCloudprovider()
	if provider != nil && !provider.IsAvailable() {
		return false
	}

	account := provider.GetCloudaccount()
	if account != nil && !account.IsAvailable() {
		return false
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

func (manager *SDiskManager) syncCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, vdisk cloudprovider.ICloudDisk, index int, syncOwnerId mcclient.IIdentityProvider) (*SDisk, error) {
	// ownerProjId := projectId

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

	diskObj, err := db.FetchByExternalId(manager, vdisk.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			vstorage, _ := vdisk.GetIStorage()

			storageObj, err := db.FetchByExternalId(StorageManager, vstorage.GetGlobalId())
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
		err = disk.syncWithCloudDisk(ctx, userCred, provider, vdisk, index, syncOwnerId)
		if err != nil {
			return nil, err
		}
		return disk, nil
	}
}

func (manager *SDiskManager) SyncDisks(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, storage *SStorage, disks []cloudprovider.ICloudDisk, syncOwnerId mcclient.IIdentityProvider) ([]SDisk, []cloudprovider.ICloudDisk, compare.SyncResult) {
	// syncOwnerId := projectId

	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, syncOwnerId))

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
		err = commondb[i].syncWithCloudDisk(ctx, userCred, provider, commonext[i], -1, syncOwnerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localDisks = append(localDisks, commondb[i])
			remoteDisks = append(remoteDisks, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		extId := added[i].GetGlobalId()
		_disk, err := db.FetchByExternalId(manager, extId)
		if err != nil && err != sql.ErrNoRows {
			//主要是显示duplicate err及 general err,方便排错
			msg := fmt.Errorf("failed to found disk by external Id %s error: %v", extId, err)
			syncResult.Error(msg)
			continue
		}
		if _disk != nil {
			disk := _disk.(*SDisk)
			err = disk.syncDiskStorage(ctx, userCred, added[i])
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
			syncMetadata(ctx, userCred, new, added[i])
			localDisks = append(localDisks, *new)
			remoteDisks = append(remoteDisks, added[i])
			syncResult.Add()
		}
	}

	return localDisks, remoteDisks, syncResult
}

func (self *SDisk) syncDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, idisk cloudprovider.ICloudDisk) error {
	extId := idisk.GetGlobalId()
	istorage, err := idisk.GetIStorage()
	if err != nil {
		log.Errorf("failed to get istorage for disk %s error: %v", extId, err)
		return err
	}
	storageExtId := istorage.GetGlobalId()
	storage, err := db.FetchByExternalId(StorageManager, storageExtId)
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
			storage, err := db.FetchByExternalId(StorageManager, storageId)
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

func (self *SDisk) syncWithCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, index int, syncOwnerId mcclient.IIdentityProvider) error {
	recycle := false
	guests := self.GetGuests()
	if provider.GetFactory().IsSupportPrepaidResources() && len(guests) == 1 && guests[0].IsPrepaidRecycle() {
		recycle = true
	}
	extDisk.Refresh()

	storage := self.GetStorage()
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
			self.BillingType = extDisk.GetBillingType()
			self.ExpiredAt = extDisk.GetExpiredAt()
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

	newName, err := db.GenerateName(manager, syncOwnerId, extDisk.GetName())
	if err != nil {
		return nil, err
	}
	disk.Name = newName
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
		disk.ExpiredAt = extDisk.GetExpiredAt()
	}

	if createAt := extDisk.GetCreatedAt(); !createAt.IsZero() {
		disk.CreatedAt = createAt
	}

	err = manager.TableSpec().Insert(&disk)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
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
	q = rangeObjectsFilter(q, rangeObjs, nil, storages.Field("zone_id"), storages.Field("manager_id"))
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
		return nil, httperrors.NewInputParameterError("Diskinfo not contains either imageID or size")
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
	return self.SVirtualResourceBase.Delete(ctx, userCred)
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

func (self *SDisk) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	extra *jsonutils.JSONDict) *jsonutils.
	JSONDict {
	if cloudprovider := self.GetCloudprovider(); cloudprovider != nil {
		extra.Add(jsonutils.NewString(cloudprovider.Provider), "provider")
	}
	if storage := self.GetStorage(); storage != nil {
		extra.Add(jsonutils.NewString(storage.GetName()), "storage")
		extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
		extra.Add(jsonutils.NewString(storage.MediumType), "medium_type")
		/*extra.Add(jsonutils.NewString(storage.ZoneId), "zone_id")
		if zone := storage.getZone(); zone != nil {
			extra.Add(jsonutils.NewString(zone.Name), "zone")
			extra.Add(jsonutils.NewString(zone.CloudregionId), "region_id")
			if region := zone.GetRegion(); region != nil {
				extra.Add(jsonutils.NewString(region.Name), "region")
			}
		}*/

		info := storage.getCloudProviderInfo()
		extra.Update(jsonutils.Marshal(&info))
	}
	guestArray := jsonutils.NewArray()
	guests, guest_status := []string{}, []string{}
	for _, guest := range self.GetGuests() {
		guests = append(guests, guest.Name)
		guest_status = append(guest_status, guest.Status)
		guestArray.Add(jsonutils.Marshal(map[string]string{"name": guest.Name, "id": guest.Id, "status": guest.Status}))
	}
	extra.Add(guestArray, "guests")
	extra.Add(jsonutils.NewString(strings.Join(guests, ",")), "guest")
	extra.Add(jsonutils.NewInt(int64(len(guests))), "guest_count")
	extra.Add(jsonutils.NewString(strings.Join(guest_status, ",")), "guest_status")

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		extra.Add(jsonutils.NewString(timeutils.FullIsoTime(pendingDeletedAt)), "auto_delete_at")
	}
	// the binded snapshot policy list
	sds, err := SnapshotPolicyDiskManager.FetchAllByDiskID(ctx, userCred, self.Id)
	if err != nil {
		return extra
	}
	spIds := make([]string, len(sds))
	for i := range sds {
		spIds[i] = sds[i].SnapshotpolicyId
	}
	sps, err := SnapshotPolicyManager.FetchAllByIds(spIds)
	if err != nil {
		return extra
	}
	if len(sps) == 0 {
		extra.Add(jsonutils.NewString(""), "snapshotpolicy_status")
	} else {
		extra.Add(jsonutils.NewString(sds[0].Status), "snapshotpolicy_status")
	}
	// check status
	// construction for snapshotpolicies attached to disk
	snapshotpoliciesJson := jsonutils.NewArray()
	for i := range sps {
		spsJson := jsonutils.Marshal(sps[i])
		spsDict := spsJson.(*jsonutils.JSONDict)
		repeatWeekdays := SnapshotPolicyManager.RepeatWeekdaysToIntArray(sps[i].RepeatWeekdays)
		timePoints := SnapshotPolicyManager.TimePointsToIntArray(sps[i].TimePoints)
		spsDict.Remove("repeat_weekdays")
		spsDict.Remove("time_points")
		spsDict.Add(jsonutils.Marshal(repeatWeekdays), "repeat_weekdays")
		spsDict.Add(jsonutils.Marshal(timePoints), "time_points")
		snapshotpoliciesJson.Add(spsDict)
	}
	extra.Add(snapshotpoliciesJson, "snapshotpolicies")
	storage := self.GetStorage()
	if storage != nil {
		manualSnapshotCount, _ := self.GetManualSnapshotCount()
		if utils.IsInStringArray(storage.StorageType, append(api.SHARED_FILE_STORAGE, api.STORAGE_LOCAL)) {
			extra.Set("manual_snapshot_count", jsonutils.NewInt(int64(manualSnapshotCount)))
			extra.Set("max_manual_snapshot_count", jsonutils.NewInt(int64(options.Options.DefaultMaxManualSnapshotCount)))
		}
	}

	return extra
}

func (self *SDisk) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, userCred, extra), nil
}

func (self *SDisk) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, userCred, extra)
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
	if self.PendingDeleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		return nil, err
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
	if len(guests) == 1 && utils.IsInStringArray(disk.GetStorage().StorageType, api.FIEL_STORAGE) {
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
		continue
	onFail:
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, err.Error(), userCred)
		reason := fmt.Sprintf("Disk auto create snapshot failed: %s", err.Error())
		notifyclient.NotifySystemError(disk.Id, disk.Name, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, reason)
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

func (self *SDisk) SaveRenewInfo(ctx context.Context, userCred mcclient.TokenCredential, bc *billing.SBillingCycle, expireAt *time.Time) error {
	_, err := db.Update(self, func() error {
		if self.BillingType != billing_api.BILLING_TYPE_PREPAID {
			self.BillingType = billing_api.BILLING_TYPE_PREPAID
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

func (manager *SDiskManager) AutoSyncExtDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {

	spds, err := manager.getAutoSnapshotDisksId(true)
	if err != nil {
		log.Errorf("Get auto snapshot ext disks id failed: %s", err)
		return
	}
	if len(spds) == 0 {
		log.Infof("CronJob AutoSyncExtDiskSnapshot: No external disk need sync snapshot")
		return
	}

	for i := 0; i < len(spds); i++ {
		disk := manager.FetchDiskById(spds[i].DiskId)

		syncResult := disk.syncSnapshots(ctx, userCred)
		if syncResult.IsError() {
			db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SYNC_SNAPSHOT_FAIL, syncResult.Result(), userCred)
			continue
		}
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SYNC_SNAPSHOT, "disk auto sync snapshot successfully", userCred)
	}
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
	region := self.GetStorage().GetRegion()

	extSnapshots, err := extDisk.GetISnapshots()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	localSnapshots := SnapshotManager.GetDiskSnapshots(self.Id)

	lockman.LockClass(ctx, SnapshotManager, db.GetLockClassKey(SnapshotManager, syncOwnerId))
	defer lockman.ReleaseClass(ctx, SnapshotManager, db.GetLockClassKey(SnapshotManager, syncOwnerId))

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
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	_, err := self.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, data)
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
		lockman.LockObject(ctx, &snapshot)
		_, err := snapshot.PerformChangeOwner(ctx, userCred, query, data)
		if err != nil {
			lockman.ReleaseObject(ctx, &snapshot)
			return nil, errors.Wrapf(err, "fail to change owner of this disk(%s)'s snapshot %s", self.Id, snapshot.Id)
		}
		lockman.ReleaseObject(ctx, &snapshot)
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
