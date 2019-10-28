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

type SDBInstanceDatabaseManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var DBInstanceDatabaseManager *SDBInstanceDatabaseManager

func init() {
	DBInstanceDatabaseManager = &SDBInstanceDatabaseManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SDBInstanceDatabase{},
			"dbinstancedatabases_tbl",
			"dbinstancedatabase",
			"dbinstancedatabases",
		),
	}
	DBInstanceDatabaseManager.SetVirtualObject(DBInstanceDatabaseManager)
}

type SDBInstanceDatabase struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	CharacterSet string `width:"32" charset:"ascii" nullable:"true" list:"user"`
	DBInstanceId string `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SDBInstanceDatabaseManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

func (self *SDBInstanceDatabaseManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceDatabaseManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceDatabase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceDatabase) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDBInstanceDatabase) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SDBInstanceDatabaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstance", ModelKeyword: "dbinstance", OwnerId: userCred},
	})
}

func (manager *SDBInstanceDatabaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SDBInstanceDatabaseManager) SyncDBInstanceDatabases(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, cloudDatabases []cloudprovider.ICloudDBInstanceDatabase) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, instance.GetOwnerId()))

	result := compare.SyncResult{}
	dbDatabases, err := instance.GetDBInstanceDatabases()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceDatabase, 0)
	commondb := make([]SDBInstanceDatabase, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceDatabase, 0)
	added := make([]cloudprovider.ICloudDBInstanceDatabase, 0)
	if err := compare.CompareSets(dbDatabases, cloudDatabases, &removed, &commondb, &commonext, &added); err != nil {
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

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudDBInstanceDatabase(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
		} else {
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromCloudDBInstanceDatabase(ctx, userCred, instance, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SDBInstanceDatabase) SyncWithCloudDBInstanceDatabase(ctx context.Context, userCred mcclient.TokenCredential, extDatabase cloudprovider.ICloudDBInstanceDatabase) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extDatabase.GetStatus()
		self.Name = extDatabase.GetName()
		self.CharacterSet = extDatabase.GetCharacterSet()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstanceDatabase.UpdateWithLock")
	}
	return nil
}

func (manager *SDBInstanceDatabaseManager) newFromCloudDBInstanceDatabase(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, extDatabase cloudprovider.ICloudDBInstanceDatabase) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	database := SDBInstanceDatabase{}
	database.SetModelManager(manager, &database)

	database.Name = extDatabase.GetName()
	database.DBInstanceId = instance.Id
	database.Status = extDatabase.GetStatus()
	database.CharacterSet = extDatabase.GetCharacterSet()
	database.ExternalId = extDatabase.GetGlobalId()

	err := manager.TableSpec().Insert(&database)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceDatabase.Insert")
	}
	return nil
}
