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

package incloudsphere

import (
	"fmt"
	"net/url"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	multicloud.InCloudSphereTags

	region *SRegion

	Id                 string `json:"id"`
	Name               string `json:"name"`
	NFSPath            string `json:"nfsPath"`
	Type               string `json:"type"`
	Description        string `json:"description"`
	HostNum            int64  `json:"hostNum"`
	VMNum              int64  `json:"vmNum"`
	ClusterNum         int64  `json:"clusterNum"`
	CfsDomainNum       int64  `json:"cfsDomainNum"`
	StorageNum         int64  `json:"storageNum"`
	ImageISONum        int64  `json:"imageIsoNum"`
	NetNum             int64  `json:"netNum"`
	NeutronNetNum      int64  `json:"neutronNetNum"`
	CPUCapacity        string `json:"cpuCapacity"`
	CPUAvailable       string `json:"cpuAvailable"`
	CPUUsed            string `json:"cpuUsed"`
	CPUUtilization     string `json:"cpuUtilization"`
	MemoryCapacity     string `json:"memoryCapacity"`
	MemoryAvailable    string `json:"memoryAvailable"`
	MemoryUsed         string `json:"memoryUsed"`
	MemoryUtilization  string `json:"memoryUtilization"`
	StorageCapacity    string `json:"storageCapacity"`
	StorageAvailable   string `json:"storageAvailable"`
	StorageUsed        string `json:"storageUsed"`
	StorageUtilization string `json:"storageUtilization"`
	DatastoreNum       int64  `json:"datastoreNum"`
	LocalstoreNum      int64  `json:"localstoreNum"`
	CfsstoreNum        int64  `json:"cfsstoreNum"`
	RawstoreNum        int64  `json:"rawstoreNum"`
	NfsstoreNum        int64  `json:"nfsstoreNum"`
	XactivestoreNum    int64  `json:"xactivestoreNum"`
	NetworkType        string `json:"networkType"`
	VswitchDtos        string `json:"vswitchDtos"`
	//SDNNetworkDtos     []interface{} `json:"sdnNetworkDtos"`
	SDNInit      bool   `json:"sdnInit"`
	SDNSpeedUp   bool   `json:"sdnSpeedUp"`
	SDNConfigDto string `json:"sdnConfigDto"`
}

func (self *SZone) GetId() string {
	return self.Id
}

func (self *SZone) GetGlobalId() string {
	return self.Id
}

func (self *SZone) GetName() string {
	return self.Name
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.region.GetHost(id)
	if err != nil {
		return nil, err
	}
	host.zone = self
	if host.DataCenterId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	return host, nil
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := self.region.GetHosts(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		hosts[i].zone = self
		ret = append(ret, &hosts[i])
	}
	return ret, nil
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.region.GetStoragesByDc(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = self
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.region.GetStorage(id)
	if err != nil {
		return nil, err
	}
	if storage.DataCenterId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	storage.zone = self
	return storage, nil
}

func (self *SRegion) GetZones() ([]SZone, error) {
	ret := []SZone{}
	return ret, self.list("/datacenters", url.Values{}, &ret)
}

func (self *SRegion) GetZone(id string) (*SZone, error) {
	ret := &SZone{region: self}
	res := fmt.Sprintf("/datacenters/%s", id)
	return ret, self.get(res, url.Values{}, ret)
}
