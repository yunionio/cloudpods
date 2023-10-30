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

package azure

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type Interface struct {
	ID string
}

type SecurityGroupPropertiesFormat struct {
	SecurityRules        []SecurityRules `json:"securityRules,omitempty"`
	DefaultSecurityRules []SecurityRules `json:"defaultSecurityRules,omitempty"`
	NetworkInterfaces    *[]Interface    `json:"networkInterfaces,omitempty"`
	Subnets              *[]SNetwork     `json:"subnets,omitempty"`
}
type SSecurityGroup struct {
	multicloud.SSecurityGroup
	AzureTags

	region     *SRegion
	Properties *SecurityGroupPropertiesFormat `json:"properties,omitempty"`
	ID         string
	Name       string
	Location   string
	Type       string
}

func (self *SSecurityGroup) GetId() string {
	return self.ID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SSecurityGroup) GetDescription() string {
	return ""
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	rules := make([]cloudprovider.ISecurityGroupRule, 0)
	if self.Properties == nil || self.Properties.SecurityRules == nil {
		return rules, nil
	}
	for i := range self.Properties.SecurityRules {
		self.Properties.SecurityRules[i].region = self.region.getZone().region
		rules = append(rules, &self.Properties.SecurityRules[i])
	}
	return rules, nil
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
}

func (region *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: region}
	params := map[string]interface{}{
		"Name":     opts.Name,
		"Type":     "Microsoft.Network/networkSecurityGroups",
		"Location": region.Name,
	}

	err := region.create("", jsonutils.Marshal(params), secgroup)
	if err != nil {
		return nil, errors.Wrapf(err, "create")
	}
	return secgroup, nil
}

func (region *SRegion) ListSecgroups() ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	err := region.list("Microsoft.Network/networkSecurityGroups", url.Values{}, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return secgroups, nil
}

func (region *SRegion) GetSecurityGroupDetails(secgroupId string) (*SSecurityGroup, error) {
	secgroup := SSecurityGroup{region: region}
	return &secgroup, region.get(secgroupId, url.Values{}, &secgroup)
}

func (self *SSecurityGroup) Refresh() error {
	sec, err := self.region.GetSecurityGroupDetails(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sec)
}

func (region *SRegion) AttachSecurityToInterfaces(secgroupId string, nicIds []string) error {
	for _, nicId := range nicIds {
		nic, err := region.GetNetworkInterface(nicId)
		if err != nil {
			return err
		}
		nic.Properties.NetworkSecurityGroup = &SSecurityGroup{ID: secgroupId}
		if err := region.update(jsonutils.Marshal(nic), nil); err != nil {
			return err
		}
	}
	return nil
}

func (region *SRegion) SetSecurityGroup(instanceId, secgroupId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	nicIds := []string{}
	for _, nic := range instance.Properties.NetworkProfile.NetworkInterfaces {
		nicIds = append(nicIds, nic.ID)
	}
	return region.AttachSecurityToInterfaces(secgroupId, nicIds)
}

func (self *SSecurityGroup) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := self.region.CreateSecurityGroupRule(self.ID, opts)
	if err != nil {
		return nil, err
	}
	rule.region = self.region
	return rule, nil
}

func (self *SSecurityGroup) Delete() error {
	if self.Properties != nil {
		if self.Properties.NetworkInterfaces != nil {
			for _, nic := range *self.Properties.NetworkInterfaces {
				nic, err := self.region.GetNetworkInterface(nic.ID)
				if err != nil {
					return errors.Wrapf(err, "get nic %s", nic.ID)
				}
				nic.Properties.NetworkSecurityGroup = nil
				err = self.region.update(jsonutils.Marshal(nic), nil)
				if err != nil {
					return errors.Wrapf(err, "update nic")
				}
			}
		}
		if self.Properties.Subnets != nil {
			for _, _net := range *self.Properties.Subnets {
				net, err := self.region.GetNetwork(_net.ID)
				if err != nil {
					return errors.Wrapf(err, "get network %s", _net.ID)
				}
				err = self.region.update(jsonutils.Marshal(net), nil)
				if err != nil {
					return errors.Wrapf(err, "update network")
				}
			}
		}
	}
	return self.region.del(self.ID)
}

func (self *SRegion) CreateSecurityGroupRule(groupId string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SecurityRules, error) {
	name := fmt.Sprintf("%s_%d", opts.String(), opts.Priority)
	name = func(name string) string {
		// 名称必须以字母或数字开头，以字母、数字或下划线结尾，并且只能包含字母、数字、下划线、句点或连字符
		ret := ""
		for _, s := range name {
			if !(unicode.IsDigit(s) || unicode.IsLetter(s) || s == '.' || s == '-' || s == '_') {
				ret += "_"
				continue
			}
			ret += string(s)
		}
		if !unicode.IsDigit(rune(name[0])) && !unicode.IsLetter(rune(name[0])) {
			ret = fmt.Sprintf("r_%s", ret)
		}
		last := len(ret) - 1
		if !unicode.IsDigit(rune(name[last])) && !unicode.IsLetter(rune(name[last])) && name[last] != '_' {
			ret = fmt.Sprintf("%s_", ret)
		}
		return ret
	}(name)
	name = strings.ReplaceAll(name, ".", "_")

	if opts.Protocol == secrules.PROTO_ANY {
		opts.Protocol = "*"
	}
	if len(opts.Ports) == 0 {
		opts.Ports = "*"
	}
	properties := map[string]interface{}{
		"access":                   opts.Action,
		"priority":                 opts.Priority,
		"protocol":                 opts.Protocol,
		"direction":                opts.Direction + "bound",
		"DestinationAddressPrefix": opts.CIDR,
		"DestinationPortRange":     opts.Ports,
		"SourcePortRange":          "*",
		"SourceAddressPrefix":      "*",
	}

	info := strings.Split(groupId, "/")
	groupName := info[len(info)-1]

	params := map[string]interface{}{
		"Name":       groupName + "/securityRules/" + name,
		"Type":       "Microsoft.Network/networkSecurityGroups",
		"properties": properties,
	}

	rule := &SecurityRules{}
	resourceGroup := strings.TrimPrefix(getResourceGroup(groupId), self.client.subscriptionId+"/")
	err := self.create(resourceGroup, jsonutils.Marshal(params), rule)
	if err != nil {
		return nil, errors.Wrapf(err, "create")
	}
	return rule, nil
}
