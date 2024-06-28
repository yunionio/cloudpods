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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/cloudinit"
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

type SCloudpodsGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SCloudpodsGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SCloudpodsGuestDriver) DoScheduleCPUFilter() bool { return true }

func (self *SCloudpodsGuestDriver) DoScheduleMemoryFilter() bool { return true }

func (self *SCloudpodsGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SCloudpodsGuestDriver) DoScheduleStorageFilter() bool { return true }

func (self *SCloudpodsGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_DEFAULT
}

func (self *SCloudpodsGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SCloudpodsGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_READY
}

func (self *SCloudpodsGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "CloudpodsGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return subtask.ScheduleRun(nil)
}

func (self *SCloudpodsGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCloudpodsGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCloudpodsGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCloudpodsGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCloudpodsGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SCloudpodsGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return true, nil
}

func (self *SCloudpodsGuestDriver) IsSupportFloppy(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (self *SCloudpodsGuestDriver) IsSupportMigrate() bool {
	return true
}

func (self *SCloudpodsGuestDriver) IsSupportLiveMigrate() bool {
	return true
}

func (self *SCloudpodsGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL_WITHOUT_ENCRYPT
}

func (self *SCloudpodsGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if guest.GetDiskIndex(disk.Id) <= 0 && guest.Status == api.VM_RUNNING {
		return httperrors.NewUnsupportOperationError("Cann't online resize root disk")
	}
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return httperrors.NewServerStatusError("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SCloudpodsGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_CLOUDPODS
	keys.Brand = brand
	keys.Hypervisor = api.HYPERVISOR_DEFAULT
	return keys
}

func (self *SCloudpodsGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SCloudpodsGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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
			SysDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_LOCAL, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_RBD, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NFS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_GPFS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
			},
			DataDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_LOCAL, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_RBD, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NFS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_GPFS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
			},
		},
	}
}

func (self *SCloudpodsGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SCloudpodsGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if len(input.UserData) > 0 {
		_, err := cloudinit.ParseUserData(input.UserData)
		if err != nil {
			return nil, err
		}
	}
	return input, nil
}
