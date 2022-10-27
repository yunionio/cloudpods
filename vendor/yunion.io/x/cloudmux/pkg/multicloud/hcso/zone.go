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

package hcso

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

var StorageTypes = []string{
	api.STORAGE_HUAWEI_SAS,
	api.STORAGE_HUAWEI_SATA,
	api.STORAGE_HUAWEI_SSD,
}

type ZoneState struct {
	Available bool `json:"available"`
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0065817728.html
type SZone struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
	region *SRegion
	ihosts []cloudprovider.ICloudHost

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ZoneState ZoneState `json:"zoneState"`
	ZoneName  string    `json:"zoneName"`

	/* 支持的磁盘种类集合 */
	storageTypes []string
}

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SZone) getStorageType() {
	if len(self.storageTypes) == 0 {
		if sts, err := self.region.GetZoneSupportedDiskTypes(self.GetId()); err == nil {
			self.storageTypes = sts
		} else {
			log.Errorf("GetZoneSupportedDiskTypes %s %s", self.GetId(), err)
			self.storageTypes = StorageTypes
		}
	}
}

func (self *SZone) fetchStorages() error {
	self.getStorageType()
	self.istorages = make([]cloudprovider.ICloudStorage, len(self.storageTypes))

	for i, sc := range self.storageTypes {
		storage := SStorage{zone: self, storageType: sc}
		self.istorages[i] = &storage
	}
	return nil
}

// 华为私有云没有直接列出host的接口，所有账号下的host都是通过VM反向解析出来的
// 当账号下没有虚拟机时，如果没有host，会导致调度找不到可用的HOST。
// 因此，为了避免上述情况始终会在每个zone下返回一台虚拟的host
func (self *SZone) getEmulatedHost() SHost {
	return SHost{
		zone:      self,
		vms:       nil,
		IsFake:    true,
		projectId: self.region.client.projectId,
		Id:        fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.GetId()),
		Name:      fmt.Sprintf("%s-%s", self.region.client.cpcfg.Name, self.GetId()),
	}
}

func (self *SZone) getHosts() ([]cloudprovider.ICloudHost, error) {
	if self.ihosts != nil {
		return self.ihosts, nil
	}

	vms, err := self.region.GetInstances()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}

	hosts := map[string]string{}
	hostVms := map[string][]SInstance{}
	for i := range vms {
		vm := vms[i]
		if vm.OSEXTAZAvailabilityZone == self.GetId() {
			hosts[vm.HostID] = vm.OSEXTSRVATTRHost
			if _, ok := hostVms[vm.HostID]; ok {
				hostVms[vm.HostID] = append(hostVms[vm.HostID], vm)
			} else {
				hostVms[vm.HostID] = []SInstance{vm}
			}
		}
	}

	fakeHost := self.getEmulatedHost()
	ihosts := []cloudprovider.ICloudHost{&fakeHost}
	for k, _ := range hosts {
		h := SHost{
			zone:      self,
			projectId: self.region.client.projectId,
			Id:        k,
			Name:      hosts[k],
		}
		for i := range hostVms[k] {
			hostVms[k][i].host = &h
		}

		h.vms = hostVms[k]
		ihosts = append(ihosts, &h)
	}

	return ihosts, nil
}

func (self *SZone) GetId() string {
	return self.ZoneName
}

func (self *SZone) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_CN, self.ZoneName)
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_EN, self.ZoneName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.ZoneName)
}

func (self *SZone) GetStatus() string {
	return "enable"
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
	return self.getHosts()
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	ihosts, err := self.getHosts()
	if err != nil {
		return nil, errors.Wrap(err, "getHosts")
	}

	for i := range ihosts {
		if ihosts[i].GetGlobalId() == id {
			return ihosts[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		err := self.fetchStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchStorages")
		}
	}
	return self.istorages, nil
}

func (self *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if self.istorages == nil {
		err := self.fetchStorages()
		if err != nil {
			return nil, errors.Wrapf(err, "fetchStorages")
		}
	}
	for i := 0; i < len(self.istorages); i += 1 {
		if self.istorages[i].GetGlobalId() == id {
			return self.istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.iwires, nil
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.GetId() == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}

func (self *SZone) getNetworkById(networkId string) *SNetwork {
	for i := 0; i < len(self.iwires); i += 1 {
		wire := self.iwires[i].(*SWire)
		net := wire.getNetworkById(networkId)
		if net != nil {
			return net
		}
	}
	return nil
}
