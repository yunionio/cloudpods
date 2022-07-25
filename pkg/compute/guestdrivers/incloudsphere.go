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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SInCloudSphereGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SInCloudSphereGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SInCloudSphereGuestDriver) DoScheduleSKUFilter() bool {
	return false
}

func (self *SInCloudSphereGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_INCLOUD_SPHERE
}

func (self *SInCloudSphereGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_INCLOUD_SPHERE
}

func (self *SInCloudSphereGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SInCloudSphereGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SInCloudSphereGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_INCLOUD_SPHERE
	keys.Brand = api.CLOUD_PROVIDER_INCLOUD_SPHERE
	keys.Hypervisor = api.HYPERVISOR_INCLOUD_SPHERE
	return keys
}

func (self *SInCloudSphereGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SInCloudSphereGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SInCloudSphereGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil // do nothing, not support securitygroup
}

func (self *SInCloudSphereGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SInCloudSphereGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SInCloudSphereGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SInCloudSphereGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SInCloudSphereGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{}, cloudprovider.ErrNotSupported
}

func (self *SInCloudSphereGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SInCloudSphereGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip", self.GetHypervisor())
}

func (self *SInCloudSphereGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SInCloudSphereGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (self *SInCloudSphereGuestDriver) RequestRemoteUpdate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, replaceTags bool) error {
	return nil
}
