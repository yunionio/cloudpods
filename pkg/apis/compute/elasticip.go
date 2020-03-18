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

type SElasticipCreateInput struct {
	apis.VirtualResourceCreateInput

	// 区域名称或Id, 建议使用Id
	// 在指定区域内创建弹性公网ip
	Cloudregion string `json:"cloudregion"`

	// swagger:ignore
	Region string
	// swagger:ignore
	RegionId string
	// swagger:ignore
	CloudregionId string

	// 子订阅Id, 建议使用Id
	// 使用指定子订阅创建弹性公网ip
	// 弹性公网ip和虚拟机在同一区域同一子订阅底下，才可以进行绑定操作
	Cloudprovider string `json:"cloudprovider"`
	// swagger:ignore
	Manager string
	// swagger:ignore
	ManagerId string

	// 计费类型: 流量或带宽
	//
	//
	//
	// | 平台		|	支持类型			|
	// | ---		|	--------			|
	// |Aliyun		| traffic, bandwidth	|
	// |腾讯云		| traffic				|
	// |Azure		| traffic				|
	// |Google		| traffic, bandwidth	|
	// |Ucloud		| traffic				|
	// |Aws			| traffic				|
	// |华为云		| traffic, bandwidth	|
	// |天翼云		| traffic, bandwidth	|
	// |KVM			| 不支持创建			|
	// |VMware		| 不支持创建			|
	// |ZStack		| traffic				|
	// |OpenStack	| traffic				|
	// default: traffic
	// enum: traffic, bandwidth
	ChargeType string `json:"charge_type"`

	// 子网名称或Id
	// 私有云创建此参数必传,例如Openstack, ZStack
	Network string `json:"network"`
	// swagger:ignore
	NetworkId string
}

type ElasticipDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SElasticip

	// 绑定资源名称
	AssociateName string `json:"associate_name"`
}

type ElasticipSyncstatusInput struct {
}
