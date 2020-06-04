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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	DestinationRanges     []string
	TargetServiceAccounts []string
	TargetTags            []string
	Allowed               []SFirewallAction
	Denied                []SFirewallAction
	Direction             string
	Disabled              bool
	SelfLink              string
	Kind                  string
}

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	vpc *SVpc

	ServiceAccount string
	Tag            string
}

func (region *SRegion) GetFirewalls(network string, maxResults int, pageToken string) ([]SFirewall, error) {
	firewalls := []SFirewall{}
	params := map[string]string{"filter": "disabled = false"}
	resource := "global/firewalls"
	if len(network) > 0 {
		params["filter"] = fmt.Sprintf(`(disabled = false) AND (network="%s")`, network)
	}
	return firewalls, region.List(resource, params, maxResults, pageToken, &firewalls)
}

func (region *SRegion) GetFirewall(id string) (*SFirewall, error) {
	firewall := &SFirewall{}
	return firewall, region.Get(id, firewall)
}

func (firewall *SFirewall) _toRules(action secrules.TSecurityRuleAction) ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
	list := firewall.Allowed
	if action == secrules.SecurityRuleDeny {
		list = firewall.Denied
	}
	for _, allow := range list {
		rule := cloudprovider.SecurityRule{
			ExternalId: firewall.SelfLink,
			SecurityRule: secrules.SecurityRule{
				Action:    action,
				Direction: secrules.DIR_IN,
				Priority:  firewall.Priority,
			},
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
		ipRanges := firewall.SourceRanges
		if rule.Direction == secrules.DIR_OUT {
			ipRanges = firewall.DestinationRanges
		}
		for _, ipRange := range ipRanges {
			if regutils.MatchCIDR(ipRange) {
				_, rule.IPNet, _ = net.ParseCIDR(ipRange)
			} else {
				rule.IPNet = &net.IPNet{
					IP:   net.ParseIP(ipRange),
					Mask: net.CIDRMask(32, 32),
				}
			}
			ports := []int{}
			for _, port := range allow.Ports {
				if strings.Index(port, "-") > 0 {
					port = strings.Replace(port, "0-", "1-", 1)
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

func (firewall *SFirewall) toRules() ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
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
	return secgroup.vpc.globalnetwork.Name
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
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
	rules := []cloudprovider.SecurityRule{}
	for _, firewall := range firewalls {
		_rules, err := firewall.toRules()
		if err != nil {
			return nil, err
		}
		rules = append(rules, _rules...)
	}
	return rules, nil
}

func (secgroup *SSecurityGroup) SyncRules(common, inAdds, outAdds, inDels, outDels []cloudprovider.SecurityRule) error {
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
