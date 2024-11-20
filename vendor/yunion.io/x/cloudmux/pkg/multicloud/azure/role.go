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

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SCloudpolicy struct {
	Id              string
	Description     string
	DisplayName     string
	IsBuildIn       bool
	IsEnabled       bool
	ResourceScopes  []string
	TemplateId      string
	Version         string
	RolePermissions []struct {
		allowedResourceActions []string
		Condition              string
	}
	InheritsPermissionsFrom []struct {
		Id string
	}
}

func (role *SCloudpolicy) GetName() string {
	return role.DisplayName
}

func (role *SCloudpolicy) GetGlobalId() string {
	return role.Id
}

func (role *SCloudpolicy) GetDescription() string {
	return role.Description
}

func (role *SCloudpolicy) GetPolicyType() api.TPolicyType {
	if role.IsBuildIn {
		return api.PolicyTypeSystem
	}
	return api.PolicyTypeCustom
}

func (role *SCloudpolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return cloudprovider.ErrNotImplemented
}

func (role *SCloudpolicy) GetDocument() (*jsonutils.JSONDict, error) {
	return jsonutils.Marshal(role).(*jsonutils.JSONDict), nil
}

func (role *SCloudpolicy) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (cli *SAzureClient) GetRoles(name string) ([]SCloudpolicy, error) {
	ret := []SCloudpolicy{}
	filter := []string{}
	if len(name) > 0 {
		filter = append(filter, fmt.Sprintf("displayName eq '%s'", name))
	}
	params := url.Values{}
	if len(filter) > 0 {
		params.Set("$filter", strings.Join(filter, " and "))
	}
	resp, err := cli._list_v2(SERVICE_GRAPH, "rolemanagement/directory/roleDefinitions", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (cli *SAzureClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	roles, err := cli.GetRoles("")
	if err != nil {
		return nil, errors.Wrap(err, "GetRoles")
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range roles {
		ret = append(ret, &roles[i])
	}
	return ret, nil
}

func (cli *SAzureClient) AssignPolicy(objectId, roleId string) error {
	body := map[string]interface{}{
		"roleDefinitionId": roleId,
		"principalId":      objectId,
		"directoryScopeId": "/",
	}
	_, err := cli._post_v2(SERVICE_GRAPH, "roleManagement/directory/roleAssignments", "", body)
	return err
}

type SPrincipalPolicy struct {
	RoleDefinitionId string
	PrincipalId      string
	Id               string
}

func (cli *SAzureClient) GetPrincipalPolicy(principalId string) ([]SPrincipalPolicy, error) {
	params := url.Values{}
	filter := []string{}
	if len(principalId) > 0 {
		filter = append(filter, fmt.Sprintf("principalId eq '%s'", principalId))
	}
	if len(filter) > 0 {
		params.Set("$filter", strings.Join(filter, " and "))
	}
	resp, err := cli._list_v2(SERVICE_GRAPH, "rolemanagement/directory/roleAssignments", "", params)
	if err != nil {
		return nil, err
	}
	ret := []SPrincipalPolicy{}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (cli *SAzureClient) DeletePrincipalPolicy(assignmentId string) error {
	res := fmt.Sprintf("roleManagement/directory/roleAssignments/%s", assignmentId)
	_, err := cli._delete_v2(SERVICE_GRAPH, res, "")
	return err
}
