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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstancePrivilegeManager struct {
	db.SResourceBaseManager
}

var DBInstancePrivilegeManager *SDBInstancePrivilegeManager

func init() {
	DBInstancePrivilegeManager = &SDBInstancePrivilegeManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SDBInstancePrivilege{},
			"dbinstanceprivileges_tbl",
			"dbinstanceprivilege",
			"dbinstanceprivileges",
		),
	}
	DBInstancePrivilegeManager.SetVirtualObject(DBInstancePrivilegeManager)
}

type SDBInstancePrivilege struct {
	db.SResourceBase
	db.SExternalizedResourceBase

	Id                   string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Privilege            string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DBInstanceaccountId  string `width:"36" charset:"ascii" name:"dbinstanceaccount_id" nullable:"false" list:"user" create:"required"`
	DBInstancedatabaseId string `width:"36" charset:"ascii" name:"dbinstancedatabase_id" nullable:"false" list:"user" create:"required"`
}

func (self *SDBInstancePrivilege) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = db.DefaultUUIDGenerator()
	}
}

func (manager *SDBInstancePrivilegeManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceAccountManager, DBInstanceDatabaseManager},
	}
}

func (self *SDBInstancePrivilegeManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstancePrivilegeManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstancePrivilege) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstancePrivilege) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDBInstancePrivilege) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SDBInstancePrivilege) GetDBInstanceAccount() (*SDBInstanceAccount, error) {
	account, err := db.FetchById(DBInstanceAccountManager, self.DBInstanceaccountId)
	if err != nil {
		return nil, err
	}
	return account.(*SDBInstanceAccount), nil
}

func (self *SDBInstancePrivilege) GetDBInstanceDatabase() (*SDBInstanceDatabase, error) {
	database, err := db.FetchById(DBInstanceDatabaseManager, self.DBInstancedatabaseId)
	if err != nil {
		return nil, err
	}
	return database.(*SDBInstanceDatabase), nil
}

func (self *SDBInstancePrivilege) GetDetailedJson() (*jsonutils.JSONDict, error) {
	result := jsonutils.NewDict()
	database, err := self.GetDBInstanceDatabase()
	if err != nil {
		return nil, err
	}
	account, err := self.GetDBInstanceAccount()
	if err != nil {
		return nil, err
	}
	result.Add(jsonutils.NewString(database.Name), "database")
	result.Add(jsonutils.NewString(account.Name), "account")
	result.Add(jsonutils.NewString(database.Id), "dbinstancedatabase_id")
	result.Add(jsonutils.NewString(self.Privilege), "privileges")
	return result, nil
}

func (manager *SDBInstancePrivilegeManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstanceaccount", ModelKeyword: "dbinstanceaccount", OwnerId: userCred},
		{Key: "dbinstancedatabase", ModelKeyword: "dbinstancedatabase", OwnerId: userCred},
	})
}

func (manager *SDBInstancePrivilegeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SDBInstancePrivilegeManager) SyncDBInstanceAccountPrivileges(ctx context.Context, userCred mcclient.TokenCredential, account *SDBInstanceAccount, cloudPrivileges []cloudprovider.ICloudDBInstanceAccountPrivilege) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, account.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, account.GetOwnerId()))

	result := compare.SyncResult{}
	dbPrivileges, err := account.GetDBInstancePrivileges()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstancePrivilege, 0)
	commondb := make([]SDBInstancePrivilege, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceAccountPrivilege, 0)
	added := make([]cloudprovider.ICloudDBInstanceAccountPrivilege, 0)
	if err := compare.CompareSets(dbPrivileges, cloudPrivileges, &removed, &commondb, &commonext, &added); err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromCloudPrivileges(ctx, userCred, account, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (manager *SDBInstancePrivilegeManager) newFromCloudPrivileges(ctx context.Context, userCred mcclient.TokenCredential, account *SDBInstanceAccount, ext cloudprovider.ICloudDBInstanceAccountPrivilege) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	privilege := SDBInstancePrivilege{}
	privilege.SetModelManager(manager, &privilege)

	privilege.DBInstanceaccountId = account.Id
	privilege.ExternalId = ext.GetGlobalId()
	privilege.Privilege = ext.GetPrivilege()

	dbName := ext.GetDBName()

	database, err := account.GetDBInstanceDatabaseByName(dbName)
	if err != nil {
		return errors.Wrapf(err, "account.GetDBInstanceDatabaseByName(%s)", dbName)
	}

	privilege.DBInstancedatabaseId = database.Id

	err = manager.TableSpec().Insert(&privilege)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceDatabase.Insert")
	}
	return nil
}
