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
	// | protocol | name    |
	// | -------- | ----    |
	// | any      | 所有协议|
	// | tcp      | TCP     |
	// | icmp     | ICMP    |
	// | udp      | UDP     |
	// enum: ["any", "tcp", "udp", "icmp"]
	Protocol string `json:"protocol"`

	// 端口列表, 参数为空代表任意端口
	// 此参数仅对protocol是tcp, udp时生效
	// 支持格式:
	// | 格式类型 | 举例    |
	// | -------- | ----    |
	// | 单端口   | 22      |
	// | 端口范围 | 100-200 |
	// | 不连续端口| 80,443 |
	// requried: false
	Ports string `json:"ports"`

	// swagger:ignore
	PortStart int
	// swagger:ignore
	PortEnd int

	// 方向
	// enum: ["in", "out"]
	// required: true
	Direction string `json:"direction"`

	// ip或cidr地址, 若指定peer_secgroup_id此参数不生效
	// example: 192.168.222.121
	CIDR string `json:"cidr"`

	// 行为
	// deny: 拒绝
	// allow: 允许
	// enum: ["deny", "allow"]
	// required: true
	Action string `json:"action"`

	// 规则描述信息
	// requried: false
	// example: test to create rule
	Description string `json:"description"`
}

type SSecgroupRuleResourceSet []SSecgroupRuleResource

type SSecgroupRuleCreateInput struct {
	apis.ResourceBaseCreateInput
	SSecgroupRuleResource

	// swagger:ignore
	Secgroup string `json:"secgroup"  yunion-deprecated-by:"secgroup_id"`

	// swagger: ignore
	Status string `json:"status"`

	// 安全组ID
	// required: true
	SecgroupId string `json:"secgroup_id"`
}

type SSecgroupRuleUpdateInput struct {
	apis.ResourceBaseUpdateInput

	Priority *int    `json:"priority"`
	Ports    *string `json:"ports"`
	// ip或cidr地址, 若指定peer_secgroup_id此参数不生效
	// example: 192.168.222.121
	CIDR *string `json:"cidr"`

	// 协议
	// required: true
	//
	//
	//
	// | protocol | name    |
	// | -------- | ----    |
	// | any      | 所有协议|
	// | tcp      | TCP     |
	// | icmp     | ICMP    |
	// | udp      | UDP     |
	// enum: ["any", "tcp", "udp", "icmp"]
	Protocol *string `json:"protocol"`

	// 行为
	// deny: 拒绝
	// allow: 允许
	// enum: ["deny", "allow"]
	// required: true
	Action *string `json:"action"`

	// 规则描述信息
	// requried: false
	// example: test to create rule
	Description string `json:"description"`
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
		if !regutils.MatchCIDR(input.CIDR) && !regutils.MatchIP4Addr(input.CIDR) && !regutils.MatchCIDR6(input.CIDR) && !regutils.MatchIP6Addr(input.CIDR) {
			return fmt.Errorf("invalid ip address: %s", input.CIDR)
		}
	} else {
		// empty CIDR means both IPv4 and IPv6
		// input.CIDR = "0.0.0.0/0"
	}

	return rule.ValidateRule()
}

type SSecgroupCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// vpc id
	// defualt: default
	VpcResourceInput
	// swagger: ignore
	CloudproviderResourceInput
	// swagger: ignore
	CloudregionResourceInput

	// swagger: ignore
	GlobalvpcId string `json:"globalvpc_id"`

	// 规则列表
	// required: false
	Rules []SSecgroupRuleCreateInput `json:"rules"`
}

type SecgroupListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	ServerResourceInput

	DBInstanceResourceInput
	ELasticcacheResourceInput

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

	VpcId string `json:"vpc_id"`

	LoadbalancerId string `json:"loadbalancer_id"`
	RegionalFilterListInput
	ManagedResourceListInput
}

type SecurityGroupRuleListInput struct {
	apis.ResourceBaseListInput
	apis.ExternalizedResourceBaseListInput
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
	RegionalFilterListInput
	ManagedResourceListInput

	// 以安全组排序
	OrderBySecgroup string `json:"order_by_secgroup"`
}

type SecgroupDetails struct {
	apis.SharableVirtualResourceDetails
	SSecurityGroup

	VpcResourceInfo
	GlobalVpcResourceInfo

	// 关联云主机数量, 不包含回收站云主机
	GuestCnt int `json:"guest_cnt,allowempty"`

	// 关联此安全组的云主机is_system为true数量, , 不包含回收站云主机
	SystemGuestCnt int `json:"system_guest_cnt,allowempty"`

	// admin_secgrp_id为此安全组的云主机数量, , 不包含回收站云主机
	AdminGuestCnt int `json:"admin_guest_cnt,allowempty"`

	// 关联LB数量
	LoadbalancerCnt int `json:"loadbalancer_cnt,allowempty"`

	// 关联RDS数量
	RdsCnt int `json:"rds_cnt,allowempty"`
	// 关联Redis数量
	RedisCnt int `json:"redis_cnt,allowempty"`

	// 所有关联的资源数量
	TotalCnt int `json:"total_cnt,allowempty"`
}

type SecurityGroupResourceInfo struct {
	// 安全组名称
	Secgroup string `json:"secgroup"`

	// VPC归属区域ID
	CloudregionId string `json:"cloudregion_id"`

	CloudregionResourceInfo

	// VPC归属云订阅ID
	ManagerId string `json:"manager_id"`

	ManagedResourceInfo
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

type ElasticcachesecgroupDetails struct {
	ElasticcacheJointResourceDetails

	SElasticcachesecgroup

	// 安全组名称
	Secgroup string `json:"secgroup"`
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

type SSecurityGroupRef struct {
	GuestCnt        int `json:"guest_cnt"`
	AdminGuestCnt   int `json:"admin_guest_cnt"`
	RdsCnt          int `json:"rds_cnt"`
	RedisCnt        int `json:"redis_cnt"`
	LoadbalancerCnt int `json:"loadbalancer_cnt"`
	TotalCnt        int `json:"total_cnt"`
}

func (self *SSecurityGroupRef) Sum() {
	self.TotalCnt = self.GuestCnt + self.AdminGuestCnt + self.RdsCnt + self.RedisCnt + self.LoadbalancerCnt
}

type SecurityGroupSyncstatusInput struct {
}
