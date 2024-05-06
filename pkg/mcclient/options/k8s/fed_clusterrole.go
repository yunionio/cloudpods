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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type FedClusterRoleCreateOptions struct {
	FedResourceCreateOptions
	// Spec FedClusterRoleSpec `json:"spec"`
	Rule []string `help:"role rule, e.g: 'apps/v1:deployments:get,watch,list'"`
}

type FedClusterRoleCreateInput struct {
	FedResourceCreateOptions
	Spec FedClusterRoleSpec `json:"spec"`
}

type FedClusterRoleSpec struct {
	Template ClusterRoleTemplate `json:"template"`
}

type ClusterRoleTemplate struct {
	Rules []PolicyRule `json:"rules"`
}

type PolicyRule struct {
	APIGroups []string `json:"apiGroups"`
	Resources []string `json:"resources"`
	Verbs     []string `json:"verbs"`
}

func parsePolicyRules(rules []string) ([]PolicyRule, error) {
	ret := make([]PolicyRule, 0)
	for _, rule := range rules {
		r, err := parsePolicyRule(rule)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *r)
	}
	return ret, nil
}

func parsePolicyRule(rule string) (*PolicyRule, error) {
	parts := strings.Split(rule, ":")
	if len(parts) != 3 {
		return nil, errors.Errorf("Invalid rule format: %q", rule)
	}
	groupStr := parts[0]
	resourceStr := parts[1]
	verbStr := parts[2]

	ret := &PolicyRule{
		Verbs:     []string{},
		APIGroups: []string{},
		Resources: []string{},
	}
	ret.APIGroups = append(ret.APIGroups, strings.Split(groupStr, ",")...)
	ret.Resources = append(ret.Resources, strings.Split(resourceStr, ",")...)
	ret.Verbs = append(ret.Verbs, strings.Split(verbStr, ",")...)
	return ret, nil
}

func (o *FedClusterRoleCreateOptions) Params() (jsonutils.JSONObject, error) {
	rules, err := parsePolicyRules(o.Rule)
	if err != nil {
		return nil, err
	}
	input := FedClusterRoleCreateInput{
		FedResourceCreateOptions: o.FedResourceCreateOptions,
		Spec: FedClusterRoleSpec{
			Template: ClusterRoleTemplate{
				Rules: rules,
			},
		},
	}
	return input.JSON(input), nil
}
