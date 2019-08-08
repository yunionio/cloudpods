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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SZStackGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SZStackGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SZStackGuestDriver) DoScheduleCPUFilter() bool { return true }

func (self *SZStackGuestDriver) DoScheduleMemoryFilter() bool { return true }

func (self *SZStackGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SZStackGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_ZSTACK
}

func (self *SZStackGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ZSTACK
}

func (self *SZStackGuestDriver) GetQuotaPlatformID() []string {
	return []string{
		api.CLOUD_ENV_PRIVATE_CLOUD,
		api.CLOUD_PROVIDER_ZSTACK,
	}
}

func (self *SZStackGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_ZSTACK_LOCAL_STORAGE
}

func (self *SZStackGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SZStackGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_ZSTACK_LOCAL_STORAGE,
		api.STORAGE_ZSTACK_CEPH,
	}
}

func (self *SZStackGuestDriver) GetMaxSecurityGroupCount() int {
	return 1
}

func (self *SZStackGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
}

func (self *SZStackGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SZStackGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SZStackGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SZStackGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SZStackGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SZStackGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SZStackGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SZStackGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	return httperrors.NewInputParameterError("%s not support create eip, it only support bind eip", self.GetHypervisor())
}

func (self *SZStackGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	if len(input.Eip) > 0 || input.EipBw > 0 {
		return nil, httperrors.NewUnsupportOperationError("%s not support create virtual machine with eip", self.GetHypervisor())
	}
	return input, nil
}

func (self *SZStackGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SZStackGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SZStackGuestDriver) IsNeedInjectPasswordByCloudInit(desc *cloudprovider.SManagedVMCreateConfig) bool {
	return true
}

func (self *SZStackGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SZStackGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	userName := "root"
	if desc.OsType == "Windows" {
		userName = "Administrator"
	}
	return userName
}

func (self *SZStackGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SZStackGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}
