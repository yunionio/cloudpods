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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/stringutils"
)

type SFirewallAction struct {
	IPProtocol string
	Ports      []string
}

type SFirewall struct {
	secgroup *SSecurityGroup

	Id                    string
	CreationTimestamp     time.Time
	Name                  string
	Description           string
	Network               string
	Priority              int
	SourceRanges          []string
	DestinationRanges     []string
	TargetServiceAccounts []string
	TargetTags            []string
	Allowed               []SFirewallAction `json:",allowempty"`
	Denied                []SFirewallAction `json:",allowempty"`
	Direction             string
	Disabled              bool
	SelfLink              string
	Kind                  string
}

func (self *SFirewall) GetGlobalId() string {
	return self.Id
}

func (self *SFirewall) GetAction() secrules.TSecurityRuleAction {
	if len(self.Allowed) > 0 {
		return secrules.SecurityRuleAllow
	}
	return secrules.SecurityRuleDeny
}

func (self *SFirewall) GetDescription() string {
	return self.Description
}

func (self *SFirewall) GetDirection() secrules.TSecurityRuleDirection {
	if strings.ToLower(self.Direction) == "ingress" {
		return secrules.DIR_IN
	}
	return secrules.DIR_OUT
}

func (self *SFirewall) GetCIDRs() []string {
	return append(self.SourceRanges, self.DestinationRanges...)
}

func (self *SFirewall) GetProtocol() string {
	ret := func() string {
		if len(self.Allowed)+len(self.Denied) == 1 {
			for _, r := range append(self.Allowed, self.Denied...) {
				return r.IPProtocol
			}
		}
		ret := []string{}
		for _, r := range append(self.Allowed, self.Denied...) {
			ret = append(ret, fmt.Sprintf("%s:%s", r.IPProtocol, strings.Join(r.Ports, ",")))
		}
		return strings.Join(ret, "|")
	}()
	if ret == "all" {
		return secrules.PROTO_ANY
	}
	return ret
}

func (self *SFirewall) GetPorts() string {
	if len(self.Allowed)+len(self.Denied) == 1 {
		for _, r := range append(self.Allowed, self.Denied...) {
			return strings.Join(r.Ports, ",")
		}
	}
	ret := []string{}
	for _, r := range append(self.Allowed, self.Denied...) {
		ret = append(ret, fmt.Sprintf("%s:%s", r.IPProtocol, strings.Join(r.Ports, ",")))
	}
	return strings.Join(ret, "|")
}

func (self *SFirewall) GetPriority() int {
	return self.Priority
}

func (self *SFirewall) Delete() error {
	return self.secgroup.gvpc.client.ecsDelete(self.SelfLink, nil)
}

func (self *SFirewall) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	params := map[string]string{
		"requestId": stringutils.UUID4(),
	}
	if len(self.SourceRanges) > 0 {
		self.SourceRanges = []string{opts.CIDR}
	}
	if len(self.DestinationRanges) > 0 {
		self.DestinationRanges = []string{opts.CIDR}
	}
	self.Priority = opts.Priority
	if len(opts.Desc) > 0 {
		self.Description = opts.Desc
	}
	action := SFirewallAction{}
	action.IPProtocol = opts.Protocol
	switch opts.Protocol {
	case secrules.PROTO_TCP, secrules.PROTO_UDP:
		if len(opts.Ports) > 0 {
			action.Ports = []string{opts.Ports}
		}
	case secrules.PROTO_ANY:
		action.IPProtocol = "all"
	}
	switch opts.Action {
	case secrules.SecurityRuleAllow:
		self.Denied = []SFirewallAction{}
		self.Allowed = []SFirewallAction{action}
	case secrules.SecurityRuleDeny:
		self.Allowed = []SFirewallAction{}
		self.Denied = []SFirewallAction{action}
	}
	resource := fmt.Sprintf("projects/%s/global/firewalls/%s", self.secgroup.gvpc.client.projectId, self.Name)
	_, err := self.secgroup.gvpc.client.ecsPatch(resource, "", params, jsonutils.Marshal(self))
	return err
}
