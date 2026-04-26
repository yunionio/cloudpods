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

package proxmox

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SProxmoxClient) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.GetInstance(id)
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (cli *SProxmoxClient) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := cli.GetHosts()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		hosts[i].cli = cli
		ret = append(ret, &hosts[i])
	}
	return ret, nil
}

func (self *SProxmoxClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.GetHost(id)
	if err != nil {
		return nil, err
	}
	host.cli = self
	return host, nil
}

func (self *SProxmoxClient) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	host, err := self.GetHosts()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStoragecache{}
	for i := range host {
		ret = append(ret, &SStoragecache{
			client: self,
			Node:   host[i].Node,
		})
	}
	return ret, nil
}

func (self *SProxmoxClient) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := self.GetIStoragecaches()
	if err != nil {
		return nil, cloudprovider.ErrNotSupported
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SProxmoxClient) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := client.GetInstances("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstances")
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (client *SProxmoxClient) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return nil, errors.ErrNotSupported
}

func (client *SProxmoxClient) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, errors.ErrNotSupported
}

func (client *SProxmoxClient) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, errors.ErrNotSupported
}

func (client *SProxmoxClient) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.ErrNotSupported
}

func (client *SProxmoxClient) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.ErrNotSupported
}

func (client *SProxmoxClient) GetCloudEnv() string {
	return ""
}

func (client *SProxmoxClient) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (client *SProxmoxClient) Refresh() error {
	return nil
}

func (client *SProxmoxClient) IsEmulated() bool {
	return true
}

func (client *SProxmoxClient) GetSysTags() map[string]string {
	return nil
}

func (client *SProxmoxClient) GetTags() (map[string]string, error) {
	return nil, errors.Wrap(errors.ErrNotImplemented, "GetTags")
}

func (client *SProxmoxClient) GetISkus() ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (client *SProxmoxClient) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpc := SVpc{
		client: client,
	}
	return []cloudprovider.ICloudVpc{&vpc}, nil
}

func (client *SProxmoxClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, errors.ErrNotSupported
}

func (client *SProxmoxClient) GetUUID() string {
	return client.host
}

func (client *SProxmoxClient) GetId() string {
	return client.GetUUID()
}

func (client *SProxmoxClient) GetName() string {
	return client.cpcfg.Name
}

func (client *SProxmoxClient) GetGlobalId() string {
	return client.GetUUID()
}

func (client *SProxmoxClient) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (client *SProxmoxClient) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	return table
}

func (client *SProxmoxClient) GetProvider() string {
	return api.CLOUD_PROVIDER_PROXMOX
}

func (cli *SProxmoxClient) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := cli.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (cli *SProxmoxClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
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
