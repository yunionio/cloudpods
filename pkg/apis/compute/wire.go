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

type WireCreateInput struct {
	apis.StandaloneResourceCreateInput

	// 带宽大小,单位: Mbps
	// default: 0
	Bandwidth int `json:"bandwidth"`

	// mtu
	// minimum: 0
	// maximum: 1000000
	// default: 0
	Mtu int `json:"mtu"`

	// vpc名称或Id
	// required: true
	Vpc string `json:"vpc"`
	// swagger:ignore
	// Deprecated
	VpcId string `json:"vpc_id" deprecated-by:"vpc"`

	// 可用区名称或Id
	// required: true
	Zone string `json:"zone"`
	// swagger:ignore
	ZoneId string
}

type WireDetails struct {
	apis.StandaloneResourceDetails
	VpcResourceInfo
	ZoneResourceInfoBase

	SWire

	// IP子网数量
	// example: 1
	Networks int `json:"networks"`
	// VPC名称
}

type WireResourceInfoBase struct {
	// 二层网络(WIRE)的名称
	Wire string `json:"wire"`
}

type WireResourceInfo struct {
	WireResourceInfoBase

	// VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo

	// 可用区ID
	ZoneId string `json:"zone_id"`

	// 可用区
	Zone string `json:"zone"`
}
