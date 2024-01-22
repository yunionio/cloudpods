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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SUCloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func (self *SUCloudGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_UCLOUD
}

func (self *SUCloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_UCLOUD
}

func (self *SUCloudGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_UCLOUD
	keys.Brand = api.CLOUD_PROVIDER_UCLOUD
	keys.Hypervisor = api.HYPERVISOR_UCLOUD
	return keys
}

func (self *SUCloudGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_UCLOUD_CLOUD_SSD
}

func (self *SUCloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SUCloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SUCloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SUCloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SUCloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SUCloudGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SUCloudGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SUCloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SUCloudGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_UCLOUD_CLOUD_SSD, api.STORAGE_UCLOUD_CLOUD_NORMAL}) {
		return fmt.Errorf("Cannot resize disk with unsupported volumes type %s", storage.StorageType)
	}

	return nil
}

func (self *SUCloudGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
			},
		},
	}
}

func init() {
	driver := SUCloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (ucloud *SUCloudGuestDriver) ValidateGuestChangeConfigInput(ctx context.Context, guest *models.SGuest, input api.ServerChangeConfigInput) (*api.ServerChangeConfigSettings, error) {
	confs, err := ucloud.SBaseGuestDriver.ValidateGuestChangeConfigInput(ctx, guest, input)
	if err != nil {
		return nil, errors.Wrap(err, "SBaseGuestDriver.ValidateGuestChangeConfigInput")
	}

	if len(input.InstanceType) > 0 {
		if !strings.HasPrefix(guest.InstanceType, confs.InstanceTypeFamily) {
			return nil, httperrors.NewInputParameterError("Cannot change config with different instance family")
		}
	}

	return confs, nil
}
