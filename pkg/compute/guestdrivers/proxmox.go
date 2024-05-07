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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/cloudinit"
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

func (self *SProxmoxGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskSize/1024%1 > 0 {
		return fmt.Errorf("Resize disk size must be an integer multiple of 1G")
	}
	return nil
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

func (self *SProxmoxGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SProxmoxGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ProxmoxGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	return subtask.ScheduleRun(nil)
}

func (self *SProxmoxGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SProxmoxGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SProxmoxGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
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

func (self *SProxmoxGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
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

func (self *SProxmoxGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SProxmoxGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return true, nil
}

func (self *SProxmoxGuestDriver) RequestRemoteUpdate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, replaceTags bool) error {
	return nil
}
