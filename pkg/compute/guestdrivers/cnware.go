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

package guestdrivers

import (
	"context"
	"fmt"
	"sort"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCNwareGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SCNwareGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SCNwareGuestDriver) DoScheduleCPUFilter() bool { return true }

func (self *SCNwareGuestDriver) DoScheduleMemoryFilter() bool { return true }

func (self *SCNwareGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SCNwareGuestDriver) DoScheduleStorageFilter() bool { return true }

func (self *SCNwareGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_CNWARE
}

func (self *SCNwareGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CNWARE
}

func (self *SCNwareGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_CNWARE
	keys.Brand = brand
	keys.Hypervisor = api.HYPERVISOR_CNWARE
	return keys
}

func (self *SCNwareGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_CNWARE_LOCAL
}

func (self *SCNwareGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SCNwareGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_CNWARE_FCSAN,
		api.STORAGE_CNWARE_IPSAN,
		api.STORAGE_CNWARE_NAS,
		api.STORAGE_CNWARE_CEPH,
		api.STORAGE_CNWARE_LOCAL,
		api.STORAGE_CNWARE_NVME,
	}
}

func (self *SCNwareGuestDriver) GetMaxSecurityGroupCount() int {
	return 0
}

func (self *SCNwareGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	switch {
	case !options.Options.LockStorageFromCachedimage:
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	case len(diskConfig.ImageId) > 0:
		var (
			image *cloudprovider.SImage
			err   error
		)
		obj, err := models.CachedimageManager.FetchById(diskConfig.ImageId)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch cachedimage %s", diskConfig.ImageId)
		}
		cachedimage := obj.(*models.SCachedimage)
		if len(cachedimage.ExternalId) > 0 || cloudprovider.TImageType(cachedimage.ImageType) != cloudprovider.ImageTypeSystem {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		storages, err := cachedimage.GetStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to GetStorages of cachedimage %s", diskConfig.ImageId)
		}
		if len(storages) == 0 {
			log.Warningf("there no storage associated with cachedimage %q", image.Id)
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		if len(storages) > 1 {
			log.Warningf("there are multiple storageCache associated with caheimage %q", image.Id)
		}
		wantStorageIds := make([]string, len(storages))
		for i := range wantStorageIds {
			wantStorageIds[i] = storages[i].GetId()
		}
		for i := range wantStorageIds {
			if utils.IsInStringArray(wantStorageIds[i], storageIds) {
				log.Infof("use storage %q in where cachedimage %q", wantStorageIds[i], image.Id)
				return &storages[i], nil
			}
		}
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	default:
		ispId := guest.GetMetadata(context.Background(), "__base_instance_snapshot_id", nil)
		if len(ispId) == 0 {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		obj, err := models.InstanceSnapshotManager.FetchById(ispId)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch InstanceSnapshot %q", ispId)
		}
		isp := obj.(*models.SInstanceSnapshot)
		ispGuest, err := isp.GetGuest()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fetch Guest of InstanceSnapshot %q", ispId)
		}
		storages, err := ispGuest.GetStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "GetStorages")
		}
		if len(storages) == 0 {
			return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
		}
		if utils.IsInStringArray(storages[0].GetId(), storageIds) {
			return &storages[0], nil
		}
		return self.SVirtualizedGuestDriver.ChooseHostStorage(host, guest, diskConfig, storageIds)
	}
}

func (self *SCNwareGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCNwareGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCNwareGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SCNwareGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SCNwareGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SCNwareGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SCNwareGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SCNwareGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip, it only support bind eip", self.GetHypervisor())
}

func (self *SCNwareGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if data.CpuSockets > data.VcpuCount {
		return nil, httperrors.NewInputParameterError("The number of cpu sockets cannot be greater than the number of cpus")
	}

	// check disk config
	if len(data.Disks) == 0 {
		return data, nil
	}
	rootDisk := data.Disks[0]
	if len(rootDisk.ImageId) == 0 {
		return data, nil
	}
	image, err := models.CachedimageManager.GetImageInfo(ctx, userCred, rootDisk.ImageId, false)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to GetImageInfo of image %q", rootDisk.ImageId)
	}
	if len(image.SubImages) <= 1 {
		return data, nil
	}
	sort.Slice(image.SubImages, func(i, j int) bool {
		return image.SubImages[i].Index < image.SubImages[j].Index
	})
	newDataDisks := make([]*api.DiskConfig, 0, len(image.SubImages)+len(data.Disks)-1)
	for i, subImage := range image.SubImages {
		nDataDisk := *rootDisk
		nDataDisk.SizeMb = subImage.MinDiskMB
		nDataDisk.Format = "vmdk"
		nDataDisk.Index = i
		if i > 0 {
			nDataDisk.ImageId = ""
		}
		newDataDisks = append(newDataDisks, &nDataDisk)
	}
	for i := 1; i < len(data.Disks); i++ {
		data.Disks[i].Index += len(image.SubImages) - 1
		newDataDisks = append(newDataDisks, data.Disks[i])
	}
	data.Disks = newDataDisks
	return data, nil
}

func (self *SCNwareGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SCNwareGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SCNwareGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SCNwareGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SCNwareGuestDriver) IsWindowsUserDataTypeNeedEncode() bool {
	return true
}

func (self *SCNwareGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
			},
		},
	}
}

func (self *SCNwareGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SCNwareGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SCNwareGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil // do nothing, not support securitygroup
}

func (self *SCNwareGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return nil
}
func (self *SCNwareGuestDriver) GetChangeInstanceTypeStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SCNwareGuestDriver) ValidateGuestChangeConfigInput(ctx context.Context, guest *models.SGuest, input api.ServerChangeConfigInput) (*api.ServerChangeConfigSettings, error) {
	confs, err := self.SBaseGuestDriver.ValidateGuestChangeConfigInput(ctx, guest, input)
	if err != nil {
		return nil, errors.Wrap(err, "SBaseGuestDriver.ValidateGuestChangeConfigInput")
	}
	sku, err := models.ServerSkuManager.FetchSkuByNameAndProvider(input.Sku, "OneCloud", true)
	if err != nil {
		return nil, errors.Wrap(err, "FetchSkuByNameAndProvider")
	}

	confs.InstanceTypeFamily = ""
	confs.InstanceType = ""
	confs.VcpuCount = sku.CpuCoreCount
	confs.VmemSize = sku.MemorySizeMB
	confs.CpuSockets = 1
	log.Infof("ValidateGuestChangeConfigInput: %+v", confs)

	if input.CpuSockets != nil && *input.CpuSockets > 0 {
		confs.CpuSockets = *input.CpuSockets
	}

	defaultStorageId := ""
	if root, _ := guest.GetSystemDisk(); root != nil {
		defaultStorageId = root.StorageId
	}
	storages, err := guest.GetStorages()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorages")
	}
	storageMap := map[string]string{}
	for _, storage := range storages {
		storageMap[storage.StorageType] = storage.Id
		if len(defaultStorageId) == 0 {
			defaultStorageId = storage.Id
		}
	}
	for i := range confs.Create {
		confs.Create[i].Format = "vmdk"
		if len(confs.Create[i].Storage) == 0 {
			// 若不指定存储类型，默认和系统盘一致
			if len(confs.Create[i].Backend) == 0 {
				confs.Create[i].Storage = defaultStorageId
			} else if storageId, ok := storageMap[confs.Create[i].Backend]; ok { // 否则和已有磁盘存储保持一致
				confs.Create[i].Storage = storageId
			}
		}
	}
	return confs, nil
}
