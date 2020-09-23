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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SElasticcache.Backup
type SElasticcacheBackupManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SElasticcacheResourceBaseManager
}

var ElasticcacheBackupManager *SElasticcacheBackupManager

func init() {
	ElasticcacheBackupManager = &SElasticcacheBackupManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SElasticcacheBackup{},
			"elasticcachebackups_tbl",
			"elasticcachebackup",
			"elasticcachebackups",
		),
	}
	ElasticcacheBackupManager.SetVirtualObject(ElasticcacheBackupManager)
}

type SElasticcacheBackup struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SElasticcacheResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`

	// ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	// 备份大小
	BackupSizeMb int `nullable:"false" list:"user" create:"optional"`

	// 备份类型, 全量|增量额
	BackupType string `width:"32" charset:"ascii" nullable:"true" create:"optional" list:"user"`

	// 备份模式，自动|手动
	BackupMode string `width:"32" charset:"ascii" nullable:"true" create:"optional" list:"user"`

	// 下载地址
	DownloadURL string `width:"512" charset:"ascii" nullable:"true" create:"optional" list:"user"`

	// 开始备份时间
	StartTime time.Time `list:"user" create:"optional"`
	// 结束备份时间
	EndTime time.Time `list:"user" create:"optional"`
}

func (manager *SElasticcacheBackupManager) SyncElasticcacheBackups(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheBackups []cloudprovider.ICloudElasticcacheBackup) compare.SyncResult {
	lockman.LockRawObject(ctx, "elastic-cache-backups", elasticcache.Id)
	defer lockman.ReleaseRawObject(ctx, "elastic-cache-backups", elasticcache.Id)

	syncResult := compare.SyncResult{}

	dbBackups, err := elasticcache.GetElasticcacheBackups()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheBackup, 0)
	commondb := make([]SElasticcacheBackup, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheBackup, 0)
	added := make([]cloudprovider.ICloudElasticcacheBackup, 0)
	if err := compare.CompareSets(dbBackups, cloudElasticcacheBackups, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheBackup(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheBackup(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheBackup(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheBackup) syncRemoveCloudElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheBackup.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheBackup) SyncWithCloudElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, extBackup cloudprovider.ICloudElasticcacheBackup) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extBackup.GetStatus()

		self.BackupSizeMb = extBackup.GetBackupSizeMb()
		self.BackupType = extBackup.GetBackupType()
		self.BackupMode = extBackup.GetBackupMode()
		self.DownloadURL = extBackup.GetDownloadURL()

		self.StartTime = extBackup.GetStartTime()
		self.EndTime = extBackup.GetEndTime()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheBackup.UpdateWithLock")
	}

	return nil
}

func (manager *SElasticcacheBackupManager) newFromCloudElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extBackup cloudprovider.ICloudElasticcacheBackup) (*SElasticcacheBackup, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	backup := SElasticcacheBackup{}
	backup.SetModelManager(manager, &backup)

	backup.ElasticcacheId = elasticcache.GetId()
	backup.Name = extBackup.GetName()
	backup.ExternalId = extBackup.GetGlobalId()
	backup.Status = extBackup.GetStatus()

	backup.BackupSizeMb = extBackup.GetBackupSizeMb()
	backup.BackupType = extBackup.GetBackupType()
	backup.BackupMode = extBackup.GetBackupMode()
	backup.DownloadURL = extBackup.GetDownloadURL()

	backup.StartTime = extBackup.GetStartTime()
	backup.EndTime = extBackup.GetEndTime()

	err := manager.TableSpec().Insert(ctx, &backup)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheBackup.Insert")
	}

	return &backup, nil
}

func (self *SElasticcacheBackup) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"elasticcache_id": self.ElasticcacheId})
}

func (manager *SElasticcacheBackupManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	cacheId := jsonutils.GetAnyString(data, []string{"elasticcache_id", "elasticcache"})
	return jsonutils.Marshal(map[string]string{"elasticcache_id": cacheId})
}

func (manager *SElasticcacheBackupManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SElasticcacheBackupManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return elasticcacheSubResourceFetchOwnerId(ctx, data)
}

func (manager *SElasticcacheBackupManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return elasticcacheSubResourceFetchOwner(q, userCred, scope)
}

func (manager *SElasticcacheBackupManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	cacheId, _ := values.GetString("elasticcache_id")
	if len(cacheId) > 0 {
		q = q.Equals("elasticcache_id", cacheId)
	}
	return q
}

func (manager *SElasticcacheBackupManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SElasticcacheBackupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var region *SCloudregion
	var ec *SElasticcache
	if id, _ := data.GetString("elasticcache"); len(id) > 0 {
		_ec, err := db.FetchByIdOrName(ElasticcacheManager, userCred, id)
		if err != nil {
			return nil, fmt.Errorf("getting elastic cache instance failed")
		}

		ec = _ec.(*SElasticcache)
	} else {
		return nil, httperrors.NewMissingParameterError("elasticcache")
	}

	region = ec.GetRegion()
	driver := region.GetDriver()
	if err := driver.AllowCreateElasticcacheBackup(ctx, userCred, ownerId, ec); err != nil {
		return nil, err
	}

	input := apis.StandaloneResourceCreateInput{}
	var err error
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	return driver.ValidateCreateElasticcacheBackupData(ctx, userCred, ownerId, data)
}

func (self *SElasticcacheBackup) GetOwnerId() mcclient.IIdentityProvider {
	return ElasticcacheManager.GetOwnerIdByElasticcacheId(self.ElasticcacheId)
}

func (self *SElasticcacheBackup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.SetStatus(userCred, api.ELASTIC_CACHE_BACKUP_STATUS_CREATING, "")
	if err := self.StartElasticcacheBackupCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Failed to create elastic cache backup error: %v", err)
	}
}

func (self *SElasticcacheBackup) StartElasticcacheBackupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheBackupCreateTask", self, userCred, jsonutils.NewDict(), parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheBackup) AllowPerformRestoreInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	// todo : fix me self.IsOwner(userCred) ||
	return db.IsAdminAllowPerform(userCred, self, "restore_instance")
}

func (self *SElasticcacheBackup) ValidatorRestoreInstanceData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ec, err := db.FetchByIdOrName(ElasticcacheManager, userCred, self.ElasticcacheId)
	if err != nil {
		return nil, fmt.Errorf("getting elastic cache instance failed")
	}

	if ec.(*SElasticcache).Status != api.ELASTIC_CACHE_STATUS_RUNNING {
		return nil, httperrors.NewConflictError("can't restore elastic cache in status %s", ec.(*SElasticcache).Status)
	}

	return data, nil
}

func (self *SElasticcacheBackup) PerformRestoreInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, err := self.ValidatorRestoreInstanceData(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}

	self.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_BACKUPRECOVERING, "")
	return nil, self.StartRestoreInstanceTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SElasticcacheBackup) StartRestoreInstanceTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheBackupRestoreInstanceTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}

	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheBackup) GetRegion() *SCloudregion {
	ieb, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return nil
	}

	return ieb.(*SElasticcache).GetRegion()
}

func (self *SElasticcacheBackup) ValidateDeleteCondition(ctx context.Context) error {
	icache, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return err
	}

	if icache.(*SElasticcache).GetProviderName() == api.CLOUD_PROVIDER_ALIYUN && len(self.ExternalId) == 0 {
		return httperrors.NewUnsupportOperationError("unsupport delete %s backups", api.CLOUD_PROVIDER_ALIYUN)
	}

	return self.ValidatePurgeCondition(ctx)
}

func (self *SElasticcacheBackup) ValidatePurgeCondition(ctx context.Context) error {
	return nil
}

// 弹性缓存备份列表
func (manager *SElasticcacheBackupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheBackupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SElasticcacheResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemFilter")
	}

	if len(input.BackupType) > 0 {
		q = q.In("backup_type", input.BackupType)
	}
	if len(input.BackupMode) > 0 {
		q = q.In("backup_mode", input.BackupMode)
	}

	return q, nil
}

func (manager *SElasticcacheBackupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheBackupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SElasticcacheResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SElasticcacheBackupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SElasticcacheResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SElasticcacheBackup) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ElasticcacheBackupDetails, error) {
	return api.ElasticcacheBackupDetails{}, nil
}

func (manager *SElasticcacheBackupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheBackupDetails {
	rows := make([]api.ElasticcacheBackupDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	cacheRows := manager.SElasticcacheResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	cacheIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ElasticcacheBackupDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ElasticcacheResourceInfo:        cacheRows[i],
		}
		backup := objs[i].(*SElasticcacheBackup)
		cacheIds[i] = backup.ElasticcacheId
	}

	caches := make(map[string]SElasticcache)
	err := db.FetchStandaloneObjectsByIds(ElasticcacheManager, cacheIds, &caches)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if cache, ok := caches[cacheIds[i]]; ok {
			virObjs[i] = &cache
			rows[i].ProjectId = cache.ProjectId
		}
	}

	projRows := ElasticcacheManager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, fields, isList)
	for i := range rows {
		rows[i].ProjectizedResourceInfo = projRows[i]
	}

	return rows
}

func (manager *SElasticcacheBackupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SElasticcacheResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SElasticcacheResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
