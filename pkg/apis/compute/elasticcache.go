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

type ElasticcacheDetails struct {
	apis.VirtualResourceDetails
	VpcResourceInfo
	ZoneResourceInfoBase

	SElasticcache

	// IP子网名称
	Network string `json:"network"`
}

type ElasticcacheResourceInfo struct {
	// 弹性缓存实例名称
	Elasticcache string `json:"elasticcache"`

	// 引擎
	Engine string `json:"engine"`
	// 引擎版本
	EngineVersion string `json:"engine_version"`

	// 归属VPC ID
	VpcId string `json:"vpc_id"`

	VpcResourceInfo

	// 归属Zone ID
	ZoneId string `json:"zone_id"`

	ZoneResourceInfoBase
}

type ELasticcacheResourceInput struct {
	// 弹性缓存实例(ID or Name)
	Elasticcache string `json:"elasticcache"`

	// swagger:ignore
	// Deprecated
	ElasticcacheId string `json:"elasticcache_id" deprecated-by:"elasticcache"`
}

type ElasticcacheFilterListInput struct {
	ELasticcacheResourceInput

	// 以弹性缓存实例名称排序
	OrderByElasticcache string `json:"order_by_elasticcache"`

	VpcFilterListInput

	ZonalFilterListBase
}

type ElasticcacheAccountDetails struct {
	apis.StatusStandaloneResourceDetails
	ElasticcacheResourceInfo

	SElasticcacheAccount
}

type ElasticcacheAclDetails struct {
	apis.StandaloneResourceDetails
	ElasticcacheResourceInfo

	SElasticcacheAcl
}

type ElasticcacheParameterDetails struct {
	apis.StandaloneResourceDetails
	ElasticcacheResourceInfo

	SElasticcacheParameter
}

type ElasticcacheSyncstatusInput struct {
}
