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
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SBingoCloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SBingoCloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SBingoCloudGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_BINGO_CLOUD
}

func (self *SBingoCloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_BINGO_CLOUD
}

func (self *SBingoCloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SBingoCloudGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
				Changeable:     true,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
				Changeable:     false,
			},
		},
	}
}

func (self *SBingoCloudGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_BINGO_CLOUD
	keys.Brand = api.CLOUD_PROVIDER_BINGO_CLOUD
	keys.Hypervisor = api.HYPERVISOR_BINGO_CLOUD
	return keys
}

func (self *SBingoCloudGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SBingoCloudGuestDriver) DoScheduleCPUFilter() bool { return false }

func (self *SBingoCloudGuestDriver) DoScheduleMemoryFilter() bool { return false }

func (self *SBingoCloudGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SBingoCloudGuestDriver) DoScheduleStorageFilter() bool { return false }

func (self *SBingoCloudGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_BINGOCLOUD_CEPH,
		api.STORAGE_BINGOCLOUD_LOCAL_FS,
		api.STORAGE_BINGOCLOUD_LOCAL_VG,
	}
}

func (self *SBingoCloudGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return self.chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SBingoCloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBingoCloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBingoCloudGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SBingoCloudGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBingoCloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SBingoCloudGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SBingoCloudGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SBingoCloudGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip, it only support bind eip", self.GetHypervisor())
}

func (self *SBingoCloudGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 1 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	if len(input.Eip) > 0 || input.EipBw > 0 {
		return nil, httperrors.NewUnsupportOperationError("%s not support create virtual machine with eip", self.GetHypervisor())
	}
	return input, nil
}

func (self *SBingoCloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SBingoCloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SBingoCloudGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SBingoCloudGuestDriver) IsWindowsUserDataTypeNeedEncode() bool {
	return true
}

func (self *SBingoCloudGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SBingoCloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SBingoCloudGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SBingoCloudGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}
