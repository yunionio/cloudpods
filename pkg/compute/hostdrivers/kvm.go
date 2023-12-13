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

package hostdrivers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/k8s/tokens"
)

type SKVMHostDriver struct {
	SVirtualizationHostDriver
}

func init() {
	driver := SKVMHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SKVMHostDriver) GetHostType() string {
	return api.HOST_TYPE_HYPERVISOR
}

func (self *SKVMHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_KVM
}

func (self *SKVMHostDriver) validateGPFS(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, input api.HostStorageCreateInput) (api.HostStorageCreateInput, error) {
	header := http.Header{}
	header.Set(mcclient.AUTH_TOKEN, userCred.GetTokenString())
	header.Set(mcclient.REGION_VERSION, "v2")
	params := jsonutils.NewDict()
	params.Set("mount_point", jsonutils.NewString(input.MountPoint))
	urlStr := fmt.Sprintf("%s/storages/is-mount-point?%s", host.ManagerUri, params.QueryString())
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", urlStr, header, nil, false)
	if err != nil {
		return input, err
	}
	if !jsonutils.QueryBoolean(res, "is_mount_point", false) {
		return input, httperrors.NewBadRequestError("%s is not mount point %s", input.MountPoint, res)
	}
	urlStr = fmt.Sprintf("%s/storages/is-local-mount-point?%s", host.ManagerUri, params.QueryString())
	_, res, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", urlStr, header, nil, false)
	if err != nil {
		return input, err
	}
	if jsonutils.QueryBoolean(res, "is_local_mount_point", false) {
		return input, httperrors.NewBadRequestError("%s is local storage mount point", input.MountPoint)
	}
	return input, nil
}

func (self *SKVMHostDriver) validateCLVM(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, input api.HostStorageCreateInput) (api.HostStorageCreateInput, error) {
	vgName, _ := storage.StorageConf.GetString("clvm_vg_name")
	if vgName == "" {
		return input, httperrors.NewInternalServerError("storage has no clvm_vg_name")
	}
	input.MountPoint = vgName

	header := http.Header{}
	header.Set(mcclient.AUTH_TOKEN, userCred.GetTokenString())
	header.Set(mcclient.REGION_VERSION, "v2")
	params := jsonutils.NewDict()
	params.Set("vg_name", jsonutils.NewString(input.MountPoint))
	urlStr := fmt.Sprintf("%s/storages/is-vg-exist?%s", host.ManagerUri, params.QueryString())
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", urlStr, header, nil, false)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SKVMHostDriver) ValidateAttachStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, input api.HostStorageCreateInput) (api.HostStorageCreateInput, error) {
	if !utils.IsInStringArray(storage.StorageType, append([]string{api.STORAGE_LOCAL, api.STORAGE_LVM}, api.SHARED_STORAGE...)) {
		return input, httperrors.NewUnsupportOperationError("Unsupport attach %s storage for %s host", storage.StorageType, host.HostType)
	}
	if storage.StorageType == api.STORAGE_RBD {
		if host.HostStatus != api.HOST_ONLINE {
			return input, httperrors.NewInvalidStatusError("Attach rbd storage require host status is online")
		}
		pool, _ := storage.StorageConf.GetString("pool")
		input.MountPoint = fmt.Sprintf("rbd:%s", pool)
	} else if utils.IsInStringArray(storage.StorageType, api.SHARED_FILE_STORAGE) {
		if len(input.MountPoint) == 0 {
			return input, httperrors.NewMissingParameterError("mount_point")
		}
		count, err := models.HoststorageManager.Query().Equals("host_id", host.Id).Equals("mount_point", input.MountPoint).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("Query host storage error %s", err)
		}
		if count > 0 {
			return input, httperrors.NewBadRequestError("Host %s already have mount point %s with other storage", host.Name, input.MountPoint)
		}
		if host.HostStatus != api.HOST_ONLINE {
			return input, httperrors.NewInvalidStatusError("Attach nfs storage require host status is online")
		}
		if storage.StorageType == api.STORAGE_GPFS {
			return self.validateGPFS(ctx, userCred, host, input)
		}
	} else if storage.StorageType == api.STORAGE_CLVM {
		return self.validateCLVM(ctx, userCred, host, storage, input)
	}
	return input, nil
}

func (self *SKVMHostDriver) RequestAttachStorage(ctx context.Context, hoststorage *models.SHoststorage, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if utils.IsInStringArray(storage.StorageType, api.SHARED_STORAGE) {
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
		if utils.IsInStringArray(storage.StorageType, api.SHARED_STORAGE) && host.HostStatus == api.HOST_ONLINE {
			log.Infof("Detach SharedStorage[%s] on host %s ...", storage.Name, host.Name)
			url := fmt.Sprintf("%s/storages/detach", host.ManagerUri)
			headers := mcclient.GetTokenHeaders(task.GetUserCred())
			body := jsonutils.NewDict()
			mountPoint, _ := task.GetParams().GetString("mount_point")
			body.Set("mount_point", jsonutils.NewString(mountPoint))
			body.Set("name", jsonutils.NewString(storage.Name))
			body.Set("storage_id", jsonutils.NewString(storage.Id))
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
	input := api.CacheImageInput{}
	task.GetParams().Unmarshal(&input)

	var srcHost *models.SHost
	if len(input.SourceHostId) > 0 {
		srcHost = models.HostManager.FetchHostById(input.SourceHostId)
		if srcHost == nil {
			return errors.Errorf("Source host %s not found", input.SourceHostId)
		}
	}

	obj, err := models.CachedimageManager.FetchById(input.ImageId)
	if err != nil {
		return errors.Wrapf(err, "Fetch cached image by image_id %s", input.ImageId)
	}
	cacheImage := obj.(*models.SCachedimage)
	zone, _ := host.GetZone()
	rangeObjs := []interface{}{zone}
	if srcHost != nil {
		rangeObjs = append(rangeObjs, srcHost)
	}
	srcHostCacheImage, err := cacheImage.ChooseSourceStoragecacheInRange(api.HOST_TYPE_HYPERVISOR, []string{host.Id}, rangeObjs)
	if err != nil {
		return errors.Wrapf(err, "Choose source storagecache")
	}
	if srcHostCacheImage != nil {
		err = srcHostCacheImage.AddDownloadRefcount()
		if err != nil {
			return err
		}

		srcHost, err := srcHostCacheImage.GetHost()
		if err != nil {
			return errors.Wrapf(err, "Get storage cached image %s host", srcHostCacheImage.GetId())
		}
		input.SrcUrl = fmt.Sprintf("%s/download/images/%s", srcHost.ManagerUri, input.ImageId)
	}

	url := fmt.Sprintf("%s/disks/image_cache", host.ManagerUri)

	input.StoragecacheId = storageCache.Id
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&input), "disk")

	header := task.GetTaskRequestHeader()

	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
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

func (self *SKVMHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	header := task.GetTaskRequestHeader()
	if len(input.SnapshotId) > 0 {
		snapObj, err := models.SnapshotManager.FetchById(input.SnapshotId)
		if err != nil {
			return errors.Wrapf(err, "SnapshotManager.FetchById(%s)", input.SnapshotId)
		}
		snapshot := snapObj.(*models.SSnapshot)
		snapshotStorage := models.StorageManager.FetchStorageById(snapshot.StorageId)
		if snapshotStorage.StorageType == api.STORAGE_LOCAL {
			snapshotHost, err := snapshotStorage.GetMasterHost()
			if err != nil {
				return errors.Wrapf(err, "GetMasterHost")
			}
			if options.Options.SnapshotCreateDiskProtocol == "url" {
				input.SnapshotUrl = fmt.Sprintf("%s/download/snapshots/%s/%s/%s", snapshotHost.ManagerUri, snapshotStorage.Id, snapshot.DiskId, snapshot.Id)
				input.SnapshotOutOfChain = snapshot.OutOfChain
			} else if options.Options.SnapshotCreateDiskProtocol == "fuse" {
				input.SnapshotUrl = fmt.Sprintf("%s/snapshots/%s/%s", snapshotHost.GetFetchUrl(true), snapshot.DiskId, snapshot.Id)
			}
			input.Protocol = options.Options.SnapshotCreateDiskProtocol
		} else if snapshotStorage.StorageType == api.STORAGE_RBD {
			input.SnapshotUrl = snapshot.Id
			input.SrcDiskId = snapshot.DiskId
			input.SrcPool, _ = snapshotStorage.StorageConf.GetString("pool")
		} else {
			input.SnapshotUrl = snapshot.Location
		}
	}

	url := fmt.Sprintf("/disks/%s/create/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(input), "disk")
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	input.Rebuild = true
	input.BackingDiskId, _ = task.GetParams().GetString("backing_disk_id")
	return self.RequestAllocateDiskOnStorage(ctx, task.GetUserCred(), host, storage, disk, task, input)
}

func (self *SKVMHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	log.Infof("Deallocating disk on host %s", host.GetName())
	header := task.GetTaskRequestHeader()

	url := fmt.Sprintf("/disks/%s/delete/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	if flatPath := disk.GetMetadata(ctx, api.DISK_META_ESXI_FLAT_FILE_PATH, nil); flatPath != "" {
		body.Set("esxi_flat_file_path", jsonutils.NewString(flatPath))
	}
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return task.ScheduleRun(nil)
		}
		return err
	}
	return nil
}

func (driver *SKVMHostDriver) RequestDeallocateBackupDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	log.Infof("Deallocating disk on host %s", host.GetName())
	header := mcclient.GetTokenHeaders(task.GetUserCred())
	url := fmt.Sprintf("/disks/%s/delete/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	header := task.GetTaskRequestHeader()

	url := fmt.Sprintf("/disks/%s/resize/%s", storage.Id, disk.Id)
	body := jsonutils.NewDict()
	content := jsonutils.NewDict()
	content.Add(jsonutils.NewInt(sizeMb), "size")
	if disk.IsEncrypted() {
		info, err := disk.GetEncryptInfo(ctx, task.GetUserCred())
		if err != nil {
			return errors.Wrap(err, "disk.GetEncryptInfo")
		}
		content.Add(jsonutils.Marshal(info), "encrypt_info")
	}
	guest := disk.GetGuest()
	if guest != nil {
		content.Add(jsonutils.NewString(guest.Id), "server_id")
	}
	body.Add(content, "disk")
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

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
	storage, _ := disk.GetStorage()
	content := map[string]string{
		"image_path":       backup,
		"image_id":         imageId,
		"storagecached_id": storage.StoragecacheId,
	}
	if disk.IsEncrypted() {
		content["encrypt_key_id"] = disk.EncryptKeyId
	}
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

func (self *SKVMHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if len(guests) > 1 {
		return nil, httperrors.NewBadRequestError("Disk attach muti guests")
	} else if len(guests) == 1 {
		if guests[0].Status != api.VM_READY {
			return nil, httperrors.NewServerStatusError("Disk attached guest status must be ready")
		}
	} else {
		return nil, httperrors.NewBadRequestError("Disk dosen't attach guest")
	}

	return data, nil
}

func (self *SKVMHostDriver) RequestResetDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	url := fmt.Sprintf("/disks/%s/reset/%s", disk.StorageId, disk.Id)

	header := task.GetTaskRequestHeader()

	if disk.IsEncrypted() {
		info, err := disk.GetEncryptInfo(ctx, task.GetUserCred())
		if err != nil {
			return errors.Wrap(err, "disk.GetEncryptInfo")
		}
		params.Add(jsonutils.Marshal(info), "encrypt_info")
	}

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMHostDriver) RequestCleanUpDiskSnapshots(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	url := fmt.Sprintf("/disks/%s/cleanup-snapshots/%s", disk.StorageId, disk.Id)

	header := task.GetTaskRequestHeader()

	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMHostDriver) PrepareConvert(host *models.SHost, image, raid string, data jsonutils.JSONObject) (*api.ServerCreateInput, error) {
	params, err := self.SBaseHostDriver.PrepareConvert(host, image, raid, data)
	if err != nil {
		return nil, err
	}
	var sysSize = "60g"
	raidConfs, _ := cmdline.FetchBaremetalDiskConfigsByJSON(data)
	if len(raidConfs) == 0 {
		raid, err = self.GetRaidScheme(host, raid)
		if err != nil {
			return nil, err
		}
		if raid != baremetal.DISK_CONF_NONE {
			raidConfs = []*api.BaremetalDiskConfig{
				{
					Conf:   raid,
					Splits: fmt.Sprintf("%s,", sysSize),
					Type:   api.DISK_TYPE_HYBRID,
				},
			}
		}
	}
	params.BaremetalDiskConfigs = raidConfs
	disks, _ := cmdline.FetchDiskConfigsByJSON(data)
	if len(disks) == 0 {
		if len(image) == 0 {
			image = options.Options.ConvertHypervisorDefaultTemplate
		}
		if len(image) == 0 {
			return nil, fmt.Errorf("Not default image specified for converting %s", self.GetHostType())
		}
		rootDisk := &api.DiskConfig{}
		if raid != baremetal.DISK_CONF_NONE {
			rootDisk = &api.DiskConfig{
				ImageId: image,
				SizeMb:  -1,
			}
		} else if host.StorageInfo.(*jsonutils.JSONArray).Length() > 1 {
			rootDisk = &api.DiskConfig{
				ImageId: image,
				SizeMb:  -1,
			}
		} else {
			rootDisk = &api.DiskConfig{
				ImageId: image,
				SizeMb:  60 * 1024, // 60g
			}
		}
		optDisk := &api.DiskConfig{
			Fs:         "ext4",
			SizeMb:     -1,
			Mountpoint: "/opt/cloud/workspace",
		}
		disks = append(disks, rootDisk, optDisk)
	}
	params.Disks = disks
	nets, _ := cmdline.FetchNetworkConfigsByJSON(data)
	if len(nets) == 0 {
		wire := host.GetMasterWire()
		if wire == nil {
			return nil, fmt.Errorf("No master wire?")
		}
		net := &api.NetworkConfig{
			Wire:       wire.GetId(),
			Private:    true,
			TryTeaming: true,
		}
		nets = append(nets, net)
	}
	params.Networks = nets

	deployConfigs, err := self.getDeployConfig(host)
	if err != nil {
		return nil, err
	}
	params.DeployConfigs = deployConfigs
	return params, nil
}

func (self *SKVMHostDriver) getDeployConfig(host *models.SHost) ([]*api.DeployConfig, error) {
	deployConf := &api.DeployConfig{
		Action: "create",
		Path:   "/etc/sysconfig/yunionauth",
	}
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
	apiServer, err := tokens.GetControlPlaneEndpoint()
	if err != nil {
		log.Errorf("Failed to get kubernetes controlplane endpoint: %v", err)
	}
	joinToken, err := tokens.GetNodeJoinToken()
	if err != nil {
		log.Errorf("Failed to get kubernetes node join token: %v", err)
	}
	authInfo += fmt.Sprintf("API_SERVER=%s\n", apiServer)
	authInfo += fmt.Sprintf("JOIN_TOKEN=%s\n", joinToken)
	if apiServer != "" {
		dockerCfg, err := tokens.GetDockerDaemonContent()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get docker daemon config")
		}
		authInfo += fmt.Sprintf("DOCKER_DAEMON_JSON=%s\n", dockerCfg)
	}
	deployConf.Content = authInfo
	return []*api.DeployConfig{deployConf}, nil
}

func (self *SKVMHostDriver) PrepareUnconvert(host *models.SHost) error {
	hoststorages := host.GetHoststorages()
	if hoststorages == nil {
		return self.SBaseHostDriver.PrepareUnconvert(host)
	}
	for i := 0; i < len(hoststorages); i++ {
		storage := hoststorages[i].GetStorage()
		if storage.IsLocal() && storage.StorageType != api.STORAGE_BAREMETAL {
			cnt, err := storage.GetDiskCount()
			if err != nil {
				return err
			}
			if cnt > 0 {
				return fmt.Errorf("Local host storage is not empty??? %s", storage.GetName())
			}
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
		if storage.StorageType != api.STORAGE_BAREMETAL {
			hs.Delete(ctx, userCred)
			if storage.IsLocal() {
				storage.Delete(ctx, userCred)
			}
		}
	}
	onK8s := host.GetMetadata(ctx, "on_kubernetes", userCred)
	hostname := host.GetMetadata(ctx, "hostname", userCred)
	if strings.ToLower(onK8s) == "true" {
		if err := self.tryCleanKubernetesData(host, hostname); err != nil {
			log.Errorf("try clean kubernetes data: %v", err)
		}
	}
	kwargs := make(map[string]interface{}, 0)
	for _, k := range []string{
		"kernel_version", "nest", "os_distribution", "os_version",
		"ovs_version", "qemu_version", "storage_type", "on_kubernetes",
	} {
		kwargs[k] = "None"
	}
	host.SetAllMetadata(ctx, kwargs, userCred)
	return self.SBaseHostDriver.FinishUnconvert(ctx, userCred, host)
}

func (self *SKVMHostDriver) tryCleanKubernetesData(host *models.SHost, hostname string) error {
	cli, err := tokens.GetCoreClient()
	if err != nil {
		return errors.Wrap(err, "get k8s client")
	}
	if hostname == "" {
		hostname = host.GetName()
	}
	return cli.Nodes().Delete(context.Background(), hostname, metav1.DeleteOptions{})
}

func (self *SKVMHostDriver) RequestSyncOnHost(ctx context.Context, host *models.SHost, task taskman.ITask) error {
	log.Infof("Deallocating disk on host %s", host.GetName())
	header := mcclient.GetTokenHeaders(task.GetUserCred())
	url := fmt.Sprintf("/hosts/%s/sync", host.Id)
	body := jsonutils.NewDict()
	desc := self.GetJsonFromHost(ctx, host)
	body.Add(desc, "desc")
	_, err := host.Request(ctx, task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMHostDriver) GetJsonFromHost(ctx context.Context, host *models.SHost) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(host.Name), "name")
	// tenant
	domainFetcher, _ := db.DefaultDomainFetcher(ctx, host.DomainId)
	if domainFetcher != nil {
		desc.Add(jsonutils.NewString(domainFetcher.GetProjectDomainId()), "domain_id")
		desc.Add(jsonutils.NewString(domainFetcher.GetProjectDomain()), "project_domain")
	}
	return desc
}

func (driver *SKVMHostDriver) RequestProbeIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, input jsonutils.JSONObject) (*jsonutils.JSONArray, error) {
	url := fmt.Sprintf("%s/hosts/%s/probe-isolated-devices", host.ManagerUri, host.GetId())
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	_, respBody, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, input, false)
	if err != nil {
		return nil, errors.Wrapf(err, "send to host %s", url)
	}
	return respBody.(*jsonutils.JSONArray), err
}
