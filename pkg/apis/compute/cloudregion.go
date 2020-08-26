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

type SCloudregionUsage struct {
	// 虚拟私有网络数量
	// example: 2
	VpcCount int `json:"vpc_count,allowempty"`
	// 可用区数量
	// example: 3
	ZoneCount int `json:"zone_count,allowempty"`
	// 云主机梳理
	// example: 2
	GuestCount int `json:"guest_count,allowempty"`
	// IP子网数量
	// example: 12
	NetworkCount int `json:"network_count,allowempty"`
	// 距离前天0点新增云主机数量
	// example: 0
	GuestIncrementCount int `json:"guest_increment_count,allowempty"`
}

type CloudregionDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	SCloudregionUsage

	SCloudregion

	// 云类型, public, private or onpremise
	// example: public
	CloudEnv string `json:"cloud_env"`
}

type CloudregionResourceInfo struct {
	// 区域的名称
	// example: Default
	Region string `json:"region"`

	// 区域的名称
	Cloudregion string `json:"cloudregion"`

	// 区域的Id
	// example: default
	RegionId string `json:"region_id"`

	// 纳管云区域的组合Id(平台+Id)
	// example: ZStack/59e7bc87-a6b3-4c39-8f02-c68e8243d4e4
	RegionExternalId string `json:"region_external_id"`

	// 纳管云区域的Id
	// example: 59e7bc87-a6b3-4c39-8f02-c68e8243d4e4
	RegionExtId string `json:"region_ext_id"`
}
