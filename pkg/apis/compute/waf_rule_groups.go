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

import "yunion.io/x/onecloud/pkg/apis"

const (
	WAF_RULE_GROUP_STATUS_AVAILABLE = "available"
	WAF_RULE_GROUP_STATUS_DELETING  = "deleting"
)

type WafRuleGroupDetails struct {
	apis.StatusInfrasResourceBaseDetails
	SWafRuleGroup
}

type WafRuleGroupListInput struct {
	apis.StatusInfrasResourceBaseListInput

	// 是否是系统RuleGroup
	IsSystem *bool `json:"is_system"`
	// 云平台
	Provider string `json:"provider"`
	// 云环境
	CloudEnv string `json:"cloud_env"`
}

type WafRuleGroupCacheDetails struct {
	apis.StatusStandaloneResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	SWafRuleGroupCache
}

type WafRuleGroupCacheListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	ManagedResourceListInput
	RegionalFilterListInput
}
