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

package aws

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

var StorageTypes = []string{
	api.STORAGE_GP2_SSD,
	api.STORAGE_GP3_SSD,
	api.STORAGE_IO1_SSD,
	api.STORAGE_IO2_SSD,
	api.STORAGE_ST1_HDD,
	api.STORAGE_SC1_HDD,
	api.STORAGE_STANDARD_HDD,
}

type SZone struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	region *SRegion
	host   *SHost

	iwires []cloudprovider.ICloudWire

	ZoneId    string // 沿用阿里云ZoneId,对应Aws ZoneName
	LocalName string
	State     string
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}

func (self *SZone) getNetworkById(networkId string) *SNetwork {
	log.Debugf("Search in wires %d", len(self.iwires))
	for i := 0; i < len(self.iwires); i += 1 {
		log.Debugf("Search in wire %s", self.iwires[i].GetName())
		wire := self.iwires[i].(*SWire)
		net := wire.getNetworkById(networkId)
		if net != nil {
			return net
		}
	}
	return nil
}

func (self *SZone) GetId() string {
	return self.ZoneId
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_CN, self.LocalName)
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_AWS_EN, self.LocalName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.ZoneId)
}

func (self *SZone) GetStatus() string {
	if self.State == "unavailable" {
		return api.ZONE_SOLDOUT
	} else {
		return api.ZONE_ENABLE
	}
}

func (self *SZone) Refresh() error {
	return nil
}

func (self *SZone) IsEmulated() bool {
	return false
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := self.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIHostById")
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	ret := []cloudprovider.ICloudStorage{}
	for i := range StorageTypes {
		storage := &SStorage{zone: self, storageType: StorageTypes[i]}
		ret = append(ret, storage)
	}
	return ret, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i += 1 {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "not found %s", id)
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, errors.Wrap(err, "GetIStorages")
	}
	for i := 0; i < len(storages); i += 1 {
		storage := storages[i].(*SStorage)
		if storage.storageType == category {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.ZoneId == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}

func (self *SRegion) TestStorageAvailable(zoneId, storageType string) (bool, error) {
	params := map[string]string{
		"AvailabilityZone": zoneId,
		"ClientToken":      utils.GenRequestId(20),
		"DryRun":           "true",
		"Size":             "125",
		"VolumeType":       storageType,
	}
	iops, ok := map[string]string{
		api.STORAGE_GP3_SSD: "3000",
		api.STORAGE_IO1_SSD: "100",
		api.STORAGE_IO2_SSD: "100",
	}[storageType]
	if ok {
		params["Iops"] = iops
	}
	ret := struct{}{}
	err := self.ec2Request("CreateVolume", params, &ret)
	if err != nil {
		if e, ok := err.(*sAwsError); ok && e.Errors.Code == "DryRunOperation" {
			return true, nil
		}
		return false, errors.Wrapf(err, "CreateVolume")
	}
	return true, nil
}
