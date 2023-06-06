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

package identity

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	OrganizationRootParent = "<-root-org-node->"
)

type TOrgType string

const (
	OrgTypeProject = TOrgType("project")
	OrgTypeDomain  = TOrgType("domain")
	OrgTypeObject  = TOrgType("object")
)

var (
	OrganizationTypes = []TOrgType{
		OrgTypeProject,
		OrgTypeDomain,
		OrgTypeObject,
	}
)

func IsValidOrgType(orgType TOrgType) bool {
	for _, t := range OrganizationTypes {
		if t == orgType {
			return true
		}
	}
	return false
}

type OrganizationListInput struct {
	apis.StandaloneResourceListInput

	Type []TOrgType `json:"type"`

	RootId string `json:"root_id"`

	RootOnly *bool `json:"root_only"`
}

type OrganizationCreateInput struct {
	apis.StandaloneResourceCreateInput

	Type TOrgType `json:"type"`

	ParentId string `json:"parent_id"`

	// swagger: ignore
	Level uint8 `json:"level,omitzero"`

	// swagger: ignore
	RootId string `json:"root_id"`

	Info *SOrganizationInfo `json:"info,omitempty"`
}

type SOrganizationInfo struct {
	Keys []string          `json:"levels,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

func (info *SOrganizationInfo) IsZero() bool {
	return len(info.Keys) == 0 && len(info.Tags) == 0
}

func (info *SOrganizationInfo) String() string {
	return jsonutils.Marshal(info).String()
}

type OrganizationUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput
}

type OrganizationPerformAddLevelsInput struct {
	Keys []string `json:"keys" help:"add keys"`
}

type OrganizationPerformBindInput struct {
	TargetId []string `json:"target_id"`

	ResourceType string `json:"resource_type"`
}
