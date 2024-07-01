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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

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

func (self *SRegion) GetSecurityGroups(ids []string, name string, offset int, limit int) ([]SSecurityGroup, int, error) {
	if limit > 100 || limit <= 0 {
		limit = 100
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
	return ""
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

func (self *SSecurityGroup) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("cvm", "sg", []string{self.SecurityGroupId}, tags, replace)
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroupRules")
	}
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range rules {
		rules[i].secgroup = self
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.SecurityGroupId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, _, err := self.GetSecurityGroups([]string{id}, "", 0, 1)
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

func (self *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["GroupName"] = opts.Name
	params["GroupDescription"] = opts.Desc
	if len(opts.ProjectId) > 0 {
		params["ProjectId"] = opts.ProjectId
	}

	idx := 0
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.%d.Value", idx)] = v
		idx++
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

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroupRules")
	}
	maxPriority := 0
	for i := range rules {
		if rules[i].GetDirection() == opts.Direction && rules[i].PolicyIndex > maxPriority {
			maxPriority = rules[i].PolicyIndex
		}
	}
	if opts.Priority > maxPriority {
		opts.Priority = maxPriority
	}
	err = self.region.CreateSecurityGroupRule(self.SecurityGroupId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSecurityGroupRule")
	}
	rules, err = self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroupRules")
	}
	for i := range rules {
		if rules[i].Direction == opts.Direction && rules[i].GetPriority() == opts.Priority {
			rules[i].secgroup = self
			return &rules[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SRegion) CreateSecurityGroupRule(groupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	prefix := "SecurityGroupPolicySet.Egress.0."
	if opts.Direction == secrules.DIR_IN {
		prefix = "SecurityGroupPolicySet.Ingress.0."
	}
	if opts.Protocol == secrules.PROTO_ANY {
		opts.Protocol = "all"
		opts.Ports = "all"
	}
	if len(opts.Ports) == 0 {
		opts.Ports = "all"
	}
	action := "accept"
	if opts.Action == secrules.SecurityRuleDeny {
		action = "drop"
	}
	if len(opts.CIDR) == 0 {
		opts.CIDR = "0.0.0.0/0"
	}
	params := map[string]string{
		"SecurityGroupId":            groupId,
		prefix + "PolicyIndex":       fmt.Sprintf("%d", opts.Priority),
		prefix + "Protocol":          strings.ToUpper(opts.Protocol),
		prefix + "PolicyDescription": opts.Desc,
		prefix + "Action":            action,
		prefix + "Port":              opts.Ports,
		prefix + "CidrBlock":         opts.CIDR,
	}
	if _, err := netutils.NewIPV6Prefix(opts.CIDR); err == nil {
		params[prefix+"Ipv6CidrBlock"] = opts.CIDR
		delete(params, prefix+"CidrBlock")
	}

	_, err := self.vpcRequest("CreateSecurityGroupPolicies", params)
	if err != nil {
		return errors.Wrapf(err, "CreateSecurityGroupPolicies")
	}
	return nil
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.ProjectId
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.SecurityGroupId)
}
