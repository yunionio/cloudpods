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

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SAwsGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAwsGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func fetchAwsUserName(desc cloudprovider.SManagedVMCreateConfig) string {
	// 非公有云官方镜像
	if desc.ImageType != "system" {
		return "root"
	}

	// 公有云官方镜像
	dist := strings.ToLower(desc.OsDistribution)
	if strings.Contains(dist, "centos") {
		return "centos"
	} else if strings.Contains(dist, "ubuntu") {
		return "ubuntu"
	} else if strings.Contains(dist, "windows") {
		return "Administrator"
	} else if strings.Contains(dist, "debian") {
		return "admin"
	} else if strings.Contains(dist, "suse") {
		return "ec2-user"
	} else if strings.Contains(dist, "fedora") {
		return "ec2-user"
	} else if strings.Contains(dist, "rhel") || strings.Contains(dist, "redhat") {
		return "ec2-user"
	} else if strings.Contains(dist, "amazon linux") {
		return "ec2-user"
	} else {
		return "ec2-user"
	}
}

func (self *SAwsGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	// return fetchAwsUserName(desc)
	return api.VM_AWS_DEFAULT_LOGIN_USER
}

func (self *SAwsGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_AWS
}

func (self *SAwsGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_GP2_SSD
}

func (self *SAwsGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SAwsGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_GP2_SSD,
		api.STORAGE_IO1_SSD,
		api.STORAGE_ST1_HDD,
		api.STORAGE_SC1_HDD,
		api.STORAGE_STANDARD_HDD,
	}
}

func (self *SAwsGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
}

func (self *SAwsGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SAwsGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SAwsGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	return self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
}

func (self *SAwsGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	// https://docs.amazonaws.cn/AWSEC2/latest/UserGuide/stop-start.html
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS && !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_IO1_SSD, api.STORAGE_STANDARD_HDD, api.STORAGE_GP2_SSD}) {
		return fmt.Errorf("Cannot resize system disk with unsupported volumes type %s", storage.StorageType)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_GP2_SSD, api.STORAGE_IO1_SSD, api.STORAGE_ST1_HDD, api.STORAGE_SC1_HDD, api.STORAGE_STANDARD_HDD}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAwsGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SAwsGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SAwsGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}
