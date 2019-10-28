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
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstanceBackupManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var DBInstanceBackupManager *SDBInstanceBackupManager

func init() {
	DBInstanceBackupManager = &SDBInstanceBackupManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SDBInstanceBackup{},
			"dbinstancebackups_tbl",
			"dbinstancebackup",
			"dbinstancebackups",
		),
	}
	DBInstanceBackupManager.SetVirtualObject(DBInstanceBackupManager)
}

type SDBInstanceBackup struct {
	db.SStatusStandaloneResourceBase
	SCloudregionResourceBase
	SManagedResourceBase
	db.SExternalizedResourceBase

	StartTime    time.Time `list:"user"`
	EndTime      time.Time `list:"user"`
	BackupMode   string    `width:"32" charset:"ascii" nullable:"true" list:"user"`
	DBNames      string    `width:"512" charset:"ascii" nullable:"true" list:"user"`
	BackupSizeMb int       `nullable:"false" list:"user"`
	DBInstanceId string    `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SDBInstanceBackupManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

func (self *SDBInstanceBackupManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SDBInstanceBackupManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SDBInstanceBackup) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SDBInstanceBackup) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SDBInstanceBackup) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SDBInstanceBackupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	return validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "dbinstance", ModelKeyword: "dbinstance", OwnerId: userCred},
		{Key: "cloudregion", ModelKeyword: "cloudregion", OwnerId: userCred},
	})
}

func (manager *SDBInstanceBackupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SDBInstanceBackupManager) getDBInstanceBackupsByInstance(instance *SDBInstance) ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	q := manager.Query().Equals("dbinstance_id", instance.Id)
	err := db.FetchModelObjects(manager, q, &backups)
	if err != nil {
		return nil, errors.Wrap(err, "getDBInstanceBackupsByInstance.FetchModelObjects")
	}
	return backups, nil
}

func (manager *SDBInstanceBackupManager) getDBInstanceBackupsByProviderId(providerId string) ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	err := fetchByManagerId(manager, providerId, &backups)
	if err != nil {
		return nil, errors.Wrapf(err, "getDBInstanceBackupsByProviderId.fetchByManagerId")
	}
	return backups, nil
}

func (self *SDBInstanceBackup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
}

func (manager *SDBInstanceBackupManager) SyncDBInstanceBackups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, cloudBackups []cloudprovider.ICloudDBInstanceBackup) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))

	result := compare.SyncResult{}
	dbBackups, err := region.GetDBInstanceBackups(provider)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDBInstanceBackup, 0)
	commondb := make([]SDBInstanceBackup, 0)
	commonext := make([]cloudprovider.ICloudDBInstanceBackup, 0)
	added := make([]cloudprovider.ICloudDBInstanceBackup, 0)
	if err := compare.CompareSets(dbBackups, cloudBackups, &removed, &commondb, &commonext, &added); err != nil {
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
		err := commondb[i].SyncWithCloudDBInstanceBackup(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
		} else {
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		err = manager.newFromCloudDBInstanceBackup(ctx, userCred, provider, region, added[i])
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func (self *SDBInstanceBackup) SyncWithCloudDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, extBackup cloudprovider.ICloudDBInstanceBackup) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extBackup.GetStatus()
		self.StartTime = extBackup.GetStartTime()
		self.EndTime = extBackup.GetEndTime()
		self.BackupSizeMb = extBackup.GetBackupSizeMb()
		self.DBNames = extBackup.GetDBNames()

		if dbinstanceId := extBackup.GetDBInstanceId(); len(dbinstanceId) > 0 {
			//有可能云上删除了实例，未删除备份
			_, err := db.FetchByExternalId(DBInstanceManager, dbinstanceId)
			if err == sql.ErrNoRows {
				self.DBInstanceId = ""
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstancebackup.UpdateWithLock")
	}
	return nil
}

func (manager *SDBInstanceBackupManager) newFromCloudDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, extBackup cloudprovider.ICloudDBInstanceBackup) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	backup := SDBInstanceBackup{}
	backup.SetModelManager(manager, &backup)

	newName, err := db.GenerateName(manager, provider.GetOwnerId(), extBackup.GetName())
	if err != nil {
		return errors.Wrap(err, "newFromCloudDBInstanceBackup.GenerateName")
	}

	backup.Name = newName
	backup.CloudregionId = region.Id
	backup.ManagerId = provider.Id
	backup.Status = extBackup.GetStatus()
	backup.StartTime = extBackup.GetStartTime()
	backup.EndTime = extBackup.GetEndTime()
	backup.BackupSizeMb = extBackup.GetBackupSizeMb()
	backup.DBNames = extBackup.GetDBNames()
	backup.BackupMode = extBackup.GetBackupMode()
	backup.ExternalId = extBackup.GetGlobalId()

	if dbinstanceId := extBackup.GetDBInstanceId(); len(dbinstanceId) > 0 {
		dbinstance, err := db.FetchByExternalId(DBInstanceManager, dbinstanceId)
		if err != nil {
			log.Warningf("failed to found dbinstance for backup %s by externalId: %s error: %v", backup.Name, dbinstanceId, err)
		} else {
			backup.DBInstanceId = dbinstance.GetId()
		}
	}

	err = manager.TableSpec().Insert(&backup)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceBackup.Insert")
	}
	return nil
}
