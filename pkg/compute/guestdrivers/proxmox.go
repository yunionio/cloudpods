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
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SProxmoxGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SProxmoxGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SProxmoxGuestDriver) DoScheduleSKUFilter() bool {
	return false
}

func (self *SProxmoxGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_PROXMOX
}

func (self *SProxmoxGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_PROXMOX
}

func (self *SProxmoxGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SProxmoxGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SProxmoxGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_PROXMOX
	keys.Brand = api.CLOUD_PROVIDER_PROXMOX
	keys.Hypervisor = api.HYPERVISOR_PROXMOX
	return keys
}

func (self *SProxmoxGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_READY
}

func (self *SProxmoxGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SProxmoxGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SProxmoxGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil // do nothing, not support securitygroup
}

func (self *SProxmoxGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SProxmoxGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SProxmoxGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SProxmoxGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SProxmoxGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{}, cloudprovider.ErrNotSupported
}

func (self *SProxmoxGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SProxmoxGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip", self.GetHypervisor())
}

func (self *SProxmoxGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SProxmoxGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (self *SProxmoxGuestDriver) RequestRemoteUpdate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, replaceTags bool) error {
	return nil
}
