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

package bingocloud

import (
	"fmt"

	"github.com/pkg/errors"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SZone struct {
	region      *SRegion
	DisplayName string `json:"displayName"`
	ZoneName    string `json:"zoneName"`
	ZoneState   string `json:"zoneState"`

	SCluster
}

func (self *SRegion) GetZones() ([]SZone, error) {
	resp, err := self.invoke("DescribeAvailabilityZones", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		AvailabilityZoneInfo struct {
			Item []SZone
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	return result.AvailabilityZoneInfo.Item, nil
}

func (self *SZone) GetName() string {
	return self.ZoneName
}
func (self *SZone) GetDisplayName() string {
	return self.DisplayName
}
func (self *SZone) GetState() string {
	return self.GetState()
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.ZoneName)
}

func (self *SZone) IsEmulated() bool {
	return false
}

func (self *SZone) Refresh() error {
	return nil
}

func (self *SZone) GetId() string {
	return self.ClusterId
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := self.region.GetHosts()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIHosts")
	}
	firstHost := true
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		if hosts[i].InstanceId != self.ClusterId {
			continue
		}
		hosts[i].zone = self
		ret = append(ret, &hosts[i])
		if firstHost {
			firstHost = false
		}
	}
	return ret, nil
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.region.GetHost(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIHostById(%s)", id)
	}
	if host.InstanceId != self.ClusterId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	host.zone = self
	return host, nil
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetStatus() string {
	return api.ZONE_ENABLE
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.region.GetStorages()
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorages")
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		if storages[i].StorageId != self.ClusterId {
			continue
		}
		storages[i].zone = self
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.region.GetStorage(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage", id)
	}
	if storage.StorageId != self.ClusterId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	storage.zone = self
	return storage, nil
}

func (self *SZone) GetSysTags() map[string]string {
	return nil
}

func (self *SZone) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SZone) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}
