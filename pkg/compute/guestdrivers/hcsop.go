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
	"database/sql"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHCSOPGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SHCSOPGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SHCSOPGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_HCSOP
}

func (self *SHCSOPGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HCSOP
}

func (self *SHCSOPGuestDriver) DoScheduleSKUFilter() bool {
	return false
}

func (self *SHCSOPGuestDriver) DoScheduleStorageFilter() bool {
	return false
}

func (self *SHCSOPGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_HCSOP
	keys.Brand = api.CLOUD_PROVIDER_HCSOP
	keys.Hypervisor = api.HYPERVISOR_HCSOP
	return keys
}

func (self *SHCSOPGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_HUAWEI_SAS
}

func (self *SHCSOPGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 40
}

func (self *SHCSOPGuestDriver) GetStorageTypes() []string {
	storages, _ := models.StorageManager.GetStorageTypesByProvider(self.GetProvider())
	return storages
}

func (self *SHCSOPGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SHCSOPGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHCSOPGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHCSOPGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHCSOPGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SHCSOPGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHCSOPGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SHCSOPGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SHCSOPGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SHCSOPGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SHCSOPGuestDriver) RemoteDeployGuestSyncHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, iVM cloudprovider.ICloudVM) (cloudprovider.ICloudHost, error) {
	if hostId := iVM.GetIHostId(); len(hostId) > 0 {
		nh, err := db.FetchByExternalIdAndManagerId(models.HostManager, hostId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", host.ManagerId)
		})
		if err != nil {
			log.Debugf("failed to found new hostId(%s) for ivm %s(%s) error: %v", hostId, guest.Name, guest.Id, err)
			if errors.Cause(err) != sql.ErrNoRows {
				return nil, errors.Wrap(err, "FetchByExternalIdAndManagerId")
			}

			// HYPERVISOR_HCSOP VM被部署到一台全新的宿主机
			zone, err := host.GetZone()
			if err != nil {
				log.Warningf("host %s GetZone: %s", host.GetId(), err)
			} else {
				_host, err := models.HostManager.NewFromCloudHost(ctx, userCred, iVM.GetIHost(), host.GetCloudprovider(), zone)
				if err != nil {
					log.Warningf("NewFromCloudHost %s: %s", iVM.GetIHostId(), err)
				} else {
					host = _host
				}
			}
		} else {
			host = nh.(*models.SHost)
		}
	}

	if host.GetId() != guest.HostId {
		guest.OnScheduleToHost(ctx, userCred, host.GetId())
	}

	return host.GetIHost(ctx)
}

func (self *SHCSOPGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}

func (self *SHCSOPGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SHCSOPGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}
