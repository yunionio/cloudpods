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
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type ElasticcacheDetails struct {
	apis.VirtualResourceDetails
	VpcResourceInfo
	ZoneResourceInfoBase

	SElasticcache

	// IP子网名称
	Network string `json:"network"`

	// 关联安全组列表
	Secgroups []apis.StandaloneShortDesc `json:"secgroups"`

	// 备可用区列表
	SlaveZoneInfos []apis.StandaloneShortDesc `json:"slave_zone_infos"`
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
	ElasticcacheId string `json:"elasticcache_id"`

	// swagger:ignore
	// Deprecated
	Elasticcache string `json:"elasticcache" yunion-deprecated-by:"elasticcache_id"`
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
	apis.ProjectizedResourceInfo
	ElasticcacheResourceInfo

	SElasticcacheAccount
	ProjectId string `json:"tenant_id"`
}

type ElasticcacheAclDetails struct {
	apis.StandaloneResourceDetails
	apis.ProjectizedResourceInfo
	ElasticcacheResourceInfo

	SElasticcacheAcl
	ProjectId string `json:"tenant_id"`
}

type ElasticcacheParameterDetails struct {
	apis.StandaloneResourceDetails
	ElasticcacheResourceInfo

	SElasticcacheParameter
}

type ElasticcacheSyncstatusInput struct {
}

type ElasticcacheRemoteUpdateInput struct {
	// 是否覆盖替换所有标签
	ReplaceTags *bool `json:"replace_tags" help:"replace all remote tags"`
}

//type SElasticcacheJointsBase struct {
//	apis.SVirtualJointResourceBase
//	// 弹性缓存实例(ID or Name)
//	ElasticcacheId string `json:"elasticcache_id"`
//}

type ElasticcacheJointResourceDetails struct {
	apis.VirtualJointResourceBaseDetails

	// 弹性缓存实例名称
	Elasticcache string `json:"elasticcache"`
	// 弹性缓存实例ID
	ElasticcacheId string `json:"elasticcache_id"`
}

type ElasticcacheJointsListInput struct {
	apis.VirtualJointResourceBaseListInput

	ElasticcacheFilterListInput
}

type ElasticcacheJointBaseUpdateInput struct {
	apis.VirtualJointResourceBaseUpdateInput
}

type ElasticcacheSecgroupsInput struct {
	// 安全组Id列表
	// 实例必须处于运行状态
	//
	//
	// | 平台		 | 最多绑定安全组数量	|
	// |-------------|-------------------	|
	// | 腾讯云       | 10     			    |
	// | 华为云       | 不支持安全组			|
	// | 阿里云       | 不支持安全组			|
	SecgroupIds []string `json:"secgroup_ids"`
}

type ElasticcacheCreateInput struct {
	apis.VirtualResourceCreateInput

	// 安全组列表
	// 腾讯云需要传此参数
	// required: false
	ElasticcacheSecgroupsInput

	// 主可用区名称或Id
	Zone string `json:"zone"`

	// 备可用区名称或Id列表
	// 默认副本与主可用区一致
	// 支持此参数的云厂商: 腾讯云
	// required: false
	SlaveZones []string `json:"slave_zones"`

	// Ip子网名称或Id,建议使用Id
	// required: true
	Network string `json:"network"`

	// 网络类型
	//  enum: vpc, cLassic
	// required: true
	NetworkType string `json:"network_type"`

	// 弹性缓存Engine
	//  enum: redis, memcached
	// required: true
	Engine string `json:"engine"`

	// 弹性缓存Engine版本
	// required: false
	EngineVersion string `json:"engine_version"`

	// 实例规格
	// required: false
	InstanceType string `json:"instance_type"`

	// 初始密码
	// required: false
	Password string `json:"password"`

	// 安全组名称或Id
	// default: default
	Secgroup string `json:"secgroup"`

	// 内网IP
	// 阿里云、华为云此参数可选，其它公有云该参数无效
	// required: false
	PrivateIp string `json:"private_ip"`

	// swagger:ignore
	VpcId string

	// swagger:ignore
	ManagerId string

	// 包年包月时间周期
	Duration string `json:"duration"`

	// 是否自动续费(仅包年包月时生效)
	// default: false
	AutoRenew bool `json:"auto_renew"`

	// swagger:ignore
	ExpiredAt time.Time `json:"expired_at"`

	// 计费方式
	// enum: postpaid, prepaid
	BillingType string
	// swagger:ignore
	BillingCycle string

	// 弹性缓存维护时间段
	// 华为云此参数可选,其它云该参数无效
	// enum: 22:00:00, 02:00:00, 06:00:00, 10:00:00, 14:00:00, 18:00:00
	// required: false
	MaintainStartTime string `json:"maintain_start_time"`
}
