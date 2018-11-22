package models

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
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
	})
}

type SHoststorage struct {
	SHostJointsBase

	MountPoint string `width:"256" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	HostId    string `width:"36" charset:"ascii" nullable:"false" list:"admin" key_index:"true" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" key_index:"true" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	Config       *jsonutils.JSONArray `nullable:"true" get:"admin"`  // Column(JSONEncodedDict, nullable=True)
	RealCapacity int                  `nullable:"true" list:"admin"` // Column(Integer, nullable=True)
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

func (self *SHoststorage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SHostJointsBase.GetExtraDetails(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
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

func (manager *SHoststorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	storageId, _ := data.GetString("storage_id")
	if len(storageId) == 0 {
		return nil, httperrors.NewInputParameterError("missing storage_id")
	}
	storageTmp, _ := StorageManager.FetchById(storageId)
	if storageTmp == nil {
		return nil, httperrors.NewInputParameterError(fmt.Sprintf("invalid storage_id %s", storageId))
	}
	storage := storageTmp.(*SStorage)
	if storage.StorageType == STORAGE_RBD {
		pool, _ := data.GetString("poll")
		data.Add(jsonutils.NewString(fmt.Sprintf("rbd:%s", pool)), "mount_point")
	}
	return manager.SJointResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SHoststorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SHostJointsBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	storage := self.GetStorage()
	if !utils.IsInStringArray(storage.StorageType, STORAGE_LOCAL_TYPES) {
		host := storage.GetMasterHost()
		log.Infof("Attach SharedStorage[%s] on host %s ...", storage.Name, host.Name)
		url := fmt.Sprintf("%s/storages/attach", host.ManagerUri)
		headers := http.Header{}
		headers.Set("X-Auth-Token", userCred.GetTokenString())
		body := jsonutils.NewDict()
		body.Set("mount_point", jsonutils.NewString(self.MountPoint))
		body.Set("name", jsonutils.NewString(storage.Name))
		body.Set("storage_id", jsonutils.NewString(storage.Id))
		body.Set("storage_conf", storage.StorageConf)
		body.Set("storage_type", jsonutils.NewString(storage.StorageType))
		if len(storage.StoragecacheId) > 0 {
			storagecache := StoragecacheManager.FetchStoragecacheById(storage.StoragecacheId)
			if storagecache != nil {
				body.Set("imagecache_path", jsonutils.NewString(
					storage.GetStorageCachePath(self.MountPoint, storagecache.Path)))
				body.Set("sotragecache_id", jsonutils.NewString(storagecache.Id))
			}
		}
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(),
			ctx, "POST", url, headers, body, false)
		if err != nil {
			log.Errorf("Host Storage Post Create Error: %s", err)
			return
		}
		self.SyncStorageStatus()
	}
}

func (self *SHoststorage) SyncStorageStatus() {
	storage := self.GetStorage()
	hostQuery := HostManager.Query().SubQuery()
	count := HoststorageManager.Query().Join(hostQuery,
		sqlchemy.AND(sqlchemy.Equals(hostQuery.Field("id"), self.HostId),
			sqlchemy.Equals(hostQuery.Field("host_status"), "online"))).Count()
	status := storage.Status
	if count >= 1 {
		status = STORAGE_ONLINE
	} else {
		status = STORAGE_OFFLINE
	}
	if status != storage.Status {
		storage.GetModelManager().TableSpec().Update(storage, func() error {
			storage.Status = status
			return nil
		})
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
	extra.Add(jsonutils.NewInt(int64(storage.Capacity-used-wasted)), "free_capacity")
	extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
	extra.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	extra.Add(jsonutils.NewBool(storage.Enabled), "enabled")
	extra.Add(jsonutils.NewFloat(float64(storage.GetOvercommitBound())), "cmtbound")
	extra.Add(jsonutils.NewInt(int64(self.GetGuestDiskCount())), "guest_disk_count")
	if len(storage.StoragecacheId) > 0 {
		storagecache := StoragecacheManager.FetchStoragecacheById(storage.StoragecacheId)
		if storagecache != nil {
			extra.Set("imagecache_path", jsonutils.NewString(storage.GetStorageCachePath(self.MountPoint, storagecache.Path)))
		}
	}
	return extra
}

func (self *SHoststorage) GetGuestDiskCount() int {
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

	return q.Count()
}

func (self *SHoststorage) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetGuestDiskCount() > 0 {
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
