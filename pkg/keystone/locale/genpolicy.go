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

package locale

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
)

var (
	allowResult = jsonutils.NewString("allow")
	denyResult  = jsonutils.NewString("deny")
)

func getAdminPolicy(services map[string][]string) jsonutils.JSONObject {
	policy := jsonutils.NewDict()
	for k := range services {
		resList := services[k]
		if len(resList) == 0 {
			policy.Add(allowResult, k)
		} else {
			resPolicy := jsonutils.NewDict()
			for i := range resList {
				resPolicy.Add(allowResult, resList[i])
			}
			policy.Add(resPolicy, k)
		}
	}
	return policy
}

func getEditActionPolicy(service, resource string) jsonutils.JSONObject {
	p := jsonutils.NewDict()
	p.Add(denyResult, "create")
	p.Add(denyResult, "delete")
	perform := jsonutils.NewDict()
	perform.Add(denyResult, "purge")
	perform.Add(denyResult, "clone")
	perform.Add(denyResult, "disable")
	if resActions, ok := adminPerformActions[service]; ok {
		if actions, ok := resActions[resource]; ok {
			for _, action := range actions {
				perform.Add(denyResult, action)
			}
		}
	}
	perform.Add(allowResult, "*")
	p.Add(perform, "perform")
	p.Add(allowResult, "*")
	return p
}

func getViewerActionPolicy() jsonutils.JSONObject {
	p := jsonutils.NewDict()
	p.Add(allowResult, "get")
	p.Add(allowResult, "list")
	p.Add(denyResult, "*")
	return p
}

func getEditorPolicy(services map[string][]string) jsonutils.JSONObject {
	policy := jsonutils.NewDict()
	if len(services) == 1 {
		for k := range services {
			if k == "*" {
				ns := make(map[string][]string)
				ns[k] = services[k]
				// expand adminPerformActions
				for s, resActions := range adminPerformActions {
					resList := make([]string, 0, len(resActions)+1)
					resList = append(resList, "*")
					for res := range resActions {
						resList = append(resList, res)
					}
					ns[s] = resList
				}
				services = ns
			}
		}
	}
	for k, resList := range services {
		resPolicy := jsonutils.NewDict()
		if len(resList) == 0 {
			resList = []string{"*"}
		}
		if len(resList) == 1 && resList[0] == "*" {
			if resActions, ok := adminPerformActions[k]; ok {
				for res := range resActions {
					resList = append(resList, res)
				}
			}
		}
		for i := range resList {
			resPolicy.Add(getEditActionPolicy(k, resList[i]), resList[i])
		}
		policy.Add(resPolicy, k)
	}
	return policy
}

func getViewerPolicy(services map[string][]string) jsonutils.JSONObject {
	policy := jsonutils.NewDict()
	for k, resList := range services {
		if len(resList) == 0 {
			resList = []string{"*"}
		}
		resPolicy := jsonutils.NewDict()
		for i := range resList {
			resPolicy.Add(getViewerActionPolicy(), resList[i])
		}
		policy.Add(resPolicy, k)
	}
	return policy
}

func addExtraPolicy(policy *jsonutils.JSONDict, extra map[string]map[string][]string) jsonutils.JSONObject {
	for s, resources := range extra {
		var resourcePolicy *jsonutils.JSONDict
		resPolicy, _ := policy.Get(s)
		if resPolicy != nil {
			resourcePolicy = resPolicy.(*jsonutils.JSONDict)
		} else {
			resourcePolicy = jsonutils.NewDict()
		}
		for r, actions := range resources {
			var actionPolicy *jsonutils.JSONDict
			actPolicy, _ := resourcePolicy.Get(r)
			if actPolicy != nil {
				actionPolicy = actPolicy.(*jsonutils.JSONDict)
			} else {
				actionPolicy = jsonutils.NewDict()
			}
			for i := range actions {
				actionPolicy.Add(allowResult, actions[i])
			}
			actionPolicy.Add(denyResult, "*")
			resourcePolicy.Add(actionPolicy, r)
		}
		policy.Add(resourcePolicy, s)
	}
	return policy
}

func GenerateAllPolicies() []SPolicyData {
	ret := make([]SPolicyData, 0)
	for i := range policyDefinitons {
		def := policyDefinitons[i]
		for _, scope := range []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
			rbacscope.ScopeDomain,
			rbacscope.ScopeProject,
		} {
			if scope.HigherEqual(def.Scope) {
				ps := generatePolicies(scope, def)
				ret = append(ret, ps...)
			}
		}
	}
	ret = append(ret, predefinedPolicyData...)
	return ret
}

type SPolicyData struct {
	Name   string
	Scope  rbacscope.TRbacScope
	Policy jsonutils.JSONObject

	Description   string
	DescriptionCN string

	AvailableRoles []string
}

func generatePolicies(scope rbacscope.TRbacScope, def sPolicyDefinition) []SPolicyData {
	level := ""
	switch scope {
	case rbacscope.ScopeSystem:
		level = "sys"
		if def.Scope == rbacscope.ScopeSystem {
			level = ""
		}
	case rbacscope.ScopeDomain:
		level = "domain"
	case rbacscope.ScopeProject:
		level = "project"
	}

	type sRoleConf struct {
		name       string
		policyFunc func(services map[string][]string) jsonutils.JSONObject
		fullName   string
		fullNameCN string
	}

	var roleConfs []sRoleConf
	if len(def.Services) > 0 {
		if len(def.AvailableRoles) == 0 || utils.IsInStringArray("admin", def.AvailableRoles) {
			roleConfs = append(roleConfs, sRoleConf{
				name:       "admin",
				policyFunc: getAdminPolicy,
				fullNameCN: "管理",
				fullName:   "full",
			})
		}
		if len(def.AvailableRoles) == 0 || utils.IsInStringArray("editor", def.AvailableRoles) {
			roleConfs = append(roleConfs, sRoleConf{
				name:       "editor",
				policyFunc: getEditorPolicy,
				fullNameCN: "编辑/操作",
				fullName:   "editor/operator",
			})
		}
		if len(def.AvailableRoles) == 0 || utils.IsInStringArray("viewer", def.AvailableRoles) {
			roleConfs = append(roleConfs, sRoleConf{
				name:       "viewer",
				policyFunc: getViewerPolicy,
				fullNameCN: "只读",
				fullName:   "read-only",
			})
		}
	} else {
		roleConfs = []sRoleConf{
			{
				name:       "",
				policyFunc: nil,
				fullNameCN: "",
				fullName:   "",
			},
		}
	}

	ret := make([]SPolicyData, 0)
	for _, role := range roleConfs {
		nameSegs := make([]string, 0)
		if len(level) > 0 {
			nameSegs = append(nameSegs, level)
		}
		if len(def.Name) > 0 {
			nameSegs = append(nameSegs, def.Name)
		}
		if len(role.name) > 0 {
			nameSegs = append(nameSegs, role.name)
		}
		name := strings.Join(nameSegs, "-")
		if name == "sys-admin" {
			name = "sysadmin"
		}
		var policy jsonutils.JSONObject
		if def.Services != nil {
			policy = role.policyFunc(def.Services)
		} else {
			policy = jsonutils.NewDict()
		}
		policy = addExtraPolicy(policy.(*jsonutils.JSONDict), def.Extra)
		desc := ""
		descCN := ""
		switch scope {
		case rbacscope.ScopeSystem:
			descCN += "全局"
			desc += "System-level"
		case rbacscope.ScopeDomain:
			descCN += "本域内"
			desc += "Domain-level"
		case rbacscope.ScopeProject:
			descCN += "本项目内"
			desc += "Project-level"
		}
		if len(role.fullName) > 0 {
			desc += " " + role.fullName
		}
		desc += " previlliges for"
		if len(def.Desc) > 0 {
			desc += " " + def.Desc
		}
		if len(def.DescCN) > 0 {
			descCN += def.DescCN
		}
		if len(role.fullNameCN) > 0 {
			descCN += role.fullNameCN
		}
		descCN += "权限"
		policyJson := jsonutils.NewDict()
		policyJson.Add(policy, "policy")
		ret = append(ret, SPolicyData{
			Name:   name,
			Scope:  scope,
			Policy: policyJson,

			Description:   strings.TrimSpace(desc),
			DescriptionCN: strings.TrimSpace(descCN),
		})
	}
	return ret
}
