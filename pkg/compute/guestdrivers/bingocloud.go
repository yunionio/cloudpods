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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBingoCloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SBingoCloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SBingoCloudGuestDriver) DoScheduleCPUFilter() bool { return false }

func (self *SBingoCloudGuestDriver) DoScheduleMemoryFilter() bool { return false }

func (self *SBingoCloudGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SBingoCloudGuestDriver) DoScheduleStorageFilter() bool { return false }

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

func (self *SBingoCloudGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_BINGO_CLOUD
	keys.Brand = api.CLOUD_PROVIDER_BINGO_CLOUD
	keys.Hypervisor = api.HYPERVISOR_BINGO_CLOUD
	return keys
}

func (self *SBingoCloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBingoCloudGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SBingoCloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SBingoCloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SBingoCloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SBingoCloudGuestDriver) RemoteDeployGuestSyncHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, iVM cloudprovider.ICloudVM) (cloudprovider.ICloudHost, error) {
	if hostId := iVM.GetIHostId(); len(hostId) > 0 {
		nh, err := db.FetchByExternalIdAndManagerId(models.HostManager, hostId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", host.ManagerId)
		})
		if err != nil {
			log.Warningf("failed to found new hostId(%s) for ivm %s(%s) error: %v", hostId, guest.Name, guest.Id, err)
		} else if nh.GetId() != guest.HostId {
			guest.OnScheduleToHost(ctx, userCred, nh.GetId())
			host = nh.(*models.SHost)
		}
	}

	return host.GetIHost(ctx)
}
