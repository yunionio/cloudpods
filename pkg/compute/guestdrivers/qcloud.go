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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SQcloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SQcloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SQcloudGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_QCLOUD
}

func (self *SQcloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_QCLOUD
	keys.Brand = api.CLOUD_PROVIDER_QCLOUD
	keys.Hypervisor = api.HYPERVISOR_QCLOUD
	return keys
}

func (self *SQcloudGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_CLOUD_PREMIUM
}

func (self *SQcloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 50
}

func (self *SQcloudGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_CLOUD_BASIC,
		api.STORAGE_CLOUD_PREMIUM,
		api.STORAGE_CLOUD_SSD,
		api.STORAGE_LOCAL_BASIC,
		api.STORAGE_LOCAL_SSD,
		api.STORAGE_CLOUD_HSSD,
	}
}

func (self *SQcloudGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SQcloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	//https://cloud.tencent.com/document/product/362/5747
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_LOCAL_BASIC, api.STORAGE_LOCAL_SSD, api.STORAGE_LOCAL_PRO}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	if disk.DiskSize/1024%10 > 0 {
		return fmt.Errorf("Resize disk size must be an integer multiple of 10G")
	}
	return nil
}

func (self *SQcloudGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}

	sysDisk := input.Disks[0]
	if sysDisk.SizeMb < 50*1024 {
		return nil, fmt.Errorf("The system disk size must be more than 50GB")
	}

	switch sysDisk.Backend {
	case api.STORAGE_CLOUD_BASIC, api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_PREMIUM:
		if sysDisk.SizeMb > 1024*1024 {
			return nil, fmt.Errorf("The %s system disk size must be less than 1024GB", sysDisk.Backend)
		}
	case api.STORAGE_LOCAL_PRO, api.STORAGE_CLOUD_HSSD: //https://cloud.tencent.com/document/product/362/2353
		return nil, fmt.Errorf("storage %s can not be system disk", sysDisk.Backend)
	}

	for i := 1; i < len(input.Disks); i++ {
		disk := input.Disks[i]
		switch disk.Backend {
		case api.STORAGE_CLOUD_BASIC:
			if disk.SizeMb < 10*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 10GB ~ 16000GB", disk.Backend)
			}
		case api.STORAGE_CLOUD_PREMIUM:
			if disk.SizeMb < 10*1024 || disk.SizeMb > 32000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 10GB ~ 32000GB", disk.Backend)
			}
		case api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_HSSD:
			if disk.SizeMb < 20*1024 || disk.SizeMb > 32000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 20GB ~ 32000GB", disk.Backend)
			}
		case api.STORAGE_LOCAL_PRO:
			return nil, httperrors.NewInputParameterError("storage %s can not be data disk", disk.Backend)
		}
		if disk.SizeMb/1024%10 > 0 {
			return nil, httperrors.NewInputParameterError("Data disk size must be an integer multiple of 10G")
		}
	}
	return input, nil
}

func (qcloud *SQcloudGuestDriver) ValidateGuestChangeConfigInput(ctx context.Context, guest *models.SGuest, input api.ServerChangeConfigInput) (*api.ServerChangeConfigSettings, error) {
	confs, err := qcloud.SBaseGuestDriver.ValidateGuestChangeConfigInput(ctx, guest, input)
	if err != nil {
		return nil, errors.Wrap(err, "SBaseGuestDriver.ValidateGuestChangeConfigInput")
	}

	if confs.CpuChanged() || confs.MemChanged() {
		disk, err := guest.GetSystemDisk()
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("failed to found system disk error: %v", err)
		}
		storage, _ := disk.GetStorage()
		if storage == nil {
			return nil, httperrors.NewResourceNotFoundError("failed to found storage for disk %s(%s)", disk.Name, disk.Id)
		}
		// 腾讯云系统盘为本地存储，不支持调整配置
		if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_LOCAL_BASIC, api.STORAGE_LOCAL_SSD, api.STORAGE_LOCAL_PRO}) {
			return nil, httperrors.NewUnsupportOperationError("The system disk is locally stored and does not support changing configuration")
		}
	}

	for _, newDisk := range confs.Create {
		switch newDisk.Backend {
		case api.STORAGE_CLOUD_BASIC:
			if newDisk.SizeMb < 10*1024 || newDisk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 10GB ~ 16000GB", newDisk.Backend)
			}
		case api.STORAGE_CLOUD_PREMIUM:
			if newDisk.SizeMb < 10*1024 || newDisk.SizeMb > 32000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 10GB ~ 32000GB", newDisk.Backend)
			}
		case api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_HSSD:
			if newDisk.SizeMb < 20*1024 || newDisk.SizeMb > 32000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 20GB ~ 32000GB", newDisk.Backend)
			}
		case api.STORAGE_LOCAL_BASIC, api.STORAGE_LOCAL_SSD, api.STORAGE_LOCAL_PRO:
			return nil, httperrors.NewUnsupportOperationError("Not support create local storage disks")
		case "": //这里Backend为空有可能会导致创建出来还是local storage,依然会出错,需要用户显式指定
			return nil, httperrors.NewInputParameterError("Please input new disk backend type")
		}
		if newDisk.SizeMb/1024%10 > 0 {
			return nil, httperrors.NewInputParameterError("Data disk size must be an integer multiple of 10G")
		}
	}
	return confs, nil
}

func (self *SQcloudGuestDriver) ValidateDetachDisk(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, disk *models.SDisk) error {
	storage, _ := disk.GetStorage()
	if storage == nil {
		return httperrors.NewResourceNotFoundError("failed to found storage for disk %s(%s)", disk.Name, disk.Id)
	}
	// 腾讯云本地盘不支持卸载
	if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_LOCAL_BASIC, api.STORAGE_LOCAL_SSD}) {
		return httperrors.NewUnsupportOperationError("The disk is locally stored and does not support detach")
	}
	return nil
}

func (self *SQcloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SQcloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SQcloudGuestDriver) IsSupportdDcryptPasswordFromSecretKey() bool {
	return false
}

func (self *SQcloudGuestDriver) IsSupportShutdownMode() bool {
	return true
}

func (self *SQcloudGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SQcloudGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
				Changeable:     false,
				DistAccounts: []cloudprovider.SDistDefaultAccount{
					{
						OsDistribution: "Ubuntu",
						DefaultAccount: "ubuntu",
						Changeable:     false,
					},
				},
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
				Changeable:     false,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_BASIC, MaxSizeGb: 16000, MinSizeGb: 10, StepSizeGb: 10, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_PREMIUM, MaxSizeGb: 16000, MinSizeGb: 50, StepSizeGb: 10, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_SSD, MaxSizeGb: 16000, MinSizeGb: 100, StepSizeGb: 10, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_HSSD, MaxSizeGb: 32000, MinSizeGb: 20, StepSizeGb: 10, Resizable: true},
			},
			SysDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_BASIC, MaxSizeGb: 500, MinSizeGb: 50, StepSizeGb: 10, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_PREMIUM, MaxSizeGb: 1024, MinSizeGb: 50, StepSizeGb: 10, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_CLOUD_SSD, MaxSizeGb: 500, MinSizeGb: 50, StepSizeGb: 10, Resizable: false},
			},
		},
	}
}

func (self *SQcloudGuestDriver) GetDefaultAccount(osType, osDist, imageType string) string {
	userName := api.VM_DEFAULT_LINUX_LOGIN_USER
	if imageType == "system" {
		if osDist == "Ubuntu" {
			userName = "ubuntu"
		}
	}
	if strings.ToLower(osType) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		userName = api.VM_DEFAULT_WINDOWS_LOGIN_USER
	}

	return userName
}

func (self *SQcloudGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SQcloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 12) || (months == 24) || (months == 36) {
		return true
	}
	return false
}

func (self *SQcloudGuestDriver) IsSupportPublicIp() bool {
	return true
}

func (self *SQcloudGuestDriver) IsSupportPublicipToEip() bool {
	return true
}

func (self *SQcloudGuestDriver) IsSupportSetAutoRenew() bool {
	return true
}
