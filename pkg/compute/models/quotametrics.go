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

package models

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func (account *SCloudaccount) GetQuotaPlatformID() []string {
	return []string{
		account.getCloudEnv(),
		account.Provider,
	}
}

func (provider *SCloudprovider) GetQuotaPlatformID() []string {
	return provider.GetCloudaccount().GetQuotaPlatformID()
}

func (base *SManagedResourceBase) GetQuotaPlatformID() []string {
	account := base.GetCloudaccount()
	if account != nil {
		return account.GetQuotaPlatformID()
	} else {
		return []string{
			api.CLOUD_ENV_ON_PREMISE, api.CLOUD_PROVIDER_ONECLOUD,
		}
	}
}

func (host *SHost) GetQuotaPlatformID() []string {
	ids := host.SManagedResourceBase.GetQuotaPlatformID()
	switch host.HostType {
	case api.HOST_TYPE_HYPERVISOR:
		ids = append(ids, api.HYPERVISOR_KVM)
	case api.HOST_TYPE_BAREMETAL:
		ids = append(ids, api.HYPERVISOR_BAREMETAL)
	case api.HOST_TYPE_KUBELET:
		ids = append(ids, api.HYPERVISOR_CONTAINER)
	}
	return ids
}

func (storage *SStorage) getQuotaPlatformID() []string {
	hosts := storage.GetAttachedHosts()
	if len(hosts) > 0 {
		return hosts[0].GetQuotaPlatformID()
	}
	return storage.SManagedResourceBase.GetQuotaPlatformID()
}

func (guest *SGuest) GetQuotaPlatformID() []string {
	host := guest.GetHost()
	if host != nil {
		return host.GetQuotaPlatformID()
	}
	return GetDriver(guest.GetHypervisor()).GetQuotaPlatformID()
}

func (disk *SDisk) GetQuotaPlatformID() []string {
	storage := disk.GetStorage()
	if storage != nil {
		return storage.getQuotaPlatformID()
	}
	return []string{}
}

func GetQuotaPlatformID(hypervisor string) []string {
	if len(hypervisor) > 0 {
		return GetDriver(hypervisor).GetQuotaPlatformID()
	}
	return []string{}
}
