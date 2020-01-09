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
	// filter by city
	City string `json:"city"`

	// filter by cloudregion
	Cloudregion string `json:"cloudregion"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	Region string `json:"region"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	RegionId string `json:"region_id"`
}

func (input RegionalFilterListInput) CloudregionStr() string {
	if len(input.Cloudregion) > 0 {
		return input.Cloudregion
	}
	if len(input.CloudregionId) > 0 {
		return input.CloudregionId
	}
	if len(input.Region) > 0 {
		return input.Region
	}
	if len(input.RegionId) > 0 {
		return input.RegionId
	}
	return ""
}

type ZonalFilterListInput struct {
	RegionalFilterListInput

	// filter by zone
	Zone string `json:"zone"`
	// swagger:ignore
	// Deprecated
	// filter by zone_id
	ZoneId string `json:"zone_id"`
	// filter by an array of zone
	Zones []string `json:"zones"`
}

func (input ZonalFilterListInput) ZoneStr() string {
	if len(input.Zone) > 0 {
		return input.Zone
	}
	if len(input.ZoneId) > 0 {
		return input.ZoneId
	}
	return ""
}

func (input ZonalFilterListInput) ZoneList() []string {
	zoneStr := input.ZoneStr()
	if len(zoneStr) > 0 {
		input.Zones = append(input.Zones, zoneStr)
	}
	return input.Zones
}

type HostFilterListInput struct {
	ZonalFilterListInput

	// filter by host
	Host string `json:"host"`
	// swagger:ignore
	// Deprecated
	// filter by host_id
	HostId string `json:"host_id"`
}

func (input HostFilterListInput) HostStr() string {
	if len(input.Host) > 0 {
		return input.Host
	}
	if len(input.HostId) > 0 {
		return input.HostId
	}
	return ""
}

type CloudregionListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput
	UsableResourceListInput
	UsableVpcResourceListInput

	// filter by city
	City string `json:"city"`
	// filter by service???
	Service string `json:"service"`
}

type ZoneListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput

	RegionalFilterListInput

	UsableResourceListInput
	UsableVpcResourceListInput

	// filter by service
	Service string `json:"service"`
}
