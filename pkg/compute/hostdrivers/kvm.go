package hostdrivers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SKVMHostDriver struct {
	SBaseHostDriver
}

func init() {
	driver := SKVMHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SKVMHostDriver) GetHostType() string {
	return models.HOST_TYPE_HYPERVISOR
}

func (self *SKVMHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (self *SKVMHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}
	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	obj, err := models.CachedimageManager.FetchById(imageId)
	if err != nil {
		return err
	}
	cacheImage := obj.(*models.SCachedimage)
	srcHostCacheImage, err := cacheImage.ChooseSourceStoragecacheInRange(models.HOST_TYPE_HYPERVISOR, []string{host.Id}, []interface{}{host.GetZone()})
	if err != nil {
		return err
	}

	content := jsonutils.NewDict()
	content.Add(jsonutils.NewString(imageId), "image_id")
	if srcHostCacheImage != nil {
		err = srcHostCacheImage.AddDownloadRefcount()
		if err != nil {
			return err
		}
		srcHost, err := srcHostCacheImage.GetHost()
		if err != nil {
			return err
		}
		srcUrl := fmt.Sprintf("%s/download/images/%s", srcHost.ManagerUri, imageId)
		content.Add(jsonutils.NewString(srcUrl), "src_url")
	}
	url := fmt.Sprintf("%s/disks/image_cache", host.ManagerUri)

	if isForce {
		content.Add(jsonutils.NewBool(true), "is_force")
	}
	content.Add(jsonutils.NewString(storageCache.Id), "storagecache_id")
	body := jsonutils.NewDict()
	body.Add(content, "disk")
	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	url := fmt.Sprintf("/disks/%s/create/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	body.Add(content, "disk")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestDeallocateDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	log.Infof("Deallocating disk on host %s", host.GetName())
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	url := fmt.Sprintf("/disks/%s/delete/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestResizeDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	url := fmt.Sprintf("/disks/%s/resize/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	content := jsonutils.NewDict()
	content.Add(jsonutils.NewInt(size), "size")
	body.Add(content, "disk")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestResizeDiskOnHostOnline(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	self.RequestResizeDiskOnHost(host, storage, disk, size, task)
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	for _, guest := range disk.GetAttachedGuests() {
		guestdisk := guest.GetGuestDisk(disk.GetId())
		url := fmt.Sprintf("/servers/%s/monitor", guest.GetId())
		body := jsonutils.NewDict()
		cmd := fmt.Sprintf("block_resize drive_%d %dM", guestdisk.Index, size)
		body.Add(jsonutils.NewString(cmd), "cmd")
		host.Request(task.GetUserCred(), "POST", url, header, body)
	}
	return nil
}

func (self *SKVMHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(map[string]string{"image_id": imageId}), "disk")
	url := fmt.Sprintf("/disks/%s/save-prepare/%s", disk.StorageId, disk.Id)
	header := http.Header{"X-Task-Id": []string{task.GetTaskId()}, "X-Region-Version": []string{"v2"}}
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	body := jsonutils.NewDict()
	backup, _ := data.GetString("backup")
	content := map[string]string{"image_path": backup, "image_id": imageId, "storagecached_id": disk.GetStorage().StoragecacheId}
	if data.Contains("format") {
		content["format"], _ = data.GetString("format")
	}
	body.Add(jsonutils.Marshal(content), "disk")
	url := fmt.Sprintf("/disks/%s/upload", disk.StorageId)
	header := http.Header{"X-Task-Id": []string{task.GetTaskId()}, "X-Region-Version": []string{"v2"}}
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestDeleteSnapshotsWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	url := fmt.Sprintf("/storages/%s/delete-snapshots", snapshot.StorageId)
	body := jsonutils.NewDict()
	body.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestResetDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	url := fmt.Sprintf("/disks/%s/reset/%s", disk.StorageId, disk.Id)
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMHostDriver) RequestCleanUpDiskSnapshots(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	url := fmt.Sprintf("/disks/%s/cleanup-snapshots/%s", disk.StorageId, disk.Id)
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMHostDriver) PrepareConvert(host *models.SHost, image, raid string, data jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	params, err := self.SBaseHostDriver.PrepareConvert(host, image, raid, data)
	if err != nil {
		return nil, err
	}
	var sysSize = "60g"
	log.Infof("%v", data)
	if data.Contains("baremetal_disk_config.0") {
		for i := 0; data.Contains(fmt.Sprintf("baremetal_disk_config.%d", i)); i++ {
			v, _ := data.Get(fmt.Sprintf("baremetal_disk_config.%d", i))
			params.Set(fmt.Sprintf("baremetal_disk_config.%d", i), v)
		}
	} else {
		raid, err = self.GetRaidScheme(host, raid)
		if err != nil {
			return nil, err
		}
		if raid != baremetal.DISK_CONF_NONE {
			params.Set("baremetal_disk_config.0", jsonutils.NewString(fmt.Sprintf("%s:(%s,)", raid, sysSize)))
		}
	}
	if data.Contains("disk.0") {
		for i := 0; data.Contains(fmt.Sprintf("disk.%d", i)); i++ {
			v, _ := data.Get(fmt.Sprintf("disk.%d", i))
			params.Set(fmt.Sprintf("disk.%d", i), v)
		}
	} else {
		if len(image) == 0 {
			image = options.Options.ConvertHypervisorDefaultTemplate
		}
		if len(image) == 0 {
			return nil, fmt.Errorf("Not default image specified for converting %s", self.GetHostType())
		}
		if raid != baremetal.DISK_CONF_NONE {
			params.Set("disk.0", jsonutils.NewString(fmt.Sprintf("%s:autoextend", image)))
		} else if host.StorageInfo.(*jsonutils.JSONArray).Length() > 1 {
			params.Set("disk.0", jsonutils.NewString(fmt.Sprintf("%s:autoextend", image)))
		} else {
			params.Set("disk.0", jsonutils.NewString(fmt.Sprintf("%s:%s", image, sysSize)))
		}
		params.Set("disk.1", jsonutils.NewString("ext4:autoextend:/opt/cloud/workspace"))
	}
	if data.Contains("net.0") {
		for i := 0; data.Contains(fmt.Sprintf("net.%d", i)); i++ {
			v, _ := data.Get(fmt.Sprintf("net.%d", i))
			params.Set(fmt.Sprintf("net.%d", i), v)
		}
	} else {
		wire := host.GetMasterWire()
		if wire == nil {
			return nil, fmt.Errorf("No master wire?")
		}
		params.Set("net.0", jsonutils.NewString(fmt.Sprintf("wire=%s:[private]", wire.GetName())))
	}
	params.Set("deploy.0.path", jsonutils.NewString("/etc/sysconfig/yunionauth"))
	params.Set("deploy.0.action", jsonutils.NewString("create"))
	log.Infof("%v", params)
	authLoc, err := url.Parse(options.Options.AuthURL)
	if err != nil {
		return nil, err
	}
	authInfo := fmt.Sprintf("YUNION_REGION=%s\n", options.Options.Region)
	authInfo += fmt.Sprintf("YUNION_KEYSTONE=%s\n", options.Options.AuthURL)
	authInfo += fmt.Sprintf("YUNION_KEYSTONE_HOST=%s\n", authLoc.Hostname())
	authInfo += fmt.Sprintf("YUNION_HOST_NAME=%s\n", host.GetName())
	authInfo += fmt.Sprintf("YUNION_HOST_ADMIN=%s\n", options.Options.AdminUser)
	authInfo += fmt.Sprintf("YUNION_HOST_PASSWORD=%s\n", options.Options.AdminPassword)
	authInfo += fmt.Sprintf("YUNION_HOST_PROJECT=%s\n", options.Options.AdminProject)
	authInfo += fmt.Sprintf("YUNION_START=yes\n")
	params.Set("deploy.0.content", jsonutils.NewString(authInfo))
	return params, nil
}

func (self *SKVMHostDriver) PrepareUnconvert(host *models.SHost) error {
	hoststorages := host.GetHoststorages()
	if hoststorages == nil {
		return self.SBaseHostDriver.PrepareUnconvert(host)
	}
	for i := 0; i < len(hoststorages); i++ {
		storage := hoststorages[i].GetStorage()
		if storage.IsLocal() && storage.StorageType != models.STORAGE_BAREMETAL && storage.GetDiskCount() > 0 {
			return fmt.Errorf("Local host storage is not empty??? %s", storage.GetName())
		}
	}
	return self.SBaseHostDriver.PrepareUnconvert(host)
}

func (self *SKVMHostDriver) FinishUnconvert(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost) error {
	for _, hs := range host.GetHoststorages() {
		storage := hs.GetStorage()
		if storage == nil {
			continue
		}
		if storage.StorageType != models.STORAGE_BAREMETAL {
			hs.Delete(ctx, userCred)
			if storage.IsLocal() {
				storage.Delete(ctx, userCred)
			}
		}
	}
	kwargs := make(map[string]interface{}, 0)
	for _, k := range []string{"kernel_version", "nest", "os_distribution", "os_version",
		"ovs_version", "qemu_version", "storage_type"} {
		kwargs[k] = "None"
	}
	host.SetAllMetadata(ctx, kwargs, userCred)
	return self.SBaseHostDriver.FinishUnconvert(ctx, userCred, host)
}
