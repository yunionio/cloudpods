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

package huawei

import (
	"net/url"
	"strconv"

	"yunion.io/x/pkg/errors"
)

type SInstanceType struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Vcpus        string       `json:"vcpus"`
	RamMB        int          `json:"ram"`            // 内存大小
	OSExtraSpecs OSExtraSpecs `json:"os_extra_specs"` // 扩展规格
}

type OSExtraSpecs struct {
	EcsPerformancetype string `json:"ecs:performancetype"`
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=ListFlavors
func (self *SRegion) GetInstanceTypes(zoneId string) ([]SInstanceType, error) {
	query := url.Values{}
	if len(zoneId) > 0 {
		query.Set("available_zone", zoneId)
	}
	resp, err := self.list(SERVICE_ECS, "cloudservers/flavors", query)
	if err != nil {
		return nil, errors.Wrapf(err, "list flavors")
	}
	ret := []SInstanceType{}
	err = resp.Unmarshal(&ret, "flavors")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, zoneId string) ([]SInstanceType, error) {
	instanceTypes, err := self.GetInstanceTypes(zoneId)
	if err != nil {
		return nil, err
	}

	ret := make([]SInstanceType, 0)
	for _, t := range instanceTypes {
		// cpu & mem 都匹配才行
		if t.Vcpus == strconv.Itoa(cpu) && t.RamMB == memMB {
			ret = append(ret, t)
		}
	}

	return ret, nil
}
