// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	VolcEngineTags

	region            *SRegion
	Description       string
	SecurityGroupId   string
	SecurityGroupName string
	VpcId             string
	CreationTime      time.Time
	UpdateTime        time.Time
	Type              string
	ProjectName       string
	ServiceManaged    bool
	Status            string
}

func (region *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (string, error) {
	params := make(map[string]string)
	params["VpcId"] = opts.VpcId
	params["SecurityGroupName"] = opts.Name

	if len(opts.ProjectId) > 0 {
		params["ProjectName"] = opts.ProjectId
	}

	if len(opts.Desc) > 0 {
		params["Description"] = opts.Desc
	}
	params["ClientToken"] = utils.GenRequestId(20)

	idx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.%d.Value", idx)] = v
		idx++
	}

	body, err := region.vpcRequest("CreateSecurityGroup", params)
	if err != nil {
		return "", errors.Wrap(err, "CreateSecurityGroup")
	}
	return body.GetString("SecurityGroupId")
}

func (region *SRegion) GetSecurityGroup(secGroupId string) (*SSecurityGroup, error) {
	secgroups, _, err := region.GetSecurityGroups("", "", []string{secGroupId}, 1, 1)
	if err != nil {
		return nil, err
	}
	for i := range secgroups {
		secgroups[i].region = region
		if secgroups[i].SecurityGroupId == secGroupId {
			return &secgroups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, secGroupId)
}

func (region *SRegion) GetSecurityGroupRules(secGroupId string) ([]SSecurityGroupRule, error) {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGroupId
	body, err := region.vpcRequest("DescribeSecurityGroupAttributes", params)
	if err != nil {
		return nil, err
	}
	ret := []SSecurityGroupRule{}
	err = body.Unmarshal(&ret, "Permissions")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal security group details fail")
	}
	return ret, nil
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	err := self.region.CreateSecurityGroupRule(self.SecurityGroupId, opts)
	if err != nil {
		return nil, err
	}
	rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].secgroup = self
		if rules[i].GetPriority() == opts.Priority &&
			rules[i].GetAction() == opts.Action &&
			rules[i].GetProtocol() == opts.Protocol &&
			rules[i].GetPorts() == opts.Ports &&
			strings.Join(rules[i].GetCIDRs(), ",") == opts.CIDR &&
			rules[i].GetDirection() == opts.Direction {
			return &rules[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (region *SRegion) CreateSecurityGroupRule(secGrpId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["SecurityGroupId"] = secGrpId
	params["Description"] = opts.Desc
	params["PortStart"] = "-1"
	params["PortEnd"] = "-1"
	if len(opts.Ports) > 0 {
		params["PortStart"] = opts.Ports
		params["PortEnd"] = opts.Ports
		if strings.Contains(opts.Ports, "-") {
			info := strings.Split(opts.Ports, "-")
			if len(info) == 2 {
				params["PortStart"] = info[0]
				params["PortEnd"] = info[1]
			}
		}
	}
	protocol := opts.Protocol
	if len(opts.Protocol) == 0 || opts.Protocol == secrules.PROTO_ANY {
		protocol = "all"
	}
	params["Protocol"] = protocol
	params["Policy"] = "drop"
	if opts.Action == secrules.SecurityRuleAllow {
		params["Policy"] = "accept"
	}
	params["CidrIp"] = "0.0.0.0/0"
	if len(opts.CIDR) > 0 {
		params["CidrIp"] = opts.CIDR
	}

	params["Priority"] = fmt.Sprintf("%d", opts.Priority)
	action := "AuthorizeSecurityGroupIngress"
	if opts.Direction == secrules.DIR_OUT {
		action = "AuthorizeSecurityGroupEgress"
	}
	_, err := region.vpcRequest(action, params)
	return err
}

func (region *SRegion) GetSecurityGroups(vpcId, name string, securityGroupIds []string, pageSize int, pageNumber int) ([]SSecurityGroup, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(name) > 0 {
		params["SecurityGroupName"] = name
	}

	for i, id := range securityGroupIds {
		params[fmt.Sprintf("SecurityGroupIds.%d", i+1)] = id
	}

	body, err := region.vpcRequest("DescribeSecurityGroups", params)
	if err != nil {
		log.Errorf("GetSecurityGroups fail %s", err)
		return nil, 0, err
	}

	secgrps := make([]SSecurityGroup, 0)
	err = body.Unmarshal(&secgrps, "SecurityGroups")
	if err != nil {
		log.Errorf("Unmarshal security groups fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return secgrps, int(total), nil
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.SecurityGroupId
}

func (secgroup *SSecurityGroup) GetName() string {
	return secgroup.SecurityGroupName
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.GetId()
}

func (secgroup *SSecurityGroup) GetCreatedAt() time.Time {
	return secgroup.CreationTime
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return secgroup.Description
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return secgroup.Status
}

func (secgroup *SSecurityGroup) Refresh() error {
	group, err := secgroup.region.GetSecurityGroup(secgroup.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(secgroup, group)
}

func (secgroup *SSecurityGroup) GetProjectId() string {
	return secgroup.ProjectName
}

func (self *SRegion) vpcTagResources(resType string, id string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	params := map[string]string{
		"ResourceType":  resType,
		"ResourceIds.1": id,
	}
	idx := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tags.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.%d.Value", idx)] = v
		idx++
	}
	_, err := self.vpcRequest("TagResources", params)
	return err
}

func (self *SRegion) vpcUntagResources(resType string, id string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	params := map[string]string{
		"ResourceType":  resType,
		"ResourceIds.1": id,
	}
	idx := 1
	for k := range tags {
		params[fmt.Sprintf("TagKeys.%d", idx)] = k
		idx++
	}
	_, err := self.vpcRequest("UntagResources", params)
	return err
}

func (secgroup *SSecurityGroup) SetTags(tags map[string]string, replace bool) error {
	oldTags, err := secgroup.GetTags()
	if err != nil {
		return errors.Wrapf(err, "GetTags")
	}
	added, removed := map[string]string{}, map[string]string{}
	for k, v := range tags {
		oldValue, ok := oldTags[k]
		if !ok {
			added[k] = v
		} else if oldValue != v {
			removed[k] = oldValue
			added[k] = v
		}
	}
	if replace {
		for k, v := range oldTags {
			newValue, ok := tags[k]
			if !ok {
				removed[k] = v
			} else if v != newValue {
				added[k] = newValue
				removed[k] = v
			}
		}
	}
	if len(removed) > 0 {
		err = secgroup.region.vpcUntagResources("securitygroup", secgroup.SecurityGroupId, removed)
		if err != nil {
			return errors.Wrapf(err, "DeleteTags %s", removed)
		}
	}
	if len(added) > 0 {
		return secgroup.region.vpcTagResources("securitygroup", secgroup.SecurityGroupId, added)
	}
	return nil
}

func (secgroup *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	rules, err := secgroup.region.GetSecurityGroupRules(secgroup.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range rules {
		rules[i].secgroup = secgroup
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return secgroup.VpcId
}

func (secgroup *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	ret := []cloudprovider.SecurityGroupReference{}
	return ret, errors.Wrapf(errors.ErrNotImplemented, "GetReferences not supported")
}

func (region *SRegion) DeleteSecurityGroup(id string) error {
	params := make(map[string]string)
	params["SecurityGroupId"] = id
	_, err := region.vpcRequest("DeleteSecurityGroup", params)
	if err != nil {
		return errors.Wrapf(err, "Delete %s", id)
	}
	return nil
}

func (secgroup *SSecurityGroup) Delete() error {
	return secgroup.region.DeleteSecurityGroup(secgroup.SecurityGroupId)
}
