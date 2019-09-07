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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGuestdiskManager struct {
	SGuestJointsManager
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

	DiskId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	ImagePath string `width:"256" charset:"ascii" nullable:"false" get:"user" create:"required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	Driver    string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	CacheMode string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	AioMode   string `width:"32" charset:"ascii" nullable:"true" get:"user" update:"user"`  // Column(VARCHAR(32, charset='ascii'), nullable=True)
	Iops      int    `nullable:"true" default:"0"`
	Bps       int    `nullable:"true" default:"0"` // Mb

	Mountpoint string `width:"256" charset:"utf8" nullable:"true" get:"user"` // Column(VARCHAR(256, charset='utf8'), nullable=True)

	Index int8 `nullable:"false" default:"0" list:"user" update:"user"` // Column(TINYINT(4), nullable=False, default=0)
}

func (manager *SGuestdiskManager) GetSlaveFieldName() string {
	return "disk_id"
}

func (manager *SGuestdiskManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SGuestdisk) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SGuestdisk) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("index") {
		if index, err := data.Int("index"); err != nil {
			return nil, err
		} else {
			guestdisk := GuestdiskManager.Query().SubQuery()
			count, err := guestdisk.Query().Filter(sqlchemy.Equals(guestdisk.Field("guest_id"), self.GuestId)).
				Filter(sqlchemy.NotEquals(guestdisk.Field("disk_id"), self.DiskId)).
				Filter(sqlchemy.Equals(guestdisk.Field("index"), index)).CountWithError()
			if err != nil {
				return nil, httperrors.NewInternalServerError("check disk index uniqueness fail %s", err)
			}
			if count > 0 {
				return nil, httperrors.NewInputParameterError("DISK Index %d has been occupied", index)
			}
		}
	}
	return self.SGuestJointsBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (joint *SGuestdisk) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SGuestdisk) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SGuestdisk) getExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	disk := self.GetDisk()
	if storage := disk.GetStorage(); storage != nil {
		extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
	}
	extra.Add(jsonutils.NewInt(int64(disk.DiskSize)), "disk_size")
	extra.Add(jsonutils.NewString(disk.Status), "status")
	extra.Add(jsonutils.NewString(disk.DiskType), "disk_type")
	if storage := disk.GetStorage(); storage != nil {
		extra.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	}
	return extra
}

func (self *SGuestdisk) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SGuestJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraInfo(extra)
}

func (self *SGuestdisk) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SGuestJointsBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = db.JointModelExtra(self, extra)
	return self.getExtraInfo(extra), nil
}

func (self *SGuestdisk) DoSave(driver string, cache string, mountpoint string) error {
	self.ImagePath = ""
	if len(driver) == 0 {
		driver = "scsi"
	}
	if len(cache) == 0 {
		cache = "none"
	}
	if len(mountpoint) > 0 {
		self.Mountpoint = mountpoint
	}
	self.Driver = driver
	self.CacheMode = cache
	self.AioMode = "native"
	return GuestdiskManager.TableSpec().Insert(self)
}

func (self *SGuestdisk) GetDisk() *SDisk {
	disk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		log.Errorf("GetDisk fail: %s", err)
		return nil
	}
	return disk.(*SDisk)
}

func (self *SGuestdisk) GetJsonDescAtHost(host *SHost) jsonutils.JSONObject {
	disk := self.GetDisk()
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(self.DiskId), "disk_id")
	desc.Add(jsonutils.NewString(self.Driver), "driver")
	desc.Add(jsonutils.NewString(self.CacheMode), "cache_mode")
	desc.Add(jsonutils.NewString(self.AioMode), "aio_mode")
	desc.Add(jsonutils.NewInt(int64(self.Iops)), "iops")
	desc.Add(jsonutils.NewInt(int64(self.Bps)), "bps")
	desc.Add(jsonutils.NewInt(int64(disk.DiskSize)), "size")
	templateId := disk.GetTemplateId()
	if len(templateId) > 0 {
		desc.Add(jsonutils.NewString(templateId), "template_id")
		storage := disk.GetStorage()
		storagecacheimg := StoragecachedimageManager.GetStoragecachedimage(storage.StoragecacheId, templateId)
		if storagecacheimg != nil {
			desc.Add(jsonutils.NewString(storagecacheimg.Path), "image_path")
		}
	}
	storage := disk.GetStorage()
	// XXX ???
	if host.HostType == api.HOST_TYPE_HYPERVISOR {
		desc.Add(jsonutils.NewString(disk.StorageId), "storage_id")
		localpath := disk.GetPathAtHost(host)
		if len(localpath) == 0 {
			desc.Add(jsonutils.JSONTrue, "migrating")
			target := host.GetLeastUsedStorage(storage.StorageType)
			desc.Add(jsonutils.NewString(target.Id), "target_storage_id")
			disk.SetStatus(nil, api.DISK_START_MIGRATE, "migration")
		} else {
			desc.Add(jsonutils.NewString(localpath), "path")
		}
	}
	desc.Add(jsonutils.NewString(disk.DiskFormat), "format")
	desc.Add(jsonutils.NewInt(int64(self.Index)), "index")

	tid := disk.GetTemplateId()
	if len(tid) > 0 {
		desc.Add(jsonutils.NewString(tid), "template_id")
	}
	if len(disk.SnapshotId) > 0 {
		needMerge := disk.GetMetadata("merge_snapshot", nil)
		if needMerge == "true" {
			desc.Set("merge_snapshot", jsonutils.JSONTrue)
		}
	}
	fs := disk.GetFsFormat()
	if len(fs) > 0 {
		desc.Add(jsonutils.NewString(fs), "fs")
	}
	if len(self.Mountpoint) > 0 {
		desc.Add(jsonutils.NewString(self.Mountpoint), "mountpoint")
	}
	dev := disk.getDev()
	if len(dev) > 0 {
		desc.Add(jsonutils.NewString(dev), "dev")
	}
	return desc
}

func (self *SGuestdisk) GetDetailedJson() *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	disk := self.GetDisk()
	storage := disk.GetStorage()
	if fs := disk.GetFsFormat(); len(fs) > 0 {
		desc.Add(jsonutils.NewString(fs), "fs")
	}
	desc.Add(jsonutils.NewString(disk.DiskType), "disk_type")
	desc.Add(jsonutils.NewInt(int64(self.Index)), "index")
	desc.Add(jsonutils.NewInt(int64(disk.DiskSize)), "size")
	desc.Add(jsonutils.NewString(disk.DiskFormat), "disk_format")
	desc.Add(jsonutils.NewString(self.Driver), "driver")
	desc.Add(jsonutils.NewString(self.CacheMode), "cache_mode")
	desc.Add(jsonutils.NewString(self.AioMode), "aio_mode")
	desc.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	desc.Add(jsonutils.NewString(storage.StorageType), "storage_type")
	desc.Add(jsonutils.NewInt(int64(self.Iops)), "iops")
	desc.Add(jsonutils.NewInt(int64(self.Bps)), "bps")

	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		desc.Add(jsonutils.NewString(imageId), "image_id")
		cachedImageObj, _ := CachedimageManager.FetchById(imageId)
		if cachedImageObj != nil {
			cachedImage := cachedImageObj.(*SCachedimage)
			desc.Add(jsonutils.NewString(cachedImage.GetName()), "image")
		}
	}

	return desc
}

func (self *SGuestdisk) GetDetailedString() string {
	disk := self.GetDisk()
	var fs string
	if len(disk.GetTemplateId()) > 0 {
		fs = "root"
	} else if len(disk.GetFsFormat()) > 0 {
		fs = disk.GetFsFormat()
	} else {
		fs = "none"
	}
	return fmt.Sprintf("disk%d:%dM/%s/%s/%s/%s/%s", self.Index, disk.DiskSize,
		disk.DiskFormat, self.Driver, self.CacheMode, self.AioMode, fs)
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
	return conf
}
