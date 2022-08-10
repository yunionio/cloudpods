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
	"net/url"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	multicloud.ProxmoxTags

	region *SRegion

	Name    string
	Nodes   int
	Quorate int
	Version int
}

type ClusterStatus struct {
	Nodes   int    `json:"nodes,omitempty"`
	Quorate int    `json:"quorate,omitempty"`
	Version int    `json:"version,omitempty"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Online  int    `json:"online,omitempty"`
	Level   string `json:"level,omitempty"`
	Nodeid  int    `json:"nodeid,omitempty"`
	Local   int    `json:"local,omitempty"`
	IP      string `json:"ip,omitempty"`
}

func (self *SZone) GetId() string {
	return self.Name
}

func (self *SZone) GetGlobalId() string {
	return self.Name
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

	return host, nil
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := self.region.GetHosts()
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
	storages, err := self.region.GetStorages()
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
		return nil, errors.Wrapf(err, "GetStorage %s", id)
	}

	storage.zone = self
	return storage, nil
}

func (self *SRegion) GetZone() (*SZone, error) {
	ret := &SZone{region: self}
	css := []ClusterStatus{}

	noneClusterMsg := true
	nodeNum := 0

	err := self.get("/cluster/status", url.Values{}, &css)
	if err != nil {
		return nil, err
	}

	for _, cs := range css {
		if cs.Type == "cluster" {
			ret.Name = cs.Name
			ret.Nodes = cs.Nodes
			ret.Quorate = cs.Quorate
			ret.Version = cs.Version
			noneClusterMsg = false
			break
		} else {
			nodeNum++
		}

	}

	if noneClusterMsg == true {
		ret.Name = "cluster"
		ret.Nodes = nodeNum
	}

	return ret, nil
}
