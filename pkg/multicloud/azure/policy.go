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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SPolicyDefinitonPropertieParameterMetadata struct {
	DisplayName       string
	Description       string
	StrongType        string
	AssignPermissions bool
}

type SPolicyDefinitonPropertieParameter struct {
	Type          string
	Metadata      SPolicyDefinitonPropertieParameterMetadata
	AllowedValues []string
	DefaultValue  []string
}

type SPolicyDefinitonProperties struct {
	DisplayName string
	PolicyType  string
	Mode        string
	Description string
	Metadata    SPolicyDefinitonPropertieMetadata
	Parameters  map[string]SPolicyDefinitonPropertieParameter
	PolicyRule  SPolicyDefinitonPropertieRule
}

type SPolicyDefinitonPropertieRuleThen struct {
	Effect string
}

type SPolicyDefinitonPropertieRuleInfo jsonutils.JSONDict

type SPolicyDefinitonPropertieRule struct {
	If   jsonutils.JSONObject
	Then SPolicyDefinitonPropertieRuleThen
}

type SPolicyDefinitonPropertieMetadata struct {
	Version  string
	Category string
}

type SPolicyDefinition struct {
	Properties SPolicyDefinitonProperties
	Id         string
	Name       string
	Type       string
}

func (client *SAzureClient) GetPolicyDefinitions() ([]SPolicyDefinition, error) {
	definitions := []SPolicyDefinition{}
	err := client.list("Microsoft.Authorization/policyDefinitions", url.Values{}, &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "Microsoft.Authorization/policyDefinitions.List")
	}
	return definitions, nil
}

func (client *SAzureClient) GetPolicyDefinition(id string) (*SPolicyDefinition, error) {
	definition := &SPolicyDefinition{}
	err := client.get(id, url.Values{}, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s", id)
	}
	return definition, nil
}

type PolicyAssignmentPropertiesParameter struct {
	Value []string
}

type PolicyAssignmentProperties struct {
	DisplayName string
	Parameters  map[string]PolicyAssignmentPropertiesParameter
}

type SPolicyAssignment struct {
	Id         string
	Properties PolicyAssignmentProperties
	values     []string
	category   string
	condition  string
	parameters *jsonutils.JSONDict
}

func (assignment *SPolicyAssignment) GetName() string {
	return assignment.Properties.DisplayName
}

func (assignment *SPolicyAssignment) GetGlobalId() string {
	return strings.ToLower(assignment.Id)
}

func (assignment *SPolicyAssignment) GetCategory() string {
	return assignment.category
}

func (assignment *SPolicyAssignment) GetCondition() string {
	return assignment.condition
}

func (assignment *SPolicyAssignment) GetParameters() *jsonutils.JSONDict {
	return assignment.parameters
}

func (client *SAzureClient) GetPolicyAssignments(defineId string) ([]SPolicyAssignment, error) {
	assignments := []SPolicyAssignment{}
	resource := "Microsoft.Authorization/policyAssignments"
	params := url.Values{}
	if len(defineId) > 0 {
		params.Set("$filter", fmt.Sprintf(`policyDefinitionId eq '%s'`, defineId))
	}
	err := client.list(resource, params, &assignments)
	if err != nil {
		return nil, errors.Wrap(err, "Microsoft.Authorization/policyAssignments.List")
	}
	return assignments, nil
}
