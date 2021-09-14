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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/apis"
)

type SSecgroupRuleResource struct {
	// 优先级, 数字越大优先级越高
	// minimum: 1
	// maximum: 100
	// required: true
	Priority *int `json:"priority"`

	// 协议
	// required: true
	//
	//
	//
	// | protocol | name	|
	// | -------- | ----	|
	// | any	  | 所有协议|
	// | tcp	  | TCP		|
	// | icmp	  | ICMP	|
	// | udp	  | UDP 	|
	// enum: any, tcp, udp, icmp
	Protocol string `json:"protocol"`

	// 端口列表, 参数为空代表任意端口
	// 此参数仅对protocol是tcp, udp时生效
	// 支持格式:
	// | 格式类型 | 举例	|
	// | -------- | ----	|
	// | 单端口	  | 22		|
	// | 端口范围 | 100-200	|
	// | 不连续端口| 80,443	|
	// requried: false
	Ports string `json:"ports"`

	// swagger:ignore
	PortStart int
	// swagger:ignore
	PortEnd int

	// 方向
	// enum: in, out
	// required: true
	Direction string `json:"direction"`

	// ip或cidr地址, 若指定peer_secgroup_id此参数不生效
	// example: 192.168.222.121
	CIDR string `json:"cidr"`

	// 行为
	// deny: 拒绝
	// allow: 允许
	// enum: deny, allow
	// required: true
	Action string `json:"action"`

	// 规则描述信息
	// requried: false
	// example: test to create rule
	Description string `json:"description"`

	// 对端安全组Id, 此参数和cidr参数互斥，并且优先级高于cidr, 同时peer_secgroup_id不能和它所在的安全组ID相同
	// required: false
	PeerSecgroupId string `json:"peer_secgroup_id"`
}

type SSecgroupRuleCreateInput struct {
	apis.ResourceBaseCreateInput
	SSecgroupRuleResource

	// swagger:ignore
	Secgroup string `json:"secgroup"  yunion-deprecated-by:"secgroup_id"`

	// 安全组ID
	// required: true
	SecgroupId string `json:"secgroup_id"`
}

type SSecgroupRuleUpdateInput struct {
	apis.ResourceBaseUpdateInput

	SSecgroupRuleResource
}

func (input *SSecgroupRuleResource) Check() error {
	priority := 1
	if input.Priority != nil {
		priority = *input.Priority
	}
	rule := secrules.SecurityRule{
		Priority:  priority,
		Direction: secrules.TSecurityRuleDirection(input.Direction),
		Action:    secrules.TSecurityRuleAction(input.Action),
		Protocol:  input.Protocol,
		PortStart: input.PortStart,
		PortEnd:   input.PortEnd,
		Ports:     []int{},
	}

	if len(input.Ports) > 0 {
		err := rule.ParsePorts(input.Ports)
		if err != nil {
			return errors.Wrapf(err, "ParsePorts(%s)", input.Ports)
		}
	}

	if len(input.CIDR) > 0 {
		if !regutils.MatchCIDR(input.CIDR) && !regutils.MatchIPAddr(input.CIDR) {
			return fmt.Errorf("invalid ip address: %s", input.CIDR)
		}
	} else {
		input.CIDR = "0.0.0.0/0"
	}

	return rule.ValidateRule()
}

type SSecgroupCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// 规则列表
	// required: false
	Rules []SSecgroupRuleCreateInput `json:"rules"`
}

type SecgroupListInput struct {
	apis.SharableVirtualResourceListInput

	ServerResourceInput

	DBInstanceResourceInput
	ELasticcacheResourceInput

	// equals
	Equals string

	// 按缓存数量排序
	// pattern:asc|desc
	OrderByCacheCnt string `json:"order_by_cache_cnt"`

	// 按缓存关联主机数排序
	// pattern:asc|desc
	OrderByGuestCnt string `json:"order_by_guest_cnt"`

	// 模糊过滤规则中含有指定ip的安全组
	// example: 10.10.2.1
	Ip string `json:"ip"`

	// 精确匹配规则中含有指定端口的安全组
	// example: 100-200
	Ports string `json:"ports"`

	// 指定过滤规则的方向(仅在指定ip或ports时生效) choices: all|in|out
	// default: all
	// example: in
	Direction string `json:"direction"`

	RegionalFilterListInput

	ManagedResourceListInput
	WithCache bool `json:"witch_cache"`
}

type SecurityGroupCacheListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	VpcFilterListInput
	SecgroupFilterListInput
}

type SecurityGroupRuleListInput struct {
	apis.ResourceBaseListInput
	SecgroupFilterListInput

	Projects []string `json:"projects"`

	// 以direction字段过滤安全组规则
	Direction string `json:"direction"`
	// 以action字段过滤安全组规则
	Action string `json:"action"`
	// 以protocol字段过滤安全组规则
	Protocol string `json:"protocol"`
	// 以ports字段过滤安全组规则
	Ports string `json:"ports"`
	// 根据ip模糊匹配安全组规则
	Ip string `json:"ip"`
}

type SecgroupResourceInput struct {
	// 过滤关联指定安全组（ID或Name）的列表结果
	SecgroupId string `json:"secgroup_id"`
	// swagger:ignore
	// Deprecated
	// filter by secgroup_id
	Secgroup string `json:"secgroup" yunion-deprecated-by:"secgroup_id"`

	// 模糊匹配安全组规则名称
	SecgroupName string `json:"secgroup_name"`
}

type SecgroupFilterListInput struct {
	SecgroupResourceInput

	// 以安全组排序
	OrderBySecgroup string `json:"order_by_secgroup"`
}

type SecgroupDetails struct {
	apis.SharableVirtualResourceDetails
	SSecurityGroup

	// 关联云主机数量
	GuestCnt int `json:"guest_cnt,allowempty"`

	// 关联此安全组的云主机is_system为true数量
	SystemGuestCnt int `json:"system_guest_cnt,allowempty"`

	// admin_secgrp_id为此安全组的云主机数量
	AdminGuestCnt int `json:"admin_guest_cnt,allowempty"`

	// 安全组缓存数量
	CacheCnt int `json:"cache_cnt,allowempty"`
	// 规则信息
	Rules []SecgroupRuleDetails `json:"rules"`
	// 入方向规则信息
	InRules []SecgroupRuleDetails `json:"in_rules"`
	// 出方向规则信息
	OutRules []SecgroupRuleDetails `json:"out_rules"`

	CloudCaches []jsonutils.JSONObject `json:"cloud_caches"`
}

type SecurityGroupResourceInfo struct {
	// 安全组名称
	Secgroup string `json:"secgroup"`
}

type GuestsecgroupListInput struct {
	GuestJointsListInput
	SecgroupFilterListInput
}

type ElasticcachesecgroupListInput struct {
	ElasticcacheJointsListInput
	SecgroupFilterListInput
}

type GuestsecgroupDetails struct {
	GuestJointResourceDetails

	SGuestsecgroup

	// 安全组名称
	Secgroup string `json:"secgroup"`
}

//type SElasticcachesecgroup struct {
//	SElasticcacheJointsBase
//	SSecurityGroupResourceBase
//}

type ElasticcachesecgroupDetails struct {
	ElasticcacheJointResourceDetails

	SElasticcachesecgroup

	// 安全组名称
	Secgroup string `json:"secgroup"`
}

type SecgroupMergeInput struct {
	// 安全组id列表
	SecgroupIds []string `json:"secgroup_ids"`

	// swagger:ignore
	// Deprecated
	Secgroups []string `json:"secgroup" yunion-deprecated-by:"secgroup_ids"`
}

type SecurityGroupPurgeInput struct {
}

type SecurityGroupCloneInput struct {
	Name        string
	Description string
}

type SecgroupImportRulesInput struct {
	Rules []SSecgroupRuleCreateInput `json:"rules"`
}

type SecgroupJsonDesc struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
