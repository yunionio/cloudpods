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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/billing"
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

type SNutanixGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SNutanixGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SNutanixGuestDriver) DoScheduleCPUFilter() bool { return true }

func (self *SNutanixGuestDriver) DoScheduleMemoryFilter() bool { return true }

func (self *SNutanixGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SNutanixGuestDriver) DoScheduleStorageFilter() bool { return true }

func (self *SNutanixGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_NUTANIX
}

func (self *SNutanixGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_NUTANIX
}

func (self *SNutanixGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SNutanixGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_NUTANIX
	keys.Brand = api.CLOUD_PROVIDER_NUTANIX
	keys.Hypervisor = api.HYPERVISOR_NUTANIX
	return keys
}

func (self *SNutanixGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SNutanixGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SNutanixGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SNutanixGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SNutanixGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SNutanixGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil // do nothing, not support securitygroup
}

func (self *SNutanixGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SNutanixGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "NutanixGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SNutanixGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SNutanixGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SNutanixGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SNutanixGuestDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SNutanixGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{}, cloudprovider.ErrNotSupported
}

func (self *SNutanixGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SNutanixGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskSize/1024%1 > 0 {
		return fmt.Errorf("Resize disk size must be an integer multiple of 1G")
	}
	return nil
}

func (self *SNutanixGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip", self.GetHypervisor())
}

func (self *SNutanixGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SNutanixGuestDriver) RequestRenewInstance(ctx context.Context, guest *models.SGuest, bc billing.SBillingCycle) (time.Time, error) {
	return time.Time{}, nil
}

func (self *SNutanixGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SNutanixGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (self *SNutanixGuestDriver) RequestRemoteUpdate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, replaceTags bool) error {
	return nil
}
