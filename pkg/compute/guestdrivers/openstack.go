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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SOpenStackGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SOpenStackGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SOpenStackGuestDriver) DoScheduleCPUFilter() bool { return true }

func (self *SOpenStackGuestDriver) DoScheduleMemoryFilter() bool { return true }

func (self *SOpenStackGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SOpenStackGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_OPENSTACK
}

func (self *SOpenStackGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackGuestDriver) GetQuotaPlatformID() []string {
	return []string{
		api.CLOUD_ENV_PRIVATE_CLOUD,
		api.CLOUD_PROVIDER_OPENSTACK,
	}
}

func (self *SOpenStackGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SOpenStackGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_OPENSTACK_ISCSI
}

func (self *SOpenStackGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SOpenStackGuestDriver) GetStorageTypes() []string {
	storages, _ := models.StorageManager.GetStorageTypesByHostType(api.HYPERVISOR_HOSTTYPE[self.GetHypervisor()])
	return storages
}

func (self *SOpenStackGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
}

func (self *SOpenStackGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING, api.VM_REBUILD_ROOT_FAIL}, nil
}

func (self *SOpenStackGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) IsNeedInjectPasswordByCloudInit(desc *cloudprovider.SManagedVMCreateConfig) bool {
	return true
}

func (self *SOpenStackGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SOpenStackGuestDriver) IsRebuildRootSupportChangeImage() bool {
	return false
}

func (self *SOpenStackGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	return httperrors.NewInputParameterError("%s not support create eip, it only support bind eip", self.GetHypervisor())
}

func (self *SOpenStackGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SOpenStackGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	var err error
	input, err = self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) >= 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	if len(input.Eip) > 0 || input.EipBw > 0 {
		return nil, httperrors.NewUnsupportOperationError("%s not support create virtual machine with eip", self.GetHypervisor())
	}
	return input, nil
}

func (self *SOpenStackGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SOpenStackGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SOpenStackGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SOpenStackGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SOpenStackGuestDriver) CancelExpireTime(
	ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest) error {
	return guest.CancelExpireTime(ctx, userCred)
}
