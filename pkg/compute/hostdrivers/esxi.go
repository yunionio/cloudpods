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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SESXiHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SESXiHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SESXiHostDriver) GetHostType() string {
	return api.HOST_TYPE_ESXI
}

func (self *SESXiHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_ESXI
}

func (self *SESXiHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SESXiHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (self *SESXiHostDriver) CheckAndSetCacheImage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
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
	var srcHostCacheImage *models.SStoragecachedimage
	// Check if storageCache has this cacheImage.
	// If no, choose source storage cache
	// else, use it
	hostCacheImage := models.StoragecachedimageManager.GetStoragecachedimage(storageCache.GetId(), cacheImage.GetId())
	if hostCacheImage == nil {
		zone, _ := host.GetZone()
		srcHostCacheImage, err = cacheImage.ChooseSourceStoragecacheInRange(api.HOST_TYPE_ESXI, []string{host.Id},
			[]interface{}{zone, host.GetCloudprovider()})
		if err != nil {
			return err
		}
	}

	type contentStruct struct {
		ImageId            string
		HostId             string
		HostIp             string
		SrcHostIp          string
		SrcPath            string
		SrcDatastore       vcenter.SVCenterAccessInfo
		Datastore          vcenter.SVCenterAccessInfo
		Format             string
		IsForce            bool
		StoragecacheId     string
		ImageType          string
		ImageExternalId    string
		StorageCacheHostIp string
	}

	content := contentStruct{}
	content.ImageId = imageId
	content.HostId = host.Id
	content.HostIp = host.AccessIp
	// format force VMDK
	format := cacheImage.GetFormat()
	if format == "qcow2" {
		format = "vmdk"
	}
	content.Format = format

	content.ImageType = cacheImage.ImageType
	content.ImageExternalId = cacheImage.ExternalId

	storage := host.GetStorageByFilePath(storageCache.Path)
	if storage == nil {
		msg := fmt.Sprintf("fail to find storage for storageCache %s", storageCache.Path)
		log.Errorf(msg)
		return errors.Error(msg)
	}

	accessInfo, err := host.GetCloudaccount().GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}
	content.Datastore = accessInfo

	if cloudprovider.TImageType(cacheImage.ImageType) == cloudprovider.ImageTypeSystem {
		var host *models.SHost
		if srcHostCacheImage != nil {
			host, err = srcHostCacheImage.GetHost()
			if err != nil {
				return errors.Wrap(err, "srcHostCacheImage.GetHost")
			}
		} else {
			host, err = storageCache.GetMasterHost()
			if err != nil {
				return errors.Wrap(err, "StorageCache.GetHost")
			}
		}
		content.StorageCacheHostIp = host.AccessIp
	} else if srcHostCacheImage != nil {
		err = srcHostCacheImage.AddDownloadRefcount()
		if err != nil {
			return err
		}
		srcHost, err := srcHostCacheImage.GetHost()
		if err != nil {
			return err
		}
		content.SrcHostIp = srcHost.AccessIp
		content.SrcPath = srcHostCacheImage.Path
		srcStorageCache := srcHostCacheImage.GetStoragecache()
		if srcStorageCache == nil {
			return errors.Wrap(errors.ErrNotFound, "StorageCacheImage.GetStoragecaceh")
		}
		srcStorage := srcHost.GetStorageByFilePath(srcStorageCache.Path)
		accessInfo, err := srcHost.GetCloudaccount().GetVCenterAccessInfo(srcStorage.ExternalId)
		if err != nil {
			return err
		}
		content.SrcDatastore = accessInfo
	}

	if !host.IsEsxiAgentReady() {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	url := "/disks/image_cache"

	if isForce {
		content.IsForce = true
	}
	content.StoragecacheId = storageCache.Id

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&content), "disk")

	header := task.GetTaskRequestHeader()

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	if err != nil {
		return err
	}
	return nil
}

func (self *SESXiHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	if !host.IsEsxiAgentReady() {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	type specStruct struct {
		Datastore vcenter.SVCenterAccessInfo
	}

	input.HostIp = host.AccessIp
	input.Format = "vmdk"

	var err error
	input.Datastore, err = host.GetCloudaccount().GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&input), "disk")

	url := fmt.Sprintf("/disks/agent/create/%s", disk.Id)

	header := task.GetTaskRequestHeader()

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	return err
}

func (self *SESXiHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	if !host.IsEsxiAgentReady() {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	guests := disk.GetGuests()
	if len(guests) == 0 {
		return fmt.Errorf("No VM associate with this disk")
	}

	if len(guests) > 1 {
		return fmt.Errorf("The disk is attached to multiple guests")
	}

	guest := guests[0]

	if guest.HostId != host.Id {
		return fmt.Errorf("The only guest is not on the host????")
	}

	type specStruct struct {
		Vm      vcenter.SVCenterAccessInfo
		Disk    vcenter.SVCenterAccessInfo
		HostIp  string
		ImageId string
	}

	spec := specStruct{}
	spec.HostIp = host.AccessIp
	spec.ImageId = imageId

	account := host.GetCloudaccount()
	accessInfo, err := account.GetVCenterAccessInfo(guest.ExternalId)
	if err != nil {
		return err
	}
	spec.Vm = accessInfo

	accessInfo, err = account.GetVCenterAccessInfo(disk.ExternalId)
	if err != nil {
		return err
	}
	spec.Disk = accessInfo

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&spec), "disk")

	url := fmt.Sprintf("/disks/agent/save-prepare/%s", disk.Id)
	header := task.GetTaskRequestHeader()

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	return err
}

func (self *SESXiHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	guest := disk.GetGuest()
	if guest == nil {
		return fmt.Errorf("unable to find guest has disk %s", disk.GetId())
	}

	iVm, err := guest.GetIVM(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetIVM")
	}
	if iVm.GetStatus() == api.VM_RUNNING {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			disks, err := iVm.GetIDisks()
			if err != nil {
				return nil, errors.Wrapf(err, "GetIDisk")
			}
			for i := range disks {
				if disks[i].GetGlobalId() == disk.ExternalId {
					err = disks[i].Resize(ctx, sizeMb)
					if err != nil {
						return nil, errors.Wrapf(err, "Resize")
					}
					return jsonutils.Marshal(map[string]int64{"disk_size": sizeMb}), nil
				}
			}
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, "disk %s", disk.Name)
		})
		return nil
	}
	spec := struct {
		HostInfo vcenter.SVCenterAccessInfo
		VMId     string
		DiskId   string
		SizeMb   int64
	}{}

	account := host.GetCloudaccount()
	accessInfo, err := account.GetVCenterAccessInfo(host.ExternalId)
	if err != nil {
		return err
	}
	spec.HostInfo = accessInfo
	spec.DiskId = disk.GetExternalId()
	spec.VMId = guest.GetExternalId()
	spec.SizeMb = sizeMb

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(spec), "disk")

	url := fmt.Sprintf("/disks/agent/resize/%s", disk.Id)
	header := task.GetTaskRequestHeader()

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	return err
}

func (self *SESXiHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {

	imagePath, _ := data.GetString("backup")
	if len(imagePath) == 0 {
		return fmt.Errorf("missing parameter backup")
	}
	// agentId, _ := data.GetString("agent_id")
	// if len(agentId) == 0 {
	// 	return fmt.Errorf("missing parameter agent_id")
	// }

	// agent := models.HostManager.FetchHostById(agentId)
	// if agent == nil {
	// 	return fmt.Errorf("cannot find host with id %s", agentId)
	// }

	storage, _ := disk.GetStorage()

	type specStruct struct {
		ImagePath      string
		ImageId        string
		StorageId      string
		StoragecacheId string
		Compress       bool `json:",allowfalse"`
	}

	spec := specStruct{}
	spec.ImageId = imageId
	spec.ImagePath = imagePath
	spec.StorageId = storage.Id
	spec.StoragecacheId = storage.StoragecacheId
	spec.Compress = false

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&spec), "disk")

	url := "/disks/agent/upload"

	header := task.GetTaskRequestHeader()

	_, err := host.EsxiRequest(ctx, httputils.POST, url, header, body)
	return err
}
