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
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SFirewallAction struct {
	IPProtocol string
	Ports      []string
}

type SFirewall struct {
	Id                    string
	CreationTimestamp     time.Time
	Name                  string
	Description           string
	Network               string
	Priority              int
	SourceRanges          []string
	TargetServiceAccounts []string
	TargetTags            []string
	Allowed               []SFirewallAction
	Denied                []SFirewallAction
	Direction             string
	Disabled              bool
	SelfLink              string
	Kind                  string
}

type FirewallSet []SFirewall

func (f FirewallSet) Len() int {
	return len(f)
}

func (f FirewallSet) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FirewallSet) Less(i, j int) bool {
	if f[i].Priority != f[j].Priority {
		return f[i].Priority < f[j].Priority
	}
	return len(f[i].Allowed) < len(f[j].Allowed)
}

type SSecurityGroup struct {
	vpc *SVpc

	ServiceAccount string
	Tag            string
}

func (region *SRegion) GetFirewalls(network string, maxResults int, pageToken string) ([]SFirewall, error) {
	firewalls := []SFirewall{}
	params := map[string]string{"filter": "disabled = false"}
	resource := "global/firewalls"
	return firewalls, region.List(resource, params, maxResults, pageToken, &firewalls)
}

func (region *SRegion) GetFirewall(id string) (*SFirewall, error) {
	firewall := &SFirewall{}
	return firewall, region.Get(id, firewall)
}

func (firewall *SFirewall) _toRules(action secrules.TSecurityRuleAction) ([]secrules.SecurityRule, error) {
	rules := []secrules.SecurityRule{}
	list := firewall.Allowed
	if action == secrules.SecurityRuleDeny {
		list = firewall.Denied
	}
	for _, allow := range list {
		rule := secrules.SecurityRule{
			Action:      action,
			Direction:   secrules.DIR_IN,
			Description: firewall.Description,
			Priority:    firewall.Priority,
		}
		if firewall.Direction == "EGRESS" {
			rule.Direction = secrules.DIR_OUT
		}
		switch allow.IPProtocol {
		case "tcp", "udp", "icmp":
			rule.Protocol = allow.IPProtocol
		case "all":
			rule.Protocol = secrules.PROTO_ANY
		default:
			return nil, fmt.Errorf("unsupport protocol %s", allow.IPProtocol)
		}
		for _, sourceRange := range firewall.SourceRanges {
			if regutils.MatchCIDR(sourceRange) {
				_, rule.IPNet, _ = net.ParseCIDR(sourceRange)
			} else {
				rule.IPNet = &net.IPNet{
					IP:   net.ParseIP(sourceRange),
					Mask: net.CIDRMask(32, 32),
				}
			}
			ports := []int{}
			for _, port := range allow.Ports {
				if strings.Index(port, "-") > 0 {
					err := rule.ParsePorts(port)
					if err != nil {
						return nil, errors.Wrapf(err, "Parse port %s", port)
					}
					rules = append(rules, rule)
				} else {
					_port, err := strconv.Atoi(port)
					if err != nil {
						return nil, errors.Wrapf(err, "Atio port %s", port)
					}
					ports = append(ports, _port)
				}
			}
			if len(ports) > 0 {
				rule.Ports = ports
				rule.PortStart = -1
				rule.PortEnd = -1
				rules = append(rules, rule)
			}
		}
	}
	return rules, nil
}

func (firewall *SFirewall) toRules() ([]secrules.SecurityRule, error) {
	rules := []secrules.SecurityRule{}
	_rules, err := firewall._toRules(secrules.SecurityRuleAllow)
	if err != nil {
		return nil, err
	}
	rules = append(rules, _rules...)
	_rules, err = firewall._toRules(secrules.SecurityRuleDeny)
	if err != nil {
		return nil, err
	}
	rules = append(rules, _rules...)
	return rules, nil
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.vpc.globalnetwork.GetGlobalId()
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	if len(secgroup.Tag) > 0 {
		return fmt.Sprintf("%s/%s", secgroup.GetId(), secgroup.Tag)
	}
	if len(secgroup.ServiceAccount) > 0 {
		return fmt.Sprintf("%s/%s", secgroup.GetId(), secgroup.ServiceAccount)
	}
	return secgroup.GetId()
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return ""
}

func (secgroup *SSecurityGroup) GetName() string {
	if len(secgroup.Tag) > 0 {
		return secgroup.Tag
	}
	if len(secgroup.ServiceAccount) > 0 {
		return secgroup.ServiceAccount
	}
	return secgroup.vpc.globalnetwork.GetName()
}

func (secgroup *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return ""
}

func (secgroup *SSecurityGroup) IsEmulated() bool {
	return false
}

func (secgroup *SSecurityGroup) Refresh() error {
	return nil
}

func (secgroup *SSecurityGroup) Delete() error {
	return nil
}

func (secgroup *SSecurityGroup) GetProjectId() string {
	return ""
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return secgroup.vpc.GetGlobalId()
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	_firewalls, err := self.vpc.region.GetFirewalls(self.vpc.globalnetwork.SelfLink, 0, "")
	if err != nil {
		return nil, err
	}
	firewalls := []SFirewall{}
	for _, firewall := range _firewalls {
		if len(self.Tag) > 0 && utils.IsInStringArray(self.Tag, firewall.TargetTags) {
			firewalls = append(firewalls, firewall)
		} else if len(self.ServiceAccount) > 0 && utils.IsInStringArray(self.ServiceAccount, firewall.TargetServiceAccounts) {
			firewalls = append(firewalls, firewall)
		} else {
			if len(self.Tag) == 0 && len(self.ServiceAccount) == 0 && len(firewall.TargetServiceAccounts) == 0 && len(firewall.TargetTags) == 0 {
				firewalls = append(firewalls, firewall)
			}
		}
	}
	sort.Sort(FirewallSet(firewalls))
	rules := []secrules.SecurityRule{}
	priority := 100
	for _, firewall := range firewalls {
		firewall.Priority = priority
		if priority > 2 {
			priority--
		}
		_rules, err := firewall.toRules()
		if err != nil {
			return nil, err
		}
		rules = append(rules, _rules...)
	}
	return rules, nil
}

func (secgroup *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISecurityGroupById(id string) (cloudprovider.ICloudSecurityGroup, error) {
	vpcs, err := region.GetIVpcs()
	if err != nil {
		return nil, errors.Wrap(err, "GetIVpcs")
	}

	for _, vpc := range vpcs {
		secgroups, err := vpc.GetISecurityGroups()
		if err != nil {
			return nil, errors.Wrap(err, "GetISecurityGroups")
		}
		for _, secgroup := range secgroups {
			if secgroup.GetGlobalId() == id {
				return secgroup, nil
			}
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetISecurityGroupByName(vpcId string, name string) (cloudprovider.ICloudSecurityGroup, error) {
	ivpc, err := region.GetIVpcById(vpcId)
	if err != nil {
		return nil, err
	}
	secgroups, err := ivpc.GetISecurityGroups()
	if err != nil {
		return nil, errors.Wrap(err, "ivpc.GetISecurityGroups")
	}
	for _, secgroup := range secgroups {
		if secgroup.GetName() == name {
			return secgroup, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return region.GetISecurityGroupByName(conf.VpcId, "")
}
