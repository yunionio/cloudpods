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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SSecurityGroupRule struct {
	secgroup *SSecurityGroup

	PortRangeMax    int64  `json:"port_range_max"`
	SecurityGroupID string `json:"security_group_id"`
	RemoteGroupId   string `json:"remote_group_id"`
	Description     string `json:"description"`
	RemoteIPPrefix  string `json:"remote_ip_prefix"`
	Protocol        string `json:"protocol"`
	Ethertype       string `json:"ethertype"`
	UpdatedAt       string `json:"updated_at"`
	Direction       string `json:"direction"`
	TenantID        string `json:"tenant_id"`
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	PortRangeMin    int64  `json:"port_range_min"`
	CreatedAt       string `json:"created_at"`
}

func (self *SRegion) GetSecurityGroupRules(secgroupId string) ([]SSecurityGroupRule, error) {
	params := map[string]string{
		"regionId":        self.GetId(),
		"securityGroupId": secgroupId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/getSecurityGroupRules", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroupRules.DoGet")
	}

	ret := make([]SSecurityGroupRule, 0)
	err = resp.Unmarshal(&ret, "returnObj", "security_group_rules")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroupRules.Unmarshal")
	}

	secgroup, err := self.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSecurityGroupRules.GetSecurityGroupDetails")
	}

	for i := range ret {
		ret[i].secgroup = secgroup
	}

	return ret, nil
}

func (self *SRegion) CreateSecurityGroupRule(groupId, direction, ethertype, protocol, remoteIpPrefix string, portRangeMin, portRangeMax int64) error {
	ruleParams := jsonutils.NewDict()
	ruleParams.Set("regionId", jsonutils.NewString(self.GetId()))
	ruleParams.Set("securityGroupId", jsonutils.NewString(groupId))
	ruleParams.Set("direction", jsonutils.NewString(direction))
	ruleParams.Set("ethertype", jsonutils.NewString(ethertype))

	if len(protocol) > 0 {
		ruleParams.Set("protocol", jsonutils.NewString(protocol))
	}

	if len(remoteIpPrefix) > 0 {
		ruleParams.Set("remoteIpPrefix", jsonutils.NewString(remoteIpPrefix))
	}

	if portRangeMin > 0 {
		ruleParams.Set("portRangeMin", jsonutils.NewInt(portRangeMin))
	}

	if portRangeMax > 0 {
		ruleParams.Set("portRangeMax", jsonutils.NewInt(portRangeMax))
	}

	params := map[string]jsonutils.JSONObject{
		"jsonStr": ruleParams,
	}

	_, err := self.client.DoPost("/apiproxy/v3/createSecurityGroupRule", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.CreateSecurityGroupRule.DoPost")
	}

	return err
}
