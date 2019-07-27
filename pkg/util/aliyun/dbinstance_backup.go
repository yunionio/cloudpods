package aliyun

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	region *SRegion

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

func (backup *SDBInstanceBackup) GetBackupMode() string {
	switch backup.BackupMode {
	case "Manual":
		return api.BACKUP_MODE_MANUAL
	default:
		return api.BACKUP_MODE_AUTOMATED
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

func (backup *SDBInstanceBackup) GetDBInstanceId() string {
	return backup.DBInstanceId
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

func (region *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	dbinstnaces, err := region.GetIDBInstances()
	if err != nil {
		return nil, err
	}
	ibackups := []cloudprovider.ICloudDBInstanceBackup{}
	for i := 0; i < len(dbinstnaces); i++ {
		_dbinstance := dbinstnaces[i].(*SDBInstance)
		_ibackup, err := _dbinstance.GetIDBInstanceBackups()
		if err != nil {
			return nil, errors.Wrapf(err, "_dbinstance(%v).GetIDBInstanceBackups", _dbinstance)
		}
		ibackups = append(ibackups, _ibackup...)
	}
	return ibackups, nil
}

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	for {
		parts, total, err := rds.region.GetDBInstanceBackups(rds.DBInstanceId, "", len(backups), 50)
		if err != nil {
			return nil, err
		}
		backups = append(backups, parts...)
		if len(backups) >= total {
			break
		}
	}

	ibackups := []cloudprovider.ICloudDBInstanceBackup{}
	for i := 0; i < len(backups); i++ {
		backups[i].region = rds.region
		ibackups = append(ibackups, &backups[i])
	}
	return ibackups, nil
}
