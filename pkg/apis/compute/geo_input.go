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

type RegionalFilterListInput struct {
	// 过滤位于指定城市区域的资源
	City string `json:"city"`

	// 过滤处于指定区域内的资源
	Cloudregion string `json:"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudregionId string `json:"cloudregion_id" deprecated-by:"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Region string `json:"region" deprecated-by:"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	RegionId string `json:"region_id" deprecated-by:"cloudregion"`
}

type ZonalFilterListInput struct {
	RegionalFilterListInput

	// 过滤处于指定可用区内的资源
	Zone string `json:"zone"`
	// swagger:ignore
	// Deprecated
	// filter by zone_id
	ZoneId string `json:"zone_id" deprecated-by:"zone"`
	// 过滤处于多个指定可用区内的资源
	Zones []string `json:"zones"`
}

func (input ZonalFilterListInput) ZoneList() []string {
	zoneStr := input.Zone
	if len(zoneStr) > 0 {
		input.Zones = append(input.Zones, zoneStr)
	}
	return input.Zones
}

type HostFilterListInput struct {
	ZonalFilterListInput

	// 过滤关联指定宿主机（ID或Name）的列表结果
	Host string `json:"host"`
	// swagger:ignore
	// Deprecated
	// filter by host_id
	HostId string `json:"host_id" deprecated-by:"host"`
}

type CloudregionListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput
	UsableResourceListInput
	UsableVpcResourceListInput

	// 过滤位于指定城市的区域
	City string `json:"city"`
	// 过滤提供特定服务的区域
	Service string `json:"service"`
}

type ZoneListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput

	RegionalFilterListInput

	UsableResourceListInput
	UsableVpcResourceListInput

	// 过滤提供特定服务的可用区
	Service string `json:"service"`
}
