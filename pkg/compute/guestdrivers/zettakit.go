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
	"yunion.io/x/pkg/util/cloudinit"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SZettaKitGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SZettaKitGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SZettaKitGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_ZETTAKIT
}

func (self *SZettaKitGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ZETTAKIT
}

func (self *SZettaKitGuestDriver) DoScheduleSKUFilter() bool {
	return false
}

func (self *SZettaKitGuestDriver) DoScheduleStorageFilter() bool {
	return false
}

func (self *SZettaKitGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_ZETTAKIT
	keys.Brand = api.CLOUD_PROVIDER_ZETTAKIT
	keys.Hypervisor = api.HYPERVISOR_ZETTAKIT
	return keys
}

func (self *SZettaKitGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_ZETTAKIT_NORMAL
}

func (self *SZettaKitGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 20
}

func (self *SZettaKitGuestDriver) GetStorageTypes() []string {
	storages, _ := models.StorageManager.GetStorageTypesByProvider(self.GetProvider())
	return storages
}

func (self *SZettaKitGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SZettaKitGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SZettaKitGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SZettaKitGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SZettaKitGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SZettaKitGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SZettaKitGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SZettaKitGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if len(input.UserData) > 0 {
		_, err := cloudinit.ParseUserData(input.UserData)
		if err != nil {
			return nil, err
		}
	}
	if len(input.Cdrom) > 0 {
		image, err := models.CachedimageManager.GetCachedimageById(ctx, userCred, input.Cdrom, false)
		if err != nil {
			return nil, err
		}
		if len(image.ExternalId) > 0 {
			hosts, err := image.GetHosts()
			if err != nil {
				return nil, err
			}
			if len(input.PreferHost) == 0 && len(hosts) == 1 {
				input.PreferHost = hosts[0].Id
			}
		}
	}
	return input, nil
}

func (self *SZettaKitGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SZettaKitGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SZettaKitGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SZettaKitGuestDriver) RemoteDeployGuestSyncHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, iVM cloudprovider.ICloudVM) (cloudprovider.ICloudHost, error) {
	if hostId := iVM.GetIHostId(); len(hostId) > 0 {
		nh, err := db.FetchByExternalIdAndManagerId(models.HostManager, hostId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", host.ManagerId)
		})
		if err != nil {
			log.Debugf("failed to found new hostId(%s) for ivm %s(%s) error: %v", hostId, guest.Name, guest.Id, err)
			if errors.Cause(err) != sql.ErrNoRows {
				return nil, errors.Wrap(err, "FetchByExternalIdAndManagerId")
			}

			// HYPERVISOR_ZETTAKIT VM被部署到一台全新的宿主机
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

func (self *SZettaKitGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SZettaKitGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return false
}

func (self *SZettaKitGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}

func (self *SZettaKitGuestDriver) RequestSyncSecgroupsOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil // do nothing, not support securitygroup
}

func (self *SZettaKitGuestDriver) GetMaxSecurityGroupCount() int {
	return 1
}

func (self *SZettaKitGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SZettaKitGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return true, nil
}
