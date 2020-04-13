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
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	err := client.ListAll("Microsoft.Authorization/policyDefinitions", &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "Microsoft.Authorization/policyDefinitions.List")
	}
	return definitions, nil
}

func (client *SAzureClient) GetPolicyDefinition(id string) (*SPolicyDefinition, error) {
	definition := &SPolicyDefinition{}
	err := client.Get(id, []string{}, definition)
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
	if len(defineId) > 0 {
		resource += ("?$filter=" + url.PathEscape("policyDefinitionId eq ") + fmt.Sprintf("'%s'", defineId))
	}
	err := client.ListAll(resource, &assignments)
	if err != nil {
		return nil, errors.Wrap(err, "Microsoft.Authorization/policyAssignments.List")
	}
	return assignments, nil
}

func (client *SAzureClient) GetICloudDefinitions() ([]cloudprovider.ICloudPolicyDefinition, error) {
	ret := []cloudprovider.ICloudPolicyDefinition{}
	definitions, err := client.GetPolicyDefinitions()
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicyDefinitions")
	}
	for i := range definitions {
		if definitions[i].Properties.PolicyRule.Then.Effect != "deny" {
			continue
		}
		rule := definitions[i].Properties.PolicyRule.If
		if rule.Contains("field") {
			field, _ := rule.GetString("field")
			if field == "location" {
				defaultValue := []string{}
				locationParameter := ""
				for k, v := range definitions[i].Properties.Parameters {
					if v.Metadata.StrongType == "location" {
						defaultValue = v.DefaultValue
						locationParameter = k
						break
					}
				}
				assignments, err := client.GetPolicyAssignments(definitions[i].Id)
				if err != nil {
					return nil, errors.Wrapf(err, "GetPolicyAssignments(%s)", definitions[i].Id)
				}
				for i := range assignments {
					location, ok := assignments[i].Properties.Parameters[locationParameter]
					if ok {
						if len(location.Value) > 0 {
							assignments[i].values = location.Value
						} else {
							assignments[i].values = defaultValue
						}
					}
					regionIds := jsonutils.NewArray()
					assignments[i].parameters = jsonutils.NewDict()
					for _, value := range assignments[i].values {
						region := client.GetRegion(value)
						if region != nil {
							regionIds.Add(jsonutils.NewString(region.GetGlobalId()))
						} else {
							log.Errorf("failed to found region %s", value)
						}
					}
					assignments[i].category = api.POLICY_DEFINITION_CATEGORY_CLOUDREGION
					if rule.Contains("in") {
						assignments[i].condition = api.POLICY_DEFINITION_CONDITION_NOT_IN
					} else if rule.Contains("notIn") {
						assignments[i].condition = api.POLICY_DEFINITION_CONDITION_IN
					}
					assignments[i].parameters.Add(regionIds, "cloudregions")
					ret = append(ret, &assignments[i])
				}
			} else if strings.Contains(field, "tags") {
				reg := regexp.MustCompile(`^\[concat\('tags\[', parameters\('\w+'\), '\]'\)\]$`)
				if !reg.MatchString(field) {
					continue
				}
				if rule.Contains("exists") {
					exists, _ := rule.Bool("exists")
					defaultValue := []string{}
					tagParameter := ""
					for k, v := range definitions[i].Properties.Parameters {
						tagParameter = k
						defaultValue = v.DefaultValue
					}
					assignments, err := client.GetPolicyAssignments(definitions[i].Id)
					if err != nil {
						return nil, errors.Wrapf(err, "GetPolicyAssignments(%s)", definitions[i].Id)
					}
					for i := range assignments {
						tag, ok := assignments[i].Properties.Parameters[tagParameter]
						if ok {
							if len(tag.Value) > 0 {
								assignments[i].values = tag.Value
							} else {
								assignments[i].values = defaultValue
							}
						}
						if len(assignments[i].values) == 0 {
							continue
						}
						tags := jsonutils.NewArray()
						for _, _tag := range assignments[i].values {
							tags.Add(jsonutils.NewString(_tag))
						}
						assignments[i].parameters = jsonutils.NewDict()
						assignments[i].category = api.POLICY_DEFINITION_CATEGORY_TAG
						if exists {
							assignments[i].condition = api.POLICY_DEFINITION_CONDITION_EXCEPT
						} else {
							assignments[i].condition = api.POLICY_DEFINITION_CONDITION_CONTAINS
						}
						assignments[i].parameters.Add(tags, "tags")
						ret = append(ret, &assignments[i])
					}
				}
			}
		}
	}
	return ret, nil
}
