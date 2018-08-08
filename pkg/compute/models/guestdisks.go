package models

import (
	"context"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/mcclient"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

type SGuestdiskManager struct {
	SGuestJointsManager
}

var GuestdiskManager *SGuestdiskManager

func init() {
	db.InitManager(func() {
		GuestdiskManager = &SGuestdiskManager{SGuestJointsManager: NewGuestJointsManager(SGuestdisk{}, "guestdisks_tbl", "guestdisk", "guestdisks", DiskManager)}
	})
}

type SGuestdisk struct {
	SGuestJointsBase

	DiskId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	ImagePath string `width:"256" charset:"ascii" nullable:"false" get:"user" create:"required"` // Column(VARCHAR(256, charset='ascii'), nullable=False)

	Driver    string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	CacheMode string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	AioMode   string `width:"32" charset:"ascii" nullable:"true" get:"user" update:"user"`  // Column(VARCHAR(32, charset='ascii'), nullable=True)

	Mountpoint string `width:"256" charset:"utf8" nullable:"true" get:"user"` // Column(VARCHAR(256, charset='utf8'), nullable=True)

	Index int8 `nullable:"false" default:"0" list:"user" update:"user"` // Column(TINYINT(4), nullable=False, default=0)
}

func (manager *SGuestdiskManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SGuestdisk) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (joint *SGuestdisk) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SGuestdisk) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SGuestdisk) getExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	disk := self.GetDisk()
	extra.Add(jsonutils.NewInt(int64(disk.DiskSize)), "disk_size")
	extra.Add(jsonutils.NewString(disk.Status), "status")
	return extra
}

func (self *SGuestdisk) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SGuestJointsBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraInfo(extra)
}

func (self *SGuestdisk) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SGuestJointsBase.GetExtraDetails(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraInfo(extra)
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
	desc.Add(jsonutils.NewInt(int64(disk.DiskSize)), "size")
	templateId := disk.GetTemplateId()
	if len(templateId) > 0 {
		desc.Add(jsonutils.NewString(templateId), "template_id")
		// hostcachedimg = Hostcachedimages.get_host_cachedimage(host, template_id)
		// if hostcachedimg is not None:
		//	desc['image_path'] = hostcachedimg.path
	}
	if host.HostType == HOST_TYPE_HYPERVISOR && disk.IsLocal() {
		desc.Add(jsonutils.NewString(disk.StorageId), "storage_id")
		localpath := disk.GetPathAtHost(host)
		if len(localpath) == 0 {
			desc.Add(jsonutils.NewString(disk.GetFetchUrl()), "url")
			storage := disk.GetStorage()
			target := host.GetLeastUsedStorage(storage.StorageType)
			desc.Add(jsonutils.NewString(target.Id), "target_storage_id")
			disk.SetStatus(nil, DISK_START_MIGRATE, "migration")
		} else {
			desc.Add(jsonutils.NewString(localpath), "path")
		}
	}
	desc.Add(jsonutils.NewString(disk.DiskFormat), "format")
	tid := disk.GetTemplateId()
	if len(tid) > 0 {
		desc.Add(jsonutils.NewString(tid), "template_id")
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
	var fs string
	if len(disk.GetTemplateId()) > 0 {
		fs = "root"
	} else if len(disk.GetFsFormat()) > 0 {
		fs = disk.GetFsFormat()
	} else {
		fs = "none"
	}
	storage := disk.GetStorage()
	desc.Add(jsonutils.NewString(fs), "fs")
	desc.Add(jsonutils.NewInt(int64(self.Index)), "index")
	desc.Add(jsonutils.NewString(fmt.Sprintf("%dM", disk.DiskSize)), "index")
	desc.Add(jsonutils.NewString(disk.DiskFormat), "disk_format")
	desc.Add(jsonutils.NewString(self.Driver), "driver")
	desc.Add(jsonutils.NewString(self.CacheMode), "cache_mode")
	desc.Add(jsonutils.NewString(self.AioMode), "aio_mode")
	desc.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	desc.Add(jsonutils.NewString(storage.StorageType), "storage_type")
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
