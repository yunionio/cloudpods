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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
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

func (cli *SESXiClient) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	hosts, err := cli.GetIHosts()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for _, host := range hosts {
		vm, err := host.GetIVMs()
		if err != nil {
			return nil, err
		}
		ret = append(ret, vm...)
	}
	return ret, nil
}

func (cli *SESXiClient) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	hosts, err := cli.GetIHosts()
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		vm, err := host.GetIVMById(id)
		if errors.Cause(err) != cloudprovider.ErrNotFound {
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
		if errors.Cause(err) != cloudprovider.ErrNotFound {
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

func (cli *SESXiClient) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	return []cloudprovider.ICloudVpc{cli.fakeVpc}, nil
}

func (client *SESXiClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, errors.ErrNotSupported
}

func (client *SESXiClient) GetId() string {
	return client.GetUUID()
}

func (client *SESXiClient) GetName() string {
	return client.cpcfg.Name
}

func (client *SESXiClient) GetGlobalId() string {
	return client.GetUUID()
}

func (client *SESXiClient) GetStatus() string {
	return "available"
}

func (client *SESXiClient) GetCloudEnv() string {
	return ""
}

func (client *SESXiClient) Refresh() error {
	return nil
}

func (client *SESXiClient) IsEmulated() bool {
	return true
}

func (client *SESXiClient) GetSysTags() map[string]string {
	return nil
}

func (client *SESXiClient) GetTags() (map[string]string, error) {
	return nil, errors.Wrap(errors.ErrNotImplemented, "GetTags")
}

func (client *SESXiClient) SetTags(tags map[string]string, replace bool) error {
	return errors.ErrNotImplemented
}

func (client *SESXiClient) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (client *SESXiClient) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return nil, errors.ErrNotSupported
}

func (client *SESXiClient) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, errors.ErrNotSupported
}

func (client *SESXiClient) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, errors.ErrNotSupported
}

func (client *SESXiClient) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.ErrNotSupported
}

func (client *SESXiClient) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.ErrNotSupported
}
