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

	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	allowResult = jsonutils.NewString("allow")
	denyResult  = jsonutils.NewString("deny")

	editorAction = getEditActionPolicy()
	viewerAction = getViewerActionPolicy()
)

func getAdminPolicy(services map[string][]string) jsonutils.JSONObject {
	policy := jsonutils.NewDict()
	for k, resList := range services {
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

func getEditActionPolicy() jsonutils.JSONObject {
	p := jsonutils.NewDict()
	p.Add(denyResult, "create")
	p.Add(denyResult, "delete")
	perform := jsonutils.NewDict()
	perform.Add(denyResult, "purge")
	perform.Add(denyResult, "clone")
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
	for k, resList := range services {
		if len(resList) == 0 {
			resList = []string{"*"}
		}
		resPolicy := jsonutils.NewDict()
		for i := range resList {
			resPolicy.Add(editorAction, resList[i])
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
			resPolicy.Add(viewerAction, resList[i])
		}
		policy.Add(resPolicy, k)
	}
	return policy
}

func addExtraPolicy(policy *jsonutils.JSONDict, extra map[string]map[string][]string) jsonutils.JSONObject {
	for s, resources := range extra {
		resourcePolicy := jsonutils.NewDict()
		for r, actions := range resources {
			actionPolicy := jsonutils.NewDict()
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
		for _, scope := range []rbacutils.TRbacScope{
			rbacutils.ScopeSystem,
			rbacutils.ScopeDomain,
			rbacutils.ScopeProject,
		} {
			if scope.HigherEqual(def.Scope) {
				ps := generatePolicies(scope, def)
				ret = append(ret, ps...)
			}
		}
	}
	return ret
}

type SPolicyData struct {
	Name   string
	Scope  rbacutils.TRbacScope
	Policy jsonutils.JSONObject

	Description   string
	DescriptionCN string
}

func generatePolicies(scope rbacutils.TRbacScope, def sPolicyDefinition) []SPolicyData {
	level := ""
	switch scope {
	case rbacutils.ScopeSystem:
		level = "sys"
		if def.Scope == rbacutils.ScopeSystem {
			level = ""
		}
	case rbacutils.ScopeDomain:
		level = "domain"
	case rbacutils.ScopeProject:
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
		roleConfs = []sRoleConf{
			{
				name:       "admin",
				policyFunc: getAdminPolicy,
				fullNameCN: "管理",
				fullName:   "full",
			},
			{
				name:       "editor",
				policyFunc: getEditorPolicy,
				fullNameCN: "编辑/操作",
				fullName:   "editor/operator",
			},
			{
				name:       "viewer",
				policyFunc: getViewerPolicy,
				fullNameCN: "只读",
				fullName:   "read-only",
			},
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
		case rbacutils.ScopeSystem:
			descCN += "全局"
			desc += "System-level"
		case rbacutils.ScopeDomain:
			descCN += "本域内"
			desc += "Domain-level"
		case rbacutils.ScopeProject:
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
