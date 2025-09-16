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

package ucloud

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://docs.ucloud.cn/api/unet-api/describe_firewall
type SSecurityGroup struct {
	multicloud.SSecurityGroup
	UcloudTags
	region *SRegion

	CreateTime    int64               `json:"CreateTime"`
	FWID          string              `json:"FWId"`
	GroupID       string              `json:"GroupId"`
	Name          string              `json:"Name"`
	Remark        string              `json:"Remark"`
	ResourceCount int                 `json:"ResourceCount"`
	Rule          []SecurityGroupRule `json:"Rule"`
	Tag           string              `json:"Tag"`
	Type          string              `json:"Type"`
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.region.client.projectId
}

func (self *SSecurityGroup) GetId() string {
	return self.FWID
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	group, err := self.region.GetSecurityGroup(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, group)
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Remark
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range self.Rule {
		self.Rule[i].secgroup = self
		ret = append(ret, &self.Rule[i])
	}
	return ret, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
}

func (self *SRegion) GetSecurityGroup(secGroupId string) (*SSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups(secGroupId, "", "")
	if err != nil {
		return nil, err
	}
	for i := range secgroups {
		secgroups[i].region = self
		if secgroups[i].GetGlobalId() == secGroupId {
			return &secgroups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, secGroupId)
}

// https://docs.ucloud.cn/api/unet-api/create_firewall
func (self *SRegion) CreateSecurityGroup(name, description string) (string, error) {
	params := NewUcloudParams()
	params.Set("Name", name)
	params.Set("Remark", description)

	rules := []string{"TCP|22|0.0.0.0/0|ACCEPT|LOW", "TCP|3389|0.0.0.0/0|ACCEPT|LOW", "ICMP||0.0.0.0/0|ACCEPT|LOW"}
	for i, rule := range rules {
		params.Set(fmt.Sprintf("Rule.%d", i), rule)
	}

	type Firewall struct {
		FWId string
	}

	firewall := Firewall{}
	err := self.DoAction("CreateFirewall", params, &firewall)
	if err != nil {
		return "", err
	}

	return firewall.FWId, nil
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	params := NewUcloudParams()
	params.Set("FWId", self.FWID)
	idx := 0
	for _, rule := range self.Rule {
		params.Set(fmt.Sprintf("Rule.%d", idx), rule.String())
		idx++
	}
	if len(opts.Ports) == 0 {
		opts.Ports = "1-65535"
	}
	if opts.Protocol == secrules.PROTO_ICMP || opts.Protocol == "gre" {
		opts.Ports = ""
	}
	if opts.Protocol == secrules.PROTO_ANY {
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "protocol any")
	}
	action := "DROP"
	if opts.Action == secrules.SecurityRuleAllow {
		action = "ACCEPT"
	}
	priority := "LOW"
	if opts.Priority == 1 {
		priority = "HIGH"
	} else if opts.Priority == 2 {
		priority = "MEDIUM"
	}
	rule := fmt.Sprintf("%s|%s|%s|%s|%s|%s", strings.ToUpper(opts.Protocol), opts.Ports, opts.CIDR, action, priority, opts.Desc)
	params.Set(fmt.Sprintf("Rule.%d", idx), rule)
	err := self.region.DoAction("UpdateFirewall", params, nil)
	if err != nil {
		return nil, err
	}
	err = self.Refresh()
	if err != nil {
		return nil, err
	}
	for i := range self.Rule {
		if self.Rule[i].String() == rule {
			self.Rule[i].secgroup = self
			return &self.Rule[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SRegion) GetSecurityGroups(secGroupId string, resourceId string, name string) ([]SSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)

	params := NewUcloudParams()
	if len(secGroupId) > 0 {
		params.Set("FWId", secGroupId)
	}

	if len(resourceId) > 0 {
		params.Set("ResourceId", resourceId)
		params.Set("ResourceType", "uhost") //  默认只支持"uhost"，云主机
	}
	err := self.DoListAll("DescribeFirewall", params, &secgroups)
	if err != nil {
		if strings.Contains(err.Error(), "not exist") {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	}

	result := []SSecurityGroup{}

	for i := range secgroups {
		if len(name) == 0 || secgroups[i].Name == name {
			secgroups[i].region = self
			result = append(result, secgroups[i])
		}
	}

	return result, nil
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.FWID)
}
