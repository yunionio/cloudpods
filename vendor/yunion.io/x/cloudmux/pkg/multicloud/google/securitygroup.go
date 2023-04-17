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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	SECGROUP_TYPE_SERVICE_ACCOUNT = "serviceAccount"
	SECGROUP_TYPE_TAG             = "tag"
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
	GoogleTags
	gvpc *SGlobalNetwork

	ServiceAccount string
	Tag            string
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

func (firewall *SFirewall) _toRules(action secrules.TSecurityRuleAction) ([]cloudprovider.SecurityRule, error) {
	rules := []cloudprovider.SecurityRule{}
	list := firewall.Allowed
	if action == secrules.SecurityRuleDeny {
		list = firewall.Denied
	}
	for _, allow := range list {
		rule := cloudprovider.SecurityRule{
			SecurityRule: secrules.SecurityRule{
				Action:    action,
				Direction: secrules.DIR_IN,
				Priority:  65535 - firewall.Priority,
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
			rule.ParseCIDR(ipRange)
			ports := []int{}
			for _, port := range allow.Ports {
				if strings.Index(port, "-") > 0 {
					if strings.HasPrefix(port, "0-") {
						port = strings.Replace(port, "0-", "1-", 1)
					}
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
			if len(allow.Ports) == 0 {
				rule.Ports = []int{}
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
	return secgroup.gvpc.GetGlobalId()
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	if len(secgroup.Tag) > 0 {
		return fmt.Sprintf("%s/%s/%s", secgroup.GetId(), SECGROUP_TYPE_TAG, secgroup.Tag)
	}
	if len(secgroup.ServiceAccount) > 0 {
		return fmt.Sprintf("%s/%s/%s", secgroup.GetId(), SECGROUP_TYPE_SERVICE_ACCOUNT, secgroup.ServiceAccount)
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
	return secgroup.gvpc.Name
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

func (self *SSecurityGroup) Delete() error {
	firewalls, err := self.gvpc.client.GetFirewalls(self.gvpc.SelfLink, 0, "")
	if err != nil {
		return err
	}
	for _, firewall := range firewalls {
		if len(self.Tag) > 0 && utils.IsInStringArray(self.Tag, firewall.TargetTags) || len(self.ServiceAccount) > 0 && utils.IsInStringArray(self.ServiceAccount, firewall.TargetServiceAccounts) {
			err = self.gvpc.client.ecsDelete(firewall.SelfLink, nil)
			if err != nil {
				return errors.Wrapf(err, "delete rule %s", firewall.SelfLink)
			}
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

func (self *SSecurityGroup) GetRules() ([]cloudprovider.SecurityRule, error) {
	_firewalls, err := self.gvpc.client.GetFirewalls(self.gvpc.SelfLink, 0, "")
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

func (region *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	ivpc, err := region.GetIVpcById(opts.VpcId)
	if err != nil {
		return nil, err
	}
	secgroups, err := ivpc.GetISecurityGroups()
	if err != nil {
		return nil, errors.Wrap(err, "ivpc.GetISecurityGroups")
	}
	for _, secgroup := range secgroups {
		if strings.ToLower(secgroup.GetName()) == strings.ToLower(opts.Name) {
			return secgroup, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SGoogleClient) CreateSecurityGroupRule(rule cloudprovider.SecurityRule, vpcId string, tag string, serviceAccount string) error {
	name := fmt.Sprintf("%s-%d", rule.String(), rule.Priority)
	if len(tag) > 0 {
		name = fmt.Sprintf("%s-%s", tag, name)
	}
	if len(serviceAccount) > 0 {
		name = fmt.Sprintf("%s-%s", serviceAccount, name)
	}

	body := map[string]interface{}{
		"name":      strings.ToLower(name),
		"priority":  rule.Priority,
		"network":   vpcId,
		"direction": "INGRESS",
	}
	if len(tag) > 0 {
		body["targetTags"] = []string{strings.ToLower(tag)}
	}
	if len(serviceAccount) > 0 {
		body["targetServiceAccounts"] = []string{serviceAccount}
	}
	if rule.Direction == secrules.DIR_OUT {
		body["direction"] = "EGRESS"
		body["destinationRanges"] = []string{rule.IPNet.String()}
	} else {
		body["sourceRanges"] = []string{rule.IPNet.String()}
	}

	protocol := strings.ToLower(rule.Protocol)
	if protocol == secrules.PROTO_ANY {
		protocol = "all"
	}

	ports := []string{}
	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			ports = append(ports, fmt.Sprintf("%d", port))
		}
	} else if rule.PortStart > 0 && rule.PortEnd > 0 {
		if rule.PortStart == rule.PortEnd {
			ports = append(ports, fmt.Sprintf("%d", rule.PortStart))
		} else {
			ports = append(ports, fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd))
		}
	}

	actionInfo := []struct {
		IPProtocol string
		Ports      []string
	}{
		{
			IPProtocol: protocol,
			Ports:      ports,
		},
	}

	if rule.Action == secrules.SecurityRuleDeny {
		body["denied"] = actionInfo
	} else {
		body["allowed"] = actionInfo
	}

	firwall := &SFirewall{}
	err := self.Insert("global/firewalls", jsonutils.Marshal(body), firwall)
	if err != nil {
		if strings.Index(err.Error(), "already exists") >= 0 {
			return nil
		}
		return errors.Wrap(err, "region.Insert")
	}
	return nil
}

func (region *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	gvpc, err := region.client.GetGlobalNetwork(opts.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetIVpcById(%s)", opts.VpcId)
	}
	secgroup := &SSecurityGroup{gvpc: gvpc, Tag: strings.ToLower(opts.Name)}
	var syncRules = func(rules []cloudprovider.SecurityRule) error {
		if len(rules) == 0 {
			return nil
		}
		offset := 65534 / (len(rules) + 1)
		for i := range rules {
			rules[i].Priority = 65533 - i*offset
			err := secgroup.gvpc.client.CreateSecurityGroupRule(rules[i], secgroup.gvpc.SelfLink, secgroup.Tag, secgroup.ServiceAccount)
			if err != nil {
				return errors.Wrapf(err, "CreateSecurityGroupRule")
			}
		}
		return nil
	}
	err = syncRules(opts.InRules)
	if err != nil {
		return nil, err
	}
	err = syncRules(opts.OutRules)
	if err != nil {
		return nil, err
	}
	return secgroup, nil
}
