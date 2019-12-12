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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SGoogleGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SGoogleGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SGoogleGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseQuotaKeys = quotas.OwnerIdQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_GOOGLE
	// ignore brand
	return keys
}

func (self *SGoogleGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_GOOGLE
}

func (self *SGoogleGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_GOOGLE_PD_STANDARD
}

func (self *SGoogleGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}
