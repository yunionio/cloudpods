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
	POLICY_DEFINITION_STATUS_READY   = "ready"
	POLICY_DEFINITION_STATUS_UNKNOWN = "unknown"

	POLICY_DEFINITION_CATEGORY_CLOUDREGION = "cloudregion"
	POLICY_DEFINITION_CATEGORY_TAG         = "tag"

	POLICY_DEFINITION_CONDITION_IN       = "in"
	POLICY_DEFINITION_CONDITION_NOT_IN   = "not_in"
	POLICY_DEFINITION_CONDITION_CONTAINS = "contains"
	POLICY_DEFINITION_CONDITION_EXCEPT   = "except"
)

type PolicyDefinitionListInput struct {
	apis.StatusStandaloneResourceListInput
	ManagedResourceListInput
}

type PolicyDefinitionCreateInput struct {
	apis.StatusStandaloneResourceCreateInput
	SPolicyDefinition
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
