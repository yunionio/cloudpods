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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDBInstanceAccountManager struct {
	db.SStatusStandaloneResourceBaseManager
	SDBInstanceResourceBaseManager
}

var DBInstanceAccountManager *SDBInstanceAccountManager

func init() {
	DBInstanceAccountManager = &SDBInstanceAccountManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SDBInstanceAccount{},
			"dbinstanceaccounts_tbl",
			"dbinstanceaccount",
			"dbinstanceaccounts",
		),
	}
	DBInstanceAccountManager.SetVirtualObject(DBInstanceAccountManager)
}

type SDBInstanceAccount struct {
	db.SStatusStandaloneResourceBase

	Host string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"optional" default:"%"`

	SDBInstanceResourceBase `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`

	// 数据库密码
	Secret string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`
}

func (manager *SDBInstanceAccountManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

func (manager *SDBInstanceAccountManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (self *SDBInstanceAccount) GetOwnerId() mcclient.IIdentityProvider {
	instance, err := self.GetDBInstance()
	if err != nil {
		log.Errorf("failed to get instance for account %s(%s)", self.Name, self.Id)
		return nil
	}
	return instance.GetOwnerId()
}

func (manager *SDBInstanceAccountManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if jsonutils.QueryBoolean(query, "admin", false) && !db.IsAllowList(rbacutils.ScopeProject, userCred, manager) {
		return false
	}
	return true
}

func (manager *SDBInstanceAccountManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	dbinstanceId, _ := data.GetString("dbinstance_id")
	if len(dbinstanceId) > 0 {
		instance, err := db.FetchById(DBInstanceManager, dbinstanceId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(DBInstanceManager, %s)", dbinstanceId)
		}
		return instance.(*SDBInstance).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SDBInstanceAccountManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := DBInstanceManager.Query("id")
		switch scope {
		case rbacutils.ScopeProject:
			sq = sq.Equals("tenant_id", userCred.GetProjectId())
			return q.In("dbinstance_id", sq.SubQuery())
		case rbacutils.ScopeDomain:
			sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
			return q.In("dbinstance_id", sq.SubQuery())
		}
	}
	return q
}

func (self *SDBInstanceAccountManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceAccount) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceAccount) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsProjectAllowUpdate(userCred, self)
}

func (self *SDBInstanceAccount) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DBInstanceAccountUpdateInput) (api.DBInstanceAccountUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = self.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrapf(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	if len(input.Name) > 0 && input.Name != self.Name {
		return input, httperrors.NewForbiddenError("not allow update rds account name")
	}
	return input, nil
}

func (self *SDBInstanceAccount) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SDBInstanceAccount) getPrivilegesDetails() ([]api.DBInstancePrivilege, error) {
	out := []api.DBInstancePrivilege{}
	privileges, err := self.GetDBInstancePrivileges()
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstancePrivileges")
	}
	for _, privilege := range privileges {
		detail, err := privilege.GetPrivilege()
		if err != nil {
			return nil, errors.Wrap(err, "GetDetailedJson")
		}
		out = append(out, detail)
	}
	return out, nil
}

func (self *SDBInstanceAccount) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, out api.DBInstanceAccountDetails) (api.DBInstanceAccountDetails, error) {
	privileges, err := self.getPrivilegesDetails()
	if err != nil {
		return out, err
	}
	out.DBInstanceprivileges = privileges
	return out, nil
}

func (self *SDBInstanceAccount) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DBInstanceAccountDetails, error) {
	return api.DBInstanceAccountDetails{}, nil
}

func (manager *SDBInstanceAccountManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceAccountDetails {
	rows := make([]api.DBInstanceAccountDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dbRows := manager.SDBInstanceResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dbinstanceIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.DBInstanceAccountDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			DBInstanceResourceInfo:          dbRows[i],
		}
		account := objs[i].(*SDBInstanceAccount)
		rows[i], _ = account.getMoreDetails(ctx, userCred, rows[i])
		dbinstanceIds[i] = account.DBInstanceId
	}

	dbinstances := make(map[string]SDBInstance)
	err := db.FetchStandaloneObjectsByIds(DBInstanceManager, dbinstanceIds, &dbinstances)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if dbinstance, ok := dbinstances[dbinstanceIds[i]]; ok {
			virObjs[i] = &dbinstance
			rows[i].ProjectId = dbinstance.ProjectId
		}
	}

	projRows := DBInstanceManager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, stringutils2.SSortedStrings{}, isList)
	for i := range rows {
		rows[i].ProjectizedResourceInfo = projRows[i]
	}

	return rows
}

// RDS账号列表
func (manager *SDBInstanceAccountManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceAccountListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SDBInstanceResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DBInstanceFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SDBInstanceAccountManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceAccountListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SDBInstanceResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DBInstanceFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SDBInstanceAccountManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SDBInstanceResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

type sRdsAccount struct {
	Name         string
	DBInstanceId string `json:"dbinstance_id"`
	Host         string
}

func (self *SDBInstanceAccount) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sRdsAccount{Name: self.Name, DBInstanceId: self.DBInstanceId, Host: self.Host})
}

func (manager *SDBInstanceAccountManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	info := sRdsAccount{}
	data.Unmarshal(&info)
	return jsonutils.Marshal(info)
}

func (manager *SDBInstanceAccountManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	info := sRdsAccount{}
	values.Unmarshal(&info)
	if len(info.DBInstanceId) > 0 {
		q = q.Equals("dbinstance_id", info.DBInstanceId)
	}
	if len(info.Name) > 0 {
		q = q.Equals("name", info.Name)
	}
	if len(info.Host) > 0 {
		q = q.Equals("host", info.Host)
	}
	return q
}

func (manager *SDBInstanceAccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DBInstanceAccountCreateInput) (*jsonutils.JSONDict, error) {
	if len(input.Password) > 0 {
		err := seclib2.ValidatePassword(input.Password)
		if err != nil {
			return nil, err
		}
	} else {
		input.Password = seclib2.RandomPassword2(12)
	}

	for _, instance := range []string{input.DBInstance, input.DBInstanceId} {
		if len(instance) > 0 {
			input.DBInstance = instance
			break
		}
	}
	if len(input.DBInstance) == 0 {
		return nil, httperrors.NewMissingParameterError("dbinstance")
	}
	_instance, err := DBInstanceManager.FetchByIdOrName(userCred, input.DBInstance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("failed to found dbinstance %s", input.DBInstance)
		}
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "DBInstanceManager.FetchByIdOrName"))
	}
	instance := _instance.(*SDBInstance)
	input.DBInstanceId = instance.Id
	if instance.Status != api.DBINSTANCE_RUNNING {
		return nil, httperrors.NewInputParameterError("DBInstance %s(%s) status is %s require status is %s", instance.Name, instance.Id, instance.Status, api.DBINSTANCE_RUNNING)
	}
	region := instance.GetRegion()
	if region == nil {
		return nil, httperrors.NewInputParameterError("failed to found region for dbinstance %s(%s)", instance.Name, instance.Id)
	}
	for i, privilege := range input.Privileges {
		database, err := instance.GetDBInstanceDatabase(privilege.Database)
		if err != nil {
			return nil, httperrors.NewInputParameterError("failed to found dbinstance %s(%s) database %s: %v", instance.Name, instance.Id, privilege.Database, err)
		}
		input.Privileges[i].DBInstancedatabaseId = database.Id
	}
	input, err = region.GetDriver().ValidateCreateDBInstanceAccountData(ctx, userCred, ownerId, instance, input)
	if err != nil {
		return nil, err
	}

	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (self *SDBInstanceAccount) SetPassword(passwd string) error {
	return self.savePassword(passwd)
}

func (self *SDBInstanceAccount) savePassword(secret string) error {
	sec, err := utils.EncryptAESBase64(self.Id, secret)
	if err != nil {
		return err
	}

	_, err = db.Update(self, func() error {
		self.Secret = sec
		return nil
	})
	return err
}

func (self *SDBInstanceAccount) GetPassword() (string, error) {
	return utils.DescryptAESBase64(self.Id, self.Secret)
}

func (self *SDBInstanceAccount) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := &api.DBInstanceAccountCreateInput{}
	data.Unmarshal(input)
	self.savePassword(input.Password)
	self.StartDBInstanceAccountCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SDBInstanceAccount) StartDBInstanceAccountCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_USER_CREATING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceAccountCreateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstanceAccount) AllowPerformGrantPrivilege(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "grant-privilege")
}

func (self *SDBInstanceAccount) PerformGrantPrivilege(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	instance, err := self.GetDBInstance()
	if err != nil {
		return nil, errors.Wrap(err, "failed to found dbinstance")
	}
	databaseStr, _ := data.GetString("database")
	if len(databaseStr) == 0 {
		return nil, httperrors.NewMissingParameterError("database")
	}
	database, err := instance.GetDBInstanceDatabase(databaseStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Failed to found database %s for dbinstance %s(%s): %v", databaseStr, instance.Name, instance.Id, err)
	}
	privilegeStr, _ := data.GetString("privilege")
	if len(privilegeStr) == 0 {
		return nil, httperrors.NewMissingParameterError("privilege")
	}
	privilege, _ := instance.GetDBInstancePrivilege(self.Id, database.Id)
	if privilege != nil {
		return nil, httperrors.NewInputParameterError("The account %s(%s) has permission %s to the database %s(%s)", self.Name, self.Id, privilege.Privilege, database.Name, database.Id)
	}

	err = instance.GetRegion().GetDriver().ValidateDBInstanceAccountPrivilege(ctx, userCred, instance, self.Name, privilegeStr)
	if err != nil {
		return nil, err
	}

	return nil, self.StartGrantPrivilegeTask(ctx, userCred, databaseStr, privilegeStr, "")
}

func (self *SDBInstanceAccount) AllowPerformSetPrivileges(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "set-privileges")
}

func (self *SDBInstanceAccount) PerformSetPrivileges(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	instance, err := self.GetDBInstance()
	if err != nil {
		return nil, errors.Wrap(err, "failed to found dbinstance")
	}

	input := api.SDBInstanceSetPrivilegesInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("failed to unmarshal input params: %v", err)
	}

	setPrivilege := map[string]map[string]string{
		"grant":  map[string]string{},
		"revoke": map[string]string{},
		"input":  map[string]string{},
	}

	for i, privilege := range input.Privileges {
		database, err := instance.GetDBInstanceDatabase(privilege.Database)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Failed to found database %s for dbinstance %s(%s): %v", privilege.Database, instance.Name, instance.Id, err)
		}
		input.Privileges[i].DBInstancedatabaseId = database.Id
		err = instance.GetRegion().GetDriver().ValidateDBInstanceAccountPrivilege(ctx, userCred, instance, self.Name, privilege.Privilege)
		if err != nil {
			return nil, err
		}
		dbPrivilege, _ := instance.GetDBInstancePrivilege(self.Id, database.Id)
		if dbPrivilege == nil {
			setPrivilege["grant"][database.Id] = privilege.Privilege
		} else if dbPrivilege.Privilege != privilege.Privilege {
			setPrivilege["grant"][database.Id] = privilege.Privilege
			setPrivilege["revoke"][database.Id] = dbPrivilege.Privilege
		}
		setPrivilege["input"][database.Id] = privilege.Privilege
	}

	dbPrivileges, err := self.GetDBInstancePrivileges()
	if err != nil {
		return nil, err
	}
	for _, privilege := range dbPrivileges {
		if _, ok := setPrivilege["input"][privilege.DBInstancedatabaseId]; !ok {
			setPrivilege["revoke"][privilege.DBInstancedatabaseId] = privilege.Privilege
		}
	}

	return nil, self.StartSetPrivilegesTask(ctx, userCred, jsonutils.Marshal(setPrivilege))
}

func (self *SDBInstanceAccount) StartSetPrivilegesTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	self.SetStatus(userCred, api.DBINSTANCE_USER_SET_PRIVILEGE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceAccountSetPrivilegesTask", self, userCred, data.(*jsonutils.JSONDict), "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil

}

func (self *SDBInstanceAccount) StartGrantPrivilegeTask(ctx context.Context, userCred mcclient.TokenCredential, database string, privilege string, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_USER_GRANT_PRIVILEGE, "")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(database), "database")
	params.Add(jsonutils.NewString(privilege), "privilege")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceAccountGrantPrivilegeTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstanceAccount) AllowPerformRevokePrivilege(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "revoke-privilege")
}

func (self *SDBInstanceAccount) PerformRevokePrivilege(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.DBINSTANCE_USER_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("Account status is not %s current status is %s", api.DBINSTANCE_USER_AVAILABLE, self.Status)
	}
	instance, err := self.GetDBInstance()
	if err != nil {
		return nil, errors.Wrap(err, "failed to found dbinstance")
	}
	if instance.Status != api.DBINSTANCE_RUNNING {
		return nil, httperrors.NewInvalidStatusError("Instance status is not %s current status is %s", api.DBINSTANCE_RUNNING, instance.Status)
	}
	databaseStr, _ := data.GetString("database")
	if len(databaseStr) == 0 {
		return nil, httperrors.NewMissingParameterError("database")
	}
	database, err := instance.GetDBInstanceDatabase(databaseStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Failed to found database %s for dbinstance %s(%s): %v", databaseStr, instance.Name, instance.Id, err)
	}

	if database.Status != api.DBINSTANCE_DATABASE_RUNNING {
		return nil, httperrors.NewInvalidStatusError("Database status is not %s current is %s", api.DBINSTANCE_DATABASE_RUNNING, database.Status)
	}

	privilege, err := instance.GetDBInstancePrivilege(self.Id, database.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewInputParameterError("Account %s(%s) does not have database %s(%s) permissions", self.Name, self.Id, database.Name, database.Id)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, self.StartRevokePrivilegeTask(ctx, userCred, databaseStr, privilege.Privilege, "")
}

func (self *SDBInstanceAccount) StartRevokePrivilegeTask(ctx context.Context, userCred mcclient.TokenCredential, database string, privilege string, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_USER_REVOKE_PRIVILEGE, "")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(database), "database")
	params.Add(jsonutils.NewString(privilege), "privilege")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceAccountRevokePrivilegeTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstanceAccount) AllowPerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "reset-password")
}

func (self *SDBInstanceAccount) PerformResetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	instance, err := self.GetDBInstance()
	if err != nil {
		return nil, err
	}
	passwdStr, _ := data.GetString("password")
	if len(passwdStr) > 0 {
		err = seclib2.ValidatePassword(passwdStr)
		if err != nil {
			return nil, err
		}
	}
	err = instance.GetRegion().GetDriver().ValidateResetDBInstancePassword(ctx, userCred, instance, self.Name)
	if err != nil {
		return nil, err
	}
	return nil, self.StartDBInstanceAccountResetPasswordTask(ctx, userCred, passwdStr)
}

func (self *SDBInstanceAccount) StartDBInstanceAccountResetPasswordTask(ctx context.Context, userCred mcclient.TokenCredential, password string) error {
	params := jsonutils.NewDict()
	if len(password) > 0 {
		params.Add(jsonutils.NewString(password), "password")
	} else {
		params.Add(jsonutils.NewString(seclib2.RandomPassword2(20)), "password")
	}
	self.SetStatus(userCred, api.DBINSTANCE_USER_RESET_PASSWD, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceAccountResetPasswordTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstanceAccount) GetDBInstancePrivileges() ([]SDBInstancePrivilege, error) {
	privileges := []SDBInstancePrivilege{}
	q := DBInstancePrivilegeManager.Query().Equals("dbinstanceaccount_id", self.Id)
	err := db.FetchModelObjects(DBInstancePrivilegeManager, q, &privileges)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstancePrivileges.FetchModelObjects for account %s", self.Id)
	}
	return privileges, nil
}

func (self *SDBInstanceAccount) GetDBInstanceDatabaseByName(dbName string) (*SDBInstanceDatabase, error) {
	q := DBInstanceDatabaseManager.Query().Equals("dbinstance_id", self.DBInstanceId).Equals("name", dbName)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		database := &SDBInstanceDatabase{}
		database.SetModelManager(DBInstanceDatabaseManager, database)
		err = q.First(database)
		if err != nil {
			return nil, err
		}
		return database, nil
	}
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	return nil, sql.ErrNoRows
}

func (manager *SDBInstanceAccountManager) SyncDBInstanceAccounts(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, cloudAccounts []cloudprovider.ICloudDBInstanceAccount) ([]SDBInstanceAccount, []cloudprovider.ICloudDBInstanceAccount, compare.SyncResult) {
	lockman.LockRawObject(ctx, "dbinstance-accounts", instance.Id)
	defer lockman.ReleaseRawObject(ctx, "dbinstance-accounts", instance.Id)

	result := compare.SyncResult{}
	localAccounts := []SDBInstanceAccount{}
	remoteAccounts := []cloudprovider.ICloudDBInstanceAccount{}
	dbAccounts, err := instance.GetDBInstanceAccounts()
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}
	accountMaps := map[string][]SDBInstanceAccount{}
	for i := range dbAccounts {
		key := fmt.Sprintf("%s:%s", dbAccounts[i].Name, dbAccounts[i].Host)
		_, ok := accountMaps[key]
		if !ok {
			accountMaps[key] = []SDBInstanceAccount{}
		}
		accountMaps[key] = append(accountMaps[key], dbAccounts[i])
	}
	remoteMaps := map[string]cloudprovider.ICloudDBInstanceAccount{}
	for i := range cloudAccounts {
		remoteMaps[fmt.Sprintf("%s:%s", cloudAccounts[i].GetName(), cloudAccounts[i].GetHost())] = cloudAccounts[i]
	}

	for key, account := range remoteMaps {
		locals, ok := accountMaps[key]
		if !ok {
			_account, err := manager.newFromCloudDBInstanceAccount(ctx, userCred, instance, account)
			if err != nil {
				result.AddError(err)
				continue
			}
			result.Add()
			remoteAccounts = append(remoteAccounts, account)
			localAccounts = append(localAccounts, *_account)
			continue
		}
		password := ""
		for i := range locals {
			if i == 0 {
				err = locals[i].SyncWithCloudDBInstanceAccount(ctx, userCred, instance, account)
				if err != nil {
					result.UpdateError(err)
					continue
				}
				result.Update()
				remoteAccounts = append(remoteAccounts, account)
				localAccounts = append(localAccounts, locals[0])
			} else {
				if passwd, err := locals[i].GetPassword(); err == nil && len(passwd) > 0 {
					password = passwd
				}
				err := locals[i].Purge(ctx, userCred)
				if err != nil {
					result.DeleteError(err)
					continue
				}
				result.Delete()
			}
		}
		if len(password) > 0 {
			locals[0].savePassword(password)
		}
	}

	for key, accounts := range accountMaps {
		_, ok := remoteMaps[key]
		if !ok {
			for i := range accounts {
				err := accounts[i].Purge(ctx, userCred)
				if err != nil {
					result.DeleteError(err)
					continue
				}
				result.Delete()
			}
		}
	}

	return localAccounts, remoteAccounts, result
}

func (self *SDBInstanceAccount) SyncWithCloudDBInstanceAccount(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extAccount cloudprovider.ICloudDBInstanceAccount) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extAccount.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstanceAccount.UpdateWithLock")
	}
	return nil
}

func (manager *SDBInstanceAccountManager) newFromCloudDBInstanceAccount(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extAccount cloudprovider.ICloudDBInstanceAccount) (*SDBInstanceAccount, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	account := SDBInstanceAccount{}
	account.SetModelManager(manager, &account)

	account.Name = extAccount.GetName()
	account.DBInstanceId = instance.Id
	account.Status = extAccount.GetStatus()
	account.Host = extAccount.GetHost()

	err := manager.TableSpec().Insert(ctx, &account)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudDBInstanceAccount.Insert")
	}
	return &account, nil
}

func (self *SDBInstanceAccount) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("dbinstance account delete do nothing")
	return nil
}

func (self *SDBInstanceAccount) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SDBInstanceAccount) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDBInstanceAccountDeleteTask(ctx, userCred, "")
}

func (self *SDBInstanceAccount) StartDBInstanceAccountDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_USER_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceAccountDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SDBInstanceAccountManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SDBInstanceResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SDBInstanceResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (manager *SDBInstanceAccountManager) InitializeData() error {
	sq := DBInstanceManager.Query("id")
	q := manager.Query().NotIn("dbinstance_id", sq.SubQuery())
	accounts := []SDBInstanceAccount{}
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range accounts {
		err = accounts[i].Purge(context.Background(), nil)
		if err != nil {
			return errors.Wrapf(err, "purge %s", accounts[i].Id)
		}
	}
	log.Debugf("SDBInstanceAccountManager cleaned %d dirty data.", len(accounts))
	return nil
}
