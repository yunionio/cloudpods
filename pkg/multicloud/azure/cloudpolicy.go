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
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SPermission struct {
	Actions        []string
	NotActions     []string
	DataActions    []string
	NotDataActions []string
}

type SRoleProperties struct {
	RoleName         string
	Type             string
	Description      string
	AssignableScopes []string
	Permissions      []SPermission
}

type SCloudpolicy struct {
	Id         string
	Type       string
	Name       string
	Properties SRoleProperties
}

func (role *SCloudpolicy) GetName() string {
	return role.Properties.RoleName
}

func (role *SCloudpolicy) GetGlobalId() string {
	return role.Properties.RoleName
}

func (role *SCloudpolicy) GetDescription() string {
	return role.Properties.Description
}

func (role *SCloudpolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return cloudprovider.ErrNotImplemented
}

func (role *SCloudpolicy) GetDocument() (*jsonutils.JSONDict, error) {
	return jsonutils.Marshal(role.Properties).(*jsonutils.JSONDict), nil
}

func (role *SCloudpolicy) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (cli *SAzureClient) GetRoles(name, policyType string) ([]SCloudpolicy, error) {
	ret := []SCloudpolicy{}
	subscriptionId, err := cli.getDefaultSubscriptionId()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultSubscriptionId")
	}
	params := url.Values{}
	filter := []string{}
	if len(name) > 0 {
		filter = append(filter, fmt.Sprintf("roleName eq '%s'", name))
	}
	if len(policyType) > 0 {
		filter = append(filter, fmt.Sprintf("Type eq '%s'", policyType))
	}
	if len(filter) > 0 {
		params.Set("$filter", strings.Join(filter, " and "))
	}
	resource := "providers/Microsoft.Authorization/roleDefinitions"
	if len(params) > 0 {
		resource = fmt.Sprintf("%s?%s", resource, params.Encode())
	}
	err = cli.listSubscriptionResource(subscriptionId, resource, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "listSubscriptionResource")
	}
	return ret, nil
}

func (cli *SAzureClient) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := cli.GetRoles("", "BuiltInRole")
	if err != nil {
		return nil, errors.Wrap(err, "GetRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (cli *SAzureClient) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := cli.GetRoles("", "CustomRole")
	if err != nil {
		return nil, errors.Wrap(err, "GetRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (cli *SAzureClient) AssignPolicy(objectId, roleName, subscriptionId string) error {
	roles, err := cli.GetRoles(roleName, "")
	if err != nil {
		return errors.Wrapf(err, "GetRoles(%s)", roleName)
	}
	if len(roles) == 0 {
		return errors.Wrap(cloudprovider.ErrNotFound, roleName)
	}
	if len(roles) > 1 {
		return errors.Wrap(cloudprovider.ErrDuplicateId, roleName)
	}
	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"roleDefinitionId": roles[0].Id,
			"principalId":      objectId,
		},
	}
	subscriptionIds := []string{subscriptionId}
	if len(subscriptionId) == 0 {
		subscriptionIds = []string{}
		for _, subscription := range cli.subscriptions {
			subscriptionIds = append(subscriptionIds, subscription.SubscriptionId)
		}
	}
	for _, subscriptionId := range subscriptionIds {
		resource := fmt.Sprintf("subscriptions/%s/providers/Microsoft.Authorization/roleAssignments/%s", subscriptionId, stringutils.UUID4())
		err = cli.Put(resource, jsonutils.Marshal(body))
		if err != nil {
			return errors.Wrapf(err, "AssignPolicy %s for subscription %s", roleName, subscriptionId)
		}
	}
	return nil
}

type SAssignmentProperties struct {
	RoleDefinitionId string
	PrincipalId      string
	PrincipalType    string
	Scope            string
}

type SAssignment struct {
	Id         string
	Name       string
	Type       string
	Properties SAssignmentProperties
}

func (cli *SAzureClient) GetAssignments(objectId string) ([]SAssignment, error) {
	ret := []SAssignment{}
	subscriptionId, err := cli.getDefaultSubscriptionId()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultSubscriptionId")
	}
	params := url.Values{}
	if len(objectId) > 0 {
		params.Set("$filter", fmt.Sprintf("principalId eq '%s'", objectId))
	}
	resource := "providers/Microsoft.Authorization/roleAssignments"
	if len(params) > 0 {
		resource = fmt.Sprintf("%s?%s", resource, params.Encode())
	}
	err = cli.listSubscriptionResource(subscriptionId, resource, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "listSubscriptionResource")
	}
	return ret, nil
}

func (cli *SAzureClient) GetRole(roleId string) (*SCloudpolicy, error) {
	role := &SCloudpolicy{}
	err := cli.Get(roleId, nil, role)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRole(%s)", roleId)
	}
	return role, nil
}

func (cli *SAzureClient) GetCloudpolicies(objectId string) ([]SCloudpolicy, error) {
	assignments, err := cli.GetAssignments(objectId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetAssignments(%s)", objectId)
	}
	ret := []SCloudpolicy{}
	for _, assignment := range assignments {
		role, err := cli.GetRole(assignment.Properties.RoleDefinitionId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetRule(%s)", assignment.Properties.RoleDefinitionId)
		}
		ret = append(ret, *role)
	}
	return ret, nil
}
