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

package qcloud

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SecurityGroupPolicy struct {
	region            *SRegion
	PolicyIndex       int                          // 安全组规则索引号。
	Protocol          string                       // 协议, 取值: TCP,UDP, ICMP。
	Port              string                       // 端口(all, 离散port, range)。
	ServiceTemplate   ServiceTemplateSpecification // 协议端口ID或者协议端口组ID。ServiceTemplate和Protocol+Port互斥。
	CidrBlock         string                       // 网段或IP(互斥)。
	SecurityGroupId   string                       // 已绑定安全组的网段或IP。
	AddressTemplate   AddressTemplateSpecification // IP地址ID或者ID地址组ID。
	Action            string                       // ACCEPT 或 DROP。
	PolicyDescription string                       // 安全组规则描述。
	direction         string
}

type ServiceTemplateSpecification struct {
	ServiceId      string //	协议端口ID，例如：ppm-f5n1f8da。
	ServiceGroupId string //	协议端口组ID，例如：ppmg-f5n1f8da。
}

type AddressTemplateSpecification struct {
	AddressId      string //	IP地址ID，例如：ipm-2uw6ujo6。
	AddressGroupId string //	IP地址组ID，例如：ipmg-2uw6ujo6。
}

type SecurityGroupPolicySet struct {
	Version string
	Egress  []SecurityGroupPolicy //	出站规则。
	Ingress []SecurityGroupPolicy //	入站规则。
}

type SSecurityGroup struct {
	region                 *SRegion
	SecurityGroupId        string    //		安全组实例ID，例如：sg-ohuuioma。
	SecurityGroupName      string    //		安全组名称，可任意命名，但不得超过60个字符。
	SecurityGroupDesc      string    //		安全组备注，最多100个字符。
	ProjectId              string    //		项目id，默认0。可在qcloud控制台项目管理页面查询到。
	IsDefault              bool      // 	是否是默认安全组，默认安全组不支持删除。
	CreatedTime            time.Time // 	安全组创建时间。
	SecurityGroupPolicySet SecurityGroupPolicySet
}

type SecurityGroupRuleSet []SecurityGroupPolicy

func (v SecurityGroupRuleSet) Len() int {
	return len(v)
}

func (v SecurityGroupRuleSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v SecurityGroupRuleSet) Less(i, j int) bool {
	if v[i].PolicyIndex < v[j].PolicyIndex {
		return true
	} else if v[i].PolicyIndex == v[j].PolicyIndex {
		return strings.Compare(v[i].String(), v[j].String()) <= 0
	}
	return false
}

func (self *SRegion) GetSecurityGroups(vpcId string, offset int, limit int) ([]SSecurityGroup, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	body, err := self.vpcRequest("DescribeSecurityGroups", params)
	if err != nil {
		log.Errorf("GetSecurityGroups fail %s", err)
		return nil, 0, err
	}

	secgrps := make([]SSecurityGroup, 0)
	err = body.Unmarshal(&secgrps, "SecurityGroupSet")
	if err != nil {
		log.Errorf("Unmarshal security groups fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return secgrps, int(total), nil
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSecurityGroup) GetVpcId() string {
	//腾讯云安全组未与vpc关联，统一使用normal
	return "normal"
}

func (self *SSecurityGroup) GetId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetDescription() string {
	return self.SecurityGroupDesc
}

func (self *SSecurityGroup) GetName() string {
	if len(self.SecurityGroupName) > 0 {
		return self.SecurityGroupName
	}
	return self.SecurityGroupId
}

func (self *SecurityGroupPolicy) String() string {
	rules := self.toRules()
	result := []string{}
	for _, rule := range rules {
		result = append(result, rule.String())
	}
	return strings.Join(result, ";")
}

func parseCIDR(cidr string) (*net.IPNet, error) {
	if strings.Index(cidr, "/") > 0 {
		_, ipnet, err := net.ParseCIDR(cidr)
		return ipnet, err
	}
	ip := net.ParseIP(cidr)
	if ip == nil {
		return nil, fmt.Errorf("Parse ip %s error", cidr)
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}, nil
}

func (self *SecurityGroupPolicy) toRules() []secrules.SecurityRule {
	result := []secrules.SecurityRule{}
	rule := secrules.SecurityRule{
		Action:    secrules.SecurityRuleAllow,
		Protocol:  secrules.PROTO_ANY,
		Direction: secrules.TSecurityRuleDirection(self.direction),
		Ports:     []int{},
		PortStart: -1,
		PortEnd:   -1,
	}
	if len(self.SecurityGroupId) != 0 {
		//安全组关联安全组的规则忽略
		return nil
	}
	if strings.ToLower(self.Action) == "drop" {
		rule.Action = secrules.SecurityRuleDeny
	}
	if utils.IsInStringArray(strings.ToLower(self.Protocol), []string{"tcp", "udp", "icmp"}) {
		rule.Protocol = strings.ToLower(self.Protocol)
	}
	if strings.Index(self.Port, ",") > 0 {
		for _, _port := range strings.Split(self.Port, ",") {
			port, err := strconv.Atoi(_port)
			if err != nil {
				log.Errorf("parse secgroup port %s %s error %v", self.Port, _port, err)
				continue
			}
			rule.Ports = append(rule.Ports, port)
		}
	} else if strings.Index(self.Port, "-") > 0 {
		ports := strings.Split(self.Port, "-")
		if len(ports) == 2 {
			portStart, err := strconv.Atoi(ports[0])
			if err != nil {
				return nil
			}
			portEnd, err := strconv.Atoi(ports[1])
			if err != nil {
				return nil
			}
			rule.PortStart, rule.PortEnd = portStart, portEnd
		}
	} else if strings.ToLower(self.Port) != "all" {
		port, err := strconv.Atoi(self.Port)
		if err != nil {
			return nil
		}
		rule.PortStart, rule.PortEnd = port, port
	}

	if len(self.AddressTemplate.AddressGroupId) > 0 {
		addressGroup, total, err := self.region.AddressGroupList(self.AddressTemplate.AddressGroupId, "", 0, 1)
		if err != nil {
			log.Errorf("Get AddressList %s failed %v", self.AddressTemplate.AddressId, err)
			return nil
		}
		if total != 1 {
			return nil
		}
		for i := 0; i < len(addressGroup[0].AddressTemplateIdSet); i++ {
			rules, err := self.getAddressRules(rule, addressGroup[0].AddressTemplateIdSet[i])
			if err != nil {
				return nil
			}
			result = append(result, rules...)
		}
	} else if len(self.AddressTemplate.AddressId) > 0 {
		rules, err := self.getAddressRules(rule, self.AddressTemplate.AddressId)
		if err != nil {
			return nil
		}
		result = append(result, rules...)
	} else if len(self.CidrBlock) > 0 {
		ipnet, err := parseCIDR(self.CidrBlock)
		if err != nil {
			return nil
		}
		rule.IPNet = ipnet
		result = append(result, rule)
	}
	return result
}

func (self *SecurityGroupPolicy) getAddressRules(rule secrules.SecurityRule, addressId string) ([]secrules.SecurityRule, error) {
	result := []secrules.SecurityRule{}
	address, total, err := self.region.AddressList(addressId, "", 0, 1)
	if err != nil {
		log.Errorf("Get AddressList %s failed %v", self.AddressTemplate.AddressId, err)
		return nil, err
	}
	if total != 1 {
		return nil, fmt.Errorf("failed to find address %s", addressId)
	}
	for _, ip := range address[0].AddressSet {
		ipnet, err := parseCIDR(ip)
		if err != nil {
			return nil, nil
		}
		rule.IPNet = ipnet
		result = append(result, rule)
	}
	return result, nil
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	secgroup, err := self.region.GetSecurityGroupDetails(self.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(secgroup.SecurityGroupPolicySet.Egress); i++ {
		secgroup.SecurityGroupPolicySet.Egress[i].direction = "out"
	}
	for i := 0; i < len(secgroup.SecurityGroupPolicySet.Ingress); i++ {
		secgroup.SecurityGroupPolicySet.Ingress[i].direction = "in"
	}
	originRules := []SecurityGroupPolicy{}
	originRules = append(originRules, secgroup.SecurityGroupPolicySet.Egress...)
	originRules = append(originRules, secgroup.SecurityGroupPolicySet.Ingress...)
	for i := 0; i < len(originRules); i++ {
		originRules[i].region = self.region
	}
	sort.Sort(SecurityGroupRuleSet(originRules))
	rules := []secrules.SecurityRule{}
	priority := 100
	for _, rule := range originRules {
		subRules := rule.toRules()
		for i := 0; i < len(subRules); i++ {
			subRules[i].Priority = priority
		}
		if len(subRules) > 0 {
			priority--
		}
		rules = append(rules, subRules...)
	}
	// 腾讯云若出方向规则默认是拒绝所有流量
	defaultDenyRule, err := secrules.ParseSecurityRule("out:deny any")
	if err != nil {
		return nil, err
	}
	defaultDenyRule.Priority = 1
	rules = append(rules, *defaultDenyRule)
	return rules, nil
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.region.GetSecurityGroupDetails(self.SecurityGroupId); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	_, err := self.region.syncSecgroupRules(self.SecurityGroupId, rules)
	return err
}

func (self *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	if len(secgroupId) > 0 {
		_, err := self.GetSecurityGroupDetails(secgroupId)
		if err != nil {
			if err != cloudprovider.ErrNotFound {
				return "", err
			}
			secgroupId = ""
		}
	}
	if len(secgroupId) == 0 {
		secgroup, err := self.CreateSecurityGroup(name, desc)
		if err != nil {
			return "", err
		}
		secgroupId = secgroup.SecurityGroupId
	}
	return self.syncSecgroupRules(secgroupId, rules)
}

func (self *SRegion) deleteAllRules(secgroupid string) error {
	params := map[string]string{"SecurityGroupId": secgroupid, "SecurityGroupPolicySet.Version": "0"}
	_, err := self.vpcRequest("ModifySecurityGroupPolicies", params)
	return err
}

func (self *SRegion) addRule(secgroupId string, policyIndex int, rule *secrules.SecurityRule) error {
	params := map[string]string{}
	params["SecurityGroupId"] = secgroupId
	direction := "Egress"
	action := "accept"
	if rule.Action == secrules.SecurityRuleDeny {
		action = "drop"
	}
	protocol := "ALL"
	if rule.Protocol != secrules.PROTO_ANY {
		protocol = rule.Protocol
	}
	if rule.Direction == secrules.DIR_IN {
		direction = "Ingress"
	}
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.PolicyIndex", direction)] = fmt.Sprintf("%d", policyIndex)
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.Action", direction)] = action
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.PolicyDescription", direction)] = rule.Description
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.Protocol", direction)] = protocol
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.CidrBlock", direction)] = rule.IPNet.String()
	if rule.Protocol == secrules.PROTO_TCP || rule.Protocol == secrules.PROTO_UDP {
		port := "ALL"
		if rule.PortEnd > 0 && rule.PortStart > 0 {
			if rule.PortStart == rule.PortEnd {
				port = fmt.Sprintf("%d", rule.PortStart)
			} else {
				port = fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd)
			}
		} else if len(rule.Ports) > 0 {
			ports := []string{}
			for _, _port := range rule.Ports {
				ports = append(ports, fmt.Sprintf("%d", _port))
			}
			port = strings.Join(ports, ",")
		}
		params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.Port", direction)] = port
	}
	_, err := self.vpcRequest("CreateSecurityGroupPolicies", params)
	if err != nil {
		log.Errorf("Create SecurityGroup rule %s error: %v", rule, err)
		return err
	}
	return nil
}

func (self *SRegion) syncSecgroupRules(secgroupid string, rules []secrules.SecurityRule) (string, error) {
	if err := self.deleteAllRules(secgroupid); err != nil {
		return "", err
	}
	egressIndex, ingressIndex := -1, -1
	for _, rule := range rules {
		policyIndex := 0
		switch rule.Direction {
		case secrules.DIR_IN:
			ingressIndex++
			policyIndex = ingressIndex
		case secrules.DIR_OUT:
			egressIndex++
			policyIndex = egressIndex
		default:
			return "", fmt.Errorf("Unknown rule direction %v for secgroup %s", rule, secgroupid)
		}

		//为什么不一次创建完成?
		//答: 因为如果只有入方向安全组规则，创建时会提示缺少出方向规则。
		//为什么不分两次，一次创建入方向规则，一次创建出方向规则?
		//答: 因为这样就不能设置优先级了，一次性创建的出或入方向的优先级必须一样。
		err := self.addRule(secgroupid, policyIndex, &rule)
		if err != nil {
			return "", err
		}
	}

	// 需要在云上加上优先级最低的 allow any 规则, 和本地语义保持一致
	egressIndex++
	rule, err := secrules.ParseSecurityRule("out:allow any")
	if err != nil {
		return "", err
	}
	err = self.addRule(secgroupid, egressIndex, rule)
	if err != nil {
		return "", err
	}
	return secgroupid, nil
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["SecurityGroupId"] = secGroupId

	body, err := self.vpcRequest("DescribeSecurityGroupPolicies", params)
	if err != nil {
		log.Errorf("DescribeSecurityGroupAttribute fail %s", err)
		return nil, err
	}

	secgrp := SSecurityGroup{SecurityGroupId: secGroupId}
	err = body.Unmarshal(&secgrp.SecurityGroupPolicySet, "SecurityGroupPolicySet")
	if err != nil {
		log.Errorf("Unmarshal security group details fail %s", err)
		return nil, err
	}
	return &secgrp, nil
}

func (self *SRegion) DeleteSecurityGroup(secGroupId string) error {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["SecurityGroupId"] = secGroupId
	_, err := self.vpcRequest("DeleteSecurityGroup", params)
	return err
}

type AddressTemplate struct {
	AddressSet          []string
	AddressTemplateId   string
	AddressTemplateName string
	CreatedTime         time.Time
}

func (self *SRegion) AddressList(addressId, addressName string, offset, limit int) ([]AddressTemplate, int, error) {
	params := map[string]string{}
	filter := 0
	if len(addressId) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "address-template-id"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = addressId
		filter++
	}
	if len(addressName) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "address-template-name"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = addressName
		filter++
	}
	params["Offset"] = fmt.Sprintf("%d", offset)
	if limit == 0 {
		limit = 20
	}
	params["Limit"] = fmt.Sprintf("%d", limit)
	body, err := self.vpcRequest("DescribeAddressTemplates", params)
	if err != nil {
		return nil, 0, err
	}
	addressTemplates := []AddressTemplate{}
	err = body.Unmarshal(&addressTemplates, "AddressTemplateSet")
	if err != nil {
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return addressTemplates, int(total), nil
}

type AddressTemplateGroup struct {
	AddressTemplateIdSet     []string
	AddressTemplateGroupName string
	AddressTemplateGroupId   string
	CreatedTime              time.Time
}

func (self *SRegion) AddressGroupList(groupId, groupName string, offset, limit int) ([]AddressTemplateGroup, int, error) {
	params := map[string]string{}
	filter := 0
	if len(groupId) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "address-template-group-id"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = groupId
		filter++
	}
	if len(groupName) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "address-template-group-name"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = groupName
		filter++
	}
	params["Offset"] = fmt.Sprintf("%d", offset)
	if limit == 0 {
		limit = 20
	}
	params["Limit"] = fmt.Sprintf("%d", limit)
	body, err := self.vpcRequest("DescribeAddressTemplateGroups", params)
	if err != nil {
		return nil, 0, err
	}
	addressTemplateGroups := []AddressTemplateGroup{}
	err = body.Unmarshal(&addressTemplateGroups, "AddressTemplateGroupSet")
	if err != nil {
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return addressTemplateGroups, int(total), nil
}

func (self *SRegion) CreateSecurityGroup(name, description string) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["GroupName"] = name
	params["GroupDescription"] = description
	if len(description) == 0 {
		params["GroupDescription"] = "Customize Create"
	}
	secgroup := SSecurityGroup{region: self}
	if body, err := self.vpcRequest("CreateSecurityGroup", params); err != nil {
		return nil, err
	} else if err := body.Unmarshal(&secgroup, "SecurityGroup"); err != nil {
		return nil, err
	}
	return &secgroup, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.SecurityGroupId)
}
