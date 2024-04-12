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

package ksyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SecurityGroupPermissionNicType string

const (
	IntranetNicType SecurityGroupPermissionNicType = "intranet"
	InternetNicType SecurityGroupPermissionNicType = "internet"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	SKsTag
	region *SRegion

	CreateTime            string        `json:"CreateTime"`
	ProductTag            string        `json:"ProductTag"`
	SecurityGroupEntrySet []SPermission `json:"SecurityGroupEntrySet"`
	SecurityGroupID       string        `json:"SecurityGroupId"`
	SecurityGroupName     string        `json:"SecurityGroupName"`
	SecurityGroupType     string        `json:"SecurityGroupType"`
	UserTag               string        `json:"UserTag"`
	VpcID                 string        `json:"VpcId"`
}

type SecurityGroupEntrySet struct {
	CidrBlock            string `json:"CidrBlock"`
	CreateTime           string `json:"CreateTime"`
	Direction            string `json:"Direction"`
	IcmpCode             int    `json:"IcmpCode,omitempty"`
	IcmpType             int    `json:"IcmpType,omitempty"`
	Policy               string `json:"Policy"`
	Priority             int    `json:"Priority"`
	ProductTag           string `json:"ProductTag"`
	Protocol             string `json:"Protocol"`
	RuleTag              string `json:"RuleTag,omitempty"`
	SecurityGroupEntryID string `json:"SecurityGroupEntryId"`
	UserTag              string `json:"UserTag"`
	PortRangeFrom        int    `json:"PortRangeFrom,omitempty"`
	PortRangeTo          int    `json:"PortRangeTo,omitempty"`
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return secgroup.VpcID
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.SecurityGroupID
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.SecurityGroupID
}

func (secgroup *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := make([]cloudprovider.ISecurityGroupRule, 0)
	for i := range secgroup.SecurityGroupEntrySet {
		secgroup.SecurityGroupEntrySet[i].region = secgroup.region
		ret = append(ret, &secgroup.SecurityGroupEntrySet[i])
	}
	return ret, nil
}

func (secgroup *SSecurityGroup) GetName() string {
	if len(secgroup.SecurityGroupName) > 0 {
		return secgroup.SecurityGroupName
	}
	return secgroup.SecurityGroupID
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (secgroup *SSecurityGroup) Refresh() error {
	group, err := secgroup.region.GetSecurityGroup(secgroup.SecurityGroupID)
	if err != nil {
		return err
	}
	return jsonutils.Update(secgroup, group)
}

func (secgroup *SSecurityGroup) GetTags() (map[string]string, error) {
	tags, err := secgroup.region.ListTags("security-group", secgroup.SecurityGroupID)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (secgroup *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	references, err := secgroup.region.DescribeSecurityGroupReferences(secgroup.SecurityGroupID)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupReferences")
	}
	ret := []cloudprovider.SecurityGroupReference{}
	for _, reference := range references {
		if reference.SecurityGroupId == secgroup.SecurityGroupID {
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

func (region *SRegion) DescribeSecurityGroupReferences(id string) ([]SecurityGroupReferences, error) {
	return nil, errors.ErrNotImplemented
}

func (region *SRegion) GetSecurityGroups(vpcId string, securityGroupIds []string) ([]SSecurityGroup, error) {
	ret := []SSecurityGroup{}
	params := map[string]string{
		"MaxResults": "1000",
	}
	if len(vpcId) > 0 {
		params["Filter.1.Name"] = "vpc-id"
		params["Filter.1.Value.1"] = vpcId
	}
	for i, secgroupId := range securityGroupIds {
		params[fmt.Sprintf("SecurityGroupId.%d", i+1)] = secgroupId
	}

	for {
		secgroupResp := struct {
			RequestID        string           `json:"RequestId"`
			SecurityGroupSet []SSecurityGroup `json:"SecurityGroupSet"`
			NextToken        string           `json:"NextToken"`
		}{}
		resp, err := region.vpcRequest("DescribeSecurityGroups", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeSecurityGroups")
		}
		err = resp.Unmarshal(&secgroupResp)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal secgroups")
		}
		ret = append(ret, secgroupResp.SecurityGroupSet...)
		if len(secgroupResp.NextToken) == 0 {
			break
		}
		params["NextToken"] = secgroupResp.NextToken
	}

	return ret, nil
}

func (region *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, err := region.GetSecurityGroups("", []string{id})
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		return &group, nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "security_group id:%s", id)
}

func (region *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (string, error) {
	return "", errors.ErrNotImplemented
}

func (region *SRegion) SetSecurityGroups(secgroupIds []string, instanceId string) error {
	return errors.ErrNotImplemented
}

func (region *SRegion) DeleteSecurityGroup(secGrpId string) error {
	return errors.ErrNotImplemented
}

func (region *SSecurityGroup) GetProjectId() string {
	return ""
}

func (sg *SSecurityGroup) Delete() error {
	return errors.ErrNotImplemented
}
