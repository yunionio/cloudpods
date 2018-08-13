package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"
)

type SHoststorageManager struct {
	SHostJointsManager
}

var HoststorageManager *SHoststorageManager

func init() {
	db.InitManager(func() {
		HoststorageManager = &SHoststorageManager{SHostJointsManager: NewHostJointsManager(SHoststorage{},
			"hoststorages_tbl", "hoststorage", "hoststorages", StorageManager)}
	})
}

type SHoststorage struct {
	SHostJointsBase

	MountPoint string `width:"256" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	HostId    string `width:"36" charset:"ascii" nullable:"false" list:"admin" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	StorageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

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

func (self *SHoststorage) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	host := self.GetHost()
	extra.Add(jsonutils.NewString(host.Name), "host")
	storage := self.GetStorage()
	extra.Add(jsonutils.NewString(storage.Name), "storage")
	extra.Add(jsonutils.NewInt(int64(storage.Capacity)), "capacity")
	used := storage.GetUsedCapacity(tristate.True)
	wasted := storage.GetUsedCapacity(tristate.False)
	extra.Add(jsonutils.NewInt(int64(used)), "used_capacity")
	extra.Add(jsonutils.NewInt(int64(wasted)), "waste_capacity")
	extra.Add(jsonutils.NewInt(int64(storage.Capacity-used-wasted)), "free_capacity")
	extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
	extra.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	if storage.Enabled {
		extra.Add(jsonutils.JSONTrue, "enabled")
	} else {
		extra.Add(jsonutils.JSONFalse, "enabled")
	}
	extra.Add(jsonutils.NewFloat(float64(storage.GetOvercommitBound())), "cmtbound")
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
