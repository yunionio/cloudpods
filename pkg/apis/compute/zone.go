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

import "yunion.io/x/onecloud/pkg/apis"

type ZoneCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// 区域名称或Id,建议使用Id
	Cloudregion string

	// swagger:ignore
	Region string
	// swagger:ignore
	RegionId string
	// swagger:ignore
	CloudregionId string
}

type ZoneGeneralUsage struct {
	// 可用区底下的宿主机数量
	// example: 3
	Hosts int `json:"hosts"`

	// 可用区底下启用的宿主机数量
	// example: 2
	HostsEnabled int `json:"hosts_enabled"`

	// 可用区底下的裸金属服务器数量
	// example: 1
	Baremetals int `json:"baremetals"`

	// 可用区底下启用的裸金属服务器数量
	// example: 1
	BaremetalsEnabled int `json:"baremetals_enabled"`

	// 可用区底下的二层网络数量
	// example: 3
	Wires int `json:"wires"`

	// 可用区底下的子网数量
	// example: 1
	Networks int `json:"networks"`

	// 可用区底下的块存储数量
	// example: 1
	Storages int `json:"storages"`
}

func (usage *ZoneGeneralUsage) IsEmpty() bool {
	if usage.Hosts > 0 {
		return false
	}
	if usage.Wires > 0 {
		return false
	}
	if usage.Networks > 0 {
		return false
	}
	if usage.Storages > 0 {
		return false
	}
	return true
}

type ZoneDetails struct {
	apis.StatusStandaloneResourceDetails
	CloudregionResourceInfo
	CloudenvResourceInfo

	ZoneGeneralUsage

	SZone
}

type ZoneResourceInfoBase struct {
	// 可用区名称
	// example: 北京一区
	Zone string `json:"zone"`

	// 纳管云的zoneId
	ZoneExtId string `json:"zone_ext_id"`
}

type Zone1ResourceInfoBase struct {
	// 可用区名称
	// example: 北京2区
	Zone1Name string `json:"zone_1_name"`

	// 纳管云的zoneId
	Zone1ExtId string `json:"zone_1_ext_id"`
}

type SlaveZoneResourceInfoBase struct {
	// 可用区名称
	// example: 北京2区
	SlaveZone string `json:"slave_zone"`

	// 纳管云的zoneId
	SlaveZoneExtId string `json:"slave_zone_ext_id"`
}

type ZoneResourceInfo struct {
	ZoneResourceInfoBase

	// 可用区的区域ID
	CloudregionId string `json:"cloudregion_id"`

	CloudregionResourceInfo
}

type ZonePurgeInput struct {
	// 云订阅Id, 若zone底下不存在任何资源,会删除当前zone
	// required: true
	ManagerId string `json:"manager_id"`
}
