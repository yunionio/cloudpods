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
)

const (
	POLICY_DEFINITION_STATUS_READY   = "ready"
	POLICY_DEFINITION_STATUS_UNKNOWN = "unknown"

	POLICY_DEFINITION_CATEGORY_CLOUDREGION  = "cloudregion"  // 地域
	POLICY_DEFINITION_CATEGORY_TAG          = "tag"          // 标签
	POLICY_DEFINITION_CATEGORY_EXPIRED      = "expired"      // 到期释放
	POLICY_DEFINITION_CATEGORY_BILLING_TYPE = "billing_type" // 付款方式
	POLICY_DEFINITION_CATEGORY_BATCH_CREATE = "batch_create" // 批量创建

	POLICY_DEFINITION_CONDITION_IN       = "in"       // 要求在
	POLICY_DEFINITION_CONDITION_NOT_IN   = "not_in"   // 要求不在
	POLICY_DEFINITION_CONDITION_CONTAINS = "contains" // 要求包含
	POLICY_DEFINITION_CONDITION_EXCEPT   = "except"   // 要求不包含

	POLICY_DEFINITION_CONDITION_IN_USE  = "in_use"  // 要求使用
	POLICY_DEFINITION_CONDITION_NOT_USE = "not_use" // 要求不使用
	POLICY_DEFINITION_CONDITION_LE      = "le"      // 要求小于或等于
)

var (
	POLICY_CONDITIONS = map[string][]string{
		POLICY_DEFINITION_CATEGORY_CLOUDREGION:  []string{POLICY_DEFINITION_CONDITION_IN, POLICY_DEFINITION_CONDITION_NOT_IN},
		POLICY_DEFINITION_CATEGORY_TAG:          []string{POLICY_DEFINITION_CONDITION_CONTAINS, POLICY_DEFINITION_CONDITION_EXCEPT},
		POLICY_DEFINITION_CATEGORY_EXPIRED:      []string{POLICY_DEFINITION_CONDITION_IN_USE, POLICY_DEFINITION_CONDITION_LE},
		POLICY_DEFINITION_CATEGORY_BILLING_TYPE: []string{POLICY_DEFINITION_CONDITION_IN_USE, POLICY_DEFINITION_CONDITION_NOT_USE},
		POLICY_DEFINITION_CATEGORY_BATCH_CREATE: []string{POLICY_DEFINITION_CONDITION_LE},
	}
)

type PolicyDefinitionListInput struct {
	apis.StatusStandaloneResourceListInput
	ManagedResourceListInput
}

type SPolicyDefinitionParameters struct {
	Cloudregions []SCloudregionPolicyDefinition
	Tags         []string
	Duration     string
	BillingType  string
	Count        *int
}

type PolicyDefinitionCreateInput struct {
	apis.StatusStandaloneResourceCreateInput
	Condition string
	Category  string

	// swagger:ignore
	Parameters SPolicyDefinitionParameters

	Cloudregions []string
	Tags         []string
	Duration     string
	BillingType  string
	Count        int

	Domains []string `json:"domains"`
}

type PolicyDefinitionDetails struct {
	apis.StatusStandaloneResourceDetails

	SPolicyDefinition
}

type PolicyDefinitionResourceInfo struct {
	Policydefinition string
}

type SCloudregionPolicyDefinition struct {
	Id   string
	Name string
}

type SCloudregionPolicyDefinitions struct {
	Cloudregions []SCloudregionPolicyDefinition
}

type PolicyDefinitionSyncstatusInput struct {
}
