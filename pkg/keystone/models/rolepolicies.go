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

package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SRolePolicyManager struct {
	db.SResourceBaseManager
}

var RolePolicyManager *SRolePolicyManager

func init() {
	RolePolicyManager = &SRolePolicyManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SRolePolicy{},
			"rolepolicy_tbl",
			"rolepolicy",
			"rolepolicies",
		),
	}
	RolePolicyManager.SetVirtualObject(RolePolicyManager)
}

type SRolePolicy struct {
	db.SResourceBase

	// 角色ID, 主键
	RoleId string `width:"128" charset:"ascii" primary:"true" list:"domain" create:"domain_optional"`
	// 项目ID，主键
	ProjectId string `width:"128" charset:"ascii" primary:"true" list:"domain" create:"domain_optional"`
	// 权限ID, 主键
	PolicyId string `width:"128" charset:"ascii" primary:"true" list:"domain" create:"domain_required"`
	// 是否需要认证
	Auth tristate.TriState `nullable:"false" default:"true" list:"domain" create:"domain_optional"`
	// 匹配的IP白名单
	Ips string `list:"domain" create:"domain_optional" update:"domain"`
}

func (manager *SRolePolicyManager) newRecord(ctx context.Context, roleId, projectId, policyId string, auth tristate.TriState, ips []netutils.IPV4Prefix) error {
	if len(roleId) == 0 {
		return errors.Wrap(httperrors.ErrNotEmpty, "roleId")
	}
	if len(policyId) == 0 {
		return errors.Wrap(httperrors.ErrNotEmpty, "policyId")
	}
	rpg := SRolePolicy{}
	rpg.RoleId = roleId
	rpg.ProjectId = projectId
	rpg.PolicyId = policyId
	rpg.Auth = auth
	ipStrs := make([]string, len(ips))
	for i, ipprefix := range ips {
		ipStrs[i] = ipprefix.String()
	}
	rpg.Ips = strings.Join(ipStrs, rbacutils.IP_PREFIX_SEP)
	rpg.SetModelManager(manager, &rpg)
	err := RolePolicyManager.TableSpec().InsertOrUpdate(ctx, &rpg)
	if err != nil {
		log.Errorf("insert role policy fail %s", err)
		return errors.Wrap(err, "insert role policy")
	}
	return nil
}

func (manager *SRolePolicyManager) deleteRecord(ctx context.Context, roleId, projectId, policyId string) error {
	rpg := SRolePolicy{}
	rpg.RoleId = roleId
	rpg.ProjectId = projectId
	rpg.PolicyId = policyId
	rpg.SetModelManager(manager, &rpg)
	_, err := db.Update(&rpg, func() error {
		return rpg.MarkDelete()
	})
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (rp *SRolePolicy) GetId() string {
	return fmt.Sprintf("%s:%s:%s", rp.RoleId, rp.ProjectId, rp.PolicyId)
}

func (rp *SRolePolicy) GetName() string {
	return getRolePolicyName(rp.GetRole(), rp.GetProject(), rp.GetPolicy())
}

func getRolePolicyName(role *SRole, project *SProject, policy *SPolicy) string {
	names := make([]string, 0)
	if role != nil {
		names = append(names, role.GetName())
	}
	if project != nil {
		names = append(names, project.GetName())
	}
	if policy != nil {
		names = append(names, policy.GetName())
	}
	return strings.Join(names, "/")
}

func (rp *SRolePolicy) GetRole() *SRole {
	role, err := RoleManager.FetchById(rp.RoleId)
	if err != nil {
		log.Errorf("RoleManaget.FetchById %s fail %s", rp.RoleId, err)
		return nil
	}
	return role.(*SRole)
}

func (rp *SRolePolicy) GetProject() *SProject {
	if len(rp.ProjectId) == 0 {
		return nil
	}
	project, err := ProjectManager.FetchById(rp.ProjectId)
	if err != nil {
		log.Errorf("ProjectManager.FetchById %s fail %s", rp.ProjectId, err)
		return nil
	}
	return project.(*SProject)
}

func (rp *SRolePolicy) GetPolicy() *SPolicy {
	policy, err := PolicyManager.FetchById(rp.PolicyId)
	if err != nil {
		log.Errorf("PolicyManaget.FetchById %s fail %s", rp.PolicyId, err)
		return nil
	}
	return policy.(*SPolicy)
}

func (manager *SRolePolicyManager) NamespaceScope() rbacutils.TRbacScope {
	return PolicyManager.NamespaceScope()
}

func (manager *SRolePolicyManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	policyQ := PolicyManager.Query()
	policyQ = PolicyManager.FilterByOwner(policyQ, owner, scope)
	subq := policyQ.SubQuery()
	q = q.Join(subq, sqlchemy.Equals(q.Field("policy_id"), subq.Field("id")))
	return q
}

func (manager *SRolePolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RolePolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}
	if len(query.RoleIds) > 0 {
		for i := range query.RoleIds {
			role, err := RoleManager.FetchByIdOrName(userCred, query.RoleIds[i])
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", RoleManager.Keyword(), query.RoleIds[i])
				} else {
					return nil, errors.Wrap(err, "RoleManager.FetchByIdOrName")
				}
			}
			query.RoleIds[i] = role.GetId()
		}
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field("role_id")),
			sqlchemy.In(q.Field("role_id"), query.RoleIds),
		))
	}
	if len(query.ProjectId) > 0 {
		project, err := ProjectManager.FetchByIdOrName(userCred, query.ProjectId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", ProjectManager.Keyword(), query.ProjectId)
			} else {
				return nil, errors.Wrap(err, "ProjectManager.FetchByIdOrName")
			}
		}
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field("project_id")),
			sqlchemy.Equals(q.Field("project_id"), project.GetId()),
		))
	}
	if len(query.PolicyId) > 0 {
		policy, err := PolicyManager.FetchByIdOrName(userCred, query.PolicyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", PolicyManager.Keyword(), query.PolicyId)
			} else {
				return nil, errors.Wrap(err, "PolicyManager.FetchByIdOrName")
			}
		}
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field("policy_id")),
			sqlchemy.Equals(q.Field("policy_id"), policy.GetId()),
		))
	}
	if query.Auth != nil {
		if *query.Auth {
			q = q.IsTrue("auth")
		} else {
			q = q.IsFalse("auth")
		}
	}
	return q, nil
}

func (manager *SRolePolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RolePolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SRolePolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SRolePolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.RolePolicyDetails {
	rows := make([]api.RolePolicyDetails, len(objs))
	resRows := manager.SResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	roleIds := make([]string, 0)
	projectIds := make([]string, 0)
	policyIds := make([]string, 0)
	for i := range rows {
		rows[i] = api.RolePolicyDetails{
			ResourceBaseDetails: resRows[i],
		}
		rp := objs[i].(*SRolePolicy)
		roleIds = append(roleIds, rp.RoleId)
		projectIds = append(projectIds, rp.ProjectId)
		policyIds = append(policyIds, rp.PolicyId)
	}

	roleMap := make(map[string]SRole)
	err := db.FetchModelObjectsByIds(RoleManager, "id", roleIds, &roleMap)
	if err != nil {
		log.Errorf("db.FetchModelObjectsByIds RoleManager fail %s", err)
		return rows
	}
	projectMap := make(map[string]SProject)
	err = db.FetchModelObjectsByIds(ProjectManager, "id", projectIds, &projectMap)
	if err != nil {
		log.Errorf("db.FetchModelObjectsByIds ProjectManager fail %s", err)
		return rows
	}
	policyMap := make(map[string]SPolicy)
	err = db.FetchModelObjectsByIds(PolicyManager, "id", policyIds, &policyMap)
	if err != nil {
		log.Errorf("db.FetchModelObjectsByIds PolicyManager fail %s", err)
		return rows
	}

	for i := range rows {
		rp := objs[i].(*SRolePolicy)
		var role *SRole
		if obj, ok := roleMap[rp.RoleId]; ok {
			role = &obj
		}
		var project *SProject
		if obj, ok := projectMap[rp.ProjectId]; ok {
			project = &obj
		}
		var policy *SPolicy
		if obj, ok := policyMap[rp.PolicyId]; ok {
			policy = &obj
		}
		rows[i].Id = rp.GetId()
		rows[i].Name = getRolePolicyName(role, project, policy)
		if role != nil {
			rows[i].Role = role.GetName()
		}
		if project != nil {
			rows[i].Project = project.GetName()
		}
		if policy != nil {
			rows[i].Policy = policy.GetName()
			rows[i].Scope = policy.Scope
			rows[i].Description = policy.Description
		}
	}

	return rows
}

func (manager *SRolePolicyManager) getMatchPolicyIds(userCred rbacutils.IRbacIdentity) ([]string, error) {
	isGuest := true
	if userCred != nil && !auth.IsGuestToken(userCred) {
		isGuest = false
	}
	return manager.getMatchPolicyIds2(isGuest, userCred.GetRoleIds(), userCred.GetProjectId(), userCred.GetLoginIp())
}

func (manager *SRolePolicyManager) getMatchPolicyIds2(isGuest bool, roleIds []string, pid string, loginIp string) ([]string, error) {
	q := manager.Query()
	if !isGuest {
		if len(roleIds) > 0 {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("role_id")),
				sqlchemy.In(q.Field("role_id"), roleIds),
			))
		}
		if len(pid) > 0 {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("project_id")),
				sqlchemy.Equals(q.Field("project_id"), pid),
			))
		}
	} else {
		q = q.IsFalse("auth")
	}
	rps := make([]SRolePolicy, 0)
	err := db.FetchModelObjects(manager, q, &rps)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchPolicies")
	}
	policyIds := stringutils2.NewSortedStrings(nil)
	// filter by login IP
	for _, rp := range rps {
		if len(loginIp) > 0 && !rp.MatchIP(loginIp) {
			continue
		}
		policyIds = stringutils2.Append(policyIds, rp.PolicyId)
	}
	return policyIds, nil
}

func (manager *SRolePolicyManager) GetMatchPolicyGroup(userCred rbacutils.IRbacIdentity, nameOnly bool) (map[rbacutils.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	policyIds, err := manager.getMatchPolicyIds(userCred)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getMatchPolicyIds")
	}
	return manager.GetPolicyGroupByIds(policyIds, nameOnly)
}

func (manager *SRolePolicyManager) GetMatchPolicyGroup2(isGuest bool, roleIds []string, pid string, loginIp string, nameOnly bool) (map[rbacutils.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	policyIds, err := manager.getMatchPolicyIds2(isGuest, roleIds, pid, loginIp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getMatchPolicyIds")
	}
	return manager.GetPolicyGroupByIds(policyIds, nameOnly)
}

func (manager *SRolePolicyManager) GetPolicyGroupByIds(policyIds []string, nameOnly bool) (map[rbacutils.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	names := make(map[rbacutils.TRbacScope][]string)
	var group rbacutils.TPolicyGroup
	if !nameOnly {
		group = rbacutils.TPolicyGroup{}
	}
	for _, id := range policyIds {
		policyObj, err := PolicyManager.FetchById(id)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				continue
			}
			return nil, nil, errors.Wrapf(err, "FetchPolicy %s", id)
		}
		policy := policyObj.(*SPolicy)
		if scopeName, ok := names[policy.Scope]; !ok {
			names[policy.Scope] = []string{policy.Name}
		} else {
			names[policy.Scope] = append(scopeName, policy.Name)
		}
		if !nameOnly {
			data, err := policy.getPolicy()
			if err != nil {
				return nil, nil, errors.Wrap(err, "getPolicy")
			}
			if set, ok := group[policy.Scope]; !ok {
				group[policy.Scope] = rbacutils.TPolicySet{data}
			} else {
				group[policy.Scope] = append(set, data)
			}
		}
	}
	return names, group, nil
}

func (rp *SRolePolicy) MatchIP(ipstr string) bool {
	return rbacutils.MatchIPStrings(rp.Ips, ipstr)
}

func (manager *SRolePolicyManager) fetchByRoleId(roleId string) ([]SRolePolicy, error) {
	q := manager.Query().Equals("role_id", roleId)
	rps := make([]SRolePolicy, 0)
	err := db.FetchModelObjects(manager, q, &rps)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return rps, nil
}

func (manager *SRolePolicyManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	parts := strings.Split(idStr, ":")
	if len(parts) == 3 {
		return q.Equals("role_id", parts[0]).Equals("project_id", parts[1]).Equals("policy_id", parts[2])
	} else {
		return q.Equals("ips", idStr)
	}
}

func (manager *SRolePolicyManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	parts := strings.Split(idStr, ":")
	if len(parts) == 3 {
		return q.Filter(sqlchemy.OR(
			sqlchemy.NotEquals(q.Field("role_id"), parts[0]),
			sqlchemy.NotEquals(q.Field("project_id"), parts[1]),
			sqlchemy.NotEquals(q.Field("policy_id"), parts[2]),
		))
	} else {
		return q
	}
}

func (manager *SRolePolicyManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return manager.FilterById(q, name)
}
