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

	host *SHost

	Zone      string
	ZoneName  string
	ZoneState string
}

func (self *SZone) GetId() string {
	return self.Zone
}

func (self *SZone) GetName() string {
	if len(self.ZoneName) > 0 {
		return self.ZoneName
	}
	return self.Zone
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

func (self *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	ret := []cloudprovider.ICloudStorage{}
	diskConfigSet, err := self.region.GetDiskConfigSet(self.Zone)
	if err != nil {
		return nil, err
	}
	cloudstorages := []string{}
	for _, diskConfig := range diskConfigSet {
		if !utils.IsInStringArray(strings.ToUpper(diskConfig.DiskType), cloudstorages) {
			cloudstorages = append(cloudstorages, strings.ToUpper(diskConfig.DiskType))
			storage := &SStorage{zone: self, storageType: diskConfig.DiskType, available: diskConfig.Available}
			ret = append(ret, storage)
		}
	}
	for _, storageType := range []string{"CLOUD_PREMIUM", "CLOUD_SSD", "CLOUD_BASIC"} {
		if !utils.IsInStringArray(storageType, cloudstorages) {
			cloudstorages = append(cloudstorages, storageType)
			storage := SStorage{zone: self, storageType: storageType, available: false}
			ret = append(ret, &storage)
		}
	}
	localstorages, err := self.region.GetZoneLocalStorages(self.Zone)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotSupported {
		return nil, errors.Wrapf(err, "GetZoneLocalStorages")
	}
	for _, localstorageType := range localstorages {
		storage := SLocalStorage{zone: self, storageType: localstorageType, available: true}
		ret = append(ret, &storage)
	}
	for _, storageType := range QCLOUD_LOCAL_STORAGE_TYPES {
		if !utils.IsInStringArray(storageType, localstorages) {
			storage := SLocalStorage{zone: self, storageType: storageType, available: false}
			ret = append(ret, &storage)
		}
	}
	return ret, nil
}

func (self *SZone) getLocalStorageByCategory(category string) (*SLocalStorage, error) {
	localstorages, err := self.region.GetZoneLocalStorages(self.Zone)
	if err != nil {
		return nil, errors.Wrapf(err, "GetZoneLocalStorages")
	}
	if utils.IsInStringArray(strings.ToUpper(category), localstorages) {
		return &SLocalStorage{zone: self, storageType: strings.ToUpper(category)}, nil
	}
	return nil, fmt.Errorf("No such storage %s", category)
}

func (self *SZone) getStorageByCategory(category string) (*SStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		storage, ok := storages[i].(*SStorage)
		if !ok {
			continue
		}
		if strings.ToLower(storage.storageType) == strings.ToLower(category) {
			return storage, nil
		}
	}
	return nil, fmt.Errorf("No such storage %s", category)
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
	return nil, cloudprovider.ErrNotFound
}

func (self *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.region.GetVpcs(nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		wire := &SWire{vpc: &vpcs[i], zone: self}
		ret = append(ret, wire)
	}
	return ret, nil
}

func (self *SZone) getCosEndpoint() string {
	return fmt.Sprintf("cos.%s.myqcloud.com", self.GetId())
}

func (self *SZone) getCosWebsiteEndpoint() string {
	return fmt.Sprintf("cos-website.%s.myqcloud.com", self.GetId())
}
