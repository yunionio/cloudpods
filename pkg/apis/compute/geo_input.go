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

type CloudregionResourceListInput struct {
	// 区域名称或ID
	CloudregionId []string `json:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Cloudregion string `json:"cloudregion" yunion-deprecated-by:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Region string `json:"region" yunion-deprecated-by:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	RegionId string `json:"region_id" yunion-deprecated-by:"cloudregion_id"`
}

type CloudregionResourceInput struct {
	// 区域名称或ID
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Cloudregion string `json:"cloudregion" yunion-deprecated-by:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Region string `json:"region" yunion-deprecated-by:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	RegionId string `json:"region_id" yunion-deprecated-by:"cloudregion_id"`
}

type RegionalFilterListInput struct {
	// 过滤位于指定城市区域的资源
	City string `json:"city"`

	CloudregionResourceListInput

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
	ZoneIds []string `json:"zone_ids"`
	// Deprecated
	// swagger:ignore
	Zones []string `json:"zones" yunion-deprecated-by:"zone_ids"`

	// 按可用区名称排序
	// pattern:asc|desc
	OrderByZone string `json:"order_by_zone"`
}

func (input ZonalFilterListBase) ZoneList() []string {
	zones := make([]string, 0)
	if len(input.ZoneIds) > 0 {
		zones = append(zones, input.ZoneIds...)
	}
	if len(input.ZoneId) > 0 {
		zones = append(zones, input.ZoneId)
	}
	return zones
}

func (input ZonalFilterListBase) FirstZone() string {
	if len(input.ZoneId) > 0 {
		return input.ZoneId
	}
	if len(input.ZoneIds) > 0 {
		return input.ZoneIds[0]
	}
	return ""
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
	// 按可用区数量排序
	// pattern:asc|desc
	OrderByZoneCount string `json:"order_by_zone_count"`
	// 按vpc数量排序
	// pattern:asc|desc
	OrderByVpcCount string `json:"order_by_vpc_count"`
	// 按虚拟机数量排序
	// pattern:asc|desc
	OrderByGuestCount string `json:"order_by_guest_count"`
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

	OrderByWires             string
	OrderByHosts             string
	OrderByHostsEnabled      string
	OrderByBaremetals        string
	OrderByBaremetalsEnabled string
}

type ZoneResourceInput struct {
	// 可用区ID或名称
	// example:zone1
	ZoneId string `json:"zone_id"`

	// swagger:ignore
	// Deprecated
	Zone string `json:"zone" yunion-deprecated-by:"zone_id"`
}
