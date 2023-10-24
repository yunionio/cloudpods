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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const ErrStorageInUse = errors.Error("StorageInUse")

type SHoststorageManager struct {
	SHostJointsManager
	SStorageResourceBaseManager
}

var HoststorageManager *SHoststorageManager

func init() {
	db.InitManager(func() {
		HoststorageManager = &SHoststorageManager{
			SHostJointsManager: NewHostJointsManager(
				"host_id",
				SHoststorage{},
				"hoststorages_tbl",
				"hoststorage",
				"hoststorages",
				StorageManager,
			),
		}
		HoststorageManager.SetVirtualObject(HoststorageManager)
		HoststorageManager.TableSpec().AddIndex(false, "host_id", "storage_id")
	})
}

type SHoststorage struct {
	SHostJointsBase

	// 宿主机Id
	HostId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" json:"host_id"`
	// 存储Id
	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" json:"storage_id" index:"true"`

	// 挂载点
	// nvme pci address
	MountPoint string `width:"256" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"required" json:"mount_point"`
	// 是否是根分区
	IsRootPartition bool `nullable:"true" default:"false" list:"domain" update:"domain" create:"optional"`

	// 配置信息
	Config *jsonutils.JSONArray `nullable:"true" get:"domain" json:"config"`
	// 真实容量大小
	RealCapacity int64 `nullable:"true" list:"domain" json:"real_capacity"`
}

func (manager *SHoststorageManager) GetMasterFieldName() string {
	return "host_id"
}

func (manager *SHoststorageManager) GetSlaveFieldName() string {
	return "storage_id"
}

func (manager *SHoststorageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HoststorageDetails {
	rows := make([]api.HoststorageDetails, len(objs))

	hostRows := manager.SHostJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	storageIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.HoststorageDetails{
			HostJointResourceDetails: hostRows[i],
		}
		storageIds[i] = objs[i].(*SHoststorage).StorageId
	}

	storages := make(map[string]SStorage)
	err := db.FetchStandaloneObjectsByIds(StorageManager, storageIds, &storages)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if storage, ok := storages[storageIds[i]]; ok {
			rows[i] = objs[i].(*SHoststorage).getExtraDetails(storage, rows[i])
		}
	}

	return rows
}

func (self *SHoststorage) GetHost() *SHost {
	host, _ := HostManager.FetchById(self.HostId)
	if host != nil {
		return host.(*SHost)
	}
	return nil
}

func (self *SHoststorage) GetStorage() *SStorage {
	storage, err := StorageManager.FetchById(self.StorageId)
	if err != nil {
		log.Errorf("Hoststorage fetch storage %q error: %v", self.StorageId, err)
	}
	if storage != nil {
		return storage.(*SStorage)
	}
	return nil
}

func (manager *SHoststorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.HostStorageCreateInput) (api.HostStorageCreateInput, error) {
	storageObj, err := validators.ValidateModel(userCred, StorageManager, &input.StorageId)
	if err != nil {
		return input, err
	}
	storage := storageObj.(*SStorage)
	hostObj, err := validators.ValidateModel(userCred, HostManager, &input.HostId)
	if err != nil {
		return input, err
	}
	host := hostObj.(*SHost)

	input, err = host.GetHostDriver().ValidateAttachStorage(ctx, userCred, host, storage, input)
	if err != nil {
		return input, err
	}

	input.JoinResourceBaseCreateInput, err = manager.SJointResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.JoinResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SHoststorage) syncLocalStorageShare(ctx context.Context, userCred mcclient.TokenCredential) {
	// sync host and local storage permissions
	host := self.GetHost()
	storage := self.GetStorage()
	if host != nil && storage != nil && storage.IsLocal() {
		shareInfo := host.GetSharedInfo()
		if !shareInfo.IsPublic {
			_, err := storage.performPrivateInternal(ctx, userCred, nil, apis.PerformPrivateInput{})
			if err != nil {
				log.Errorf("attach storage: private local storage fail %s", err)
			}
		} else {
			input := apis.PerformPublicDomainInput{
				Scope:           string(shareInfo.PublicScope),
				SharedDomainIds: shareInfo.SharedDomains,
			}
			_, err := storage.performPublicInternal(ctx, userCred, nil, input)
			if err != nil {
				log.Errorf("attach storage: public local storage fail %s", err)
			}
		}
	}
}

func (self *SHoststorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SHostJointsBase.PostCreate(ctx, userCred, ownerId, query, data)

	self.syncLocalStorageShare(ctx, userCred)

	if err := self.StartHostStorageAttachTask(ctx, userCred); err != nil {
		log.Errorf("failed to attach storage error: %v", err)
		self.Detach(ctx, userCred)
	}
}

func (self *SHoststorage) StartHostStorageAttachTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	host := self.GetHost()
	params := jsonutils.NewDict()
	params.Set("storage_id", jsonutils.NewString(self.StorageId))
	task, err := taskman.TaskManager.NewTask(ctx, "HostStorageAttachTask", host, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHoststorage) StartHostStorageDetachTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	if host := self.GetHost(); host.HostStatus == api.HOST_ONLINE {
		params := jsonutils.NewDict()
		params.Set("storage_id", jsonutils.NewString(self.StorageId))
		params.Set("mount_point", jsonutils.NewString(self.MountPoint))
		task, err := taskman.TaskManager.NewTask(ctx, "HostStorageDetachTask", host, userCred, params, "", "", nil)
		if err != nil {
			return err
		}
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SHoststorage) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.StartHostStorageDetachTask(ctx, userCred)
	self.SyncStorageStatus(userCred)
}

func (self *SHoststorage) SyncStorageStatus(userCred mcclient.TokenCredential) {
	storage := self.GetStorage()
	status := api.STORAGE_OFFLINE
	hosts, _ := storage.GetAttachedHosts()
	for _, host := range hosts {
		if host.HostStatus == api.HOST_ONLINE {
			status = api.STORAGE_ONLINE
		}
	}
	if status != storage.Status {
		storage.SetStatus(userCred, status, "SyncStorageStatus")
	}
}

func (self *SHoststorage) getExtraDetails(storage SStorage, out api.HoststorageDetails) api.HoststorageDetails {
	out.Storage = storage.Name
	out.Capacity = storage.Capacity
	if storage.StorageConf != nil {
		out.StorageConf = storage.StorageConf
	}
	used := storage.GetUsedCapacity(tristate.True)
	wasted := storage.GetUsedCapacity(tristate.False)
	out.UsedCapacity = used
	out.WasteCapacity = wasted
	out.FreeCapacity = storage.GetFreeCapacity()
	out.StorageType = storage.StorageType
	out.MediumType = storage.MediumType
	out.Enabled = storage.Enabled.Bool()
	out.Cmtbound = storage.GetOvercommitBound()

	//extra.Add(jsonutils.NewInt(int64(self.GetGuestDiskCount())), "guest_disk_count")

	out.GuestDiskCount, _ = self.GetGuestDiskCount()

	if len(storage.StoragecacheId) > 0 {
		storagecache := StoragecacheManager.FetchStoragecacheById(storage.StoragecacheId)
		if storagecache != nil {
			out.StoragecacheId = storagecache.Id
			out.ImagecachePath = storage.GetStorageCachePath(self.MountPoint, storagecache.Path)
		}
	}
	return out
}

func (self *SHoststorage) GetGuestDiskCount() (int, error) {
	guestdisks := GuestdiskManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()
	disks := DiskManager.Query().SubQuery()

	q := guestdisks.Query()
	q = q.Join(guests, sqlchemy.AND(sqlchemy.IsFalse(guests.Field("deleted")),
		sqlchemy.Equals(guests.Field("id"), guestdisks.Field("guest_id")),
		sqlchemy.Equals(guests.Field("host_id"), self.HostId)))
	q = q.Join(disks, sqlchemy.AND(sqlchemy.IsFalse(disks.Field("deleted")),
		sqlchemy.Equals(disks.Field("id"), guestdisks.Field("disk_id")),
		sqlchemy.Equals(disks.Field("storage_id"), self.StorageId)))

	return q.CountWithError()
}

func (self *SHoststorage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	cnt, err := self.GetGuestDiskCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestDiskCount fail %s", err)
	}
	if cnt > 0 {
		return errors.Wrap(ErrStorageInUse, "guest on the host are using disks on this storage")
	}
	return self.SHostJointsBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SHoststorage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SHoststorage) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SHoststorageManager) GetStorages(hostId string) ([]SHoststorage, error) {
	hoststorage := make([]SHoststorage, 0)
	hoststorages := HoststorageManager.Query().SubQuery()
	err := hoststorages.Query().Equals("host_id", hostId).All(&hoststorage)
	if err != nil {
		return nil, err
	}
	return hoststorage, nil
}

func (self *SHoststorage) syncWithCloudHostStorage(userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage) error {
	diff, err := db.Update(self, func() error {
		self.MountPoint = extStorage.GetMountPoint()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SHoststorageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HoststorageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemFilter(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStorageResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SHoststorageManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HoststorageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.OrderByExtraFields(ctx, q, userCred, query.HostJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SStorageResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StorageFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStorageResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SHoststorageManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SHostJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SHostJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SStorageResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SStorageResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SStorageResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
