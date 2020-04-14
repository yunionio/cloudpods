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

type PolicyDefinitionResourceListInput struct {
	Policydefinition string
}

type PolicyAssignmentListInput struct {
	apis.DomainLevelResourceListInput
	ManagedResourceListInput

	PolicyDefinitionResourceListInput
}

type PolicyAssignmentCreateInput struct {
	apis.DomainLevelResourceCreateInput

	// swagger:ignore
	PolicydefinitionId string `json:"policydefinition_id"`
	Policydefinition   string `json:"policydefinition"`
}

type SPolicyDefinitionDetails struct {
}

type PolicyAssignmentDetails struct {
	apis.DomainLevelResourceDetails
	PolicyDefinitionResourceInfo

	SPolicyAssignment
}
