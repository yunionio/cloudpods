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
	"yunion.io/x/cloudmux/pkg/multicloud/bingocloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
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
	return []string{api.VM_RUNNING, api.VM_READY}, nil
}

func (self *SBingoCloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_RUNNING, api.VM_READY}, nil
}

func (self *SBingoCloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_RUNNING, api.VM_READY}, nil
}

func (self *SBingoCloudGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_RUNNING, api.VM_READY}, nil
}

func (self *SBingoCloudGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SBingoCloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SBingoCloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SBingoCloudGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SBingoCloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SBingoCloudGuestDriver) IsSupportMigrate() bool {
	return true
}

func (self *SBingoCloudGuestDriver) IsSupportLiveMigrate() bool {
	return true
}

func (self *SBingoCloudGuestDriver) CheckLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error {
	if len(guest.BackupHostId) > 0 {
		return httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_SUSPEND}) {
		if input.MaxBandwidthMb != nil && *input.MaxBandwidthMb < 50 {
			return httperrors.NewBadRequestError("max bandwidth must gratethan 100M")
		}
		cdrom := guest.GetCdrom()
		if cdrom != nil && len(cdrom.ImageId) > 0 {
			return httperrors.NewBadRequestError("Cannot live migrate with cdrom")
		}
		devices, err := guest.GetIsolatedDevices()
		if err != nil {
			return errors.Wrapf(err, "GetIsolatedDevices")
		}
		if len(devices) > 0 {
			return httperrors.NewBadRequestError("Cannot live migrate with isolated devices")
		}
		if len(input.PreferHost) > 0 {
			err := checkAssignHost(ctx, userCred, input.PreferHost)
			if err != nil {
				return errors.Wrap(err, "checkAssignHost")
			}
		}
	}
	return nil
}

func (self *SBingoCloudGuestDriver) RequestMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "guest.GetIVM")
		}
		iHost, err := models.HostManager.FetchById(input.PreferHostId)
		if err != nil {
			return nil, errors.Wrapf(err, "models.HostManager.FetchById(%s)", input.PreferHostId)
		}
		host := iHost.(*models.SHost)
		hostExternalId := host.ExternalId
		if err = iVM.LiveMigrateVM(hostExternalId); err != nil {
			return nil, errors.Wrapf(err, "iVM.LiveMigrateVM(%s)", hostExternalId)
		}
		_, err = db.Update(guest, func() error {
			guest.HostId = host.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "db.Update guest.hostId")
		}
		return nil, nil
	})
	return nil
}

func (self *SBingoCloudGuestDriver) RequestLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iVM, err := guest.GetIVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "guest.GetIVM")
		}
		iHost, err := models.HostManager.FetchById(input.PreferHostId)
		if err != nil {
			return nil, errors.Wrapf(err, "models.HostManager.FetchById(%s)", input.PreferHostId)
		}
		host := iHost.(*models.SHost)
		hostExternalId := host.ExternalId
		if err = iVM.LiveMigrateVM(hostExternalId); err != nil {
			return nil, errors.Wrapf(err, "iVM.LiveMigrateVM(%s)", hostExternalId)
		}
		_, err = db.Update(guest, func() error {
			guest.HostId = host.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "db.Update guest.hostId")
		}
		return nil, nil
	})
	return nil
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

func (self *SBingoCloudGuestDriver) RequestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host, _ := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost(ctx)
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil {
			return nil, err
		}
		vm := ivm.(*bingocloud.SInstance)
		err = vm.SuspendVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "VM.SuspendVM for bingocloud")
		}
		return nil, nil
	})
	return nil
}

func (self *SBingoCloudGuestDriver) RequestResumeOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host, _ := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost(ctx)
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil {
			return nil, err
		}
		vm := ivm.(*bingocloud.SInstance)
		err = vm.ResumeVM(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "VM.ResumeVM for bingocloud")
		}
		return nil, nil
	})
	return nil
}

func (self *SBingoCloudGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, vmemSize int64) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		host, _ := guest.GetHost()
		if host == nil {
			return nil, errors.Error("fail to get host of guest")
		}
		ihost, err := host.GetIHost(ctx)
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil {
			return nil, err
		}
		vm := ivm.(*bingocloud.SInstance)
		err = vm.UpdateInstanceType(instanceType)
		if err != nil {
			return nil, errors.Wrap(err, "VM.UpdateInstanceType for bingocloud")
		}
		return nil, nil
	})
	return nil
}
