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

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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

func (self *SAliyunGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseQuotaKeys = quotas.OwnerIdQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_ALIYUN
	// ignore brand
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
		api.STORAGE_CLOUD_ESSD,
		api.STORAGE_CLOUD_ESSD_PL2,
		api.STORAGE_CLOUD_ESSD_PL3,
		api.STORAGE_PUBLIC_CLOUD,
		api.STORAGE_EPHEMERAL_SSD,
	}
}

func (self *SAliyunGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
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

func (self *SAliyunGuestDriver) GetChangeConfigStatus() ([]string, error) {
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
		}
		if i == 0 {
			minGB = 20
			maxGB = 500
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
	return api.VM_READY
}

func (self *SAliyunGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SAliyunGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	userName := "root"
	if desc.OsType == "Windows" {
		userName = "Administrator"
	}
	return userName
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
