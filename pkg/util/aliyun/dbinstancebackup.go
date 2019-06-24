package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase

	BackupDBNames             string
	BackupIntranetDownloadURL string
	BackupDownloadURL         string
	BackupEndTime             time.Time
	BackupId                  string
	BackupLocation            string
	BackupMethod              string
	BackupMode                string
	BackupScale               string
	BackupSize                int
	BackupStartTime           time.Time
	BackupStatus              string
	BackupType                string
	DBInstanceId              string
	HostInstanceID            int
	MetaStatus                string
	StoreStatus               string
}

func (backup *SDBInstanceBackup) GetId() string {
	return backup.BackupId
}

func (backup *SDBInstanceBackup) GetGlobalId() string {
	return backup.BackupId
}

func (backup *SDBInstanceBackup) GetName() string {
	return backup.BackupId
}

func (backup *SDBInstanceBackup) GetStartTime() time.Time {
	return backup.BackupStartTime
}

func (backup *SDBInstanceBackup) GetEndTime() time.Time {
	return backup.BackupEndTime
}

func (backup *SDBInstanceBackup) GetBackupType() string {
	switch backup.BackupType {
	case "FullBackup":
		return api.BACKUP_TYPE_FULL_BACKUP
	case "IncrementalBackup":
		return api.BACKUP_TYPE_INCREMENTALBACKUP
	default:
		log.Errorf("Unknown backup type %s", backup.BackupType)
		return api.BACKUP_TYPE_UNKNOWN
	}
}

func (backup *SDBInstanceBackup) GetStatus() string {
	switch backup.BackupStatus {
	case "Success":
		return api.DBINSTANCE_BACKUP_READY
	case "Failed":
		return api.DBINSTANCE_BACKUP_FAILED
	default:
		return api.DBINSTANCE_BACKUP_UNKNOWN
	}
}

func (backup *SDBInstanceBackup) GetBackupSizeMb() int {
	return backup.BackupSize / 1024 / 1024
}

func (backup *SDBInstanceBackup) GetDBNames() string {
	return backup.BackupDBNames
}

func (backup *SDBInstanceBackup) GetDownloadURL() string {
	return backup.BackupDownloadURL
}

func (backup *SDBInstanceBackup) GetIntranetDownloadURL() string {
	return backup.BackupIntranetDownloadURL
}

func (region *SRegion) GetDBInstanceBackups(instanceId, backupId string, offset int, limit int) ([]SDBInstanceBackup, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := map[string]string{
		"RegionId":     region.RegionId,
		"PageSize":     fmt.Sprintf("%d", limit),
		"PageNumber":   fmt.Sprintf("%d", (offset/limit)+1),
		"DBInstanceId": instanceId,
	}
	if len(backupId) > 0 {
		params["BackupId"] = backupId
	}
	body, err := region.rdsRequest("DescribeBackups", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "DescribeBackups")
	}
	backups := []SDBInstanceBackup{}
	err = body.Unmarshal(&backups, "Items", "Backup")
	if err != nil {
		return nil, 0, errors.Wrap(err, "Unmarshal")
	}
	total, _ := body.Int("TotalRecordCount")
	return backups, int(total), nil
}
