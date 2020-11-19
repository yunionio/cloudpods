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

package monitor

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type SuggestSysRuleConfigSupportTypes struct {
	Types         []SuggestDriverType `json:"types"`
	ResourceTypes []string            `json:"resource_types"`
}

type SuggestSysRuleConfigCreateInput struct {
	apis.StandaloneResourceCreateInput
	apis.ScopedResourceCreateInput

	// Type is suggestsysrule driver type
	Type *SuggestDriverType `json:"type"`
	// RuleId is SSuggestSysRule model object id
	// RuleId *string `json:"rule_id"`
	// ResourceType is suggestsysrule driver resource type
	ResourceType *MonitorResourceType `json:"resource_type"`
	// ResourceId is suggest alert result resource id
	ResourceId *string `json:"resource_id"`
	// IgnoreAlert means whether or not show SSuggestSysAlert results for current scope
	IgnoreAlert bool `json:"ignore_alert"`
}

type SuggestSysRuleConfigUpdateInput struct {
	apis.Meta

	// IgnoreAlert means whether or not show SSuggestSysAlert results for current scope
	IgnoreAlert *bool `json:"ignore_alert"`
}

type SuggestSysRuleConfigDetails struct {
	apis.StandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	RuleId      string `json:"rule_id"`
	Rule        string `json:"rule"`
	RuleEnabled bool   `json:"rule_enabled"`
	ResName     string `json:"res_name"`
}

type SuggestSysRuleConfigListInput struct {
	apis.StandaloneResourceListInput
	apis.ScopedResourceBaseListInput
	Type         *SuggestDriverType   `json:"type"`
	ResourceType *MonitorResourceType `json:"resource_type"`
	IgnoreAlert  *bool                `json:"ignore_alert"`
}

type SuggestSysRuleConfigTypeInfo struct {
	Name string `json:"name"`
}
