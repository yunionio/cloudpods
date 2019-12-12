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
)

// SElasticcache.Backup
type SElasticcacheBackupManager struct {
	db.SStatusStandaloneResourceBaseManager
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

	ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	BackupSizeMb int    `nullable:"false" list:"user" create:"optional"`
	BackupType   string `width:"32" charset:"ascii" nullable:"true" create:"optional" list:"user"` // 全量|增量额
	BackupMode   string `width:"32" charset:"ascii" nullable:"true" create:"optional" list:"user"` //  自动|手动
	DownloadURL  string `width:"512" charset:"ascii" nullable:"true" create:"optional" list:"user"`

	StartTime time.Time `list:"user" create:"optional"`
	EndTime   time.Time `list:"user" create:"optional"`
}

func (manager *SElasticcacheBackupManager) SyncElasticcacheBackups(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheBackups []cloudprovider.ICloudElasticcacheBackup) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))

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

func (self *SElasticcacheBackup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	icache, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err == nil {
		ec := icache.(*SElasticcache)
		provider := ec.GetCloudprovider()
		region := ec.GetRegion()
		zone := ec.GetZone()
		info := MakeCloudProviderInfo(region, zone, provider)
		extra.Update(jsonutils.Marshal(&info))

		info2 := jsonutils.NewDict()
		info2.Set("engine", jsonutils.NewString(ec.Engine))
		info2.Set("engine_version", jsonutils.NewString(ec.EngineVersion))
		extra.Update(info2)
	}

	return extra
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

	err := manager.TableSpec().Insert(&backup)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheBackup.Insert")
	}

	return &backup, nil
}

func (manager *SElasticcacheBackupManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	return jsonutils.GetAnyString(data, []string{"elasticcache_id", "elasticcache"})
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

func (manager *SElasticcacheBackupManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("elasticcache_id", parentId)
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

func (self *SElasticcacheBackup) GetIRegion() (cloudprovider.ICloudRegion, error) {
	_eb, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return nil, err
	}

	eb := _eb.(*SElasticcache)
	provider, err := eb.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovider for elastic cache %s: %s", eb.Name, err)
	}
	region := eb.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find region for elastic cache %s", self.Name)
	}
	return provider.GetIRegionById(region.ExternalId)
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
