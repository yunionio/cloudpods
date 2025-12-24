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

package baidu

import (
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
)

type SInstanceType struct {
	SpecId              string
	CpuCount            int
	MemoryCapacityInGb  int
	EphemeralDiskCount  int
	CpuModel            string
	CpuGhz              string
	EnableJumboFrame    bool
	NicIpv4Quota        int
	NicIpv6Quota        int
	NetworkBandwidth    string
	NetworkPackage      string
	NetEthQueueCount    string
	NetEthMaxQueueCount string
	ProductType         string
	Spec                string
	ZoneName            string
	GroupId             string
	SystemDiskType      []string
	DataDiskType        []string
}

func (region *SRegion) GetInstanceTypes(zoneName string, specs []string, specIds []string) ([]SInstanceType, error) {
	params := url.Values{}
	if len(zoneName) > 0 {
		params.Set("zoneName", zoneName)
	}
	if len(specs) > 0 {
		params.Set("specs", strings.Join(specs, ","))
	}
	if len(specIds) > 0 {
		params.Set("specIds", strings.Join(specIds, ","))
	}

	ret := []SInstanceType{}
	resp, err := region.bccList("v2/instance/flavorSpec", params)
	if err != nil {
		return nil, errors.Wrap(err, "list instance")
	}
	part := struct {
		ZoneResources []struct {
			ZoneName     string
			ebsResource  struct{}
			BccResources struct {
				FlavorGroups []struct {
					GroupId string
					Flavors []SInstanceType
				}
			}
		}
	}{}

	err = resp.Unmarshal(&part)
	if err != nil {
		return nil, err
	}

	for _, zoneResource := range part.ZoneResources {
		for _, flavorGroup := range zoneResource.BccResources.FlavorGroups {
			ret = append(ret, flavorGroup.Flavors...)
		}
	}

	return ret, nil
}
