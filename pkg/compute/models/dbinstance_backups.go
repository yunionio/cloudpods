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
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDBInstanceBackupManager struct {
	db.SVirtualResourceBaseManager
}

var DBInstanceBackupManager *SDBInstanceBackupManager

func init() {
	DBInstanceBackupManager = &SDBInstanceBackupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDBInstanceBackup{},
			"dbinstancebackups_tbl",
			"dbinstancebackup",
			"dbinstancebackups",
		),
	}
	DBInstanceBackupManager.SetVirtualObject(DBInstanceBackupManager)
}

type SDBInstanceBackup struct {
	db.SVirtualResourceBase
	SCloudregionResourceBase
	SManagedResourceBase
	db.SExternalizedResourceBase

	Engine        string    `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	EngineVersion string    `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	StartTime     time.Time `list:"user"`
	EndTime       time.Time `list:"user"`
	BackupMode    string    `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	DBNames       string    `width:"512" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	BackupSizeMb  int       `nullable:"false" list:"user"`
	DBInstanceId  string    `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`
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
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}

	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = managedResourceFilterByDomain(q, query, "", nil)
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
	instanceV := validators.NewModelIdOrNameValidator("dbinstance", "dbinstance", userCred)
	err := instanceV.Validate(data)
	if err != nil {
		return nil, err
	}
	input := &api.SDBInstanceBackupCreateInput{}
	err = data.Unmarshal(input)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Failed to unmarshal input params: %v", err)
	}
	input.BackupMode = api.BACKUP_MODE_MANUAL
	input.DBNames = strings.Join(input.Databases, ",")
	instance := instanceV.Model.(*SDBInstance)
	if instance.Status != api.DBINSTANCE_RUNNING {
		return nil, httperrors.NewInputParameterError("DBInstance %s(%s) status is %s require status is %s", instance.Name, instance.Id, instance.Status, api.DBINSTANCE_RUNNING)
	}
	input.Engine = instance.Engine
	input.EngineVersion = instance.EngineVersion
	input.ManagerId = instance.ManagerId
	region := instance.GetRegion()
	if region == nil {
		return nil, httperrors.NewInputParameterError("failed to found region for dbinstance %s(%s)", instance.Name, instance.Id)
	}
	input.CloudregionId = region.Id
	input, err = region.GetDriver().ValidateCreateDBInstanceBackupData(ctx, userCred, ownerId, instance, input)
	if err != nil {
		return nil, err
	}

	return input.JSON(input), nil
}

func (self *SDBInstanceBackup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	instance, _ := self.GetDBInstance()
	if instance != nil {
		self.SetProjectInfo(ctx, userCred, instance.ProjectId, instance.DomainId)
	}
	self.StartDBInstanceBackupCreateTask(ctx, userCred, nil, "")
}

func (self *SDBInstanceBackup) StartDBInstanceBackupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_BACKUP_CREATING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceBackupCreateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDBInstanceBackup) GetIRegion() (cloudprovider.ICloudRegion, error) {
	driver, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to found region for rds backup %s(%s)", self.Name, self.Id)
	}
	return driver.GetIRegionById(region.ExternalId)
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

func (self *SDBInstanceBackup) GetDBInstance() (*SDBInstance, error) {
	if len(self.DBInstanceId) > 0 {
		instance, err := DBInstanceManager.FetchById(self.DBInstanceId)
		if err != nil {
			return nil, err
		}
		return instance.(*SDBInstance), nil
	}
	return nil, fmt.Errorf("empty dbinstance id")
}

func (self *SDBInstanceBackup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	dbinstance, err := self.GetDBInstance()
	if err == nil {
		extra.Add(jsonutils.NewString(dbinstance.Name), "dbinstance")
	}
	cloudregionInfo := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if cloudregionInfo != nil {
		extra.Update(cloudregionInfo)
	}

	accountInfo := self.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if accountInfo != nil {
		extra.Update(accountInfo)
	}

	return extra
}

func (manager *SDBInstanceBackupManager) SyncDBInstanceBackups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, instance *SDBInstance, region *SCloudregion, cloudBackups []cloudprovider.ICloudDBInstanceBackup) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))

	result := compare.SyncResult{}
	dbBackups, err := region.GetDBInstanceBackups(provider, instance)
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
		err := removed[i].RealDelete(ctx, userCred)
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
		self.Engine = extBackup.GetEngine()
		self.EngineVersion = extBackup.GetEngineVersion()
		self.DBNames = extBackup.GetDBNames()

		if dbinstanceId := extBackup.GetDBInstanceId(); len(dbinstanceId) > 0 {
			//有可能云上删除了实例，未删除备份
			_instance, err := db.FetchByExternalId(DBInstanceManager, dbinstanceId)
			if err == sql.ErrNoRows {
				self.DBInstanceId = ""
			}
			if _instance != nil {
				instance := _instance.(*SDBInstance)
				self.ProjectId = instance.ProjectId
				self.DomainId = instance.DomainId
			}
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudDBInstancebackup.UpdateWithLock")
	}

	if len(self.ProjectId) == 0 {
		provider := self.GetCloudprovider()
		SyncCloudProject(userCred, self, provider.GetOwnerId(), extBackup, self.ManagerId)
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
	backup.Engine = extBackup.GetEngine()
	backup.EngineVersion = extBackup.GetEngineVersion()
	backup.EndTime = extBackup.GetEndTime()
	backup.BackupSizeMb = extBackup.GetBackupSizeMb()
	backup.DBNames = extBackup.GetDBNames()
	backup.BackupMode = extBackup.GetBackupMode()
	backup.ExternalId = extBackup.GetGlobalId()

	if dbinstanceId := extBackup.GetDBInstanceId(); len(dbinstanceId) > 0 {
		_dbinstance, err := db.FetchByExternalId(DBInstanceManager, dbinstanceId)
		if err != nil {
			log.Warningf("failed to found dbinstance for backup %s by externalId: %s error: %v", backup.Name, dbinstanceId, err)
		} else {
			instance := _dbinstance.(*SDBInstance)
			backup.DBInstanceId = instance.Id
			backup.ProjectId = instance.ProjectId
			backup.DomainId = instance.DomainId
		}
	}

	err = manager.TableSpec().Insert(&backup)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceBackup.Insert")
	}

	if len(backup.ProjectId) == 0 {
		SyncCloudProject(userCred, &backup, provider.GetOwnerId(), extBackup, backup.ManagerId)
	}

	return nil
}

func (self *SDBInstanceBackup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("dbinstance backup delete do nothing")
	return nil
}

func (self *SDBInstanceBackup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SDBInstanceBackup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDBInstanceBackupDeleteTask(ctx, userCred, "")
}

func (self *SDBInstanceBackup) StartDBInstanceBackupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(userCred, api.DBINSTANCE_BACKUP_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceBackupDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
