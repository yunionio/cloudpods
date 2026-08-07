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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/apis"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
)

type SecgroupListOptions struct {
	_ struct{} `mcp-desc:"列出安全组。可用 search/server/vpc-id 等过滤；详情 climc_secgroup_show，规则 climc_secgroup_rule_list"`

	baseoptions.BaseListOptions

	Equals         string `help:"Secgroup ID or Name, filter secgroups whose rules equals the specified one" mcp:"true"`
	Server         string `help:"Filter secgroups bound to specified server" mcp:"true"`
	Ip             string `help:"Filter secgroup by ip" mcp:"true"`
	Ports          string `help:"Filter secgroup by ports" mcp:"true"`
	Direction      string `help:"Filter secgroup by direction" choices:"all|in|out" mcp:"true"`
	DBInstance     string `help:"Filter secgroups bound to specified rds" json:"dbinstance" mcp:"true"`
	Cloudregion    string `help:"Filter secgroups by region" mcp:"true"`
	VpcId          string `mcp:"true"`
	Cloudaccount   string `help:"Filter secgroups by account" mcp:"true"`
	LoadbalancerId string `mcp:"true"`

	IpSetId []string `help:"Filter secgroups by ip set" mcp:"true"`
}

func (opts *SecgroupListOptions) Params() (jsonutils.JSONObject, error) {
	return baseoptions.ListStructToParams(opts)
}

type SecgroupCreateOptions struct {
	_ struct{} `mcp-desc:"创建安全组。NAME 必填；可传 rules（安全规则字符串数组）与 vpc-id。创建后可用 climc_secgroup_rule_create 继续加规则"`

	baseoptions.BaseCreateOptions
	VpcId string   `mcp:"true"`
	Tags  []string `mcp:"true"`
	Rules []string `help:"security rule to create" mcp:"true"`
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
	params.Remove("tags")
	tags := map[string]string{}
	for _, tag := range opts.Tags {
		info := strings.Split(tag, "=")
		if len(info) != 2 {
			return nil, fmt.Errorf("invalid tag %s, tag should like key=value", tag)
		}
		tags["user:"+info[0]] = info[1]
	}
	if len(tags) > 0 {
		params.Set("__meta__", jsonutils.Marshal(tags))
	}
	return params, nil
}

type SecgroupIdOptions struct {
	ID string `help:"ID or Name of security group"`
}

func (opts *SecgroupIdOptions) GetId() string {
	return opts.ID
}

func (opts *SecgroupIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

// SecgroupShowOptions / SecgroupDeleteOptions 单独包装，避免与其它 Perform 复用 IdOptions 时注册。
type SecgroupShowOptions struct {
	_ struct{} `mcp-desc:"查询安全组详情。ID 可用 climc_secgroup_list 返回的 id/name"`

	SecgroupIdOptions
}

type SecgroupDeleteOptions struct {
	_ struct{} `mcp-desc:"删除安全组。若尚不知 id，先用 climc_secgroup_list 定位；确认无虚机绑定后再删"`

	SecgroupIdOptions
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
	ACTION      string `help:"Action of rule" choices:"allow|deny"`
	PRIORITY    int    `help:"Priority for rule, range 1 ~ 100"`
	Cidr        string `help:"IP or CIDR for rule"`
	Description string `help:"Description for rule"`
	Ports       string `help:"Port for rule"`
}

func (opts *SecgroupsAddRuleOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}

type SecgroupCloneOptions struct {
	SecgroupIdOptions
	NAME string `help:"Name of new secgroup"`
	Desc string `help:"Description of new secgroup"`
}

func (opts *SecgroupCloneOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"name": opts.NAME, "description": opts.Desc}), nil
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

type SecgroupCleanOptions struct {
}

func (opts *SecgroupCleanOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ServerNetworkSecgroupListOptions struct {
	baseoptions.BaseListOptions

	Server       string `help:"Server Id or name"`
	Secgroup     string `help:"Secgroup Id or name"`
	NetworkIndex *int   `help:"Server network index"`
	IsAdmin      bool   `help:"Is admin secgroup"`
}

func (opts *ServerNetworkSecgroupListOptions) Params() (jsonutils.JSONObject, error) {
	return baseoptions.ListStructToParams(opts)
}
