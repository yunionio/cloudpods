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
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SDBInstanceAccountManager struct {
	db.SVirtualResourceBaseManager
}

var DBInstanceAccountManager *SDBInstanceAccountManager

func init() {
	DBInstanceAccountManager = &SDBInstanceAccountManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDBInstanceAccount{},
			"dbinstanceaccounts_tbl",
			"dbinstanceaccount",
			"dbinstanceaccounts",
		),
	}
	DBInstanceAccountManager.SetVirtualObject(DBInstanceAccountManager)
}

type SDBInstanceAccount struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	Secret       string `width:"256" charset:"ascii" nullable:"false" list:"domain" create:"optional"`
	DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SDBInstanceAccountManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

func (self *SDBInstanceAccountManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceAccountManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceAccount) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceAccount) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SDBInstanceAccount) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SDBInstanceAccount) getPrivilegesDetails() (*jsonutils.JSONArray, error) {
	result := jsonutils.NewArray()
	privileges, err := self.GetDBInstancePrivileges()
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstancePrivileges")
	}
	for _, privilege := range privileges {
		detail, err := privilege.GetDetailedJson()
		if err != nil {
			return nil, errors.Wrap(err, "GetDetailedJson")
		}
		result.Add(detail)
	}
	return result, nil
}

func (self *SDBInstanceAccount) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra, _ = self.getMoreDetails(ctx, userCred, extra)
	return extra
}

func (self *SDBInstanceAccount) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, extra *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	privileges, err := self.getPrivilegesDetails()
	if err != nil {
		return extra, err
	}
	extra.Add(privileges, "dbinstanceprivileges")
	return extra, nil
}

func (self *SDBInstanceAccount) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, userCred, extra)
}

func (manager *SDBInstanceAccountManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstance", ModelKeyword: "dbinstance", OwnerId: userCred},
	})
}

func (manager *SDBInstanceAccountManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	parentId, _ := data.GetString("dbinstance_id")
	return parentId
}

func (manager *SDBInstanceAccountManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("dbinstance_id", parentId)
	}
	return q
}

func (manager *SDBInstanceAccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := &api.SDBInstanceAccountCreateInput{}
	instanceV := validators.NewModelIdOrNameValidator("dbinstance", "dbinstance", userCred)
	err := instanceV.Validate(data)
	if err != nil {
		return nil, err
	}
	err = data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input params error: %v", err)
	}
	instance := instanceV.Model.(*SDBInstance)
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
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := &api.SDBInstanceAccountCreateInput{}
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

func (self *SDBInstanceAccount) GetDBInstance() (*SDBInstance, error) {
	instance, err := DBInstanceManager.FetchById(self.DBInstanceId)
	if err != nil {
		return nil, err
	}
	return instance.(*SDBInstance), nil
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
		if !seclib2.MeetComplxity(passwdStr) {
			return nil, httperrors.NewWeakPasswordError()
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
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))

	result := compare.SyncResult{}
	dbAccounts, err := instance.GetDBInstanceAccounts()
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	localAccounts := []SDBInstanceAccount{}
	remoteAccounts := []cloudprovider.ICloudDBInstanceAccount{}

	removed := make([]SDBInstanceAccount, 0)
	commondb := make([]SDBInstanceAccount, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceAccount, 0)
	added := make([]cloudprovider.ICloudDBInstanceAccount, 0)
	if err := compare.CompareSets(dbAccounts, cloudAccounts, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].Purge(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstanceAccount(ctx, userCred, instance, commonext[i])
		if err != nil {
			result.UpdateError(err)
		} else {
			result.Update()
			localAccounts = append(localAccounts, commondb[i])
			remoteAccounts = append(remoteAccounts, commonext[i])
		}
	}

	for i := 0; i < len(added); i++ {
		account, err := manager.newFromCloudDBInstanceAccount(ctx, userCred, instance, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			localAccounts = append(localAccounts, *account)
			remoteAccounts = append(remoteAccounts, added[i])
			result.Add()
		}
	}
	return localAccounts, remoteAccounts, result
}

func (self *SDBInstanceAccount) SyncWithCloudDBInstanceAccount(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extAccount cloudprovider.ICloudDBInstanceAccount) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.ProjectId = instance.ProjectId
		self.DomainId = instance.DomainId
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
	account.ExternalId = extAccount.GetGlobalId()
	account.ProjectId = instance.ProjectId
	account.DomainId = instance.DomainId

	err := manager.TableSpec().Insert(&account)
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
	return self.SVirtualResourceBase.Delete(ctx, userCred)
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
