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
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SAwsGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SAwsGuestDriver) IsWindowsUserDataTypeNeedEncode() bool {
	return true
}

func (self *SAwsGuestDriver) GetWindowsUserDataType() string {
	return cloudprovider.CLOUD_EC2
}

func (self *SAwsGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_AWS_DEFAULT_LOGIN_USER,
				Changeable:     false,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
				Changeable:     false,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GP2_SSD, MaxSizeGb: 16384, MinSizeGb: 1, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GP3_SSD, MaxSizeGb: 16384, MinSizeGb: 1, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_IO1_SSD, MaxSizeGb: 16384, MinSizeGb: 4, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_ST1_HDD, MaxSizeGb: 16384, MinSizeGb: 500, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_SC1_HDD, MaxSizeGb: 16384, MinSizeGb: 500, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_STANDARD_HDD, MaxSizeGb: 1024, MinSizeGb: 1, StepSizeGb: 1, Resizable: true},
			},
			SysDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_GP2_SSD, MaxSizeGb: 16384, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_IO1_SSD, MaxSizeGb: 16384, MinSizeGb: 4, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_ST1_HDD, MaxSizeGb: 16384, MinSizeGb: 500, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_SC1_HDD, MaxSizeGb: 16384, MinSizeGb: 500, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_STANDARD_HDD, MaxSizeGb: 1024, MinSizeGb: 1, StepSizeGb: 1, Resizable: false},
			},
		},
	}
}

func (self *SAwsGuestDriver) GetDefaultAccount(osType, osDist, imageType string) string {
	if strings.ToLower(osType) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		return api.VM_DEFAULT_WINDOWS_LOGIN_USER
	}
	return api.VM_AWS_DEFAULT_LOGIN_USER
}

func (self *SAwsGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_AWS
}

func (self *SAwsGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_AWS
}

func (self *SAwsGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_AWS
	keys.Brand = api.CLOUD_PROVIDER_AWS
	keys.Hypervisor = api.HYPERVISOR_AWS
	return keys
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
		api.STORAGE_GP3_SSD,
		api.STORAGE_IO1_SSD,
		api.STORAGE_IO2_SSD,
		api.STORAGE_ST1_HDD,
		api.STORAGE_SC1_HDD,
		api.STORAGE_STANDARD_HDD,
	}
}

func (self *SAwsGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
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

func (self *SAwsGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SAwsGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SAwsGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if len(input.Eip) > 0 || input.EipBw > 0 {
		// 未明确指定network时，由调度器进行调度，跳过support_eip检查
		if len(input.Networks) > 0 && len(input.Networks[0].Network) > 0 {
			inetwork, err := db.FetchByIdOrName(ctx, models.NetworkManager, userCred, input.Networks[0].Network)
			if err != nil {
				return nil, errors.Wrap(err, "SAwsGuestDriver.ValidateCreateData.Networks.FetchByIdOrName")
			}

			support_eip := inetwork.(*models.SNetwork).GetMetadataJson(ctx, "ext:support_eip", nil)
			if support_eip != nil {
				if ok, _ := support_eip.Bool(); !ok {
					return nil, httperrors.NewInputParameterError("network %s associated route table has no internet gateway attached.", inetwork.GetName())
				}
			}
		}
	}

	return self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
}

func (self *SAwsGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	// https://docs.amazonaws.cn/AWSEC2/latest/UserGuide/stop-start.html
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS && !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_IO1_SSD, api.STORAGE_IO2_SSD, api.STORAGE_STANDARD_HDD, api.STORAGE_GP2_SSD, api.STORAGE_GP3_SSD}) {
		return fmt.Errorf("Cannot resize system disk with unsupported volumes type %s", storage.StorageType)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_GP2_SSD, api.STORAGE_GP3_SSD, api.STORAGE_IO1_SSD, api.STORAGE_IO2_SSD, api.STORAGE_ST1_HDD, api.STORAGE_SC1_HDD, api.STORAGE_STANDARD_HDD}) {
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
