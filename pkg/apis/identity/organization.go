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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	OrganizationLabelSeparator = "/"

	OrganizationRootParent = "<-root-org-node->"

	OrganizationStatusInit       = "init"
	OrganizationStatusReady      = "ready"
	OrganizationStatusSync       = "sync"
	OrganizationStatusSyncFailed = "sync_failed"
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
	apis.EnabledStatusInfrasResourceBaseListInput

	Type []TOrgType `json:"type"`

	Key string `json:"key"`
}

type OrganizationCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	Type TOrgType `json:"type"`

	// swagger: ignore
	Level int `json:"level,omitzero"`

	// key
	Key []string `json:"key"`

	// keys
	// swagger: ignore
	Keys string `json:"keys"`
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
	apis.EnabledStatusInfrasResourceBaseUpdateInput
}

type OrganizationPerformAddLevelsInput struct {
	Key []string `json:"key" help:"add keys"`
}

type OrganizationPerformAddNodeInput struct {
	Tags   map[string]string
	Weight int
}

type OrganizationPerformSyncInput struct {
	ResourceType string

	Reset *bool
}

type OrganizationNodePerformBindInput struct {
	TargetId []string `json:"target_id"`

	ResourceType string `json:"resource_type"`
}

func IsValidLabel(val string) bool {
	return !strings.Contains(val, OrganizationLabelSeparator)
}

func trimLabel(label string) string {
	return strings.Trim(label, OrganizationLabelSeparator+" ")
}

func JoinLabels(seg ...string) string {
	if len(seg) == 0 {
		return ""
	}
	buf := strings.Builder{}
	buf.WriteString(trimLabel(seg[0]))
	for _, s := range seg[1:] {
		buf.WriteString(OrganizationLabelSeparator)
		buf.WriteString(trimLabel(s))
	}
	return buf.String()
}

func SplitLabel(label string) []string {
	parts := strings.Split(label, OrganizationLabelSeparator)
	ret := make([]string, 0)
	for _, p := range parts {
		p = trimLabel(p)
		if len(p) > 0 {
			ret = append(ret, p)
		}
	}
	return ret
}

type OrganizationNodeUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	Weight *int `json:"weight"`
}

type OrganizationNodeListInput struct {
	apis.StandaloneResourceListInput

	OrgId string `json:"org_id"`

	OrgType TOrgType `json:"org_type"`

	Level int `json:"level"`
}

type SProjectOrganization struct {
	Id    string
	Name  string
	Keys  []string
	Nodes []SProjectOrganizationNode
}

type SProjectOrganizationNode struct {
	Id     string
	Labels []string
}
