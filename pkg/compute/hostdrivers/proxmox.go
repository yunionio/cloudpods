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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
)

type SProxmoxHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SProxmoxHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SProxmoxHostDriver) GetHostType() string {
	return api.HOST_TYPE_PROXMOX
}

func (self *SProxmoxHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_PROXMOX
}

func (self *SProxmoxHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SProxmoxHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (driver *SProxmoxHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return -1
}

func (self *SProxmoxHostDriver) CheckAndSetCacheImage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	input := api.CacheImageInput{}
	task.GetParams().Unmarshal(&input)

	if len(input.ImageId) == 0 {
		return fmt.Errorf("no image_id params")
	}

	obj, err := models.CachedimageManager.FetchById(input.ImageId)
	if err != nil {
		return err
	}
	cacheImage := obj.(*models.SCachedimage)

	cacheInput := vcenter.ImageCacheInput{}
	cacheInput.ImageId = input.ImageId
	cacheInput.ImageName = cacheImage.Name
	cacheInput.ImageExternalId = cacheImage.ExternalId
	cacheInput.Format = input.Format
	cacheInput.IsForce = input.IsForce
	cacheInput.ImageType = cacheImage.ImageType
	cacheInput.HostId = host.ExternalId
	cacheInput.HostIp = host.AccessIp
	cacheInput.Datastore, err = host.GetCloudaccount().GetVCenterAccessInfo(storageCache.ManagerId)
	if err != nil {
		return err
	}

	if !host.IsEsxiAgentReady() {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&cacheInput), "disk")

	header := task.GetTaskRequestHeader()

	url := "/proxmox/disks/image_cache"

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	if err != nil {
		return err
	}
	return nil
}

func (self *SProxmoxHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
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

	url := fmt.Sprintf("/disks/proxmox/resize/%s", disk.Id)
	header := task.GetTaskRequestHeader()

	_, err = host.EsxiRequest(ctx, httputils.POST, url, header, body)
	return err
}
