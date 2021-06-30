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
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	WAF_RULE_STATUS_AVAILABLE     = "available"
	WAF_RULE_STATUS_DELETING      = "deleting"
	WAF_RULE_STATUS_CREATING      = "creating"
	WAF_RULE_STATUS_CREATE_FAILED = "create_failed"
	WAF_RULE_STATUS_DELETE_FAILED = "delete_failed"
	WAF_RULE_STATUS_UPDATING      = "updating"
	WAF_RULE_STATUS_UPDATE_FAILED = "update_failed"
	WAF_RULE_STATUS_UNKNOWN       = "unknown"
)

type WafRuleListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	// WAF实例Id
	WafInstanceId string `json:"waf_instance_id"`

	// WAF规则组Id
	WafRuleGroupId string `json:"waf_rule_group_id"`
}

type WafRuleCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// WAF实例Id
	WafInstanceId string `json:"waf_instance_id"`

	// 优先级,不可重复
	// Azure优先级范围1-100
	Priority int `json:"priority"`
	// 匹配后默认行为
	Action *cloudprovider.DefaultAction `json:"action"`
	// enmu: and, or, not
	StatementCondition string `json:"statement_condition"`

	// swagger: ignore
	// WAF规则组Id
	WafRuleGroupId string `json:"waf_rule_group_id"`

	// 条件表达式
	Statements []cloudprovider.SWafStatement
}

type WafRuleDetails struct {
	apis.StatusStandaloneResourceDetails
	SWafRule

	Statements []cloudprovider.SWafStatement
}

type WafRuleUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput
	// 条件表达式
	Statements []cloudprovider.SWafStatement
}
