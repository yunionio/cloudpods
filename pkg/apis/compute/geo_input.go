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

package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type CloudregionResourceInput struct {
	// 区域名称或ID
	Cloudregion string `json:"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudregionId string `json:"cloudregion_id" "yunion:deprecated-by":"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Region string `json:"region" "yunion:deprecated-by":"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	RegionId string `json:"region_id" "yunion:deprecated-by":"cloudregion"`
}

type RegionalFilterListInput struct {
	// 过滤位于指定城市区域的资源
	City string `json:"city"`

	CloudregionResourceInput

	// 按区域名称过滤
	OrderByRegion string `json:"order_by_region"`
	// 按城市过滤
	OrderByCity string `json:"order_by_city"`
}

type ZonalFilterListInput struct {
	RegionalFilterListInput

	ZonalFilterListBase
}

type ZonalFilterListBase struct {
	ZoneResourceInput

	// 过滤处于多个指定可用区内的资源
	Zones []string `json:"zones"`

	// 按可用区名称排序
	// pattern:asc|desc
	OrderByZone string `json:"order_by_zone"`
}

func (input ZonalFilterListBase) ZoneList() []string {
	zoneStr := input.Zone
	if len(zoneStr) > 0 {
		input.Zones = append(input.Zones, zoneStr)
	}
	return input.Zones
}

func (input ZonalFilterListBase) FirstZone() string {
	if len(input.Zone) > 0 {
		return input.Zone
	}
	if len(input.Zones) > 0 {
		return input.Zones[0]
	}
	return ""
}
func (input ZonalFilterListInput) ZoneList() []string {
	zoneStr := input.Zone
	if len(zoneStr) > 0 {
		input.Zones = append(input.Zones, zoneStr)
	}
	return input.Zones
}

type CloudregionListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput
	UsableResourceListInput
	UsableVpcResourceListInput

	CapabilityListInput

	// 过滤位于指定城市的区域
	City string `json:"city"`
	// 过滤提供特定服务的区域
	Service string `json:"service"`

	// 云环境
	Environment []string `json:"environment"`
}

type ZoneListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput

	RegionalFilterListInput

	UsableResourceListInput
	UsableVpcResourceListInput

	// 过滤提供特定服务的可用区
	Service string `json:"service"`

	Location []string `json:"location"`
	Contacts []string `json:"contacts"`
}

type ZoneResourceInput struct {
	// 可用区ID或名称
	// example:zone1
	Zone string `json:"zone"`

	// swagger:ignore
	// Deprecated
	ZoneId string `json:"zone_id" "yunion:deprecated-by":"zone"`
}
