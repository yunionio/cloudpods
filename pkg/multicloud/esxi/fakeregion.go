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

package esxi

import (
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func (cli *SESXiClient) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SESXiClient) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SESXiClient) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, err
	}

	ihosts := make([]cloudprovider.ICloudHost, 0)
	for i := 0; i < len(dcs); i += 1 {
		dcIHosts, err := dcs[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		ihosts = append(ihosts, dcIHosts...)
	}
	return ihosts, nil
}

func (cli *SESXiClient) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	hosts, err := cli.GetIHosts()
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		vm, err := host.GetIVMById(id)
		if err != cloudprovider.ErrNotFound {
			return vm, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SESXiClient) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for _, storage := range storages {
		disk, err := storage.GetIDiskById(id)
		if err != cloudprovider.ErrNotFound {
			return disk, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SESXiClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return cli.FindHostByIp(id)
}

func (cli *SESXiClient) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, err
	}

	iStorages := make([]cloudprovider.ICloudStorage, 0)
	for i := 0; i < len(dcs); i += 1 {
		dcIStorages, err := dcs[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStorages = append(iStorages, dcIStorages...)
	}
	return iStorages, nil
}

func (cli *SESXiClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	iStorages, err := cli.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(iStorages); i += 1 {
		if iStorages[i].GetGlobalId() == id {
			return iStorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SESXiClient) GetProvider() string {
	return api.CLOUD_PROVIDER_VMWARE
}

func (cli *SESXiClient) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storages, err := cli.GetIStorages()
	if err != nil {
		return nil, err
	}
	caches := make([]cloudprovider.ICloudStoragecache, 0)
	cacheIds := make([]string, 0)
	for i := range storages {
		iCache := storages[i].GetIStoragecache()
		if !utils.IsInStringArray(iCache.GetGlobalId(), cacheIds) {
			caches = append(caches, iCache)
			cacheIds = append(cacheIds, iCache.GetGlobalId())
		}
	}
	return caches, nil
}

func (cli *SESXiClient) GetIStoragecacheById(idstr string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := cli.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == idstr {
			return caches[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SESXiClient) GetISkus() ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SESXiClient) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}
