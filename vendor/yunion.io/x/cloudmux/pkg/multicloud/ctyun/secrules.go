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

package ctyun

import (
	"fmt"
	"net"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SSecurityGroupRule struct {
	secgroup *SSecurityGroup

	Direction       string
	Priority        int
	Ethertype       string
	Protocol        string
	DestCidrIP      string
	Description     string
	Origin          string
	CreateTime      time.Time
	Id              string
	Action          string
	SecurityGroupId string
	Range           string
}

func (self *SSecurityGroupRule) toRule() (cloudprovider.SecurityRule, error) {
	ret := cloudprovider.SecurityRule{Id: self.Id, Name: self.Description}

	rule := secrules.SecurityRule{
		Direction: secrules.DIR_IN,
		Action:    secrules.SecurityRuleAllow,
		Protocol:  strings.ToLower(self.Protocol),
		Priority:  self.Priority,
	}
	if self.Direction == "egress" {
		rule.Direction = secrules.DIR_OUT
	}
	if self.Action != "accept" {
		rule.Action = secrules.SecurityRuleDeny
	}

	var err error
	_, rule.IPNet, err = net.ParseCIDR(self.DestCidrIP)
	if err != nil {
		return ret, err
	}
	if self.Range != "Any" {
		err = rule.ParsePorts(self.Range)
		if err != nil {
			return ret, err
		}
	}

	ret.SecurityRule = rule
	return ret, nil
}

func (self *SRegion) CreateSecurityGroupRule(groupId string, r secrules.SecurityRule) error {
	priority := 100 - r.Priority
	if priority < 1 || priority > 100 {
		priority = 100
	}
	rule := map[string]interface{}{
		"direction":   "egress",
		"action":      "accept",
		"priority":    priority,
		"protocol":    strings.ToUpper(r.Protocol),
		"ethertype":   "IPv4",
		"destCidrIp":  r.IPNet.String(),
		"description": r.Description,
		"range":       "1-65535",
	}
	api := "/v4/vpc/create-security-group-egress"
	if r.Direction == secrules.DIR_IN {
		rule["direction"] = "ingress"
		api = "/v4/vpc/create-security-group-ingress"
	}
	if len(r.Ports) == 1 {
		rule["range"] = fmt.Sprintf("%d", r.Ports[0])
	} else if r.PortStart > 0 && r.PortEnd > 0 {
		rule["range"] = fmt.Sprintf("%d-%d", r.PortStart, r.PortEnd)
	}
	if r.Action == secrules.SecurityRuleDeny {
		rule["action"] = "drop"
	}
	params := map[string]interface{}{
		"securityGroupID":    groupId,
		"clientToken":        utils.GenRequestId(20),
		"securityGroupRules": []map[string]interface{}{rule},
	}
	_, err := self.post(SERVICE_VPC, api, params)
	return err
}
