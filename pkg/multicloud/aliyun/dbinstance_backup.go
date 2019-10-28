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

package aliyun

import (
	"fmt"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

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

func (backup *SDBInstanceBackup) GetEngine() string {
	instance, _ := backup.region.GetDBInstanceDetail(backup.DBInstanceId)
	if instance != nil {
		return instance.Engine
	}
	return ""
}

func (backup *SDBInstanceBackup) GetEngineVersion() string {
	instance, _ := backup.region.GetDBInstanceDetail(backup.DBInstanceId)
	if instance != nil {
		return instance.EngineVersion
	}
	return ""
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

func (rds *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	params := map[string]string{
		"DBInstanceId": rds.DBInstanceId,
		"BackupMethod": "Snapshot",
	}
	if len(conf.Databases) > 0 {
		params["BackupStrategy"] = "db"
		params["DBName"] = strings.Join(conf.Databases, ",")
		params["BackupMethod"] = "Logical"
	}
	body, err := rds.region.rdsRequest("CreateBackup", params)
	if err != nil {
		return "", errors.Wrap(err, "CreateBackup")
	}
	jobId, err := body.GetString("BackupJobId")
	if err != nil {
		return "", errors.Wrap(err, "body.BackupJobId")
	}
	return "", rds.region.waitBackupCreateComplete(rds.DBInstanceId, jobId)
}

func (backup *SDBInstanceBackup) Delete() error {
	return backup.region.DeleteDBInstanceBackup(backup.DBInstanceId, backup.BackupId)
}

func (region *SRegion) DeleteDBInstanceBackup(instanceId string, backupId string) error {
	params := map[string]string{
		"DBInstanceId": instanceId,
		"BackupId":     backupId,
	}
	_, err := region.rdsRequest("DeleteBackup", params)
	return err
}

type SDBInstanceBackupJob struct {
	BackupProgressStatus string
	Process              string
	JobMode              string
	TaskAction           string
	BackupStatus         string
	BackupJobId          string
}

type SDBInstanceBackupJobs struct {
	BackupJob []SDBInstanceBackupJob
}

func (region *SRegion) GetDBInstanceBackupJobs(instanceId, jobId string) (*SDBInstanceBackupJobs, error) {
	params := map[string]string{
		"DBInstanceId": instanceId,
		"ClientToken":  utils.GenRequestId(20),
		"BackupMode":   "Manual",
	}
	if len(jobId) > 0 {
		params["BackupJobId"] = jobId
	}
	body, err := region.rdsRequest("DescribeBackupTasks", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeBackupTasks")
	}

	jobs := SDBInstanceBackupJobs{}

	err = body.Unmarshal(&jobs, "Items")
	if err != nil {
		return nil, errors.Wrapf(err, "body.Unmarshal(%s)", body)
	}

	return &jobs, nil
}

func (region *SRegion) waitBackupCreateComplete(instanceId, jobId string) error {
	for i := 0; i < 20*40; i++ {
		jobs, err := region.GetDBInstanceBackupJobs(instanceId, jobId)
		if err != nil {
			return errors.Wrapf(err, "region.GetDBInstanceBackupJobs(%s, %s)", instanceId, jobId)
		}
		if len(jobs.BackupJob) == 0 {
			return nil
		}
		for _, job := range jobs.BackupJob {
			log.Infof("instance %s backup job %s status: %s(%s)", instanceId, jobId, job.BackupStatus, job.Process)
			if job.BackupStatus == "Finished" && job.BackupJobId == jobId {
				return nil
			}
		}
		time.Sleep(time.Second * 3)
	}
	return fmt.Errorf("timeout for waiting create job complete")
}
