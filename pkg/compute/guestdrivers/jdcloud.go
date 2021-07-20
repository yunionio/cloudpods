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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SJDcloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SJDcloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SJDcloudGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_JDCLOUD
}

func (self *SJDcloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_JDCLOUD
}

func (self *SJDcloudGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_JDCLOUD
	keys.Brand = api.CLOUD_PROVIDER_JDCLOUD
	keys.Hypervisor = api.HYPERVISOR_JDCLOUD
	return keys
}

func (self *SJDcloudGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_JDCLOUD_GP1
}

// 系统盘：
// local：不能指定大小，默认为40GB
// cloud：取值范围: [40,500]GB，并且不能小于镜像的最小系统盘大小，如果没有指定，默认以镜像中的系统盘大小为准
func (self *SJDcloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 40
}

// 云硬盘大小，单位为 GiB；ssd.io1 类型取值范围[20,16000]GB,步长为10G;
// ssd.gp1 类型取值范围[20,16000]GB,步长为10G;
// hdd.std1 类型取值范围[20,16000]GB,步长为10G
func (self *SJDcloudGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_JDCLOUD_DEFAULT_WINDOWS_LOGIN_USER,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_JDCLOUD_STD, MaxSizeGb: 16000, MinSizeGb: 20, StepSizeGb: 10, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_JDCLOUD_GP1, MaxSizeGb: 16000, MinSizeGb: 20, StepSizeGb: 10, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_JDCLOUD_IO1, MaxSizeGb: 16000, MinSizeGb: 20, StepSizeGb: 10, Resizable: true},
			},
			SysDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_JDCLOUD_STD, MaxSizeGb: 500, MinSizeGb: 40, StepSizeGb: 10, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_JDCLOUD_GP1, MaxSizeGb: 500, MinSizeGb: 40, StepSizeGb: 10, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_JDCLOUD_IO1, MaxSizeGb: 500, MinSizeGb: 40, StepSizeGb: 10, Resizable: false},
			},
		},
	}
}

func (self *SJDcloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}
