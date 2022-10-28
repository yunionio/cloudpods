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

package qcloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type InstanceChargeType string

const (
	PrePaidInstanceChargeType  InstanceChargeType = "PREPAID"
	PostPaidInstanceChargeType InstanceChargeType = "POSTPAID_BY_HOUR"
	CdhPaidInstanceChargeType  InstanceChargeType = "CDHPAID"
	DefaultInstanceChargeType                     = PostPaidInstanceChargeType
)

type SZone struct {
	multicloud.SResourceBase
	QcloudTags
	region *SRegion

	iwires []cloudprovider.ICloudWire

	host *SHost

	istorages []cloudprovider.ICloudStorage

	instanceTypes []string
	refreshTime   time.Time
	localstorages []string
	cloudstorages []string

	Zone      string
	ZoneName  string
	ZoneState string
}

func (self *SZone) GetId() string {
	return self.Zone
}

func (self *SZone) GetName() string {
	return self.ZoneName
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	en := self.ZoneName
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SZone) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.region.GetGlobalId(), self.Zone)
}

func (self *SZone) IsEmulated() bool {
	return false
}

func (self *SZone) Refresh() error {
	// do nothing
	return nil
}

func (self *SZone) GetStatus() string {
	if self.ZoneState == "AVAILABLE" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_SOLDOUT
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host := self.getHost()
	if host.GetGlobalId() == id {
		return host, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return []cloudprovider.ICloudHost{self.getHost()}, nil
}

func (self *SZone) getHost() *SHost {
	if self.host == nil {
		self.host = &SHost{zone: self}
	}
	return self.host
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

type SDiskConfigSet struct {
	Available      bool
	DeviceClass    string
	DiskChargeType string
	DiskType       string
	DiskUsage      string
	InstanceFamily string
	MaxDiskSize    int
	MinDiskSize    int
	Zone           string
}

func (self *SRegion) GetDiskConfigSet(zoneName string) ([]SDiskConfigSet, error) {
	params := map[string]string{}
	params["Region"] = self.Region
	params["Zones.0"] = zoneName
	params["InquiryType"] = "INQUIRY_CBS_CONFIG"
	body, err := self.cbsRequest("DescribeDiskConfigQuota", params)
	if err != nil {
		return nil, err
	}
	diskConfigSet := []SDiskConfigSet{}
	return diskConfigSet, body.Unmarshal(&diskConfigSet, "DiskConfigSet")
}

func (self *SZone) fetchStorages() error {
	self.istorages = []cloudprovider.ICloudStorage{}
	diskConfigSet, err := self.region.GetDiskConfigSet(self.Zone)
	if err != nil {
		return err
	}
	self.cloudstorages = []string{}
	for _, diskConfig := range diskConfigSet {
		if !utils.IsInStringArray(strings.ToUpper(diskConfig.DiskType), self.cloudstorages) {
			self.cloudstorages = append(self.cloudstorages, strings.ToUpper(diskConfig.DiskType))
			storage := SStorage{zone: self, storageType: diskConfig.DiskType, available: diskConfig.Available}
			self.istorages = append(self.istorages, &storage)
		}
	}
	for _, storageType := range []string{"CLOUD_PREMIUM", "CLOUD_SSD", "CLOUD_BASIC"} {
		if !utils.IsInStringArray(storageType, self.cloudstorages) {
			self.cloudstorages = append(self.cloudstorages, storageType)
			storage := SStorage{zone: self, storageType: storageType, available: false}
			self.istorages = append(self.istorages, &storage)
		}
	}
	self.localstorages, err = self.region.GetZoneLocalStorages(self.Zone)
	if err != nil {
		log.Errorf("falied to fetch local storage for zone %s", self.Zone)
	}
	for _, localstorageType := range self.localstorages {
		storage := SLocalStorage{zone: self, storageType: localstorageType, available: true}
		self.istorages = append(self.istorages, &storage)
	}
	for _, storageType := range QCLOUD_LOCAL_STORAGE_TYPES {
		if !utils.IsInStringArray(storageType, self.localstorages) {
			storage := SLocalStorage{zone: self, storageType: storageType, available: false}
			self.istorages = append(self.istorages, &storage)
		}
	}
	return nil
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

func (self *SZone) getLocalStorageByCategory(category string) (*SLocalStorage, error) {
	if len(self.localstorages) == 0 {
		err := self.fetchStorages()
		if err != nil {
			return nil, errors.Wrap(err, "fetchStorages")
		}
	}
	if utils.IsInStringArray(strings.ToUpper(category), self.localstorages) {
		return &SLocalStorage{zone: self, storageType: strings.ToUpper(category)}, nil
	}
	return nil, fmt.Errorf("No such storage %s", category)
}

func (self *SZone) validateStorageType(category string) error {
	if len(self.localstorages) == 0 || len(self.cloudstorages) == 0 {
		err := self.fetchStorages()
		if err != nil {
			return errors.Wrap(err, "fetchStorages")
		}
	}
	if utils.IsInStringArray(strings.ToUpper(category), self.localstorages) || utils.IsInStringArray(strings.ToUpper(category), self.cloudstorages) {
		return nil
	}
	return fmt.Errorf("No such storage %s", category)
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		if utils.IsInStringArray(storages[i].GetStorageType(), self.localstorages) {
			continue
			//return &SStorage{zone: self, storageType: strings.ToUpper(storages[i].GetStorageType())}, nil
		}
		storage := storages[i].(*SStorage)
		if strings.ToLower(storage.storageType) == strings.ToLower(category) {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
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

func (self *SZone) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
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

func (self *SZone) fetchInstanceTypes() {
	self.instanceTypes = []string{}
	params := map[string]string{}
	params["Region"] = self.region.Region
	params["Filters.0.Name"] = "zone"
	params["Filters.0.Values.0"] = self.Zone
	if body, err := self.region.cvmRequest("DescribeInstanceTypeConfigs", params, true); err != nil {
		log.Errorf("DescribeInstanceTypeConfigs error: %v", err)
	} else if configSet, err := body.GetArray("InstanceTypeConfigSet"); err != nil {
		log.Errorf("Get InstanceTypeConfigSet error: %v", err)
	} else {
		for _, config := range configSet {
			if instanceType, err := config.GetString("InstanceType"); err == nil && !utils.IsInStringArray(instanceType, self.instanceTypes) {
				self.instanceTypes = append(self.instanceTypes, instanceType)
			}
		}
		self.refreshTime = time.Now()
	}
}

func (self *SZone) getAvaliableInstanceTypes() []string {
	if self.instanceTypes == nil || len(self.instanceTypes) == 0 || time.Now().Sub(self.refreshTime).Hours() > refreshHours() {
		self.fetchInstanceTypes()
	}
	return self.instanceTypes
}

func refreshHours() float64 {
	return 5
}

func (self *SZone) getCosEndpoint() string {
	return fmt.Sprintf("cos.%s.myqcloud.com", self.GetId())
}

func (self *SZone) getCosWebsiteEndpoint() string {
	return fmt.Sprintf("cos-website.%s.myqcloud.com", self.GetId())
}
