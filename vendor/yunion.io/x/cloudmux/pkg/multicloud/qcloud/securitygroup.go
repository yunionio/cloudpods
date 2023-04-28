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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SecurityGroupPolicy struct {
	PolicyIndex       int                          // 安全组规则索引号。
	Protocol          string                       // 协议, 取值: TCP,UDP, ICMP。
	Port              string                       // 端口(all, 离散port, range)。
	ServiceTemplate   ServiceTemplateSpecification // 协议端口ID或者协议端口组ID。ServiceTemplate和Protocol+Port互斥。
	CidrBlock         string                       // 网段或IP(互斥)。
	SecurityGroupId   string                       // 已绑定安全组的网段或IP。
	AddressTemplate   AddressTemplateSpecification // IP地址ID或者ID地址组ID。
	Action            string                       // ACCEPT 或 DROP。
	PolicyDescription string                       // 安全组规则描述。
}

func (self *SecurityGroupPolicy) IsNeedSkip() bool {
	if len(self.SecurityGroupId) > 0 || len(self.AddressTemplate.AddressGroupId) > 0 || len(self.AddressTemplate.AddressId) > 0 {
		return true
	}
	if len(self.ServiceTemplate.ServiceId) > 0 || len(self.ServiceTemplate.ServiceGroupId) > 0 {
		return true
	}
	return false
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
	QcloudTags
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

func (self *SecurityGroupPolicy) toRule(direction secrules.TSecurityRuleDirection) cloudprovider.SecurityRule {
	rule := cloudprovider.SecurityRule{
		SecurityRule: secrules.SecurityRule{
			Action:      secrules.SecurityRuleAllow,
			Protocol:    secrules.PROTO_ANY,
			Direction:   direction,
			Priority:    100000 - self.PolicyIndex,
			Description: self.PolicyDescription,
			Ports:       []int{},
			PortStart:   -1,
			PortEnd:     -1,
		},
	}
	if strings.ToLower(self.Action) == "drop" {
		rule.Action = secrules.SecurityRuleDeny
	}
	if utils.IsInStringArray(strings.ToLower(self.Protocol), []string{"tcp", "udp", "icmp"}) {
		rule.Protocol = strings.ToLower(self.Protocol)
	}
	if strings.ToLower(self.Port) != "all" {
		rule.ParsePorts(self.Port)
	}
	rule.ParseCIDR(self.CidrBlock)
	return rule
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	policySet, err := self.region.DescribeSecurityGroupPolicies(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupPolicies")
	}
	var toRules = func(policy []SecurityGroupPolicy, direction secrules.TSecurityRuleDirection) []cloudprovider.SecurityRule {
		ret := []cloudprovider.SecurityRule{}
		for i := 0; i < len(policy); i++ {
			rule := policy[i]
			if rule.IsNeedSkip() {
				continue
			}
			subRule := rule.toRule(direction)
			ret = append(ret, subRule)
		}
		return ret
	}
	rules := []cloudprovider.SecurityRule{}
	rules = append(rules, toRules(policySet.Egress, secrules.DIR_OUT)...)
	rules = append(rules, toRules(policySet.Ingress, secrules.DIR_IN)...)
	return rules, nil
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.SecurityGroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, _, err := self.GetSecurityGroups([]string{id}, "", "", 0, 1)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].SecurityGroupId == id {
			groups[i].region = self
			return &groups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) addSecgroupRules(secgroupId string, in, out []cloudprovider.SecurityRule) error {
	if len(in)+len(out) == 0 {
		return nil
	}
	params := map[string]string{
		"SecurityGroupId": secgroupId,
		"SortPolicys":     "True",
	}
	var addParams = func(direction secrules.TSecurityRuleDirection, rules []cloudprovider.SecurityRule) {
		for idx := 0; idx < len(rules); idx++ {
			params = convertSecgroupRule(idx, rules[len(rules)-1-idx], params)
		}
	}
	addParams(secrules.DIR_IN, in)
	addParams(secrules.DIR_OUT, out)
	_, err := self.vpcRequest("ModifySecurityGroupPolicies", params)
	if err != nil {
		return errors.Wrapf(err, "ModifySecurityGroupPolicies")
	}
	return nil
}

func convertSecgroupRule(policyIndex int, rule cloudprovider.SecurityRule, params map[string]string) map[string]string {
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
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.%d.Action", direction, policyIndex)] = action
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.%d.PolicyDescription", direction, policyIndex)] = rule.Description
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.%d.Protocol", direction, policyIndex)] = protocol
	params[fmt.Sprintf("SecurityGroupPolicySet.%s.%d.CidrBlock", direction, policyIndex)] = rule.IPNet.String()
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
		params[fmt.Sprintf("SecurityGroupPolicySet.%s.%d.Port", direction, policyIndex)] = port
	}
	return params
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

func (self *SRegion) DeleteSecgroupRule(secId string, direction string, index int) error {
	params := map[string]string{
		"SecurityGroupId": secId,
	}
	switch direction {
	case secrules.DIR_IN:
		params["SecurityGroupPolicySet.Ingress.0.PolicyIndex"] = fmt.Sprintf("%d", index)
	case secrules.DIR_OUT:
		params["SecurityGroupPolicySet.Egress.0.PolicyIndex"] = fmt.Sprintf("%d", index)
	}
	_, err := self.vpcRequest("DeleteSecurityGroupPolicies", params)
	return errors.Wrapf(err, "DeleteSecurityGroupPolicies")
}

func (self *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["GroupName"] = opts.Name
	params["GroupDescription"] = opts.Desc
	if len(opts.ProjectId) > 0 {
		params["ProjectId"] = opts.ProjectId
	}

	if len(opts.Desc) == 0 {
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
	if opts.OnCreated != nil {
		opts.OnCreated(secgroup.SecurityGroupId)
	}
	return &secgroup, self.addSecgroupRules(secgroup.SecurityGroupId, opts.InRules, opts.OutRules)
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.ProjectId
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.SecurityGroupId)
}
