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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	o "yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SUserManager struct {
	SEnabledIdentityBaseResourceManager
}

var UserManager *SUserManager

func init() {
	UserManager = &SUserManager{
		SEnabledIdentityBaseResourceManager: NewEnabledIdentityBaseResourceManager(
			SUser{},
			"user",
			"user",
			"users",
		),
	}
	UserManager.SetVirtualObject(UserManager)
	db.InitManager(func() {
		UserManager.TableSpec().ColumnSpec("lang").SetDefault(options.Options.DefaultUserLanguage)
	})
}

/*
+--------------------+-------------+------+-----+---------+-------+
| Field              | Type        | Null | Key | Default | Extra |
+--------------------+-------------+------+-----+---------+-------+
| id                 | varchar(64) | NO   | PRI | NULL    |       |
| extra              | text        | YES  |     | NULL    |       |
| enabled            | tinyint(1)  | YES  |     | NULL    |       |
| default_project_id | varchar(64) | YES  | MUL | NULL    |       |
| created_at         | datetime    | YES  |     | NULL    |       |
| last_active_at     | date        | YES  |     | NULL    |       |
| domain_id          | varchar(64) | NO   | MUL | NULL    |       |
+--------------------+-------------+------+-----+---------+-------+
*/

type SUser struct {
	SEnabledIdentityBaseResource

	Email  string `width:"64" charset:"utf8" nullable:"true" index:"true" list:"domain" update:"domain" create:"domain_optional"`
	Mobile string `width:"20" charset:"ascii" nullable:"true" index:"true" list:"domain" update:"domain" create:"domain_optional"`

	Displayname string `with:"128" charset:"utf8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	LastActiveAt time.Time `nullable:"true" list:"domain"`

	LastLoginIp     string `nullable:"true" list:"domain"`
	LastLoginSource string `nullable:"true" list:"domain"`

	IsSystemAccount tristate.TriState `nullable:"false" default:"false" list:"domain" update:"admin" create:"admin_optional"`

	// deprecated
	DefaultProjectId string `width:"64" charset:"ascii" nullable:"true"`

	AllowWebConsole tristate.TriState `nullable:"false" default:"true" list:"domain" update:"domain" create:"domain_optional"`
	EnableMfa       tristate.TriState `nullable:"false" default:"false" list:"domain" update:"domain" create:"domain_optional"`

	Lang string `width:"8" charset:"ascii" nullable:"false" list:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SUserManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{GroupManager},
		{ProjectManager},
	}
}

func (manager *SUserManager) InitializeData() error {
	q := manager.Query().IsNullOrEmpty("name")
	users := make([]SUser, 0)
	err := db.FetchModelObjects(manager, q, &users)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	for i := range users {
		extUser, err := manager.FetchUserExtended(users[i].Id, "", "", "")
		if err != nil {
			return errors.Wrap(err, "FetchUserExtended")
		}
		name := extUser.LocalName
		if len(name) == 0 {
			name = extUser.DomainName
		}
		var desc, email, mobile, dispName string
		if users[i].Extra != nil {
			desc, _ = users[i].Extra.GetString("description")
			email, _ = users[i].Extra.GetString("email")
			mobile, _ = users[i].Extra.GetString("mobile")
			dispName, _ = users[i].Extra.GetString("displayname")
		}
		_, err = db.Update(&users[i], func() error {
			users[i].Name = name
			if len(email) > 0 {
				users[i].Email = email
			}
			if len(mobile) > 0 {
				users[i].Mobile = mobile
			}
			if len(dispName) > 0 {
				users[i].Displayname = dispName
			}
			if len(desc) > 0 {
				users[i].Description = desc
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update")
		}
	}
	err = manager.initSystemAccount()
	if err != nil {
		return errors.Wrap(err, "initSystemAccount")
	}
	return manager.initSysUser(context.TODO())
}

func (manager *SUserManager) initSystemAccount() error {
	q := manager.Query().IsNotEmpty("default_project_id")
	users := make([]SUser, 0)
	err := db.FetchModelObjects(manager, q, &users)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	for i := range users {
		_, err = db.Update(&users[i], func() error {
			users[i].IsSystemAccount = tristate.True
			users[i].DefaultProjectId = ""
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update")
		}
	}
	return nil
}

func (manager *SUserManager) initSysUser(ctx context.Context) error {
	q := manager.Query().Equals("name", api.SystemAdminUser)
	q = q.Equals("domain_id", api.DEFAULT_DOMAIN_ID)
	cnt, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "query")
	}
	if cnt == 1 {
		// if ResetAdminUserPassword is true, reset sysadmin password
		if o.Options.ResetAdminUserPassword {
			usr := SUser{}
			usr.SetModelManager(manager, &usr)
			err = q.First(&usr)
			if err != nil {
				return errors.Wrap(err, "ResetAdminUserPassword Query user")
			}
			err = usr.initLocalData(o.Options.BootstrapAdminUserPassword)
			if err != nil {
				return errors.Wrap(err, "initLocalData")
			}
		}
		return nil
	}
	if cnt > 2 {
		// ???
		log.Fatalf("duplicate sysadmin account???")
	}
	// insert
	usr := SUser{}
	usr.Name = api.SystemAdminUser
	usr.DomainId = api.DEFAULT_DOMAIN_ID
	usr.Enabled = tristate.True
	usr.IsSystemAccount = tristate.True
	usr.AllowWebConsole = tristate.False
	usr.EnableMfa = tristate.False
	usr.Description = "Boostrap system default admin user"
	usr.SetModelManager(manager, &usr)

	err = manager.TableSpec().Insert(ctx, &usr)
	if err != nil {
		return errors.Wrap(err, "insert")
	}
	err = usr.initLocalData(o.Options.BootstrapAdminUserPassword)
	if err != nil {
		return errors.Wrap(err, "initLocalData")
	}
	return nil
}

/*
 Fetch extended userinfo by Id or name + domainId or name + domainName
*/
func (manager *SUserManager) FetchUserExtended(userId, userName, domainId, domainName string) (*api.SUserExtended, error) {
	if len(userId) == 0 && len(userName) == 0 {
		return nil, sqlchemy.ErrEmptyQuery
	}

	localUsers := LocalUserManager.Query().SubQuery()
	// nonlocalUsers := NonlocalUserManager.Query().SubQuery()
	users := UserManager.Query().SubQuery()
	domains := DomainManager.Query().SubQuery()
	// idmappings := IdmappingManager.Query().SubQuery()

	q := users.Query(
		users.Field("id"),
		users.Field("name"),
		users.Field("displayname"),
		users.Field("email"),
		users.Field("mobile"),
		users.Field("enabled"),
		users.Field("default_project_id"),
		users.Field("created_at"),
		users.Field("last_active_at"),
		users.Field("domain_id"),
		users.Field("is_system_account"),
		localUsers.Field("id", "local_id"),
		localUsers.Field("name", "local_name"),
		localUsers.Field("failed_auth_count", "local_failed_auth_count"),
		domains.Field("name", "domain_name"),
		domains.Field("enabled", "domain_enabled"),
		// idmappings.Field("domain_id", "idp_id"),
		// idmappings.Field("local_id", "idp_name"),
	)

	q = q.Join(domains, sqlchemy.Equals(users.Field("domain_id"), domains.Field("id")))
	q = q.LeftJoin(localUsers, sqlchemy.Equals(localUsers.Field("user_id"), users.Field("id")))
	// q = q.LeftJoin(idmappings, sqlchemy.Equals(users.Field("id"), idmappings.Field("public_id")))

	if len(userId) > 0 {
		q = q.Filter(sqlchemy.Equals(users.Field("id"), userId))
	} else if len(userName) > 0 {
		q = q.Filter(sqlchemy.Equals(users.Field("name"), userName))
		if len(domainId) == 0 && len(domainName) == 0 {
			domainId = api.DEFAULT_DOMAIN_ID
		}
		if len(domainId) > 0 {
			q = q.Filter(sqlchemy.Equals(domains.Field("id"), domainId))
		} else if len(domainName) > 0 {
			q = q.Filter(sqlchemy.Equals(domains.Field("name"), domainName))
		}
	}

	extUser := api.SUserExtended{}
	err := q.First(&extUser)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.ErrUserNotFound
		}
		return nil, errors.Wrap(err, "query")
	}

	if extUser.LocalId > 0 {
		extUser.IsLocal = true
	}

	return &extUser, nil
}

func VerifyPassword(user *api.SUserExtended, passwd string) error {
	return localUserVerifyPassword(user, passwd)
}

func localUserVerifyPassword(user *api.SUserExtended, passwd string) error {
	passes, err := PasswordManager.fetchByLocaluserId(user.LocalId)
	if err != nil {
		return errors.Wrap(err, "fetchPassword")
	}
	if len(passes) == 0 {
		return errors.Error("no valid password")
	}
	// password expiration check skip system account
	if passes[0].IsExpired() && !user.IsSystemAccount {
		return errors.Error("password expires")
	}
	err = seclib2.BcryptVerifyPassword(passwd, passes[0].PasswordHash)
	if err == nil {
		return nil
	}
	return httperrors.ErrWrongPassword
}

// 用户列表
func (manager *SUserManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.UserListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ListItemFilter")
	}

	if len(query.Email) > 0 {
		q = q.Equals("email", query.Email)
	}
	if len(query.Mobile) > 0 {
		q = q.Equals("mobile", query.Mobile)
	}
	if len(query.Displayname) > 0 {
		q = q.Equals("displayname", query.Displayname)
	}
	if query.AllowWebConsole != nil {
		if *query.AllowWebConsole {
			q = q.IsTrue("allow_web_console")
		} else {
			q = q.IsFalse("allow_web_console")
		}
	}
	if query.EnableMfa != nil {
		if *query.EnableMfa {
			q = q.IsTrue("enable_mfa")
		} else {
			q = q.IsFalse("enable_mfa")
		}
	}

	groupStr := query.GroupId
	if len(groupStr) > 0 {
		groupObj, err := GroupManager.FetchByIdOrName(userCred, groupStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), groupStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := UsergroupManager.Query("user_id").Equals("group_id", groupObj.GetId())
		q = q.In("id", subq.SubQuery())
	}

	projectStr := query.ProjectId
	if len(projectStr) > 0 {
		project, err := ProjectManager.FetchByIdOrName(userCred, projectStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), projectStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchProjectUserIdsQuery(project.GetId())
		q = q.In("id", subq.SubQuery())
	}

	roleStr := query.RoleId
	if len(roleStr) > 0 {
		role, err := RoleManager.FetchByIdOrName(userCred, roleStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), roleStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchRoleUserIdsQuery(role.GetId())
		q = q.In("id", subq.SubQuery())
	}

	if len(query.IdpId) > 0 {
		idpObj, err := IdentityProviderManager.FetchByIdOrName(userCred, query.IdpId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", IdentityProviderManager.Keyword(), query.IdpId)
			} else {
				return nil, errors.Wrap(err, "IdentityProviderManager.FetchByIdOrName")
			}
		}
		subq := IdmappingManager.FetchPublicIdsExcludesQuery(idpObj.GetId(), api.IdMappingEntityUser, nil)
		q = q.In("id", subq.SubQuery())
	}

	if len(query.IdpEntityId) > 0 {
		subq := IdmappingManager.Query("public_id").Equals("local_id", query.IdpEntityId).Equals("entity_type", api.IdMappingEntityUser)
		q = q.Equals("id", subq.SubQuery())
	}

	return q, nil
}

func (manager *SUserManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.UserListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SUserManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SUserManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = manager.SEnabledIdentityBaseResourceManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)
	isSystem := jsonutils.QueryBoolean(query, "system", false)
	if isSystem {
		var isAllow bool
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList, "system")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			isSystem = false
		}
	}
	if !isSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system_account")), sqlchemy.IsFalse(q.Field("is_system_account"))))
	}
	return q
}

func (manager *SUserManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.UserCreateInput,
) (api.UserCreateInput, error) {
	var err error
	if len(input.Password) > 0 && (input.SkipPasswordComplexityCheck == nil || !*input.SkipPasswordComplexityCheck) {
		err = validatePasswordComplexity(input.Password)
		if err != nil {
			return input, errors.Wrap(err, "validatePasswordComplexity")
		}
	}
	input.EnabledIdentityBaseResourceCreateInput, err = manager.SEnabledIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledIdentityBaseResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ValidateCreateData")
	}

	if len(input.IdpId) > 0 {
		_, err := IdentityProviderManager.FetchIdentityProviderById(input.IdpId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", IdentityProviderManager.Keyword(), input.IdpId)
			} else {
				return input, errors.Wrap(err, "IdentityProviderManager.FetchIdentityProviderById")
			}
		}
	}

	quota := SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		User:                 1,
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, &quota)
	if err != nil {
		return input, errors.Wrapf(err, "CheckSetPendingQuota")
	}

	return input, nil
}

func (user *SUser) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.UserUpdateInput) (api.UserUpdateInput, error) {
	if len(input.Name) > 0 {
		if user.IsAdminUser() {
			return input, httperrors.NewForbiddenError("cannot alter sysadmin user name")
		}
	}
	if !user.IsLocal() {
		data := jsonutils.Marshal(input)
		for _, k := range []string{
			"name",
			// "displayname",
			// "email",
			// "mobile",
			"password",
		} {
			if data.Contains(k) {
				return input, httperrors.NewForbiddenError("field %s is readonly", k)
			}
		}
	}
	if len(input.Password) > 0 && (input.SkipPasswordComplexityCheck == nil || *input.SkipPasswordComplexityCheck == false) {
		passwd := input.Password
		usrExt, err := UserManager.FetchUserExtended(user.Id, "", "", "")
		if err != nil {
			return input, errors.Wrap(err, "UserManager.FetchUserExtended")
		}
		if !usrExt.IsLocal {
			return input, errors.Wrap(httperrors.ErrForbidden, "cannot update password for non-local user")
		}
		skipHistoryCheck := false
		if user.IsSystemAccount.Bool() {
			skipHistoryCheck = true
		}
		err = PasswordManager.validatePassword(usrExt.LocalId, passwd, skipHistoryCheck)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid password: %s", err)
		}
	}
	var err error
	input.EnabledIdentityBaseUpdateInput, err = user.SEnabledIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, input.EnabledIdentityBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledIdentityBaseResource.ValidateUpdateData")
	}

	return input, nil
}

func (user *SUser) ValidateUpdateCondition(ctx context.Context) error {
	// if user.IsReadOnly() {
	// 	return httperrors.NewForbiddenError("readonly")
	// }
	return user.SEnabledIdentityBaseResource.ValidateUpdateCondition(ctx)
}

func (manager *SUserManager) fetchUserById(uid string) (*SUser, error) {
	obj, err := manager.FetchById(uid)
	if err != nil {
		return nil, errors.Wrap(err, "manager.FetchById")
	}
	return obj.(*SUser), nil
}

func (user *SUser) IsAdminUser() bool {
	return user.Name == api.SystemAdminUser && user.DomainId == api.DEFAULT_DOMAIN_ID
}

func (user *SUser) GetGroupCount() (int, error) {
	q := UsergroupManager.Query().Equals("user_id", user.Id)
	return q.CountWithError()
}

func (user *SUser) GetProjectCount() (int, error) {
	q := AssignmentManager.fetchUserProjectIdsQuery(user.Id)
	return q.CountWithError()
}

func (user *SUser) GetCredentialCount() (int, error) {
	q := CredentialManager.Query().Equals("user_id", user.Id)
	return q.CountWithError()
}

func (manager *SUserManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.UserDetails {
	rows := make([]api.UserDetails, len(objs))

	identRows := manager.SEnabledIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	userIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.UserDetails{
			EnabledIdentityBaseResourceDetails: identRows[i],
		}
		userIds[i] = objs[i].(*SUser).Id
		rows[i] = userExtra(objs[i].(*SUser), rows[i])
	}

	idpsMaps, err := fetchIdmappings(userIds, api.IdMappingEntityUser)
	if err != nil {
		log.Errorf("fetchIdmappings fail %s", err)
		return rows
	}

	for i := range rows {
		if idps, ok := idpsMaps[userIds[i]]; ok {
			if len(idps) > 0 {
				// rows[i].IdpResourceInfo = idps[0].IdpResourceInfo
				rows[i].Idps = make([]api.IdpResourceInfo, len(idps))
				for j := range idps {
					rows[i].Idps[j] = idps[j].IdpResourceInfo
				}
			}
		}
	}

	return rows
}

func userExtra(user *SUser, out api.UserDetails) api.UserDetails {
	out.GroupCount, _ = user.GetGroupCount()
	out.ProjectCount, _ = user.GetProjectCount()
	out.CredentialCount, _ = user.GetCredentialCount()

	localUser, _ := LocalUserManager.fetchLocalUser(user.Id, user.DomainId, 0)
	if localUser != nil {
		if localUser.FailedAuthCount > 0 {
			out.FailedAuthCount = localUser.FailedAuthCount
			out.FailedAuthAt = localUser.FailedAuthAt
		}
		localPass, _ := PasswordManager.FetchLastPassword(localUser.Id)
		if localPass != nil && !localPass.ExpiresAt.IsZero() {
			out.PasswordExpiresAt = localPass.ExpiresAt
		}
		out.IsLocal = true
	} else {
		out.IsLocal = false
	}

	external, update, _ := user.getExternalResources()
	if len(external) > 0 {
		out.ExtResource = jsonutils.Marshal(external)
		out.ExtResourcesLastUpdate = update
		if update.IsZero() {
			update = time.Now()
		}
		nextUpdate := update.Add(time.Duration(options.Options.FetchScopeResourceCountIntervalSeconds) * time.Second)
		out.ExtResourcesNextUpdate = nextUpdate
	}

	return out
}

func (user *SUser) initLocalData(passwd string) error {
	localUsr, err := LocalUserManager.register(user.Id, user.DomainId, user.Name)
	if err != nil {
		return errors.Wrap(err, "register localuser")
	}
	if len(passwd) > 0 {
		err = PasswordManager.savePassword(localUsr.Id, passwd, user.IsSystemAccount.Bool())
		if err != nil {
			return errors.Wrap(err, "save password")
		}
	}
	return nil
}

func (user *SUser) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	user.SEnabledIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)

	// set password
	passwd, _ := data.GetString("password")
	err := user.initLocalData(passwd)
	if err != nil {
		log.Errorf("fail to register localUser %s", err)
		return
	}

	// link idp
	idpId, _ := data.GetString("idp_id")
	if len(idpId) > 0 {
		idpEntityId, _ := data.GetString("idp_entity_id")
		if len(idpEntityId) > 0 {
			_, err := IdmappingManager.RegisterIdMapWithId(ctx, idpId, idpEntityId, api.IdMappingEntityUser, user.Id)
			if err != nil {
				log.Errorf("IdmappingManager.RegisterIdMapWithId fail %s", err)
			}
		}
	}

	// clean user quota
	pendingUsage := &SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()},
		User:                 1,
	}
	quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, true)
}

func (user *SUser) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	user.SEnabledIdentityBaseResource.PostUpdate(ctx, userCred, query, data)

	passwd, _ := data.GetString("password")
	if len(passwd) > 0 {
		usrExt, err := UserManager.FetchUserExtended(user.Id, "", "", "")
		if err != nil {
			log.Errorf("UserManager.FetchUserExtended fail %s", err)
			return
		}
		err = PasswordManager.savePassword(usrExt.LocalId, passwd, user.IsSystemAccount.Bool())
		if err != nil {
			log.Errorf("fail to set password %s", err)
			return
		}
		logclient.AddActionLogWithContext(ctx, user, logclient.ACT_UPDATE_PASSWORD, nil, userCred, true)
	}
	if enabled, _ := data.Bool("enabled"); enabled {
		localUser, err := LocalUserManager.fetchLocalUser(user.Id, user.DomainId, 0)
		if err != nil {
			if err == sql.ErrNoRows {
				return
			}
			log.Errorf("unable to fetch localUser of user %q in domain %q: %v", user.Id, user.DomainId, err)
			return
		}
		if err = localUser.ClearFailedAuth(); err != nil {
			log.Errorf("unable to clear failed auth: %v", err)
		}
	}
}

func (user *SUser) ValidateDeleteCondition(ctx context.Context) error {
	idMappings, err := user.getIdmappings()
	if err != nil {
		return errors.Wrap(err, "getIdmappings")
	}
	if !user.IsLocal() && len(idMappings) > 0 {
		for _, idmaping := range idMappings {
			idp, err := IdentityProviderManager.FetchIdentityProviderById(idmaping.IdpId)
			if err != nil && errors.Cause(err) == sql.ErrNoRows {
				return errors.Wrap(err, "IdentityProviderManager.FetchIdentityProviderById")
			}
			if idp != nil && idp.IsSso.IsFalse() {
				return httperrors.NewForbiddenError("cannot delete non-local non-sso user")
			}
		}
	}
	err = user.ValidatePurgeCondition(ctx)
	if err != nil {
		return errors.Wrap(err, "ValidatePurgeCondition")
	}
	return user.SIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (user *SUser) ValidatePurgeCondition(ctx context.Context) error {
	external, _, _ := user.getExternalResources()
	if len(external) > 0 {
		return httperrors.NewNotEmptyError("user contains external resources")
	}
	if user.IsAdminUser() {
		return httperrors.NewForbiddenError("cannot delete system user")
	}
	return nil
}

func (user *SUser) getExternalResources() (map[string]int, time.Time, error) {
	return ScopeResourceManager.getScopeResource("", "", user.Id)
}

func (user *SUser) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := AssignmentManager.projectRemoveAllUser(ctx, userCred, user)
	if err != nil {
		return errors.Wrap(err, "AssignmentManager.projectRemoveAllUser")
	}

	err = UsergroupManager.delete(user.Id, "")
	if err != nil {
		return errors.Wrap(err, "UsergroupManager.delete")
	}

	localUser, err := LocalUserManager.delete(user.Id, user.DomainId)
	if err != nil {
		return errors.Wrap(err, "LocalUserManager.delete")
	}

	if localUser != nil {
		err = PasswordManager.delete(localUser.Id)
		if err != nil {
			return errors.Wrap(err, "PasswordManager.delete")
		}
	}

	err = IdmappingManager.deleteByPublicId(user.Id, api.IdMappingEntityUser)
	if err != nil {
		return errors.Wrap(err, "IdmappingManager.deleteByPublicId")
	}

	return user.SEnabledIdentityBaseResource.Delete(ctx, userCred)
}

func (user *SUser) UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(ctxObjs) != 1 {
		return nil, httperrors.NewInputParameterError("not supported update context")
	}
	group, ok := ctxObjs[0].(*SGroup)
	if !ok {
		return nil, httperrors.NewInputParameterError("not supported update context %s", ctxObjs[0].Keyword())
	}
	if user.DomainId != group.DomainId {
		return nil, httperrors.NewInputParameterError("cannot join user and group in differnt domain")
	}
	if group.IsReadOnly() {
		return nil, httperrors.NewForbiddenError("cannot join read-only group")
	}
	return nil, UsergroupManager.add(ctx, userCred, user, group)
}

func (user *SUser) DeleteInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(ctxObjs) != 1 {
		return nil, httperrors.NewInputParameterError("not supported update context")
	}
	group, ok := ctxObjs[0].(*SGroup)
	if !ok {
		return nil, httperrors.NewInputParameterError("not supported update context %s", ctxObjs[0].Keyword())
	}
	if group.IsReadOnly() {
		return nil, httperrors.NewForbiddenError("cannot leave read-only group")
	}
	return nil, UsergroupManager.remove(ctx, userCred, user, group)
}

func (manager *SUserManager) TraceLoginV2(ctx context.Context, token *mcclient.TokenCredentialV2) {
	s := tokenV2LoginSession(token)
	manager.traceLoginEvent(ctx, token, s, token.Context)
}

func (manager *SUserManager) TraceLoginV3(ctx context.Context, token *mcclient.TokenCredentialV3) {
	s := tokenV3LoginSession(token)
	manager.traceLoginEvent(ctx, token, s, token.Token.Context)
}

func (manager *SUserManager) traceLoginEvent(ctx context.Context, token mcclient.TokenCredential, s sLoginSession, authCtx mcclient.SAuthContext) {
	usr, err := manager.fetchUserById(token.GetUserId())
	if err != nil {
		// very unlikely
		log.Errorf("fetchUserById fail %s", err)
		return
	}
	db.Update(usr, func() error {
		usr.LastActiveAt = time.Now().UTC()
		usr.LastLoginIp = authCtx.Ip
		usr.LastLoginSource = authCtx.Source
		return nil
	})
	db.OpsLog.LogEvent(usr, "auth", &s, token)
	// to reduce auth event, log web console login only
	if authCtx.Source == mcclient.AuthSourceWeb {
		logclient.AddActionLogWithContext(ctx, usr, logclient.ACT_AUTHENTICATE, &s, token, true)
		return
	}
	// ignore any other auth source
}

type sLoginSession struct {
	Version         string
	Source          string
	Ip              string
	Project         string
	ProjectId       string
	ProjectDomain   string
	ProjectDomainId string
	Token           string
}

func tokenV2LoginSession(token *mcclient.TokenCredentialV2) sLoginSession {
	s := sLoginSession{}
	s.Version = "v2"
	s.Source = token.Context.Source
	s.Ip = token.Context.Ip
	s.Project = token.Token.Tenant.Name
	s.ProjectId = token.Token.Tenant.Id
	s.ProjectDomain = token.Token.Tenant.Domain.Name
	s.ProjectDomainId = token.Token.Tenant.Domain.Id
	s.Token = token.Token.Id
	return s
}

func tokenV3LoginSession(token *mcclient.TokenCredentialV3) sLoginSession {
	s := sLoginSession{}
	s.Version = "v3"
	s.Source = token.Token.Context.Source
	s.Ip = token.Token.Context.Ip
	s.Project = token.Token.Project.Name
	s.ProjectId = token.Token.Project.Id
	s.ProjectDomain = token.Token.Project.Domain.Name
	s.ProjectDomainId = token.Token.Project.Domain.Id
	s.Token = token.Id
	return s
}

func (manager *SUserManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (user *SUser) getIdmappings() ([]SIdmapping, error) {
	return IdmappingManager.FetchEntities(user.Id, api.IdMappingEntityUser)
}

func (user *SUser) IsLocal() bool {
	usr, _ := LocalUserManager.fetchLocalUser(user.Id, user.DomainId, 0)
	if usr != nil {
		return true
	}
	return false
}

func (user *SUser) LinkedWithIdp(idpId string) bool {
	idmaps, _ := user.getIdmappings()
	for i := range idmaps {
		if idmaps[i].IdpId == idpId {
			return true
		}
	}
	return false
}

func (manager *SUserManager) FetchUsersInDomain(domainId string, excludes []string) ([]SUser, error) {
	q := manager.Query().Equals("domain_id", domainId).NotIn("id", excludes)
	usrs := make([]SUser, 0)
	err := db.FetchModelObjects(manager, q, &usrs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return usrs, nil
}

func (user *SUser) UnlinkIdp(idpId string) error {
	return IdmappingManager.deleteAny(idpId, api.IdMappingEntityUser, user.Id)
}

func (user *SUser) AllowPerformJoin(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SJoinProjectsInput,
) bool {
	return db.IsAdminAllowPerform(userCred, user, "join")
}

// 用户加入项目
func (user *SUser) PerformJoin(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SJoinProjectsInput,
) (jsonutils.JSONObject, error) {
	err := joinProjects(user, true, ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "joinProjects")
	}
	return nil, nil
}

func joinProjects(ident db.IModel, isUser bool, ctx context.Context, userCred mcclient.TokenCredential, input api.SJoinProjectsInput) error {
	err := input.Validate()
	if err != nil {
		return httperrors.NewInputParameterError("%v", err)
	}

	projects := make([]*SProject, 0)
	roles := make([]*SRole, 0)
	roleIds := make([]string, 0)

	for i := range input.Roles {
		obj, err := RoleManager.FetchByIdOrName(userCred, input.Roles[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), input.Roles[i])
			} else {
				return httperrors.NewGeneralError(err)
			}
		}
		role := obj.(*SRole)
		roles = append(roles, role)
		roleIds = append(roleIds, role.Id)
	}

	for i := range input.Projects {
		obj, err := ProjectManager.FetchByIdOrName(userCred, input.Projects[i])
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), input.Projects[i])
			} else {
				return httperrors.NewGeneralError(err)
			}
		}
		project := obj.(*SProject)
		err = validateJoinProject(userCred, project, roleIds)
		if err != nil {
			return errors.Wrapf(err, "validateJoinProject %s(%s)", project.Id, project.Name)
		}
		projects = append(projects, project)
	}

	for i := range projects {
		for j := range roles {
			if isUser {
				err = AssignmentManager.ProjectAddUser(ctx, userCred, projects[i], ident.(*SUser), roles[j])
			} else {
				err = AssignmentManager.projectAddGroup(ctx, userCred, projects[i], ident.(*SGroup), roles[j])
			}
			if err != nil {
				return httperrors.NewGeneralError(err)
			}
		}
	}

	return nil
}

func (user *SUser) AllowPerformLeave(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SLeaveProjectsInput,
) bool {
	return db.IsAdminAllowPerform(userCred, user, "leave")
}

// 用户退出项目
func (user *SUser) PerformLeave(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.SLeaveProjectsInput,
) (jsonutils.JSONObject, error) {
	err := leaveProjects(user, true, ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func leaveProjects(ident db.IModel, isUser bool, ctx context.Context, userCred mcclient.TokenCredential, input api.SLeaveProjectsInput) error {
	for i := range input.ProjectRoles {
		projObj, err := ProjectManager.FetchByIdOrName(userCred, input.ProjectRoles[i].Project)
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), input.ProjectRoles[i].Project)
			} else {
				return httperrors.NewGeneralError(err)
			}
		}
		roleObj, err := RoleManager.FetchByIdOrName(userCred, input.ProjectRoles[i].Role)
		if err != nil {
			if err == sql.ErrNoRows {
				return httperrors.NewResourceNotFoundError2(RoleManager.Keyword(), input.ProjectRoles[i].Role)
			} else {
				return httperrors.NewGeneralError(err)
			}
		}
		if isUser {
			err = AssignmentManager.projectRemoveUser(ctx, userCred, projObj.(*SProject), ident.(*SUser), roleObj.(*SRole))
		} else {
			err = AssignmentManager.projectRemoveGroup(ctx, userCred, projObj.(*SProject), ident.(*SGroup), roleObj.(*SRole))
		}
		if err != nil {
			return httperrors.NewGeneralError(err)
		}
	}
	return nil
}

func (manager *SUserManager) LockUser(uid string, reason string) error {
	usrObj, err := manager.FetchById(uid)
	if err != nil {
		return errors.Wrapf(err, "manager.FetchById %s", uid)
	}
	usr := usrObj.(*SUser)
	_, err = db.Update(usr, func() error {
		usr.Enabled = tristate.False
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	db.OpsLog.LogEvent(usr, db.ACT_DISABLE, reason, GetDefaultAdminCred())
	logclient.AddSimpleActionLog(usr, logclient.ACT_DISABLE, reason, GetDefaultAdminCred(), false)
	return nil
}

func (manager *SUserManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	log.Debugf("owner: %s scope %s", jsonutils.Marshal(owner), scope)
	if owner != nil && scope == rbacutils.ScopeProject {
		// if user has project level privilege, returns all users in user's project
		subq := AssignmentManager.fetchProjectUserIdsQuery(owner.GetProjectId())
		q = q.In("id", subq.SubQuery())
		return q
	}
	return manager.SEnabledIdentityBaseResourceManager.FilterByOwner(q, owner, scope)
}

func (user *SUser) GetUsages() []db.IUsage {
	usage := SIdentityQuota{
		SBaseDomainQuotaKeys: quotas.SBaseDomainQuotaKeys{DomainId: user.DomainId},
		User:                 1,
	}
	return []db.IUsage{
		&usage,
	}
}

func (user *SUser) AllowPerformLinkIdp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.UserLinkIdpInput,
) bool {
	return db.IsAdminAllowPerform(userCred, user, "link-idp")
}

// 用户和IDP的指定entityId关联
func (user *SUser) PerformLinkIdp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.UserLinkIdpInput,
) (jsonutils.JSONObject, error) {
	idp, err := IdentityProviderManager.FetchIdentityProviderById(input.IdpId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", IdentityProviderManager.Keyword(), input.IdpId)
		} else {
			return nil, errors.Wrap(err, "IdentityProviderManager.FetchIdentityProviderById")
		}
	}
	// check accessibility
	if (len(idp.DomainId) > 0 && idp.DomainId != user.DomainId) || (len(idp.TargetDomainId) > 0 && idp.TargetDomainId != user.DomainId) {
		return nil, errors.Wrap(httperrors.ErrForbidden, "identity domain not accessible")
	} else if len(idp.DomainId) == 0 && len(idp.TargetDomainId) == 0 && idp.AutoCreateUser.IsTrue() {

	}
	_, err = IdmappingManager.RegisterIdMapWithId(ctx, input.IdpId, input.IdpEntityId, api.IdMappingEntityUser, user.Id)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.RegisterIdMapWithId")
	}
	return nil, nil
}

func (user *SUser) AllowPerformUnlinkIdp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.UserUnlinkIdpInput,
) bool {
	return db.IsAdminAllowPerform(userCred, user, "unlink-idp")
}

// 用户和IDP的指定entityId解除关联
func (user *SUser) PerformUnlinkIdp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.UserUnlinkIdpInput,
) (jsonutils.JSONObject, error) {
	err := IdmappingManager.deleteAny(input.IdpId, api.IdMappingEntityUser, user.Id)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.deleteAny")
	}
	return nil, nil
}

func GetUserLangForKeyStone(uids []string) (map[string]string, error) {
	simpleUsers := make([]struct {
		Id   string
		Lang string
	}, 0, len(uids))
	q := UserManager.Query()
	if len(uids) == 0 {
		return nil, nil
	} else if len(uids) == 1 {
		q = q.Equals("id", uids[0])
	} else {
		q = q.In("id", uids)
	}
	err := q.All(&simpleUsers)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string, len(simpleUsers))
	for i := range simpleUsers {
		ret[simpleUsers[i].Id] = simpleUsers[i].Lang
	}
	return ret, nil
}
