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

type VpcDetails struct {
	apis.StandaloneResourceDetails
	SVpc
	CloudproviderInfo

	// 二层网络数量
	// example: 1
	WireCount int `json:"wire_count"`
	// IP子网个数
	// example: 2
	NetworkCount int `json:"network_count"`
	// 路由表个数
	// example: 0
	RoutetableCount int `json:"routetable_count"`
	// NAT网关个数
	// example: 0
	NatgatewayCount int `json:"natgateway_count"`
}
