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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDBInstanceBackupManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SDBInstanceResourceBaseManager
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

	SDBInstanceResourceBase `width:"36" charset:"ascii" name:"dbinstance_id" nullable:"false" list:"user" create:"required" index:"true"`

	// RDS引擎
	// example: MySQL
	Engine string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required" json:"engine"`
	// RDS引擎版本
	// example: 5.7
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required" json:"engine_version"`
	// 备份开始时间
	StartTime time.Time `list:"user" json:"start_time"`
	// 备份结束时间
	EndTime time.Time `list:"user" json:"end_time"`
	// 备份模式
	BackupMode string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"backup_mode"`
	// 备份数据库名称
	DBNames string `width:"512" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"db_names"`
	// 备份大小
	// example: 32
	BackupSizeMb int `nullable:"false" list:"user" json:"backup_size_mb"`

	// 备份方式 Logical|Physical
	BackupMethod string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"backup_method"`
}

func (manager *SDBInstanceBackupManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{DBInstanceManager},
	}
}

// RDS备份列表
func (manager *SDBInstanceBackupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceBackupListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	dbQuery := api.DBInstanceFilterListInput{
		DBInstanceFilterListInputBase: query.DBInstanceFilterListInputBase,
	}
	q, err = manager.SDBInstanceResourceBaseManager.ListItemFilter(ctx, q, userCred, dbQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.ListItemFilter")
	}

	if len(query.Engine) > 0 {
		q = q.In("engine", query.Engine)
	}
	if len(query.EngineVersion) > 0 {
		q = q.In("engine_version", query.EngineVersion)
	}
	if len(query.BackupMode) > 0 {
		q = q.In("backup_mode", query.BackupMode)
	}
	if len(query.DBNames) > 0 {
		q = q.Contains("db_names", query.DBNames)
	}

	return q, nil
}

func (manager *SDBInstanceBackupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DBInstanceBackupListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	dbQuery := api.DBInstanceFilterListInput{
		DBInstanceFilterListInputBase: query.DBInstanceFilterListInputBase,
	}
	q, err = manager.SDBInstanceResourceBaseManager.OrderByExtraFields(ctx, q, userCred, dbQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SDBInstanceBackupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SDBInstanceResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SDBInstanceBackupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DBInstanceBackupCreateInput) (*jsonutils.JSONDict, error) {
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
	input.BackupMode = api.BACKUP_MODE_MANUAL
	input.DBNames = strings.Join(input.Databases, ",")
	input.Engine = instance.Engine
	input.EngineVersion = instance.EngineVersion
	provider := instance.GetCloudprovider()
	if provider == nil {
		return nil, httperrors.NewInvalidStatusError("DBinstance has not valid cloudprovider")
	}
	input.ManagerId = provider.Id

	if instance.Status != api.DBINSTANCE_RUNNING {
		return nil, httperrors.NewInputParameterError("DBInstance %s(%s) status is %s require status is %s", instance.Name, instance.Id, instance.Status, api.DBINSTANCE_RUNNING)
	}
	region := instance.GetRegion()
	if region == nil {
		return nil, httperrors.NewInputParameterError("failed to found region for dbinstance %s(%s)", instance.Name, instance.Id)
	}
	input.CloudregionId = region.Id
	input, err = region.GetDriver().ValidateCreateDBInstanceBackupData(ctx, userCred, ownerId, instance, input)
	if err != nil {
		return nil, err
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
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
	task, err := taskman.TaskManager.NewTask(ctx, "DBInstanceBackupCreateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	instance, err := self.GetDBInstance()
	if err != nil {
		return errors.Wrap(err, "GetDBInstance")
	}
	instance.SetStatus(userCred, api.DBINSTANCE_BACKING_UP, "")
	self.SetStatus(userCred, api.DBINSTANCE_BACKUP_CREATING, "")
	task.ScheduleRun(nil)
	return nil
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

func (self *SDBInstanceBackup) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.DBInstanceBackupDetails, error) {
	return api.DBInstanceBackupDetails{}, nil
}

func (manager *SDBInstanceBackupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DBInstanceBackupDetails {
	rows := make([]api.DBInstanceBackupDetails, len(objs))

	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dbRows := manager.SDBInstanceResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DBInstanceBackupDetails{
			VirtualResourceDetails:     virtRows[i],
			ManagedResourceInfo:        manRows[i],
			CloudregionResourceInfo:    regRows[i],
			DBInstanceResourceInfoBase: dbRows[i].DBInstanceResourceInfoBase,
		}
	}

	return rows
}

func (self *SDBInstanceBackup) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

// 同步RDS备份状态
func (self *SDBInstanceBackup) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskSyncstatusInput) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("DBInstance backup has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DBInstanceBackupSyncstatusTask", "")
}

func (backup *SDBInstanceBackup) GetIRegion() (cloudprovider.ICloudRegion, error) {
	region := backup.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid cloudregion")
	}
	provider, err := backup.GetDriver()
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (backup *SDBInstanceBackup) GetIDBInstanceBackup() (cloudprovider.ICloudDBInstanceBackup, error) {
	if len(backup.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	if len(backup.DBInstanceId) > 0 {
		rds, err := backup.GetDBInstance()
		if err != nil {
			return nil, errors.Wrapf(err, "GetDBInstance")
		}
		iRds, err := rds.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrapf(err, "GetIDBInstance")
		}
		backups, err := iRds.GetIDBInstanceBackups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetIDBInstanceBackups")
		}
		for i := range backups {
			if backups[i].GetGlobalId() == backup.ExternalId {
				return backups[i], nil
			}
		}
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "search backup %s", backup.ExternalId)
	}

	iRegion, err := backup.GetIRegion()
	if err != nil {
		return nil, errors.Wrap(err, "backup.GetIRegion")
	}
	return iRegion.GetIDBInstanceBackupById(backup.ExternalId)
}

func (manager *SDBInstanceBackupManager) SyncDBInstanceBackups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, instance *SDBInstance, region *SCloudregion, cloudBackups []cloudprovider.ICloudDBInstanceBackup) compare.SyncResult {
	lockman.LockRawObject(ctx, "dbinstance-backups", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "dbinstance-backups", fmt.Sprintf("%s-%s", provider.Id, region.Id))

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
		err := commondb[i].SyncWithCloudDBInstanceBackup(ctx, userCred, commonext[i], provider)
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

func (self *SDBInstanceBackup) SyncWithCloudDBInstanceBackup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extBackup cloudprovider.ICloudDBInstanceBackup,
	provider *SCloudprovider,
) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.ExternalId = extBackup.GetGlobalId()
		self.Status = extBackup.GetStatus()
		self.StartTime = extBackup.GetStartTime()
		self.EndTime = extBackup.GetEndTime()
		self.BackupSizeMb = extBackup.GetBackupSizeMb()
		self.Engine = extBackup.GetEngine()
		self.EngineVersion = extBackup.GetEngineVersion()
		self.DBNames = extBackup.GetDBNames()
		self.BackupMethod = string(extBackup.GetBackupMethod())

		if dbinstanceId := extBackup.GetDBInstanceId(); len(dbinstanceId) > 0 {
			//有可能云上删除了实例，未删除备份
			_instance, err := db.FetchByExternalIdAndManagerId(DBInstanceManager, dbinstanceId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				return q.Equals("manager_id", provider.Id)
			})
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
		SyncCloudProject(userCred, self, provider.GetOwnerId(), extBackup, provider.Id)
	}

	return nil
}

func (manager *SDBInstanceBackupManager) newFromCloudDBInstanceBackup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	region *SCloudregion,
	extBackup cloudprovider.ICloudDBInstanceBackup,
) error {
	backup := SDBInstanceBackup{}
	backup.SetModelManager(manager, &backup)

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
	backup.BackupMethod = string(extBackup.GetBackupMethod())
	backup.ExternalId = extBackup.GetGlobalId()

	if dbinstanceId := extBackup.GetDBInstanceId(); len(dbinstanceId) > 0 {
		_dbinstance, err := db.FetchByExternalIdAndManagerId(DBInstanceManager, dbinstanceId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", provider.Id)
		})
		if err != nil {
			log.Warningf("failed to found dbinstance for backup %s by externalId: %s error: %v", backup.Name, dbinstanceId, err)
		} else {
			instance := _dbinstance.(*SDBInstance)
			backup.DBInstanceId = instance.Id
			backup.ProjectId = instance.ProjectId
			backup.DomainId = instance.DomainId
		}
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, provider.GetOwnerId(), extBackup.GetName())
		if err != nil {
			return errors.Wrap(err, "newFromCloudDBInstanceBackup.GenerateName")
		}
		backup.Name = newName
		return manager.TableSpec().Insert(ctx, &backup)
	}()
	if err != nil {
		return errors.Wrapf(err, "newFromCloudDBInstanceBackup.Insert")
	}

	if len(backup.ProjectId) == 0 {
		SyncCloudProject(userCred, &backup, provider.GetOwnerId(), extBackup, provider.Id)
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

func (self *SDBInstanceBackup) GetCloudprovider() *SCloudprovider {
	return self.SManagedResourceBase.GetCloudprovider()
}

func (self *SDBInstanceBackup) GetRegion() *SCloudregion {
	return self.SCloudregionResourceBase.GetRegion()
}

func (manager *SDBInstanceBackupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny("dbinstance") {
		q, err = manager.SDBInstanceResourceBaseManager.ListItemExportKeys(ctx, q, userCred, stringutils2.NewSortedStrings([]string{"dbinstance"}))
		if err != nil {
			return nil, errors.Wrap(err, "SDBInstanceResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (self *SDBInstanceBackup) GetChangeOwnerCandidateDomainIds() []string {
	return self.SManagedResourceBase.GetChangeOwnerCandidateDomainIds()
}

func (self *SDBInstanceBackup) fillRdsConfig(output *api.DBInstanceCreateInput) error {
	if self.Status != api.DBINSTANCE_BACKUP_READY {
		return fmt.Errorf("backup %s status is %s require %s", self.Name, self.Status, api.DBINSTANCE_BACKUP_READY)
	}
	if len(self.DBInstanceId) == 0 {
		if len(self.Engine) == 0 {
			return fmt.Errorf("backup engine %s is unknown", self.Name)
		}
		output.Engine = self.Engine
		if len(self.EngineVersion) == 0 {
			return fmt.Errorf("backup engine version %s is unknown", self.Name)
		}
		output.EngineVersion = self.EngineVersion
		return nil
	}
	rds, err := self.GetDBInstance()
	if err != nil {
		return errors.Wrapf(err, "backup.GetDBInstance")
	}
	if len(output.NetworkId) == 0 {
		networks, err := rds.GetDBNetworks()
		if err != nil {
			return errors.Wrapf(err, "GetDBNetworks")
		}
		if len(networks) > 0 {
			output.NetworkId = networks[0].NetworkId
		}
	}

	if output.VcpuCount == 0 {
		output.VcpuCount = rds.VcpuCount
	}
	if output.VmemSizeMb == 0 {
		output.VmemSizeMb = rds.VmemSizeMb
	}
	if output.DiskSizeGB == 0 {
		output.DiskSizeGB = rds.DiskSizeGB
	}
	if output.Port == 0 {
		output.Port = rds.Port
	}
	if len(output.Category) == 0 {
		output.Category = rds.Category
	}
	if len(output.StorageType) == 0 {
		output.StorageType = rds.StorageType
	}
	output.Engine = rds.Engine
	output.EngineVersion = rds.EngineVersion
	if len(output.InstanceType) == 0 {
		output.InstanceType = rds.InstanceType
	}
	if len(output.VpcId) == 0 {
		output.VpcId = rds.VpcId
	}
	if len(output.Zone1) == 0 {
		output.Zone1 = rds.Zone1
	}
	if len(output.Zone2) == 0 {
		output.Zone2 = rds.Zone2
	}
	if len(output.Zone3) == 0 {
		output.Zone3 = rds.Zone3
	}
	return nil
}
