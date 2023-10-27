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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGuestdiskManager struct {
	SGuestJointsManager
	SDiskResourceBaseManager
}

var GuestdiskManager *SGuestdiskManager

func init() {
	db.InitManager(func() {
		GuestdiskManager = &SGuestdiskManager{
			SGuestJointsManager: NewGuestJointsManager(
				SGuestdisk{},
				"guestdisks_tbl",
				"guestdisk",
				"guestdisks",
				DiskManager,
			),
		}
		GuestdiskManager.SetVirtualObject(GuestdiskManager)
		GuestdiskManager.TableSpec().AddIndex(true, "disk_id", "guest_id")
	})

}

type SGuestdisk struct {
	SGuestJointsBase

	SDiskResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// DiskId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	ImagePath string `width:"256" charset:"ascii" nullable:"false" get:"user" create:"required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	Driver    string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	CacheMode string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	AioMode   string `width:"32" charset:"ascii" nullable:"true" get:"user" update:"user"`  // Column(VARCHAR(32, charset='ascii'), nullable=True)
	Iops      int    `nullable:"true" default:"0"`
	Bps       int    `nullable:"true" default:"0"` // Mb

	Mountpoint string `width:"256" charset:"utf8" nullable:"true" get:"user"` // Column(VARCHAR(256, charset='utf8'), nullable=True)

	Index     int8 `nullable:"false" default:"0" list:"user" update:"user"` // Column(TINYINT(4), nullable=False, default=0)
	BootIndex int8 `nullable:"false" default:"-1" list:"user" update:"user"`
}

func (manager *SGuestdiskManager) GetSlaveFieldName() string {
	return "disk_id"
}

func (self *SGuestdisk) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.GuestdiskUpdateInput) (api.GuestdiskUpdateInput, error) {
	if input.Index != nil {
		index := *input.Index
		guestdisk := GuestdiskManager.Query().SubQuery()
		count, err := guestdisk.Query().Filter(sqlchemy.Equals(guestdisk.Field("guest_id"), self.GuestId)).
			Filter(sqlchemy.NotEquals(guestdisk.Field("disk_id"), self.DiskId)).
			Filter(sqlchemy.Equals(guestdisk.Field("index"), index)).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("check disk index uniqueness fail %s", err)
		}
		if count > 0 {
			return input, httperrors.NewInputParameterError("DISK Index %d has been occupied", index)
		}
	}
	var err error
	input.GuestJointBaseUpdateInput, err = self.SGuestJointsBase.ValidateUpdateData(ctx, userCred, query, input.GuestJointBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SGuestJointsBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SGuestdiskManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestDiskDetails {
	rows := make([]api.GuestDiskDetails, len(objs))

	guestRows := manager.SGuestJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	diskIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GuestDiskDetails{
			GuestJointResourceDetails: guestRows[i],
		}
		diskIds[i] = objs[i].(*SGuestdisk).DiskId
	}

	disks := make(map[string]SDisk)
	err := db.FetchStandaloneObjectsByIds(DiskManager, diskIds, &disks)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if disk, ok := disks[diskIds[i]]; ok {
			rows[i].Disk = disk.Name
			rows[i].Status = disk.Status
			rows[i].DiskSize = disk.DiskSize
			rows[i].DiskType = disk.DiskType
			storage, _ := disk.GetStorage()
			if storage != nil {
				rows[i].StorageType = storage.StorageType
				rows[i].MediumType = storage.MediumType
			}
		}
	}

	return rows
}

func (self *SGuestdisk) DoSave(ctx context.Context, driver string, cache string, mountpoint string) error {
	self.ImagePath = ""
	if len(driver) == 0 {
		driver = options.Options.DefaultDiskDriver
	}
	if len(cache) == 0 {
		cache = options.Options.DefaultDiskCacheMode
	}
	if len(mountpoint) > 0 {
		self.Mountpoint = mountpoint
	}
	self.Driver = driver
	self.CacheMode = cache
	if cache == "none" {
		self.AioMode = "native"
	} else {
		self.AioMode = "threads"
	}
	return GuestdiskManager.TableSpec().Insert(ctx, self)
}

func (self *SGuestdisk) GetDisk() *SDisk {
	disk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		log.Errorf("GetDisk %s fail: %s", self.DiskId, err)
		return nil
	}
	return disk.(*SDisk)
}

func (self *SGuestdisk) GetJsonDescAtHost(ctx context.Context, host *SHost) *api.GuestdiskJsonDesc {
	disk := self.GetDisk()
	return self.GetDiskJsonDescAtHost(ctx, host, disk)
}

func (self *SGuestdisk) GetDiskJsonDescAtHost(ctx context.Context, host *SHost, disk *SDisk) *api.GuestdiskJsonDesc {
	desc := &api.GuestdiskJsonDesc{
		DiskId:     disk.Id,
		Driver:     self.Driver,
		CacheMode:  self.CacheMode,
		AioMode:    self.AioMode,
		Iops:       self.Iops,
		Throughput: disk.Throughput,
		Bps:        self.Bps,
		Size:       disk.DiskSize,
	}
	desc.TemplateId = disk.GetTemplateId()
	storage, _ := disk.GetStorage()
	desc.StorageType = storage.StorageType
	if len(desc.TemplateId) > 0 {
		storagecacheimg := StoragecachedimageManager.GetStoragecachedimage(storage.StoragecacheId, desc.TemplateId)
		if storagecacheimg != nil {
			desc.ImagePath = storagecacheimg.Path
		}
	}
	if host.HostType == api.HOST_TYPE_HYPERVISOR {
		desc.StorageId = disk.StorageId
		localpath := disk.GetPathAtHost(host)
		if len(localpath) == 0 {
			desc.Migrating = true
			// not used yet
			// disk.SetStatus(nil, api.DISK_START_MIGRATE, "migration")
		} else {
			desc.Path = localpath
		}
	}
	desc.Format = disk.DiskFormat
	desc.Index = self.Index
	bootIndex := self.BootIndex
	desc.BootIndex = &bootIndex

	if len(disk.SnapshotId) > 0 {
		needMerge := disk.GetMetadata(ctx, "merge_snapshot", nil)
		if needMerge == "true" {
			desc.MergeSnapshot = true
		}
		if desc.MergeSnapshot {
			if url, err := disk.GetSnapshotFuseUrl(); err != nil {
				log.Errorf("failed get snapshot fuse url: %s", err)
			} else {
				desc.Url = url
			}
		}
	}
	if fpath := disk.GetMetadata(ctx, api.DISK_META_REMOTE_ACCESS_PATH, nil); len(fpath) > 0 {
		guest := self.getGuest()
		if sid := guest.GetMetadata(ctx, api.SERVER_META_CONVERT_FROM_ESXI, nil); len(sid) > 0 {
			desc.EsxiFlatFilePath = fpath
		} else {
			desc.Url = fpath
		}
		desc.MergeSnapshot = true
	}
	desc.Fs = disk.GetFsFormat()
	desc.Mountpoint = self.Mountpoint
	desc.Dev = disk.getDev()
	desc.IsSSD = disk.IsSsd
	return desc
}

func (self *SGuestdisk) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestdisk) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

// DEPRECATE: will be remove in future, use ToDiskConfig
func (self *SGuestdisk) ToDiskInfo() DiskInfo {
	disk := self.GetDisk()
	if disk == nil {
		return DiskInfo{}
	}
	info := disk.ToDiskInfo()
	info.Driver = self.Driver
	info.Cache = self.CacheMode
	return info
}

func (self *SGuestdisk) ToDiskConfig() *api.DiskConfig {
	disk := self.GetDisk()
	if disk == nil {
		return nil
	}
	conf := disk.ToDiskConfig()
	conf.Index = int(self.Index)
	conf.Mountpoint = self.Mountpoint
	conf.Driver = self.Driver
	conf.Cache = self.CacheMode
	return conf
}

func (self *SGuestdisk) SetBootIndex(bootIndex int8) error {
	_, err := db.Update(self, func() error {
		self.BootIndex = bootIndex
		return nil
	})
	return err
}

func (manager *SGuestdiskManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestdiskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemFilter(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}
	q, err = manager.SDiskResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DiskFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemFilter")
	}

	if len(query.Driver) > 0 {
		q = q.In("driver", query.Driver)
	}
	if len(query.CacheMode) > 0 {
		q = q.In("cache_mode", query.CacheMode)
	}
	if len(query.AioMode) > 0 {
		q = q.In("aio_mode", query.AioMode)
	}

	return q, nil
}

func (manager *SGuestdiskManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestdiskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.OrderByExtraFields(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.OrderByExtraFields")
	}
	q, err = manager.SDiskResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DiskFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDiskResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGuestdiskManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SDiskResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SDiskResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
