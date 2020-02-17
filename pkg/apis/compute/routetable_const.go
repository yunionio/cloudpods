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

const (
	ROUTE_TABLE_TYPE_VPC = "VPC" // VPC路由器
	ROUTE_TABLE_TYPE_VBR = "VBR" // 边界路由器
)

const (
	ROUTE_ENTRY_TYPE_CUSTOM = "Custom" // 自定义路由
	ROUTE_ENTRY_TYPE_SYSTEM = "System" // 系统路由
)

const (
	Next_HOP_TYPE_INSTANCE        = "Instance"              // ECS实例。
	Next_HOP_TYPE_HAVIP           = "HaVip"                 // 高可用虚拟IP。
	Next_HOP_TYPE_VPN             = "VpnGateway"            // VPN网关。
	Next_HOP_TYPE_NAT             = "NatGateway"            // NAT网关。
	Next_HOP_TYPE_NETWORK         = "NetworkInterface"      // 辅助弹性网卡。
	Next_HOP_TYPE_ROUTER          = "RouterInterface"       // 路由器接口。
	Next_HOP_TYPE_IPV6            = "IPv6Gateway"           // IPv6网关。
	Next_HOP_TYPE_INTERNET        = "InternetGateway"       // Internet网关。
	Next_HOP_TYPE_EGRESS_INTERNET = "EgressInternetGateway" // egress only Internet网关。
)
