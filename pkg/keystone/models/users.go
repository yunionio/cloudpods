package models

import (
	"fmt"
	"time"

	"yunion.io/x/sqlchemy"

	"context"
	"database/sql"
	"yunion.io/x/jsonutils"

	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
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

	Email  string `width:"64" charset:"ascii" nullable:"true" index:"true" list:"admin" update:"admin" create:"admin_optional"`
	Mobile string `width:"20" charset:"ascii" nullable:"true" index:"true" list:"admin" update:"admin" create:"admin_optional"`

	Displayname string `with:"128" charset:"utf8" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`

	LastActiveAt     time.Time `nullable:"true" list:"admin"`
	DefaultProjectId string    `width:"64" charset:"ascii" index:"true" list:"admin" update:"admin" create:"admin_optional"`
}

func (manager *SUserManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{GroupManager},
		{ProjectManager},
	}
}

func (manager *SUserManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	users := make([]SUser, 0)
	err := db.FetchModelObjects(manager, q, &users)
	if err != nil {
		return err
	}
	for i := range users {
		extUser, err := manager.FetchUserExtended(users[i].Id, "", "", "")
		if err != nil {
			return err
		}
		name := extUser.Name
		desc, _ := users[i].Extra.GetString("description")
		email, _ := users[i].Extra.GetString("email")
		mobile, _ := users[i].Extra.GetString("mobile")
		db.Update(&users[i], func() error {
			users[i].Name = name
			users[i].Email = email
			users[i].Mobile = mobile
			users[i].Description = desc
			return nil
		})
	}
	return nil
}

type SUserExtended struct {
	Id               string
	Name             string
	Enabled          bool
	DefaultProjectId string
	CreatedAt        time.Time
	LastActiveAt     time.Time
	DomainId         string

	LocalId      int
	LocalName    string
	NonlocalName string
	DomainName   string
	IsLocal      bool
}

/*
 Fetch extended userinfo by Id or name + domainId or name + domainName
*/
func (manager *SUserManager) FetchUserExtended(userId, userName, domainId, domainName string) (*SUserExtended, error) {
	if len(userId) == 0 && len(userName) == 0 {
		return nil, sqlchemy.ErrEmptyQuery
	}

	localUsers := LocalUserManager.Query().SubQuery()
	nonlocalUsers := NonlocalUserManager.Query().SubQuery()
	users := UserManager.Query().SubQuery()
	domains := ProjectManager.Query().SubQuery()

	q := users.Query(
		users.Field("id"),
		users.Field("enabled"),
		users.Field("default_project_id"),
		users.Field("created_at"),
		users.Field("last_active_at"),
		users.Field("domain_id"),
		localUsers.Field("id", "local_id"),
		localUsers.Field("name", "local_name"),
		nonlocalUsers.Field("name", "nonlocal_name"),
		domains.Field("name", "domain_name"),
	)
	q = q.Join(domains, sqlchemy.AND(
		sqlchemy.IsTrue(domains.Field("is_domain")),
		sqlchemy.Equals(users.Field("domain_id"), domains.Field("id")),
	))
	q = q.LeftJoin(localUsers, sqlchemy.Equals(localUsers.Field("user_id"), users.Field("id")))
	q = q.LeftJoin(nonlocalUsers, sqlchemy.Equals(nonlocalUsers.Field("user_id"), users.Field("id")))

	if len(userId) > 0 {
		q = q.Filter(sqlchemy.Equals(users.Field("id"), userId))
	} else if len(userName) > 0 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(localUsers.Field("name"), userName),
			sqlchemy.Equals(nonlocalUsers.Field("name"), userName),
		))
		if len(domainId) == 0 && len(domainName) == 0 {
			domainId = api.DEFAULT_DOMAIN_ID
		}
		if len(domainId) > 0 {
			q = q.Filter(sqlchemy.Equals(domains.Field("id"), domainId))
		} else if len(domainName) > 0 {
			q = q.Filter(sqlchemy.Equals(domains.Field("name"), domainName))
		}
	}

	extUser := SUserExtended{}
	err := q.First(&extUser)
	if err != nil {
		return nil, err
	}

	if len(extUser.NonlocalName) > 0 {
		extUser.IsLocal = false
		extUser.Name = extUser.NonlocalName
	} else {
		extUser.IsLocal = true
		extUser.Name = extUser.LocalName
	}
	return &extUser, nil
}

func (user *SUserExtended) VerifyPassword(passwd string) error {
	if user.IsLocal {
		return user.localUserVerifyPassword(passwd)
	} else {
		return fmt.Errorf("not implemented")
	}
}

func (user *SUserExtended) localUserVerifyPassword(passwd string) error {
	passes, err := PasswordManager.fetchByLocaluserId(user.LocalId)
	if err != nil {
		return err
	}
	if len(passes) == 0 {
		return nil
	}
	for i := range passes {
		err = seclib2.BcryptVerifyPassword(passwd, passes[i].PasswordHash)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid password")
}

func (manager *SUserManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	groupStr := jsonutils.GetAnyString(query, []string{"group_id"})
	if len(groupStr) > 0 {
		groupObj, err := GroupManager.FetchById(groupStr)
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

	projectStr := jsonutils.GetAnyString(query, []string{"project_id"})
	if len(projectStr) > 0 {
		project, err := ProjectManager.FetchProjectById(projectStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ProjectManager.Keyword(), projectStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		subq := AssignmentManager.fetchProjectUserIdsQuery(project.Id)
		q = q.In("id", subq.SubQuery())
	}

	return q, nil
}

func (user *SUser) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		if user.IsAdminUser() {
			return nil, httperrors.NewForbiddenError("cannot alter name of system user")
		}
	}
	return user.SEnabledIdentityBaseResource.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SUserManager) fetchUserById(uid string) *SUser {
	obj, _ := manager.FetchById(uid)
	if obj != nil {
		return obj.(*SUser)
	}
	return nil
}

func (user *SUser) IsAdminUser() bool {
	return user.Name == options.Options.AdminUserName && user.DomainId == options.Options.AdminUserDomainId
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

func (user *SUser) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := user.SEnabledIdentityBaseResource.GetCustomizeColumns(ctx, userCred, query)
	return userExtra(user, extra)
}

func (user *SUser) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := user.SEnabledIdentityBaseResource.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return userExtra(user, extra), nil
}

func userExtra(user *SUser, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	domain := user.GetDomain()
	if domain != nil {
		extra.Add(jsonutils.NewString(domain.Name), "domain")
	}
	grpCnt, _ := user.GetGroupCount()
	extra.Add(jsonutils.NewInt(int64(grpCnt)), "group_count")
	prjCnt, _ := user.GetProjectCount()
	extra.Add(jsonutils.NewInt(int64(prjCnt)), "project_count")
	credCnt, _ := user.GetCredentialCount()
	extra.Add(jsonutils.NewInt(int64(credCnt)), "credential_count")
	return extra
}

func (user *SUser) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	user.SEnabledIdentityBaseResource.PostCreate(ctx, userCred, ownerProjId, query, data)

	localUsr, err := LocalUserManager.register(user.Id, user.DomainId, user.Name)
	if err != nil {
		log.Errorf("fail to register localUser %s", err)
		return
	}
	passwd, _ := data.GetString("password")
	if len(passwd) > 0 {
		err = PasswordManager.savePassword(localUsr.Id, passwd)
		if err != nil {
			log.Errorf("fail to set password %s", err)
			return
		}
	}
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
		err = PasswordManager.savePassword(usrExt.LocalId, passwd)
		if err != nil {
			log.Errorf("fail to set password %s", err)
			return
		}
	}
}

func (user *SUser) ValidateDeleteCondition(ctx context.Context) error {
	grpCnt, _ := user.GetGroupCount()
	if grpCnt > 0 {
		return httperrors.NewNotEmptyError("group contains user")
	}
	prjCnt, _ := user.GetProjectCount()
	if prjCnt > 0 {
		return httperrors.NewNotEmptyError("user joins project")
	}
	return user.SIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (user *SUser) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	user.SEnabledIdentityBaseResource.PostDelete(ctx, userCred)

	localUser, err := LocalUserManager.delete(user.Id, user.DomainId)
	if err != nil {
		log.Errorf("LocalUserManager.delete fail %s", err)
		return
	}

	err = PasswordManager.delete(localUser.Id)
	if err != nil {
		log.Errorf("PasswordManager.delete fail %s", err)
		return
	}
}

func (user *SUser) UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if user.GetDomain().IsReadOnly() {
		return nil, httperrors.NewForbiddenError("readonly domain")
	}
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
	return nil, UsergroupManager.add(ctx, userCred, user, group)
}

func (user *SUser) DeleteInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []db.IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if user.GetDomain().IsReadOnly() {
		return nil, httperrors.NewForbiddenError("readonly domain")
	}
	if len(ctxObjs) != 1 {
		return nil, httperrors.NewInputParameterError("not supported update context")
	}
	group, ok := ctxObjs[0].(*SGroup)
	if !ok {
		return nil, httperrors.NewInputParameterError("not supported update context %s", ctxObjs[0].Keyword())
	}
	return nil, UsergroupManager.remove(ctx, userCred, user, group)
}
