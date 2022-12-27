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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SZone struct {
	multicloud.SResourceBase
	CloudpodsTags
	region *SRegion

	api.ZoneDetails
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

func (self *SZone) GetStatus() string {
	return self.Status
}

func (self *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SZone) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SRegion) GetZones() ([]SZone, error) {
	zones := []SZone{}
	return zones, self.list(&modules.Zones, nil, &zones)
}

func (self *SRegion) GetZone(id string) (*SZone, error) {
	zone := &SZone{region: self}
	return zone, self.cli.get(&modules.Zones, id, nil, &zone)
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zone, err := self.GetZone(id)
	if err != nil {
		return nil, err
	}
	return zone, nil
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
