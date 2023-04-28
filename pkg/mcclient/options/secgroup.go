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

package options

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/apis"
)

type SecgroupListOptions struct {
	BaseListOptions

	Equals       string `help:"Secgroup ID or Name, filter secgroups whose rules equals the specified one"`
	Server       string `help:"Filter secgroups bound to specified server"`
	Ip           string `help:"Filter secgroup by ip"`
	Ports        string `help:"Filter secgroup by ports"`
	Direction    string `help:"Filter secgroup by ports" choices:"all|in|out"`
	DBInstance   string `help:"Filter secgroups bound to specified rds" json:"dbinstance"`
	Cloudregion  string `help:"Filter secgroups by region"`
	Cloudaccount string `help:"Filter secgroups by account"`
	WithCache    bool   `help:"Whether to bring cache information"`
}

func (opts *SecgroupListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type SecgroupCreateOptions struct {
	BaseCreateOptions
	Rules []string `help:"security rule to create"`
}

func (opts *SecgroupCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("rules")
	rules := []secrules.SecurityRule{}
	for i, ruleStr := range opts.Rules {
		rule, err := secrules.ParseSecurityRule(ruleStr)
		if err != nil {
			return nil, errors.Wrapf(err, "ParseSecurityRule(%s)", ruleStr)
		}
		rule.Priority = i + 1
		rules = append(rules, *rule)
	}
	if len(rules) > 0 {
		params.Add(jsonutils.Marshal(rules), "rules")
	}
	return params, nil
}

type SecgroupIdOptions struct {
	ID string `help:"ID or Name of security group destination"`
}

func (opts *SecgroupIdOptions) GetId() string {
	return opts.ID
}

func (opts *SecgroupIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SecgroupMergeOptions struct {
	SecgroupIdOptions
	SECGROUPS []string `help:"source IDs or Names of secgroup"`
}

func (opts *SecgroupMergeOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string][]string{"secgroup_ids": opts.SECGROUPS}), nil
}

type SecgroupsAddRuleOptions struct {
	SecgroupIdOptions
	DIRECTION   string `help:"Direction of rule" choices:"in|out"`
	PROTOCOL    string `help:"Protocol of rule" choices:"any|tcp|udp|icmp"`
	ACTION      string `help:"Actin of rule" choices:"allow|deny"`
	PRIORITY    int    `help:"Priority for rule, range 1 ~ 100"`
	Cidr        string `help:"IP or CIRD for rule"`
	Description string `help:"Desciption for rule"`
	Ports       string `help:"Port for rule"`
}

func (opts *SecgroupsAddRuleOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}

type SecurityGroupCacheOptions struct {
	SecgroupIdOptions
	VPC_ID string `help:"ID or Name of vpc"`
}

func (opts *SecurityGroupCacheOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}

type SecurityGroupUncacheSecurityGroup struct {
	SecgroupIdOptions
	CACHE string `help:"ID of secgroup cache"`
}

func (opts *SecurityGroupUncacheSecurityGroup) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}

type SecgroupChangeOwnerOptions struct {
	SecgroupIdOptions
	apis.ProjectizedResourceInput
}

type SecgroupImportRulesOptions struct {
	SecgroupIdOptions

	RULE []string `help:"rule pattern: rule|priority eg: in:allow any 1"`
}

func (opts *SecgroupImportRulesOptions) Params() (jsonutils.JSONObject, error) {
	rules := jsonutils.NewArray()
	for _, rule := range opts.RULE {
		priority := 1
		var r *secrules.SecurityRule = nil
		var err error
		info := strings.Split(rule, "|")
		switch len(info) {
		case 1:
		case 2:
			priority, err = strconv.Atoi(info[1])
			if err != nil {
				return nil, errors.Wrapf(err, "Parse rule %s priority %s", rule, info[1])
			}
		default:
			return nil, fmt.Errorf("invalid rule %s", rule)
		}
		r, err = secrules.ParseSecurityRule(info[0])
		if err != nil {
			return nil, errors.Wrapf(err, "ParseSecurityRule(%s)", rule)
		}
		r.Priority = priority
		rules.Add(jsonutils.Marshal(r))
	}
	return jsonutils.Marshal(map[string]*jsonutils.JSONArray{"rules": rules}), nil
}
