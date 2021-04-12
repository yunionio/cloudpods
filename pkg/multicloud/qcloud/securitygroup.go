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
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	multicloud.SSecurityGroup
	region                 *SRegion
	SecurityGroupId        string    //		安全组实例ID，例如：sg-ohuuioma。
	SecurityGroupName      string    //		安全组名称，可任意命名，但不得超过60个字符。
	SecurityGroupDesc      string    //		安全组备注，最多100个字符。
	ProjectId              string    //		项目id，默认0。可在qcloud控制台项目管理页面查询到。
	IsDefault              bool      // 	是否是默认安全组，默认安全组不支持删除。
	CreatedTime            time.Time // 	安全组创建时间。
	SecurityGroupPolicySet SecurityGroupPolicySet
}

func (self *SRegion) GetSecurityGroups(ids []string, vpcId string, name string, offset int, limit int) ([]SSecurityGroup, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	if len(name) > 0 {
		params["Filters.0.Name"] = "security-group-name"
		params["Filters.0.Values.0"] = name
	}

	for idx, id := range ids {
		params[fmt.Sprintf("SecurityGroupIds.%d", idx)] = id
	}

	resp, err := self.vpcRequest("DescribeSecurityGroups", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeSecurityGroups")
	}

	secgrps := make([]SSecurityGroup, 0)
	err = resp.Unmarshal(&secgrps, "SecurityGroupSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalCount")
	return secgrps, int(total), nil
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

type ReferredSecurityGroup struct {
	SecurityGroupId          string
	ReferredSecurityGroupIds []string
}

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	references, err := self.region.DescribeSecurityGroupReferences(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupReferences")
	}
	ret := []cloudprovider.SecurityGroupReference{}
	for _, refer := range references {
		if refer.SecurityGroupId == self.SecurityGroupId {
			for _, id := range refer.ReferredSecurityGroupIds {
				ret = append(ret, cloudprovider.SecurityGroupReference{
					Id: id,
				})
			}
		}
	}
	return ret, nil
}

func (self *SRegion) DescribeSecurityGroupReferences(id string) ([]ReferredSecurityGroup, error) {
	params := map[string]string{
		"Region":             self.Region,
		"SecurityGroupIds.0": id,
	}
	resp, err := self.vpcRequest("DescribeSecurityGroupReferences", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupReferences")
	}
	ret := []ReferredSecurityGroup{}
	err = resp.Unmarshal(&ret, "ReferredSecurityGroupSet")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SecurityGroupPolicy) toRules() []cloudprovider.SecurityRule {
	result := []cloudprovider.SecurityRule{}
	rule := cloudprovider.SecurityRule{
		ExternalId: fmt.Sprintf("%d", self.PolicyIndex),
		SecurityRule: secrules.SecurityRule{
			Action:    secrules.SecurityRuleAllow,
			Protocol:  secrules.PROTO_ANY,
			Direction: secrules.TSecurityRuleDirection(self.direction),
			Priority:  self.PolicyIndex,
			Ports:     []int{},
			PortStart: -1,
			PortEnd:   -1,
		},
	}
	if len(self.SecurityGroupId) != 0 {
		rule.ParseCIDR("0.0.0.0/0")
		rule.PeerSecgroupId = self.SecurityGroupId
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
	} else if len(self.SecurityGroupId) > 0 {
		rule.PeerSecgroupId = self.SecurityGroupId
		result = append(result, rule)
	} else if len(self.CidrBlock) > 0 {
		rule.ParseCIDR(self.CidrBlock)
		result = append(result, rule)
	}
	return result
}

func (self *SecurityGroupPolicy) getAddressRules(rule cloudprovider.SecurityRule, addressId string) ([]cloudprovider.SecurityRule, error) {
	result := []cloudprovider.SecurityRule{}
	address, total, err := self.region.AddressList(addressId, "", 0, 1)
	if err != nil {
		log.Errorf("Get AddressList %s failed %v", self.AddressTemplate.AddressId, err)
		return nil, err
	}
	if total != 1 {
		return nil, fmt.Errorf("failed to find address %s", addressId)
	}
	for _, ip := range address[0].AddressSet {
		rule.ParseCIDR(ip)
		result = append(result, rule)
	}
	return result, nil
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	policySet, err := self.region.DescribeSecurityGroupPolicies(self.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(policySet.Egress); i++ {
		policySet.Egress[i].direction = "out"
	}
	for i := 0; i < len(policySet.Ingress); i++ {
		policySet.Ingress[i].direction = "in"
	}
	originRules := []SecurityGroupPolicy{}
	originRules = append(originRules, policySet.Egress...)
	originRules = append(originRules, policySet.Ingress...)
	for i := 0; i < len(originRules); i++ {
		originRules[i].region = self.region
	}
	rules := []cloudprovider.SecurityRule{}
	for _, rule := range originRules {
		subRules := rule.toRules()
		rules = append(rules, subRules...)
	}
	return rules, nil
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	groups, total, err := self.region.GetSecurityGroups([]string{self.SecurityGroupId}, "", "", 0, 0)
	if err != nil {
		return err
	}
	if total < 1 {
		return cloudprovider.ErrNotFound
	}
	return jsonutils.Update(self, groups[0])
}

func (self *SSecurityGroup) deleteRules(rules []cloudprovider.SecurityRule, direction string) error {
	ids := []string{}
	for _, r := range rules {
		ids = append(ids, r.ExternalId)
	}
	if len(ids) > 0 {
		err := self.region.DeleteRules(self.SecurityGroupId, direction, ids)
		if err != nil {
			return errors.Wrapf(err, "deleteRules(%s)", ids)
		}
	}
	return nil
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	rules := append(common, append(inAdds, outAdds...)...)
	sort.Sort(cloudprovider.SecurityRuleSet(rules))
	return self.region.syncSecgroupRules(self.SecurityGroupId, rules)
}

func (self *SRegion) syncSecgroupRules(secgroupid string, rules []cloudprovider.SecurityRule) error {
	err := self.deleteAllRules(secgroupid)
	if err != nil {
		return errors.Wrap(err, "deleteAllRules")
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
			return fmt.Errorf("Unknown rule direction %v for secgroup %s", rule, secgroupid)
		}

		//为什么不一次创建完成?
		//答: 因为如果只有入方向安全组规则，创建时会提示缺少出方向规则。
		//为什么不分两次，一次创建入方向规则，一次创建出方向规则?
		//答: 因为这样就不能设置优先级了，一次性创建的出或入方向的优先级必须一样。
		err := self.AddRule(secgroupid, policyIndex, rule)
		if err != nil {
			return errors.Wrap(err, "AddRule")
		}
	}
	return nil
}

func (self *SRegion) deleteAllRules(secgroupid string) error {
	params := map[string]string{"SecurityGroupId": secgroupid, "SecurityGroupPolicySet.Version": "0"}
	_, err := self.vpcRequest("ModifySecurityGroupPolicies", params)
	return err
}

func (self *SRegion) DeleteRules(secgroupId, direction string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	params := map[string]string{"SecurityGroupId": secgroupId}
	for idx, id := range ids {
		params[fmt.Sprintf("SecurityGroupPolicySet.%s.%d.PolicyIndex", direction, idx)] = id
	}
	_, err := self.vpcRequest("DeleteSecurityGroupPolicies", params)
	return err
}

func (self *SRegion) AddRule(secgroupId string, policyIndex int, rule cloudprovider.SecurityRule) error {
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
	if len(rule.PeerSecgroupId) > 0 {
		params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.SecurityGroupId", direction)] = rule.PeerSecgroupId
	} else {
		params[fmt.Sprintf("SecurityGroupPolicySet.%s.0.CidrBlock", direction)] = rule.IPNet.String()
	}
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

func (self *SRegion) DescribeSecurityGroupPolicies(secGroupId string) (*SecurityGroupPolicySet, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["SecurityGroupId"] = secGroupId

	body, err := self.vpcRequest("DescribeSecurityGroupPolicies", params)
	if err != nil {
		log.Errorf("DescribeSecurityGroupAttribute fail %s", err)
		return nil, err
	}

	policies := SecurityGroupPolicySet{}
	err = body.Unmarshal(&policies, "SecurityGroupPolicySet")
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal")
	}
	return &policies, nil
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

func (self *SRegion) CreateSecurityGroup(name, projectId, description string) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["GroupName"] = name
	params["GroupDescription"] = description
	if len(projectId) > 0 {
		params["ProjectId"] = projectId
	}

	if len(description) == 0 {
		params["GroupDescription"] = "Customize Create"
	}
	secgroup := SSecurityGroup{region: self}
	body, err := self.vpcRequest("CreateSecurityGroup", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSecurityGroup")
	}
	err = body.Unmarshal(&secgroup, "SecurityGroup")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}
	return &secgroup, nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.ProjectId
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.SecurityGroupId)
}
