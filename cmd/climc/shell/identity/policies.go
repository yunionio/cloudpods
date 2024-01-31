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
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/identity"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func createPolicy(s *mcclient.ClientSession, name string, genName string, policy string, domain string, enabled bool, disabled bool, desc string, scope string, isSystem *bool, objectags, projecttags, domaintags tagutils.TTagSet, orgNodeId []string) error {
	params := jsonutils.NewDict()
	if len(genName) > 0 {
		params.Add(jsonutils.NewString(genName), "generate_name")
		params.Add(jsonutils.NewString(genName), "type")
	} else if len(name) > 0 {
		params.Add(jsonutils.NewString(name), "name")
		params.Add(jsonutils.NewString(name), "type")
	} else {
		return fmt.Errorf("mising name")
	}
	params.Add(jsonutils.NewString(policy), "policy")
	if len(domain) > 0 {
		params.Add(jsonutils.NewString(domain), "project_domain_id")
	}
	if enabled {
		params.Add(jsonutils.JSONTrue, "enabled")
	} else if disabled {
		params.Add(jsonutils.JSONFalse, "enabled")
	}
	if len(desc) > 0 {
		params.Add(jsonutils.NewString(desc), "description")
	}
	if len(scope) > 0 {
		params.Add(jsonutils.NewString(scope), "scope")
	}
	if isSystem != nil {
		if *isSystem {
			params.Add(jsonutils.JSONTrue, "is_system")
		} else {
			params.Add(jsonutils.JSONFalse, "is_system")
		}
	}
	if len(objectags) > 0 {
		params.Add(jsonutils.Marshal(objectags), "object_tags")
	}
	if len(projecttags) > 0 {
		params.Add(jsonutils.Marshal(projecttags), "project_tags")
	}
	if len(domaintags) > 0 {
		params.Add(jsonutils.Marshal(domaintags), "domain_tags")
	}

	if len(orgNodeId) > 0 {
		params.Add(jsonutils.NewStringArray(orgNodeId), "org_node_id")
	}

	result, err := modules.Policies.Create(s, params)
	if err != nil {
		return err
	}

	printObject(result)

	return nil
}

func createPolicyFromJson(s *mcclient.ClientSession, old jsonutils.JSONObject, argsName, genName, argsDomain, argsScope string) error {
	policy, _ := old.GetString("policy")
	enabled, _ := old.Bool("enabled")
	desc, _ := old.GetString("description")
	scope, _ := old.GetString("scope")
	if len(argsScope) > 0 {
		scope = argsScope
	}
	isSystem, _ := old.Bool("is_system")
	var objectTags, projectTags, domainTags tagutils.TTagSet
	if old.Contains("object_tags") {
		objectTags = make(tagutils.TTagSet, 0)
		old.Unmarshal(&objectTags, "object_tags")
	}
	if old.Contains("project_tags") {
		projectTags = make(tagutils.TTagSet, 0)
		old.Unmarshal(&projectTags, "project_tags")
	}
	if old.Contains("domain_tags") {
		domainTags = make(tagutils.TTagSet, 0)
		old.Unmarshal(&domainTags, "domain_tags")
	}
	var nodeIds []string
	if old.Contains("org_node_id") {
		nodeIds = make([]string, 0)
		old.Unmarshal(&nodeIds, "org_node_id")
	}
	return createPolicy(s, argsName, genName, policy, argsDomain, enabled, !enabled, desc, scope, &isSystem, objectTags, projectTags, domainTags, nodeIds)
}

func init() {
	cmd := shell.NewResourceCmd(&modules.Policies)
	cmd.List(&options.PolicyListOptions{})
	cmd.Perform("user-metadata", &baseoptions.ResourceMetadataOptions{})
	cmd.Perform("set-user-metadata", &baseoptions.ResourceMetadataOptions{})
	cmd.GetProperty(&options.PolicyGetPropertyTagValuePairOptions{})
	cmd.GetProperty(&options.PolicyGetPropertyTagValueTreeOptions{})
	cmd.GetProperty(&options.PolicyGetPropertyDomainTagValuePairOptions{})
	cmd.GetProperty(&options.PolicyGetPropertyDomainTagValueTreeOptions{})

	type PolicyCreateOptions struct {
		Domain   string `help:"domain of the policy"`
		NAME     string `help:"name of the policy"`
		FILE     string `help:"path to policy file"`
		Enabled  bool   `help:"create policy enabled"`
		Disabled bool   `help:"create policy disabled"`
		Desc     string `help:"policy description"`
		Scope    string `help:"scope of policy"`
		IsSystem *bool  `help:"create system policy" negative:"no-system"`

		ProjectTags string `help:"project tags"`
		DomainTags  string `help:"domain tags"`
		ObjectTags  string `help:"object tags"`

		OrgNodeId []string `help:"node ids of organiazation tree node"`
	}
	R(&PolicyCreateOptions{}, "policy-create", "Create a new policy", func(s *mcclient.ClientSession, args *PolicyCreateOptions) error {
		policyBytes, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}

		objectTags := baseoptions.SplitTag(args.ObjectTags)
		projectTags := baseoptions.SplitTag(args.ProjectTags)
		domainTags := baseoptions.SplitTag(args.DomainTags)
		return createPolicy(s, args.NAME, "", string(policyBytes), args.Domain, args.Enabled, args.Disabled, args.Desc, args.Scope, args.IsSystem, objectTags, projectTags, domainTags, args.OrgNodeId)
	})

	type PolicyExportOptions struct {
		ID string `json:"id"`
	}
	R(&PolicyExportOptions{}, "policy-export", "Export a policy", func(s *mcclient.ClientSession, args *PolicyExportOptions) error {
		ret, err := modules.Policies.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		fmt.Println(ret.PrettyString())
		return nil
	})

	type PolicyImportOptions struct {
		FILE   string `json:"id"`
		Domain string `help:"domain of the policy"`
	}
	R(&PolicyImportOptions{}, "policy-import", "Import a policy", func(s *mcclient.ClientSession, args *PolicyImportOptions) error {
		cont, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}
		jsonCont, err := jsonutils.Parse(cont)
		if err != nil {
			return err
		}
		name, _ := jsonCont.GetString("name")
		return createPolicyFromJson(s, jsonCont, "", name, args.Domain, "")
	})

	type PolicyCloneOptions struct {
		Scope  string `help:"scope of policy"`
		Domain string `help:"domain of the policy"`
		OLD    string `help:"name or id of the old policy"`
		NAME   string `help:"name of the policy"`
	}
	R(&PolicyCloneOptions{}, "policy-clone", "Clone a policy", func(s *mcclient.ClientSession, args *PolicyCloneOptions) error {
		old, err := modules.Policies.Get(s, args.OLD, nil)
		if err != nil {
			return err
		}
		return createPolicyFromJson(s, old, args.NAME, "", args.Domain, args.Scope)
	})

	type PolicyPatchOptions struct {
		ID       string `help:"ID of policy"`
		File     string `help:"path to policy file"`
		Type     string `help:"policy type"`
		Enabled  bool   `help:"update policy enabled"`
		Disabled bool   `help:"update policy disabled"`
		Desc     string `help:"Description"`
		IsSystem bool   `help:"is_system"`

		IsNotSystem bool `help:"negative is_system"`

		TagsAction string `help:"how to update tags" choices:"add|remove|replace"`

		ProjectTags string `help:"project tags"`
		DomainTags  string `help:"domain tags"`
		ObjectTags  string `help:"object tags"`

		OrgNodeId []string `help:"node ids of organiazation tree node"`
	}
	updateFunc := func(s *mcclient.ClientSession, args *PolicyPatchOptions) error {
		policyId, err := modules.Policies.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.File) > 0 {
			policyBytes, err := ioutil.ReadFile(args.File)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(string(policyBytes)), "policy")
		}
		if args.Enabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.IsSystem {
			params.Add(jsonutils.JSONTrue, "is_system")
		}
		if args.IsNotSystem {
			params.Add(jsonutils.JSONFalse, "is_system")
		}

		if len(args.ObjectTags) > 0 {
			tags := baseoptions.SplitTag(args.ObjectTags)
			params.Add(jsonutils.Marshal(tags), "object_tags")
		}
		if len(args.ProjectTags) > 0 {
			tags := baseoptions.SplitTag(args.ProjectTags)
			params.Add(jsonutils.Marshal(tags), "project_tags")
		}
		if len(args.DomainTags) > 0 {
			tags := baseoptions.SplitTag(args.DomainTags)
			params.Add(jsonutils.Marshal(tags), "domain_tags")
		}

		if len(args.TagsAction) > 0 {
			params.Add(jsonutils.NewString(args.TagsAction), "tag_update_policy")
		}

		if len(args.OrgNodeId) > 0 {
			params.Add(jsonutils.NewStringArray(args.OrgNodeId), "org_node_id")
		}

		result, err := modules.Policies.Update(s, policyId, params)
		if err != nil {
			return err
		}

		printObject(result)

		return nil
	}
	R(&PolicyPatchOptions{}, "policy-patch", "Patch policy", updateFunc)
	R(&PolicyPatchOptions{}, "policy-update", "Update policy", updateFunc)

	type PolicyPublicOptions struct {
		ID            string   `help:"ID of policy to update" json:"-"`
		Scope         string   `help:"sharing scope" choices:"system|domain"`
		SharedDomains []string `help:"share to domains"`
	}
	R(&PolicyPublicOptions{}, "policy-public", "Mark a policy public", func(s *mcclient.ClientSession, args *PolicyPublicOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Policies.PerformAction(s, args.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PolicyPrivateOptions struct {
		ID string `help:"ID of policy to update" json:"-"`
	}
	R(&PolicyPrivateOptions{}, "policy-private", "Mark a policy private", func(s *mcclient.ClientSession, args *PolicyPrivateOptions) error {
		result, err := modules.Policies.PerformAction(s, args.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PolicyDeleteOptions struct {
		ID string `help:"ID of policy"`
	}
	R(&PolicyDeleteOptions{}, "policy-delete", "Delete policy", func(s *mcclient.ClientSession, args *PolicyDeleteOptions) error {
		policyId, err := modules.Policies.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.Policies.Delete(s, policyId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PolicyShowOptions struct {
		ID     string `help:"ID of policy"`
		Format string `help:"policy format, default yaml" default:"yaml" choices:"yaml|json"`
		Save   string `help:"save policy data into a file"`
	}
	R(&PolicyShowOptions{}, "policy-show", "Show policy", func(s *mcclient.ClientSession, args *PolicyShowOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString(args.Format), "format")
		result, err := modules.Policies.Get(s, args.ID, query)
		if err != nil {
			return err
		}
		printObject(result)
		if len(args.Save) > 0 {
			if args.Format == "yaml" {
				yaml, _ := result.GetString("policy")
				fileutils2.FilePutContents(args.Save, yaml, false)
			} else {
				p, _ := result.Get("policy")
				fileutils2.FilePutContents(args.Save, p.PrettyString(), false)
			}
		}
		return nil
	})

	type PolicyEditOptions struct {
		ID string `help:"ID of policy"`
	}
	R(&PolicyEditOptions{}, "policy-edit", "Edit and update policy", func(s *mcclient.ClientSession, args *PolicyEditOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString("yaml"), "format")
		result, err := modules.Policies.Get(s, args.ID, query)
		if err != nil {
			return err
		}
		policyId, err := result.GetString("id")
		if err != nil {
			return err
		}
		yaml, err := result.GetString("policy")
		if err != nil {
			return err
		}

		yaml, err = shellutils.Edit(yaml)
		if err != nil {
			return err
		}

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(yaml), "policy")

		result, err = modules.Policies.Patch(s, policyId, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type PolicyBindRoleOptions struct {
		POLICY string `json:"-" help:"policy Id or name"`
		api.PolicyBindRoleInput
	}
	R(&PolicyBindRoleOptions{}, "policy-bind-role", "Policy bind role", func(s *mcclient.ClientSession, args *PolicyBindRoleOptions) error {
		params := jsonutils.Marshal(args)
		log.Debugf("params: %s", params)
		result, err := modules.Policies.PerformAction(s, args.POLICY, "bind-role", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PolicyAdminCapableOptions struct {
		User          string   `help:"For user"`
		UserDomain    string   `help:"Domain for user"`
		Project       string   `help:"Role assignments for project"`
		ProjectDomain string   `help:"Domain for project"`
		Role          []string `help:"Role name list"`
		RoleId        []string `help:"Role Id list"`
	}
	R(&PolicyAdminCapableOptions{}, "policy-admin-capable", "Check admin capable", func(s *mcclient.ClientSession, args *PolicyAdminCapableOptions) error {
		auth.InitFromClientSession(s)
		policy.EnableGlobalRbac(15*time.Second, false, 1)

		var token mcclient.TokenCredential
		if len(args.User) > 0 {
			token = &mcclient.SSimpleToken{
				Domain:        args.UserDomain,
				User:          args.User,
				Project:       args.Project,
				ProjectDomain: args.ProjectDomain,
				Roles:         strings.Join(args.Role, ","),
				RoleIds:       strings.Join(args.RoleId, ","),
			}
		} else {
			token = s.GetToken()
		}

		fmt.Println("Token", token)

		for _, scope := range []rbacscope.TRbacScope{
			rbacscope.ScopeSystem,
			rbacscope.ScopeDomain,
			rbacscope.ScopeProject,
			rbacscope.ScopeUser,
			rbacscope.ScopeNone,
		} {
			fmt.Printf("%s: ", scope)
			capable := policy.PolicyManager.IsScopeCapable(token, scope)
			fmt.Println(capable)
		}

		return nil
	})

	type PolicyExplainOptions struct {
		User          string   `help:"For user"`
		UserDomain    string   `help:"Domain for user"`
		Project       string   `help:"Role assignments for project"`
		Role          []string `help:"Roles"`
		RoleId        []string `help:"Role Id list"`
		Request       []string `help:"explain request, in format of key:scope:service:resource:action:extra"`
		Name          string   `help:"policy name"`
		Debug         bool     `help:"enable RBAC debug"`
		ProjectDomain string   `help:"Domain name"`
		Ip            string   `help:"login IP"`
	}
	R(&PolicyExplainOptions{}, "policy-explain", "Explain policy result", func(s *mcclient.ClientSession, args *PolicyExplainOptions) error {
		err := log.SetLogLevelByString(log.Logger(), "debug")
		if err != nil {
			log.Fatalf("Set log level %q: %v", "debug", err)
		}
		if args.Debug {
			rbacutils.ShowMatchRuleDebug = true
		}
		auth.InitFromClientSession(s)
		policy.EnableGlobalRbac(15*time.Second, false, 1)
		if args.Debug {
			consts.EnableRbacDebug()
		}

		req := jsonutils.NewDict()
		for i := 0; i < len(args.Request); i += 1 {
			parts := strings.Split(args.Request[i], ":")
			if len(parts) < 3 {
				return fmt.Errorf("invalid request, should be in the form of key:[system|domain|project]:service[:resource:action:extra]")
			}
			key := parts[0]
			data := make([]jsonutils.JSONObject, 0)
			for i := 1; i < len(parts); i += 1 {
				data = append(data, jsonutils.NewString(parts[i]))
			}
			req.Add(jsonutils.NewArray(data...), key)
		}
		fmt.Println("Request:", req.String())

		var token mcclient.TokenCredential
		if len(args.User) > 0 {
			usrParams := jsonutils.NewDict()
			if len(args.UserDomain) > 0 {
				usrDom, err := modules.Domains.Get(s, args.UserDomain, nil)
				if err != nil {
					return fmt.Errorf("search user domain %s fail %s", args.UserDomain, err)
				}
				usrDomId, _ := usrDom.Get("id")
				usrParams.Add(usrDomId, "domain_id")
			}
			usr, err := modules.UsersV3.Get(s, args.User, usrParams)
			if err != nil {
				return fmt.Errorf("search user %s fail %s", args.User, err)
			}
			usrId, _ := usr.GetString("id")
			usrName, _ := usr.GetString("name")
			usrDomId, _ := usr.GetString("domain_id")
			usrDom, _ := usr.GetString("domain")

			projParams := jsonutils.NewDict()
			if len(args.ProjectDomain) > 0 {
				projDom, err := modules.Domains.Get(s, args.ProjectDomain, nil)
				if err != nil {
					return fmt.Errorf("search project domain %s fail %s", args.ProjectDomain, err)
				}
				projDomId, _ := projDom.Get("id")
				projParams.Add(projDomId, "domain_id")
			}
			proj, err := modules.Projects.Get(s, args.Project, projParams)
			if err != nil {
				return fmt.Errorf("search project %s fail %s", args.Project, err)
			}
			projId, _ := proj.GetString("id")
			projName, _ := proj.GetString("name")
			projDom, _ := proj.GetString("domain")
			projDomId, _ := proj.GetString("domain_id")
			token = &mcclient.SSimpleToken{
				Domain:          usrDom,
				DomainId:        usrDomId,
				User:            usrName,
				UserId:          usrId,
				Project:         projName,
				ProjectId:       projId,
				ProjectDomain:   projDom,
				ProjectDomainId: projDomId,
				Roles:           strings.Join(args.Role, ","),
				RoleIds:         strings.Join(args.RoleId, ","),
				Context: mcclient.SAuthContext{
					Ip: args.Ip,
				},
				Token: "faketoken",
			}
		} else {
			token = s.GetToken()
		}

		result, err := policy.ExplainRpc(context.Background(), token, req, args.Name)
		if err != nil {
			return err
		}
		printObject(result)

		/*for _, r := range args.Role {
			fmt.Println("role", r, "matched policies:", policy.PolicyManager.RoleMatchPolicies(r))
		}

		fmt.Println("userCred:", token)
		for _, scope := range []rbacutils.TRbacScope{
			rbacutils.ScopeSystem,
			rbacutils.ScopeDomain,
			rbacutils.ScopeProject,
			rbacutils.ScopeUser,
			rbacutils.ScopeNone,
		} {
			m := policy.PolicyManager.MatchedPolicyNames(scope, token)
			fmt.Println("matched", scope, "policies:", m)
		}

		fmt.Println("all_policies", policy.PolicyManager.AllPolicies())*/
		return nil
	})
}
