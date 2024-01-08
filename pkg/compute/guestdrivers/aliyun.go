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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SAliyunGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAliyunGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SAliyunGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_ALIYUN
}

func (self *SAliyunGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_ALIYUN
	keys.Brand = api.CLOUD_PROVIDER_ALIYUN
	keys.Hypervisor = api.HYPERVISOR_ALIYUN
	return keys
}

func (self *SAliyunGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_CLOUD_EFFICIENCY
}

func (self *SAliyunGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 20
}

func (self *SAliyunGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_CLOUD_EFFICIENCY,
		api.STORAGE_CLOUD_SSD,
		api.STORAGE_CLOUD_ESSD_PL0,
		api.STORAGE_CLOUD_ESSD,
		api.STORAGE_CLOUD_ESSD_PL2,
		api.STORAGE_CLOUD_ESSD_PL3,
		api.STORAGE_CLOUD_ESSD_ENTRY,
		api.STORAGE_CLOUD_AUTO,
		api.STORAGE_PUBLIC_CLOUD,
		api.STORAGE_EPHEMERAL_SSD,
	}
}

func (self *SAliyunGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SAliyunGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) IsAllowSaveImageOnRunning() bool {
	return true
}

func (self *SAliyunGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_PUBLIC_CLOUD, api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_EFFICIENCY}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAliyunGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	for i, disk := range input.Disks {
		minGB := -1
		maxGB := -1
		switch disk.Backend {
		case api.STORAGE_CLOUD_EFFICIENCY, api.STORAGE_CLOUD_SSD, api.STORAGE_CLOUD_ESSD:
			minGB = 20
			maxGB = 32768
		case api.STORAGE_CLOUD_ESSD_PL2:
			minGB = 461
			maxGB = 32768
		case api.STORAGE_CLOUD_ESSD_PL3:
			minGB = 1261
			maxGB = 32768
		case api.STORAGE_PUBLIC_CLOUD:
			minGB = 5
			maxGB = 2000
		case api.STORAGE_EPHEMERAL_SSD:
			minGB = 5
			maxGB = 800
		case api.STORAGE_CLOUD_AUTO, api.STORAGE_CLOUD_ESSD_PL0:
			minGB = 40
			maxGB = 32768
		case api.STORAGE_CLOUD_ESSD_ENTRY:
			minGB = 10
			maxGB = 32768
		}
		if i == 0 && (disk.SizeMb < 20*1024 || disk.SizeMb > 500*1024) {
			return nil, httperrors.NewInputParameterError("The system disk size must be in the range of 20GB ~ 500Gb")
		}
		if disk.SizeMb < minGB*1024 || disk.SizeMb > maxGB*1024 {
			return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of %dGB ~ %dGB", disk.Backend, minGB, maxGB)
		}
	}
	if input.EipBw > 100 {
		return nil, httperrors.NewInputParameterError("%s requires that the eip bandwidth must be less than 100Mbps", self.GetHypervisor())
	}
	return input, nil
}

func (self *SAliyunGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SAliyunGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SAliyunGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
				Changeable:     false,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
				Changeable:     false,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_CLOUD_EFFICIENCY, MaxSizeGb: 32768, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_SSD, MaxSizeGb: 32768, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_PL0, MaxSizeGb: 32768, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD, MaxSizeGb: 32768, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_PL2, MaxSizeGb: 32768, MinSizeGb: 461, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_PL3, MaxSizeGb: 32768, MinSizeGb: 1261, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_ENTRY, MaxSizeGb: 32768, MinSizeGb: 10, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_AUTO, MaxSizeGb: 32768, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_PUBLIC_CLOUD, MaxSizeGb: 2000, MinSizeGb: 5, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_EPHEMERAL_SSD, MaxSizeGb: 800, MinSizeGb: 5, StepSizeGb: 1, Resizable: true},
			},
			SysDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_CLOUD_EFFICIENCY, MaxSizeGb: 500, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_SSD, MaxSizeGb: 500, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_PL0, MaxSizeGb: 32768, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD, MaxSizeGb: 500, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_PL2, MaxSizeGb: 32768, MinSizeGb: 461, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_PL3, MaxSizeGb: 32768, MinSizeGb: 1261, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_ESSD_ENTRY, MaxSizeGb: 32768, MinSizeGb: 10, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_CLOUD_AUTO, MaxSizeGb: 32768, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
			},
		},
	}
}

func (self *SAliyunGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SAliyunGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	weeks := bc.GetWeeks()
	if weeks >= 1 && weeks <= 4 {
		return true
	}
	months := bc.GetMonths()
	if (months >= 1 && months <= 10) || (months == 12) || (months == 24) || (months == 36) || (months == 48) || (months == 60) {
		return true
	}
	return false
}

func (self *SAliyunGuestDriver) IsSupportShutdownMode() bool {
	return true
}

func (self *SAliyunGuestDriver) IsSupportPublicipToEip() bool {
	return true
}

func (self *SAliyunGuestDriver) IsSupportSetAutoRenew() bool {
	return true
}

func (self *SAliyunGuestDriver) IsSupportPublicIp() bool {
	return true
}

func (self *SAliyunGuestDriver) RemoteActionAfterGuestCreated(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, iVM cloudprovider.ICloudVM, desc *cloudprovider.SManagedVMCreateConfig) {
	if desc.PublicIpBw > 0 {
		publicIp, err := iVM.AllocatePublicIpAddress()
		if err != nil {
			logclient.AddSimpleActionLog(guest, logclient.ACT_ALLOCATE, errors.Wrapf(err, "iVM.AllocatePublicIpAddress"), userCred, false)
			return
		}
		log.Infof("AllocatePublicIpAddress for instance %s %s", guest.Name, publicIp)
	}
}
