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

package k8s

import (
	"yunion.io/x/jsonutils"
)

type FedApiResourecesOptions struct{}

func (opts *FedApiResourecesOptions) GetId() string {
	return "api-resources"
}

func (opts *FedApiResourecesOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type FedClusterUsersOptions struct{}

func (opts *FedClusterUsersOptions) GetId() string {
	return "cluster-users"
}

func (opts *FedClusterUsersOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type FedClusterUserGroupsOptions struct{}

func (opts *FedClusterUserGroupsOptions) GetId() string {
	return "cluster-user-groups"
}

func (opts *FedClusterUserGroupsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type FedRoleListOptions struct {
	FedNamespaceResourceListOptions
}

type FedRoleCreateOptions struct {
	FedNamespaceResourceCreateOptions
	Rule []string `help:"role rule, e.g: 'apps/v1:deployments:get,watch,list'"`
}

type FedRoleCreateInput struct {
	FedNamespaceResourceCreateInput
	Spec FedRoleSpec `json:"spec"`
}

type FedRoleSpec struct {
	Template RoleTemplate `json:"template"`
}

type RoleTemplate struct {
	Rules []PolicyRule `json:"rules"`
}

func (o *FedRoleCreateOptions) Params() (jsonutils.JSONObject, error) {
	rules := make([]PolicyRule, 0)
	for _, rule := range o.Rule {
		ret, err := parsePolicyRule(rule)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *ret)
	}
	input := FedRoleCreateInput{
		FedNamespaceResourceCreateInput: o.FedNamespaceResourceCreateOptions.ToInput(),
		Spec: FedRoleSpec{
			Template: RoleTemplate{
				Rules: rules,
			},
		},
	}
	return input.JSON(input), nil
}
