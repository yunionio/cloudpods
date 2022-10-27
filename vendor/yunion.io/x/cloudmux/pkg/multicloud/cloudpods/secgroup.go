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

package cloudpods

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SResourceBase
	CloudpodsTags
	region *SRegion

	api.SecgroupDetails
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetId() string {
	return self.Id
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.Id
}

func (self *SSecurityGroup) GetStatus() string {
	return self.Status
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.TenantId
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	ret := []cloudprovider.SecurityRule{}
	for _, r := range self.Rules {
		if len(r.PeerSecgroupId) > 0 {
			continue
		}
		rule := cloudprovider.SecurityRule{ExternalId: r.Id}
		rule.Action = secrules.TSecurityRuleAction(r.Action)
		rule.Priority = int(r.Priority)
		rule.Protocol = r.Protocol
		rule.Description = r.Description
		rule.Direction = secrules.TSecurityRuleDirection(r.Direction)
		rule.ParseCIDR(r.CIDR)
		rule.ParsePorts(r.Ports)
		ret = append(ret, rule)
	}
	return ret, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SSecurityGroup) GetReferences() ([]cloudprovider.SecurityGroupReference, error) {
	params := map[string]interface{}{
		"secgroup_id": self.Id,
	}
	servers := []SInstance{}
	err := self.region.list(&modules.Servers, params, &servers)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.SecurityGroupReference{}
	for i := range servers {
		ret = append(ret, cloudprovider.SecurityGroupReference{
			Id:   servers[i].Id,
			Name: servers[i].Name,
		})
	}
	return ret, nil
}

func (self *SRegion) DeleteSecRule(id string) error {
	return self.cli.delete(&modules.SecGroupRules, id)
}

func (self *SRegion) CreateSecRule(secId string, rule cloudprovider.SecurityRule) error {
	input := api.SSecgroupRuleCreateInput{}
	input.SecgroupId = secId
	input.Priority = &rule.Priority
	input.Action = string(rule.Action)
	input.Protocol = rule.Protocol
	input.Direction = string(rule.Direction)
	input.Description = rule.Description
	if rule.IPNet != nil {
		input.CIDR = rule.IPNet.String()
	}

	if len(rule.Ports) > 0 {
		ports := []string{}
		for _, port := range rule.Ports {
			ports = append(ports, fmt.Sprintf("%d", port))
		}
		input.Ports = strings.Join(ports, ",")
	} else if rule.PortStart > 0 && rule.PortEnd > 0 {
		input.Ports = fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd)
	}
	ret := struct{}{}
	return self.create(&modules.SecGroupRules, input, &ret)
}

func (self *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
	for _, r := range append(inDels, outDels...) {
		err := self.region.DeleteSecRule(r.Id)
		if err != nil {
			return errors.Wrapf(err, "delete rule %s", r.Id)
		}
	}
	for _, r := range append(inAdds, outAdds...) {
		err := self.region.CreateSecRule(self.Id, r)
		if err != nil {
			return errors.Wrapf(err, "create rule %s", r)
		}
	}
	return nil
}

func (self *SSecurityGroup) Delete() error {
	return self.region.cli.delete(&modules.SecGroups, self.Id)
}

func (self *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	params := map[string]interface{}{
		"cloud_env": "onpremise",
	}
	ret := []SSecurityGroup{}
	return ret, self.cli.list(&modules.SecGroups, params, &ret)
}

func (self *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	secgroup := &SSecurityGroup{region: self}
	return secgroup, self.cli.get(&modules.SecGroups, id, nil, secgroup)
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	params := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Desc,
		"rules":       opts.Rules,
	}
	if len(opts.ProjectId) > 0 {
		params["project_id"] = opts.ProjectId
	}
	secgroup := &SSecurityGroup{region: self}
	return secgroup, self.create(&modules.SecGroups, params, secgroup)
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.region.GetSecurityGroups()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self.region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup, err := self.GetSecurityGroup(secgroupId)
	if err != nil {
		return nil, err
	}
	return secgroup, nil
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup, err := self.GetSecurityGroup(opts.Name)
	if err != nil {
		return nil, err
	}
	return secgroup, nil
}
