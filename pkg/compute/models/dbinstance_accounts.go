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

type SDBInstanceAccountManager struct {
	db.SStatusStandaloneResourceBaseManager
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
	db.SExternalizedResourceBase

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
	return db.IsAdminAllowUpdate(userCred, self)
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

func (self *SDBInstanceAccount) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	privileges, err := self.getPrivilegesDetails()
	if err != nil {
		return nil, err
	}
	extra.Add(privileges, "dbinstanceprivileges")
	return extra, nil
}

func (manager *SDBInstanceAccountManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstance", ModelKeyword: "dbinstance", OwnerId: userCred},
	})
}

func (manager *SDBInstanceAccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
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
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstanceAccount(ctx, userCred, commonext[i])
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

func (self *SDBInstanceAccount) SyncWithCloudDBInstanceAccount(ctx context.Context, userCred mcclient.TokenCredential, extAccount cloudprovider.ICloudDBInstanceAccount) error {
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
	account.ExternalId = extAccount.GetGlobalId()

	err := manager.TableSpec().Insert(&account)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudDBInstanceAccount.Insert")
	}
	return &account, nil
}
