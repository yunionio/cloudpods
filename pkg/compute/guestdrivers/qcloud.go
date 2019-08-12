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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
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
	}
}

func (self *SQcloudGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
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

func (self *SQcloudGuestDriver) GetChangeConfigStatus() ([]string, error) {
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
	if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_LOCAL_BASIC, api.STORAGE_LOCAL_SSD}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SQcloudGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
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
	switch sysDisk.Backend {
	case api.STORAGE_CLOUD_BASIC, api.STORAGE_CLOUD_SSD:
		if sysDisk.SizeMb > 500*1024 {
			return nil, fmt.Errorf("The %s system disk size must be less than 500GB", sysDisk.Backend)
		}
	case api.STORAGE_CLOUD_PREMIUM:
		if sysDisk.SizeMb > 1024*1024 {
			return nil, fmt.Errorf("The %s system disk size must be less than 1024GB", sysDisk.Backend)
		}
	}

	for i := 1; i < len(input.Disks); i++ {
		disk := input.Disks[i]
		switch disk.Backend {
		case api.STORAGE_CLOUD_BASIC:
			if disk.SizeMb < 10*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 10GB ~ 16000GB", disk.Backend)
			}
		case api.STORAGE_CLOUD_PREMIUM:
			if disk.SizeMb < 50*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 50GB ~ 16000GB", disk.Backend)
			}
		case api.STORAGE_CLOUD_SSD:
			if disk.SizeMb < 100*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 100GB ~ 16000GB", disk.Backend)
			}
		}
	}
	return input, nil
}

func (self *SQcloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SQcloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SQcloudGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SQcloudGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	userName := "root"
	if desc.ImageType == "system" {
		if desc.OsDistribution == "Ubuntu" {
			userName = "ubuntu"
		}
		if desc.OsType == "Windows" {
			userName = "Administrator"
		}
	}
	return userName
}

func (self *SQcloudGuestDriver) GetGuestSecgroupVpcid(guest *models.SGuest) (string, error) {
	return api.NORMAL_VPC_ID, nil
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
