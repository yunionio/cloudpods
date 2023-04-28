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

package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SecGroupRulesListOptions struct {
	options.BaseListOptions
	Secgroup     string   `help:"Secgroup ID or Name"`
	SecgroupName string   `help:"Search rules by fuzzy secgroup name"`
	Projects     []string `help:"Filter rules by project"`
	Direction    string   `help:"filter Direction of rule" choices:"in|out"`
	Protocol     string   `help:"filter Protocol of rule" choices:"any|tcp|udp|icmp"`
	Action       string   `help:"filter Actin of rule" choices:"allow|deny"`
	Ports        string   `help:"filter Ports of rule"`
	Ip           string   `help:"filter cidr of rule"`
}

func (opts *SecGroupRulesListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type SecGroupRulesCreateOptions struct {
	SECGROUP string `help:"Secgroup ID or Name" metavar:"Secgroup"`
	RULE     string `json:"-"`
	Priority int64  `help:"priority of Rule" default:"50"`
	Desc     string `help:"Description" json:"description"`
}

func (opts *SecGroupRulesCreateOptions) Params() (jsonutils.JSONObject, error) {
	rule, err := secrules.ParseSecurityRule(opts.RULE)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid rule %s", opts.RULE)
	}
	return jsonutils.Marshal(map[string]interface{}{
		"direction":   rule.Direction,
		"action":      rule.Action,
		"protocol":    rule.Protocol,
		"cidr":        rule.IPNet.String(),
		"ports":       rule.GetPortsString(),
		"priority":    opts.Priority,
		"description": opts.Desc,
		"secgroup_id": opts.SECGROUP,
	}), nil
}

type SecGroupRulesUpdateOptions struct {
	options.BaseIdOptions
	Name     string `help:"New name of rule"`
	Priority int64  `help:"priority of Rule"`
	Protocol string `help:"Protocol of rule" choices:"any|tcp|udp|icmp"`
	Ports    string `help:"Ports of rule"`
	Cidr     string `help:"Cidr of rule"`
	Action   string `help:"filter Actin of rule" choices:"allow|deny"`
	Desc     string `help:"Description" metavar:"Description"`
}

func (opts *SecGroupRulesUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Name) > 0 {
		params.Add(jsonutils.NewString(opts.Name), "name")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if opts.Priority > 0 {
		params.Add(jsonutils.NewInt(opts.Priority), "priority")
	}
	if len(opts.Protocol) > 0 {
		params.Add(jsonutils.NewString(opts.Protocol), "protocol")
	}
	if len(opts.Ports) > 0 {
		params.Add(jsonutils.NewString(opts.Ports), "ports")
	}
	if len(opts.Cidr) > 0 {
		params.Add(jsonutils.NewString(opts.Cidr), "cidr")
	}
	if len(opts.Action) > 0 {
		params.Add(jsonutils.NewString(opts.Action), "action")
	}
	return params, nil
}
