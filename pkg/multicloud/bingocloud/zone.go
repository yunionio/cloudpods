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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SZone struct {
	multicloud.SResourceBase
	multicloud.STagBase

	region      *SRegion
	ZoneName    string
	DisplayName string
	ZoneState   string
}

func (self *SZone) GetGlobalId() string {
	return self.ZoneName
}

func (self *SZone) GetId() string {
	return self.ZoneName
}

func (self *SZone) GetName() string {
	return self.DisplayName
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	return cloudprovider.SModelI18nTable{}
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetStatus() string {
	if self.ZoneState == "available" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_DISABLE
}

func (self *SRegion) GetZones() ([]SZone, error) {
	resp, err := self.invoke("DescribeAvailabilityZones", nil)
	if err != nil {
		return nil, err
	}
	ret := []SZone{}
	return ret, resp.Unmarshal(&ret, "availabilityZoneInfo")
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, errors.Wrapf(err, "GetZones")
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].GetGlobalId() == id {
			return zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
