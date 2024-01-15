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

package aliyun

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// {"CreationTime":"2017-03-19T13:37:48Z","Description":"System created security group.","SecurityGroupId":"sg-j6cannq0xxj2r9z0yxwl","SecurityGroupName":"sg-j6cannq0xxj2r9z0yxwl","Tags":{"Tag":[]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}
// {"Description":"System created security group.","InnerAccessPolicy":"Accept","Permissions":{"Permission":[{"CreateTime":"2017-03-19T13:37:54Z","Description":"","DestCidrIp":"","DestGroupId":"","DestGroupName":"","DestGroupOwnerAccount":"","Direction":"ingress","IpProtocol":"ALL","NicType":"intranet","Policy":"Accept","PortRange":"-1/-1","Priority":110,"SourceCidrIp":"0.0.0.0/0","SourceGroupId":"","SourceGroupName":"","SourceGroupOwnerAccount":""},{"CreateTime":"2017-03-19T13:37:55Z","Description":"","DestCidrIp":"0.0.0.0/0","DestGroupId":"","DestGroupName":"","DestGroupOwnerAccount":"","Direction":"egress","IpProtocol":"ALL","NicType":"intranet","Policy":"Accept","PortRange":"-1/-1","Priority":110,"SourceCidrIp":"","SourceGroupId":"","SourceGroupName":"","SourceGroupOwnerAccount":""}]},"RegionId":"cn-hongkong","RequestId":"FBFE0950-5F2D-40DE-8C3C-E5A62AE7F7DA","SecurityGroupId":"sg-j6cannq0xxj2r9z0yxwl","SecurityGroupName":"sg-j6cannq0xxj2r9z0yxwl","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}

type SecurityGroupPermissionNicType string

const (
	IntranetNicType SecurityGroupPermissionNicType = "intranet"
	InternetNicType SecurityGroupPermissionNicType = "internet"
)

type SPermissions struct {
	Permission []SPermission
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	AliyunTags

	region            *SRegion
	CreationTime      time.Time
	Description       string
	SecurityGroupId   string
	SecurityGroupName string
	VpcId             string
	InnerAccessPolicy string
	Permissions       SPermissions
	RegionId          string
	ResourceGroupId   string
}

func (self *SSecurityGroup) GetVpcId() string {
	return self.VpcId
}

func (self *SSecurityGroup) GetId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags(ALIYUN_SERVICE_ECS, "securitygroup", self.SecurityGroupId, tags, replace)
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := make([]cloudprovider.ISecurityGroupRule, 0)
	rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].region = self.region
		ret = append(ret, &rules[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) GetName() string {
	if len(self.SecurityGroupName) > 0 {
		return self.SecurityGroupName
	}
	return self.SecurityGroupId
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

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	references, err := self.region.DescribeSecurityGroupReferences(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupReferences")
	}
	ret := []cloudprovider.SecurityGroupReference{}
	for _, reference := range references {
		if reference.SecurityGroupId == self.SecurityGroupId {
			for _, sec := range reference.ReferencingSecurityGroups.ReferencingSecurityGroup {
				ret = append(ret, cloudprovider.SecurityGroupReference{
					Id: sec.SecurityGroupId,
				})
			}
		}
	}
	return ret, nil
}

type ReferencingSecurityGroup struct {
	AliUid          string
	SecurityGroupId string
}

type ReferencingSecurityGroups struct {
	ReferencingSecurityGroup []ReferencingSecurityGroup
}

type SecurityGroupReferences struct {
	SecurityGroupId           string
	ReferencingSecurityGroups ReferencingSecurityGroups
}

func (self *SRegion) DescribeSecurityGroupReferences(id string) ([]SecurityGroupReferences, error) {
	params := map[string]string{
		"RegionId":          self.RegionId,
		"SecurityGroupId.1": id,
	}
	resp, err := self.ecsRequest("DescribeSecurityGroupReferences", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupReferences")
	}
	ret := []SecurityGroupReferences{}
	err = resp.Unmarshal(&ret, "SecurityGroupReferences", "SecurityGroupReference")
	return ret, errors.Wrapf(err, "resp.Unmarshal")
}

func (self *SRegion) GetSecurityGroups(vpcId, name string, securityGroupIds []string) ([]SSecurityGroup, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["MaxResults"] = "100"
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(name) > 0 {
		params["SecurityGroupName"] = name
	}

	if len(securityGroupIds) > 0 {
		params["SecurityGroupIds"] = jsonutils.Marshal(securityGroupIds).String()
	}

	ret := []SSecurityGroup{}
	for {
		part := struct {
			SecurityGroups struct {
				SecurityGroup []SSecurityGroup
			}
			NextToken string
		}{}
		resp, err := self.ecsRequest("DescribeSecurityGroups", params)
		if err != nil {
			return nil, err
		}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SecurityGroups.SecurityGroup...)
		if len(part.NextToken) == 0 || len(part.SecurityGroups.SecurityGroup) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, err := self.GetSecurityGroups("", "", []string{id})
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

func (self *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (string, error) {
	params := make(map[string]string)
	params["VpcId"] = opts.VpcId
	params["SecurityGroupName"] = opts.Name
	params["Description"] = opts.Desc
	params["ClientToken"] = utils.GenRequestId(20)
	if len(opts.ProjectId) > 0 {
		params["ResourceGroupId"] = opts.ProjectId
	}

	tagIdx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tag.%d.Key", tagIdx)] = k
		params[fmt.Sprintf("Tag.%d.Value", tagIdx)] = v
		tagIdx += 1
	}

	body, err := self.ecsRequest("CreateSecurityGroup", params)
	if err != nil {
		return "", errors.Wrap(err, "CreateSecurityGroup")
	}
	return body.GetString("SecurityGroupId")
}

func (self *SRegion) modifySecurityGroup(secGrpId string, name string, desc string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["SecurityGroupName"] = name
	if len(desc) > 0 {
		params["Description"] = desc
	}
	_, err := self.ecsRequest("ModifySecurityGroupAttribute", params)
	return err
}

func (self *SRegion) CreateSecurityGroupRule(secGrpId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SecurityGroupId"] = secGrpId
	params["Permissions.1.NicType"] = string(IntranetNicType)
	params["Permissions.1.Description"] = opts.Desc
	params["Permissions.1.PortRange"] = "-1/-1"
	params["Permissions.1.IpProtocol"] = opts.Protocol
	if opts.Protocol == secrules.PROTO_ANY {
		params["Permissions.1.IpProtocol"] = "all"
	}
	if opts.Protocol == secrules.PROTO_TCP || opts.Protocol == secrules.PROTO_UDP {
		if len(opts.Ports) == 0 {
			params["Permissions.1.PortRange"] = "1/65535"
		} else {
			params["Permissions.1.PortRange"] = fmt.Sprintf("%s/%s", opts.Ports, opts.Ports)
			if strings.Contains(opts.Ports, "-") {
				params["Permissions.1.PortRange"] = strings.ReplaceAll(opts.Ports, "-", "/")
			}
		}
	}
	params["Permissions.1.Policy"] = "drop"
	if opts.Action == secrules.SecurityRuleAllow {
		params["Permissions.1.Policy"] = "accept"
	}

	action := "AuthorizeSecurityGroup"
	params["Permissions.1.Priority"] = fmt.Sprintf("%d", opts.Priority)
	if opts.Direction == secrules.SecurityRuleIngress {
		params["Permissions.1.SourceCidrIp"] = "0.0.0.0/0"
		if len(opts.CIDR) > 0 {
			params["Permissions.1.SourceCidrIp"] = opts.CIDR
		}
	} else {
		params["Permissions.1.DestCidrIp"] = "0.0.0.0/0"
		if len(opts.CIDR) > 0 {
			params["Permissions.1.DestCidrIp"] = opts.CIDR
		}
		action = "AuthorizeSecurityGroupEgress"
	}
	_, err := self.ecsRequest(action, params)
	return err
}

func (self *SRegion) SetSecurityGroups(secgroupIds []string, instanceId string) error {
	params := map[string]string{"InstanceId": instanceId}
	for _, secgroupId := range secgroupIds {
		params["SecurityGroupId"] = secgroupId
		if _, err := self.ecsRequest("JoinSecurityGroup", params); err != nil {
			return err
		}
	}
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return err
	}
	for _, _secgroupId := range instance.SecurityGroupIds.SecurityGroupId {
		if !utils.IsInStringArray(_secgroupId, secgroupIds) {
			if err := self.leaveSecurityGroup(_secgroupId, instanceId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SRegion) leaveSecurityGroup(secgroupId, instanceId string) error {
	params := map[string]string{"InstanceId": instanceId, "SecurityGroupId": secgroupId}
	_, err := self.ecsRequest("LeaveSecurityGroup", params)
	return err
}

func (self *SRegion) DeleteSecurityGroup(secGrpId string) error {
	params := make(map[string]string)
	params["SecurityGroupId"] = secGrpId

	_, err := self.ecsRequest("DeleteSecurityGroup", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteSecurityGroup")
	}
	return nil
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.SecurityGroupId)
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroupRules")
	}
	ruleIds := []string{}
	for _, r := range rules {
		ruleIds = append(ruleIds, r.SecurityGroupRuleId)
	}
	err = self.region.CreateSecurityGroupRule(self.SecurityGroupId, opts)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 3; i++ {
		rules, err := self.region.GetSecurityGroupRules(self.SecurityGroupId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecurityGroupRules")
		}
		for i := range rules {
			rule := rules[i]
			if !utils.IsInStringArray(rule.SecurityGroupRuleId, ruleIds) {
				rule.region = self.region
				return &rule, nil
			}
		}
		time.Sleep(time.Second * 3)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.ResourceGroupId
}
