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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHoststorageManager struct {
	SHostJointsManager
}

var HoststorageManager *SHoststorageManager

func init() {
	db.InitManager(func() {
		HoststorageManager = &SHoststorageManager{
			SHostJointsManager: NewHostJointsManager(
				SHoststorage{},
				"hoststorages_tbl",
				"hoststorage",
				"hoststorages",
				StorageManager,
			),
		}
		HoststorageManager.SetVirtualObject(HoststorageManager)
	})
}

type SHoststorage struct {
	SHostJointsBase

	MountPoint string `width:"256" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	HostId    string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	Config       *jsonutils.JSONArray `nullable:"true" get:"admin"`  // Column(JSONEncodedDict, nullable=True)
	RealCapacity int64                `nullable:"true" list:"admin"` // Column(Integer, nullable=True)
}

func (manager *SHoststorageManager) GetMasterFieldName() string {
	return "host_id"
}

func (manager *SHoststorageManager) GetSlaveFieldName() string {
	return "storage_id"
}

func (joint *SHoststorage) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SHoststorage) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SHoststorage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (self *SHoststorage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra), nil
}

func (self *SHoststorage) GetHost() *SHost {
	host, _ := HostManager.FetchById(self.HostId)
	if host != nil {
		return host.(*SHost)
	}
	return nil
}

func (self *SHoststorage) GetStorage() *SStorage {
	storage, _ := StorageManager.FetchById(self.StorageId)
	if storage != nil {
		return storage.(*SStorage)
	}
	return nil
}

func (manager *SHoststorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	storageId, _ := data.GetString("storage_id")
	if len(storageId) == 0 {
		return nil, httperrors.NewMissingParameterError("storage_id")
	}
	storageTmp, _ := StorageManager.FetchById(storageId)
	if storageTmp == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find storage %s to attach host", storageId)
	}
	storage := storageTmp.(*SStorage)

	hostId, _ := data.GetString("host_id")
	if len(hostId) == 0 {
		return nil, httperrors.NewMissingParameterError("host_id")
	}
	hostTmp, _ := HostManager.FetchById(hostId)
	if hostTmp == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find host %s to attach storage", hostId)
	}
	host := hostTmp.(*SHost)

	if err := host.GetHostDriver().ValidateAttachStorage(ctx, userCred, host, storage, data); err != nil {
		return nil, err
	}

	input := apis.JoinResourceBaseCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal JoinResourceBaseCreateInput fail %s", err)
	}
	input, err = manager.SJointResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (self *SHoststorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SHostJointsBase.PostCreate(ctx, userCred, ownerId, query, data)

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
	for _, host := range storage.GetAttachedHosts() {
		if host.HostStatus == api.HOST_ONLINE {
			status = api.STORAGE_ONLINE
		}
	}
	if status != storage.Status {
		storage.SetStatus(userCred, status, "SyncStorageStatus")
	}
}

func (self *SHoststorage) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	host := self.GetHost()
	extra.Add(jsonutils.NewString(host.Name), "host")
	storage := self.GetStorage()
	extra.Add(jsonutils.NewString(storage.Name), "storage")
	extra.Add(jsonutils.NewInt(int64(storage.Capacity)), "capacity")
	if storage.StorageConf != nil {
		extra.Set("storage_conf", storage.StorageConf)
	}
	used := storage.GetUsedCapacity(tristate.True)
	wasted := storage.GetUsedCapacity(tristate.False)
	extra.Add(jsonutils.NewInt(int64(used)), "used_capacity")
	extra.Add(jsonutils.NewInt(int64(wasted)), "waste_capacity")
	extra.Add(jsonutils.NewInt(int64(storage.GetFreeCapacity())), "free_capacity")
	extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
	extra.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	extra.Add(jsonutils.NewBool(storage.Enabled.Bool()), "enabled")
	extra.Add(jsonutils.NewFloat(float64(storage.GetOvercommitBound())), "cmtbound")

	//extra.Add(jsonutils.NewInt(int64(self.GetGuestDiskCount())), "guest_disk_count")

	extra = db.FetchModelExtraCountProperties(self, extra)

	if len(storage.StoragecacheId) > 0 {
		storagecache := StoragecacheManager.FetchStoragecacheById(storage.StoragecacheId)
		if storagecache != nil {
			extra.Set("imagecache_path", jsonutils.NewString(storage.GetStorageCachePath(self.MountPoint, storagecache.Path)))
			extra.Set("storagecache_id", jsonutils.NewString(storagecache.Id))
		}
	}
	return extra
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

func (self *SHoststorage) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.GetGuestDiskCount()
	if err != nil {
		return httperrors.NewInternalServerError("GetGuestDiskCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("guest on the host are using disks on this storage")
	}
	return self.SHostJointsBase.ValidateDeleteCondition(ctx)
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
