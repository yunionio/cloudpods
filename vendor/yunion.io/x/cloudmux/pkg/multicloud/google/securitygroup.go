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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	GoogleTags
	gvpc *SGlobalNetwork

	Tag string

	Rules []SFirewall
}

func (self *SGoogleClient) GetFirewalls(network string, maxResults int, pageToken string) ([]SFirewall, error) {
	firewalls := []SFirewall{}
	params := map[string]string{"filter": "disabled = false"}
	resource := "global/firewalls"
	if len(network) > 0 {
		params["filter"] = fmt.Sprintf(`(disabled = false) AND (network="%s")`, network)
	}
	return firewalls, self._ecsListAll("GET", resource, params, &firewalls)
}

func (self *SGoogleClient) GetFirewall(id string) (*SFirewall, error) {
	firewall := &SFirewall{}
	return firewall, self.ecsGet("global/firewalls", id, firewall)
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.Tag
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.Tag
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return ""
}

func (self *SSecurityGroup) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (secgroup *SSecurityGroup) GetName() string {
	return secgroup.Tag
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (secgroup *SSecurityGroup) Refresh() error {
	rules, err := secgroup.gvpc.client.GetFirewalls(secgroup.gvpc.SelfLink, 0, "")
	if err != nil {
		return err
	}
	secgroup.Rules = []SFirewall{}
	for i := range rules {
		if len(rules[i].TargetTags) > 0 && rules[i].TargetTags[0] == secgroup.Tag {
			secgroup.Rules = append(secgroup.Rules, rules[i])
		}
	}
	if len(secgroup.Rules) == 0 {
		return cloudprovider.ErrNotFound
	}
	return nil
}

func (self *SSecurityGroup) Delete() error {
	for _, rule := range self.Rules {
		err := self.gvpc.client.ecsDelete(rule.SelfLink, nil)
		if err != nil {
			return errors.Wrapf(err, "delete")
		}
	}
	return nil
}

func (secgroup *SSecurityGroup) GetProjectId() string {
	return ""
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return secgroup.gvpc.GetGlobalId()
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	ret := []cloudprovider.ISecurityGroupRule{}
	for i := range self.Rules {
		self.Rules[i].secgroup = self
		ret = append(ret, &self.Rules[i])
	}
	return ret, nil
}

func (self *SGoogleClient) CreateSecurityGroupRule(globalnetworkId, tag string, opts *cloudprovider.SecurityGroupRuleCreateOptions) (*SFirewall, error) {
	name := fmt.Sprintf("%s-%d-auto-%d", opts.String(), opts.Priority, time.Now().Unix())
	body := map[string]interface{}{
		"name":        strings.ToLower(name),
		"description": opts.Desc,
		"priority":    opts.Priority,
		"network":     globalnetworkId,
		"direction":   "INGRESS",
		"targetTags":  []string{strings.ToLower(tag)},
	}
	if opts.Direction == secrules.DIR_OUT {
		body["direction"] = "EGRESS"
		body["destinationRanges"] = []string{opts.CIDR}
	} else {
		body["sourceRanges"] = []string{opts.CIDR}
	}

	protocol := strings.ToLower(opts.Protocol)
	if protocol == secrules.PROTO_ANY {
		protocol = "all"
	}

	actionInfo := []struct {
		IPProtocol string
		Ports      []string
	}{
		{
			IPProtocol: protocol,
		},
	}
	if len(opts.Ports) > 0 {
		actionInfo[0].Ports = []string{opts.Ports}
	}

	if opts.Action == secrules.SecurityRuleDeny {
		body["denied"] = actionInfo
	} else {
		body["allowed"] = actionInfo
	}

	firwall := &SFirewall{}
	err := self.Insert("global/firewalls", jsonutils.Marshal(body), firwall)
	if err != nil {
		return nil, errors.Wrap(err, "region.Insert")
	}
	return firwall, nil
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	rule, err := self.gvpc.client.CreateSecurityGroupRule(self.gvpc.SelfLink, self.Tag, opts)
	if err != nil {
		return nil, err
	}
	rule.secgroup = self
	return rule, nil
}

func (self *SGlobalNetwork) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	secgroup := &SSecurityGroup{gvpc: self, Tag: strings.ToLower(opts.Name)}
	groups, err := self.GetISecurityGroups()
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].GetGlobalId() == secgroup.GetGlobalId() {
			return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, secgroup.GetGlobalId())
		}
	}
	rule, err := secgroup.gvpc.client.CreateSecurityGroupRule(secgroup.gvpc.SelfLink, secgroup.Tag, &cloudprovider.SecurityGroupRuleCreateOptions{
		Desc:      "default allow out",
		Priority:  65535,
		Protocol:  secrules.PROTO_ANY,
		Direction: secrules.DIR_OUT,
		CIDR:      "0.0.0.0/0",
		Action:    secrules.SecurityRuleAllow,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSecurityGroupRule")
	}
	secgroup.Rules = []SFirewall{*rule}
	return secgroup, nil
}
