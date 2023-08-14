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

package qingcloud

import "strings"

type SZone struct {
	region *SRegion

	Status   string
	ZoneId   string
	RegionId string
}

func (self *SRegion) GetZones() ([]SZone, error) {
	params := map[string]string{}
	resp, err := self.ec2Request("DescribeZones", params)
	if err != nil {
		return nil, err
	}
	ret := []SZone{}
	err = resp.Unmarshal(&ret, "zone_set")
	if err != nil {
		return nil, err
	}
	result := []SZone{}
	for i := range ret {
		if strings.HasPrefix(ret[i].ZoneId, self.Region) {
			result = append(result, ret[i])
		}
	}
	return result, nil
}
