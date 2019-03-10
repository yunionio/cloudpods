package hostdrivers

import (
	"context"
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SKVMHostDriver struct {
	SVirtualizationHostDriver
}

func init() {
	driver := SKVMHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SKVMHostDriver) GetHostType() string {
	return models.HOST_TYPE_HYPERVISOR
}

func (self *SKVMHostDriver) ValidateAttachStorage(host *models.SHost, storage *models.SStorage, data *jsonutils.JSONDict) error {
	if !utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_LOCAL, models.STORAGE_RBD}) {
		return httperrors.NewUnsupportOperationError("Unsupport attach %s storage for %s host", storage.StorageType, host.HostType)
	}
	if storage.StorageType == models.STORAGE_RBD {
		if host.HostStatus != models.HOST_ONLINE {
			return httperrors.NewInvalidStatusError("Attach rbd storage require host status is online")
		}
		pool, _ := storage.StorageConf.GetString("pool")
		data.Set("mount_point", jsonutils.NewString(fmt.Sprintf("rbd:%s", pool)))
	}
	return nil
}

func (self *SKVMHostDriver) RequestAttachStorage(ctx context.Context, hoststorage *models.SHoststorage, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if storage.StorageType == models.STORAGE_RBD {
			log.Infof("Attach SharedStorage[%s] on host %s ...", storage.Name, host.Name)
			url := fmt.Sprintf("%s/storages/attach", host.ManagerUri)
			headers := mcclient.GetTokenHeaders(task.GetUserCred())
			data := map[string]interface{}{
				"mount_point":  hoststorage.MountPoint,
				"name":         storage.Name,
				"storage_id":   storage.Id,
				"storage_conf": storage.StorageConf,
				"storage_type": storage.StorageType,
			}
			if len(storage.StoragecacheId) > 0 {
				storagecache := models.StoragecacheManager.FetchStoragecacheById(storage.StoragecacheId)
				if storagecache != nil {
					data["imagecache_path"] = storage.GetStorageCachePath(hoststorage.MountPoint, storagecache.Path)
					data["storagecache_id"] = storagecache.Id
				}
			}
			_, resp, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, headers, jsonutils.Marshal(data), false)
			return resp, err
		}
		return nil, nil
	})
	return nil
}

func (self *SKVMHostDriver) RequestDetachStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if storage.StorageType == models.STORAGE_RBD && host.HostStatus == models.HOST_ONLINE {
			log.Infof("Detach SharedStorage[%s] on host %s ...", storage.Name, host.Name)
			url := fmt.Sprintf("%s/storages/detach", host.ManagerUri)
			headers := mcclient.GetTokenHeaders(task.GetUserCred())
			body := jsonutils.NewDict()
			mountPoint, _ := task.GetParams().GetString("mount_point")
			body.Set("mount_point", jsonutils.NewString(mountPoint))
			body.Set("name", jsonutils.NewString(storage.Name))
			_, resp, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, headers, body, false)
			return resp, err
		}
		return nil, nil
	})
	return nil
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
	format, _ := params.GetString("format")
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

	type contentStruct struct {
		ImageId        string
		Format         string
		SrcUrl         string
		IsForce        bool
		StoragecacheId string
	}

	content := contentStruct{}
	content.ImageId = imageId
	content.Format = format

	if srcHostCacheImage != nil {
		err = srcHostCacheImage.AddDownloadRefcount()
		if err != nil {
			return err
		}
		srcHost, err := srcHostCacheImage.GetHost()
		if err != nil {
			return err
		}
		content.SrcUrl = fmt.Sprintf("%s/download/images/%s", srcHost.ManagerUri, imageId)
	}
	url := fmt.Sprintf("%s/disks/image_cache", host.ManagerUri)

	if isForce {
		content.IsForce = true
	}
	content.StoragecacheId = storageCache.Id
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&content), "disk")

	header := task.GetTaskRequestHeader()

	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMHostDriver) RequestUncacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	type contentStruct struct {
		ImageId        string
		StoragecacheId string
	}

	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}

	content := contentStruct{}
	content.ImageId = imageId
	content.StoragecacheId = storageCache.Id

	url := fmt.Sprintf("%s/disks/image_cache", host.ManagerUri)

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&content), "disk")

	header := task.GetTaskRequestHeader()

	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	header := task.GetTaskRequestHeader()
	if snapshotId, err := content.GetString("snapshot"); err == nil {
		iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
		snapshot := iSnapshot.(*models.SSnapshot)
		snapshotStorage := models.StorageManager.FetchStorageById(snapshot.StorageId)
		snapshotHost := snapshotStorage.GetMasterHost()
		if options.Options.SnapshotCreateDiskProtocol == "url" {
			content.Set("snapshot_url",
				jsonutils.NewString(fmt.Sprintf("%s/download/snapshots/%s/%s/%s",
					snapshotHost.ManagerUri, snapshotStorage.Id, snapshot.DiskId, snapshot.Id)))
			content.Set("snapshot_out_of_chain", jsonutils.NewBool(snapshot.OutOfChain))
		} else if options.Options.SnapshotCreateDiskProtocol == "fuse" {
			content.Set("snapshot_url", jsonutils.NewString(fmt.Sprintf("%s/snapshots/%s/%s",
				snapshotHost.GetFetchUrl(), snapshot.DiskId, snapshot.Id)))
		}
		content.Set("protocol", jsonutils.NewString(options.Options.SnapshotCreateDiskProtocol))
	}

	url := fmt.Sprintf("/disks/%s/create/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	body.Add(content, "disk")
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	content.Add(jsonutils.JSONTrue, "rebuild")
	return self.RequestAllocateDiskOnStorage(ctx, host, storage, disk, task, content)
}

func (self *SKVMHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	log.Infof("Deallocating disk on host %s", host.GetName())
	header := task.GetTaskRequestHeader()

	url := fmt.Sprintf("/disks/%s/delete/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (driver *SKVMHostDriver) RequestDeallocateBackupDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	log.Infof("Deallocating disk on host %s", host.GetName())
	header := mcclient.GetTokenHeaders(task.GetUserCred())
	url := fmt.Sprintf("/disks/%s/delete/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, guest *models.SGuest, sizeMb int64, task taskman.ITask) error {
	header := task.GetTaskRequestHeader()

	url := fmt.Sprintf("/disks/%s/resize/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	content := jsonutils.NewDict()
	content.Add(jsonutils.NewInt(sizeMb), "size")
	if guest != nil {
		content.Add(jsonutils.NewString(guest.Id), "server_id")
	}
	body.Add(content, "disk")
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

/*
func (self *SKVMHostDriver) RequestResizeDiskOnHostOnline(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	header := task.GetTaskRequestHeader()

	guests := disk.GetAttachedGuests()
	if len(guests) == 0 {
		return fmt.Errorf("no valid guest")
	}
	if len(guests) > 1 {
		return fmt.Errorf("cannot resize across more than 1 guest")
	}
	guest := guests[0]

	guestdisk := guest.GetGuestDisk(disk.GetId())
	url := fmt.Sprintf("/servers/%s/monitor", guest.GetId())
	body := jsonutils.NewDict()
	cmd := fmt.Sprintf("block_resize drive_%d %dM", guestdisk.Index, sizeMb)
	body.Add(jsonutils.NewString(cmd), "cmd")

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	})

	return nil
}
*/

func (self *SKVMHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(map[string]string{"image_id": imageId}), "disk")
	url := fmt.Sprintf("/disks/%s/save-prepare/%s", disk.StorageId, disk.Id)

	header := task.GetTaskRequestHeader()

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
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

	header := task.GetTaskRequestHeader()

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestDeleteSnapshotsWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	url := fmt.Sprintf("/storages/%s/delete-snapshots", snapshot.StorageId)
	body := jsonutils.NewDict()
	body.Set("disk_id", jsonutils.NewString(snapshot.DiskId))

	header := task.GetTaskRequestHeader()

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestResetDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	url := fmt.Sprintf("/disks/%s/reset/%s", disk.StorageId, disk.Id)

	header := task.GetTaskRequestHeader()

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMHostDriver) RequestCleanUpDiskSnapshots(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	url := fmt.Sprintf("/disks/%s/cleanup-snapshots/%s", disk.StorageId, disk.Id)

	header := task.GetTaskRequestHeader()

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMHostDriver) PrepareConvert(host *models.SHost, image, raid string, data jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	params, err := self.SBaseHostDriver.PrepareConvert(host, image, raid, data)
	if err != nil {
		return nil, err
	}
	var sysSize = "60g"
	if data.Contains("baremetal_disk_config.0") {
		jsonArray := jsonutils.GetArrayOfPrefix(data, "baremetal_disk_config")
		for i := 0; i < len(jsonArray); i += 1 { // } .Contains(fmt.Sprintf("baremetal_disk_config.%d", i)); i++ {
			// v, _ := data.Get(fmt.Sprintf("baremetal_disk_config.%d", i))
			params.Set(fmt.Sprintf("baremetal_disk_config.%d", i), jsonArray[i])
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
		jsonArray := jsonutils.GetArrayOfPrefix(data, "disk")
		for i := 0; i < len(jsonArray); i += 1 { // } data.Contains(fmt.Sprintf("disk.%d", i)); i++ {
			// v, _ := data.Get(fmt.Sprintf("disk.%d", i))
			params.Set(fmt.Sprintf("disk.%d", i), jsonArray[i])
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
		jsonArray := jsonutils.GetArrayOfPrefix(data, "net")
		for i := 0; i < len(jsonArray); i += 1 { // } data.Contains(fmt.Sprintf("net.%d", i)); i++ {
			// v, _ := data.Get(fmt.Sprintf("net.%d", i))
			params.Set(fmt.Sprintf("net.%d", i), jsonArray[i])
		}
	} else {
		wire := host.GetMasterWire()
		if wire == nil {
			return nil, fmt.Errorf("No master wire?")
		}
		params.Set("net.0", jsonutils.NewString(fmt.Sprintf("wire=%s:[private]:[try-teaming]", wire.GetName())))
	}
	params.Set("deploy.0.path", jsonutils.NewString("/etc/sysconfig/yunionauth"))
	params.Set("deploy.0.action", jsonutils.NewString("create"))
	log.Infof("%v", params)
	authLoc, err := url.Parse(options.Options.AuthURL)
	if err != nil {
		return nil, err
	}
	portStr := authLoc.Port()
	useSsl := ""
	if authLoc.Scheme == "https" {
		useSsl = "yes"
		if len(portStr) == 0 {
			portStr = "443"
		}
	} else {
		if len(portStr) == 0 {
			portStr = "80"
		}
	}
	authInfo := fmt.Sprintf("YUNION_REGION=%s\n", options.Options.Region)
	authInfo += fmt.Sprintf("YUNION_KEYSTONE=%s\n", options.Options.AuthURL)
	authInfo += fmt.Sprintf("YUNION_KEYSTONE_HOST=%s\n", authLoc.Hostname())
	authInfo += fmt.Sprintf("YUNION_KEYSTONE_PORT=%s\n", portStr)
	authInfo += fmt.Sprintf("YUNION_KEYSTONE_USE_SSL=%s\n", useSsl)
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
