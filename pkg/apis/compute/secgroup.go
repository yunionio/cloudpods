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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/apis"
)

type SSecgroupRuleCreateInput struct {
	apis.ResourceBaseCreateInput

	// 优先级, 数字越大优先级越高
	// minimum: 1
	// maximum: 100
	// required: true
	Priority int `json:"priority"`

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

	// ip或cidr地址
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

	// 仅单独创建安全组规则时需要指定安全组
	// required: true
	Secgroup string `json:"secgroup"`

	// swagger:ignore
	SecgroupId string
}

func (input *SSecgroupRuleCreateInput) Check() error {
	rule := secrules.SecurityRule{
		Priority:  input.Priority,
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

	ServerFilterListInput

	// equals
	Equals string

	// 按缓存数量排序
	// pattern:asc|desc
	OrderByCacheCnt string `json:"order_by_cache_cnt"`

	// 按缓存关联主机数排序
	// pattern:asc|desc
	OrderByGuestCnt string `json:"order_by_guest_cnt"`

	// 是否被修改
	IsDirty *bool `json:"is_dirty"`
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

	// 以direction字段过滤安全组规则
	Direction string `json:"direction"`
	// 以action字段过滤安全组规则
	Action string `json:"action"`
	// 以protocol字段过滤安全组规则
	Protocol string `json:"protocol"`
}

type SecgroupResourceInput struct {
	// 过滤关联指定安全组（ID或Name）的列表结果
	Secgroup string `json:"secgroup"`
	// swagger:ignore
	// Deprecated
	// filter by secgroup_id
	SecgroupId string `json:"secgroup_id" "yunion:deprecated-by":"secgroup"`
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
	GuestCnt int `json:"guest_cnt"`
	// 安全组缓存数量
	CacheCnt int `json:"cache_cnt"`
	// 规则信息
	Rules string `json:"rules"`
	// 入方向规则信息
	InRules string `json:"in_rules"`
	// 出方向规则信息
	OutRules string `json:"out_rules"`
}

type SecurityGroupResourceInfo struct {
	// 安全组名称
	Secgroup string `json:"secgroup"`
}

type GuestsecgroupListInput struct {
	GuestJointsListInput
	SecgroupFilterListInput
}

type GuestsecgroupDetails struct {
	GuestJointResourceDetails

	SGuestsecgroup

	// 安全组名称
	Secgroup string `json:"secgroup"`
}
