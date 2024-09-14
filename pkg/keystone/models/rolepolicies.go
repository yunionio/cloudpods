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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
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
	Auth tristate.TriState `default:"true" list:"domain" create:"domain_optional"`
	// 匹配的IP白名单
	Ips string `list:"domain" create:"domain_optional" update:"domain"`
	// 匹配开始时间
	ValidSince time.Time `list:"domain" create:"domain_optional" update:"domain"`
	// 匹配结束时间
	ValidUntil time.Time `list:"domain" create:"domain_optional" update:"domain"`
}

func (manager *SRolePolicyManager) newRecord(ctx context.Context, roleId, projectId, policyId string, auth tristate.TriState, ips []netutils.IPV4Prefix, validSince, validUntil time.Time) error {
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
	rpg.ValidSince = validSince
	rpg.ValidUntil = validUntil
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

func (manager *SRolePolicyManager) NamespaceScope() rbacscope.TRbacScope {
	return PolicyManager.NamespaceScope()
}

func (manager *SRolePolicyManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	policyQ := PolicyManager.Query()
	policyQ = PolicyManager.FilterByOwner(ctx, policyQ, PolicyManager, userCred, owner, scope)
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
			role, err := RoleManager.FetchByIdOrName(ctx, userCred, query.RoleIds[i])
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
		project, err := ProjectManager.FetchByIdOrName(ctx, userCred, query.ProjectId)
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
		policy, err := PolicyManager.FetchByIdOrName(ctx, userCred, query.PolicyId)
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

func (manager *SRolePolicyManager) getMatchPolicyIds(userCred rbacutils.IRbacIdentity, tm time.Time) ([]string, error) {
	isGuest := true
	if userCred != nil && len(userCred.GetProjectId()) > 0 && len(userCred.GetRoleIds()) > 0 && !auth.IsGuestToken(userCred) {
		isGuest = false
	}
	return manager.getMatchPolicyIds2(isGuest, userCred.GetRoleIds(), userCred.GetProjectId(), userCred.GetLoginIp(), tm)
}

func (manager *SRolePolicyManager) getMatchPolicyIds2(isGuest bool, roleIds []string, pid string, loginIp string, tm time.Time) ([]string, error) {
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
		if !tm.IsZero() && !rp.MatchTime(tm) {
			continue
		}
		policyIds = stringutils2.Append(policyIds, rp.PolicyId)
	}
	return policyIds, nil
}

func appendPolicy(ctx context.Context, names map[rbacscope.TRbacScope][]string, policies rbacutils.TPolicyGroup, scope rbacscope.TRbacScope, policyName string, nameOnly bool) (map[rbacscope.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	if utils.IsInStringArray(policyName, names[scope]) {
		return names, policies, nil
	}
	policyObj, err := PolicyManager.FetchByName(ctx, nil, policyName)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, nil, errors.Wrapf(err, "FetchPolicy %s", policyName)
	}
	if policyObj == nil {
		// empty policy
		return names, policies, nil
	}
	names[scope] = append(names[scope], policyName)
	if !nameOnly {
		data, err := policyObj.(*SPolicy).getPolicy()
		if err != nil {
			return nil, nil, errors.Wrap(err, "getPolicy")
		}
		policies[scope] = append(policies[scope], *data)
	}
	return names, policies, nil
}

func (manager *SRolePolicyManager) GetMatchPolicyGroup(userCred rbacutils.IRbacIdentity, tm time.Time, nameOnly bool) (map[rbacscope.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	policyIds, err := manager.getMatchPolicyIds(userCred, tm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getMatchPolicyIds")
	}
	return manager.GetPolicyGroupByIds(policyIds, nameOnly)
}

type sUserProjectPair struct {
	userId    string
	projectId string
}

func (p sUserProjectPair) GetProjectId() string {
	return p.projectId
}

func (p sUserProjectPair) GetRoleIds() []string {
	if len(p.userId) == 0 || len(p.projectId) == 0 {
		return nil
	}
	roles, _ := AssignmentManager.FetchUserProjectRoles(p.userId, p.projectId)
	ret := make([]string, len(roles))
	for i := range roles {
		ret[i] = roles[i].Id
	}
	return ret
}

func (p sUserProjectPair) GetUserId() string {
	return p.userId
}

func (p sUserProjectPair) GetLoginIp() string {
	return ""
}

func (p sUserProjectPair) GetTokenString() string {
	if len(p.userId) == 0 {
		return auth.GUEST_TOKEN
	}
	return p.userId
}

func (manager *SRolePolicyManager) GetMatchPolicyGroupByInput(ctx context.Context, userId, projectId string, tm time.Time, nameOnly bool) (map[rbacscope.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	return manager.GetMatchPolicyGroupByCred(
		ctx,
		sUserProjectPair{
			userId:    userId,
			projectId: projectId,
		},
		tm, nameOnly,
	)
}

func (manager *SRolePolicyManager) GetMatchPolicyGroupByCred(ctx context.Context, userCred api.IRbacIdentityWithUserId, tm time.Time, nameOnly bool) (map[rbacscope.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	names, policies, err := manager.GetMatchPolicyGroup(userCred, tm, nameOnly)
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetMatchPolicyGroup")
	}
	if !options.Options.EnableDefaultDashboardPolicy {
		return names, policies, nil
	}
	userId := userCred.GetUserId()
	if len(userId) == 0 {
		// anonymous access
		log.Debugf("anomymouse accessed policies: %s", jsonutils.Marshal(names))
		return names, policies, nil
	}
	usr, err := UserManager.fetchUserById(userId)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "user id %s of userCred not found", userId)
	}
	// append dashboard policy only when there are matched policies
	if len(names) > 0 && usr.AllowWebConsole.IsTrue() {
		// add web console policy
		for scope := range names {
			consolePolicyName := ""
			switch scope {
			case rbacscope.ScopeSystem:
				consolePolicyName = options.Options.SystemDashboardPolicy
			case rbacscope.ScopeDomain:
				consolePolicyName = options.Options.DomainDashboardPolicy
			case rbacscope.ScopeProject:
				consolePolicyName = options.Options.ProjectDashboardPolicy
			}
			if len(consolePolicyName) == 0 {
				continue
			}
			names, policies, err = appendPolicy(ctx, names, policies, scope, consolePolicyName, nameOnly)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "appendConsolePolicy %s %s", scope, consolePolicyName)
			}
		}
	}
	return names, policies, nil
}

func (manager *SRolePolicyManager) GetMatchPolicyGroup2(isGuest bool, roleIds []string, pid string, loginIp string, tm time.Time, nameOnly bool) (map[rbacscope.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	policyIds, err := manager.getMatchPolicyIds2(isGuest, roleIds, pid, loginIp, tm)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getMatchPolicyIds")
	}
	return manager.GetPolicyGroupByIds(policyIds, nameOnly)
}

func (manager *SRolePolicyManager) GetPolicyGroupByIds(policyIds []string, nameOnly bool) (map[rbacscope.TRbacScope][]string, rbacutils.TPolicyGroup, error) {
	names := make(map[rbacscope.TRbacScope][]string)
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
				group[policy.Scope] = rbacutils.TPolicySet{*data}
			} else {
				group[policy.Scope] = append(set, *data)
			}
		}
	}
	return names, group, nil
}

func (rp *SRolePolicy) MatchIP(ipstr string) bool {
	result := rbacutils.MatchIPStrings(rp.Ips, ipstr)
	return result
}

func (rp *SRolePolicy) MatchTime(tm time.Time) bool {
	if !rp.ValidSince.IsZero() && tm.Before(rp.ValidSince) {
		return false
	}
	if !rp.ValidUntil.IsZero() && tm.After(rp.ValidUntil) {
		return false
	}
	return true
}

func (manager *SRolePolicyManager) fetchByRoleId(roleId string) ([]SRolePolicy, error) {
	q := manager.Query().Equals("role_id", roleId)
	rps := make([]SRolePolicy, 0)
	err := db.FetchModelObjects(manager, q, &rps)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return rps, nil
}

func (manager *SRolePolicyManager) fetchByPolicyId(policyId string) ([]SRolePolicy, error) {
	q := manager.Query().Equals("policy_id", policyId)
	rps := make([]SRolePolicy, 0)
	err := db.FetchModelObjects(manager, q, &rps)
	if err != nil {
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

func (manager *SRolePolicyManager) isBootstrapRolePolicy() (bool, error) {
	cnt, err := manager.Query().CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "CountWithError")
	}
	return (cnt == 0), nil
}
