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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"
)

type SDBInstanceAccountPrivilegeManager struct {
	db.SResourceBaseManager
}

var DBInstanceAccountPrivilegeManager *SDBInstanceAccountPrivilegeManager

func init() {
	DBInstanceAccountPrivilegeManager = &SDBInstanceAccountPrivilegeManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SDBInstanceAccountPrivilege{},
			"dbinstanceaccountprivilege_tbl",
			"dbinstanceaccountprivilege",
			"dbinstanceaccountprivileges",
		),
	}
	DBInstanceAccountPrivilegeManager.SetVirtualObject(DBInstanceAccountPrivilegeManager)
}

type SDBInstanceAccountPrivilege struct {
	db.SResourceBase
	db.SExternalizedResourceBase

	Privilege            string `width:"32" charset:"ascii" nullable:"false" list:"user"`
	DBInstanceaccountId  string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
	DBInstancedatabaseId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SDBInstanceAccountPrivilegeManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceAccountManager, DBInstanceDatabaseManager},
	}
}

func (self *SDBInstanceAccountPrivilegeManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceAccountPrivilegeManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceAccountPrivilege) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceAccountPrivilege) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDBInstanceAccountPrivilege) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SDBInstanceAccountPrivilegeManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
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

func (manager *SDBInstanceAccountPrivilegeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SDBInstanceAccountPrivilegeManager) getPrivilegesByAccount(account *SDBInstanceAccount) ([]SDBInstanceAccountPrivilege, error) {
	privileges := []SDBInstanceAccountPrivilege{}
	q := manager.Query().Equals("dbinstanceaccount_id", account.Id)
	err := db.FetchModelObjects(manager, q, &privileges)
	if err != nil {
		return nil, errors.Wrap(err, "getPrivilegesByAccount.FetchModelObjects")
	}
	return privileges, nil
}

func (manager *SDBInstanceAccountPrivilegeManager) SyncDBInstanceAccountPrivileges(ctx context.Context, userCred mcclient.TokenCredential, account *SDBInstanceAccount, cloudPrivileges []cloudprovider.ICloudDBInstanceAccountPrivilege) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, account.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, account.GetOwnerId()))

	result := compare.SyncResult{}
	dbPrivileges, err := manager.getPrivilegesByAccount(account)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceAccountPrivilege, 0)
	commondb := make([]SDBInstanceAccountPrivilege, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceDatabase, 0)
	added := make([]cloudprovider.ICloudDBInstanceDatabase, 0)
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

func (manager *SDBInstanceAccountPrivilegeManager) newFromCloudPrivileges(ctx context.Context, userCred mcclient.TokenCredential, account *SDBInstanceAccount, ext cloudprovider.ICloudDBInstanceAccountPrivilege) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	privilege := SDBInstanceAccountPrivilege{}
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
