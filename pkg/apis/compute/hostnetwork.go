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

import "yunion.io/x/cloudmux/pkg/apis/compute"

type HostnetworkDetails struct {
	HostJointResourceDetails

	SHostnetwork

	// IP子网名称
	Network string `json:"network"`

	// 二层网络名称
	Wire string `json:"wire"`
	// 二层网络ID
	WireId string `json:"wire_id"`

	NicType compute.TNicType `json:"nic_type"`
}

type HostnetworkListInput struct {
	HostJointsListInput
	NetworkFilterListInput

	// IP地址
	IpAddr []string `json:"ip_addr"`
	// MAC地址
	MacAddr []string `json:"mac_addr"`
}
