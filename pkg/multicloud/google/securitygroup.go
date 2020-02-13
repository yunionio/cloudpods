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
	if len(network) > 0 {
		params["filter"] = fmt.Sprintf(`(disabled = false) AND (network="%s")`, network)
	}
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
			Description: firewall.SelfLink,
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
	rules, err := secgroup.GetRules()
	if err != nil {
		return errors.Wrap(err, "GetRules")
	}
	for _, rule := range rules {
		err = secgroup.vpc.region.DeleteSecgroupRule(rule.Description, rule)
		if err != nil {
			return errors.Wrapf(err, "DeleteSecgroupRule(%s)", rule.Description)
		}
	}
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

func (region *SRegion) DeleteSecgroupRule(ruleId string, rule secrules.SecurityRule) error {
	firwall, err := region.GetFirewall(ruleId)
	if err != nil {
		return errors.Wrap(err, "region.GetFirewall")
	}
	currentRule, err := firwall.toRules()
	if err != nil {
		return errors.Wrap(err, "firwall.toRules")
	}
	if len(currentRule) > 1 {
		for _, _rule := range currentRule {
			if _rule.String() != rule.String() {
				for _, tag := range firwall.TargetTags {
					err = region.CreateSecurityGroupRule(_rule, firwall.Network, tag, "")
					if err != nil {
						return errors.Wrap(err, "region.CreateSecurityGroupRule")
					}
				}
				for _, serviceAccount := range firwall.TargetServiceAccounts {
					err = region.CreateSecurityGroupRule(_rule, firwall.Network, "", serviceAccount)
					if err != nil {
						return errors.Wrap(err, "region.CreateSecurityGroupRule")
					}
				}
				if len(firwall.TargetTags)+len(firwall.TargetServiceAccounts) == 0 {
					err = region.CreateSecurityGroupRule(_rule, firwall.Network, "", "")
					if err != nil {
						return errors.Wrap(err, "region.CreateSecurityGroupRule")
					}
				}
			}
		}
	}
	return region.Delete(firwall.SelfLink)
}

func (secgroup *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	if len(rules) == 0 {
		rules = append(rules, *secrules.MustParseSecurityRule("in:deny any"))
	}
	currentRule, err := secgroup.GetRules()
	if err != nil {
		return errors.Wrap(err, "secgroup.GetRules")
	}
	sort.Sort(secrules.SecurityRuleSet(rules))
	region := secgroup.vpc.region
	deleteRules := map[string]secrules.SecurityRule{}
	addRules := []secrules.SecurityRule{}
	i, j := 0, 0
	for i < len(rules) || j < len(currentRule) {
		if i < len(rules) && j < len(currentRule) {
			currentRuleStr := currentRule[j].String()
			ruleStr := rules[i].String()
			cmp := strings.Compare(currentRuleStr, ruleStr)
			if cmp == 0 {
				i += 1
				j += 1
			} else if cmp > 0 {
				// delete rule
				deleteRules[currentRule[j].Description] = currentRule[j]
				j += 1
			} else {
				rules[i].Priority = 101 - rules[i].Priority
				addRules = append(addRules, rules[i])
				i += 1
			}
		} else if i >= len(rules) {
			// delete rule
			deleteRules[currentRule[j].Description] = currentRule[j]
			j += 1
		} else if j >= len(currentRule) {
			// add rule
			rules[i].Priority = 101 - rules[i].Priority
			addRules = append(addRules, rules[i])
			err = region.CreateSecurityGroupRule(rules[i], secgroup.vpc.globalnetwork.SelfLink, secgroup.Tag, secgroup.ServiceAccount)
			if err != nil {
				return errors.Wrapf(err, "region.CreateSecurityGroupRule(%s)", rules[i].String())
			}
			i += 1
		}
	}
	for id, rule := range deleteRules {
		err = region.DeleteSecgroupRule(id, rule)
		if err != nil {
			return errors.Wrapf(err, "DeleteSecgroupRule(%s)", id)
		}
	}
	for _, rule := range addRules {
		err = region.CreateSecurityGroupRule(rule, secgroup.vpc.globalnetwork.SelfLink, secgroup.Tag, secgroup.ServiceAccount)
		if err != nil {
			return errors.Wrapf(err, "CreateSecurityGroupRule(%s)", rule.String())
		}
	}
	return nil
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
		if strings.ToLower(secgroup.GetName()) == strings.ToLower(name) {
			return secgroup, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateSecurityGroupRule(rule secrules.SecurityRule, vpcId string, tag string, serviceAccount string) error {
	name := fmt.Sprintf("%s-%d", rule.String(), rule.Priority)
	if len(tag) > 0 {
		name = fmt.Sprintf("for-tag-%s-%s", tag, name)
	}
	if len(serviceAccount) > 0 {
		name = fmt.Sprintf("for-service-account-%s-%s", serviceAccount, name)
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
	} else {
		body["sourceRanges"] = []string{rule.IPNet.String()}
	}

	protocol := string(rule.Protocol)
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
	err := region.Insert("global/firewalls", jsonutils.Marshal(body), firwall)
	if err != nil {
		if strings.Index(err.Error(), "already exists") >= 0 {
			return nil
		}
		return errors.Wrap(err, "region.Insert")
	}
	return nil
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	conf.VpcId = fmt.Sprintf("%s/%s", region.GetGlobalId(), conf.VpcId)
	ivpc, err := region.GetIVpcById(conf.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetIVpcById(%s)", conf.VpcId)
	}

	vpc := ivpc.(*SVpc)

	secgroup := &SSecurityGroup{vpc: vpc, Tag: strings.ToLower(conf.Name)}
	return secgroup, nil
}
