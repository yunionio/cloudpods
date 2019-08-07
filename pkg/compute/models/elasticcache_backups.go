package models

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	BackupSizeMb int    `nullable:"false" list:"user"`
	BackupType   string `width:"32" charset:"ascii" nullable:"true" list:"user"` // 全量|增量额
	BackupMode   string `width:"32" charset:"ascii" nullable:"true" list:"user"` //  自动|手动
	DownloadURL  string `width:"512" charset:"ascii" nullable:"true" list:"user"`

	StartTime time.Time `list:"user"`
	EndTime   time.Time `list:"user"`
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
		self.DownloadURL = extBackup.GetDownloadURL()

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
