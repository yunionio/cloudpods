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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	AliyunTags
	region *SRegion

	Engine                    string
	EngineVersion             string
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
	return backup.Engine
}

func (backup *SDBInstanceBackup) GetEngineVersion() string {
	return backup.EngineVersion
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
		backups[i].Engine = rds.Engine
		backups[i].EngineVersion = rds.EngineVersion
		ibackups = append(ibackups, &backups[i])
	}
	return ibackups, nil
}

func (self *SRegion) CreateDBInstanceBackup(rdsId string, databases []string) (string, error) {
	rds, err := self.GetDBInstanceDetail(rdsId)
	if err != nil {
		return "", errors.Wrapf(err, "GetDBInstanceDetail")
	}
	params := map[string]string{
		"DBInstanceId": rdsId,
	}
	switch rds.Engine {
	case api.DBINSTANCE_TYPE_MYSQL:
		if utils.IsInStringArray(rds.EngineVersion, []string{"5.7", "8.0"}) && ((utils.IsInStringArray(rds.GetStorageType(), []string{
			api.ALIYUN_DBINSTANCE_STORAGE_TYPE_CLOUD_ESSD,
			api.ALIYUN_DBINSTANCE_STORAGE_TYPE_CLOUD_SSD,
		}) && rds.GetCategory() == api.ALIYUN_DBINSTANCE_CATEGORY_HA) ||
			(rds.GetStorageType() == api.ALIYUN_DBINSTANCE_STORAGE_TYPE_CLOUD_SSD &&
				rds.GetCategory() == api.ALIYUN_DBINSTANCE_CATEGORY_BASIC)) {
			params["BackupMethod"] = "Snapshot"
		} else {
			params["BackupMethod"] = "Physical"
			if len(databases) > 0 {
				params["BackupStrategy"] = "db"
				params["DBName"] = strings.Join(databases, ",")
				params["BackupMethod"] = "Logical"
			}
		}
	case api.DBINSTANCE_TYPE_MARIADB:
		params["BackupMethod"] = "Snapshot"
	case api.DBINSTANCE_TYPE_SQLSERVER:
		params["BackupMethod"] = "Physical"
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		if rds.GetStorageType() == api.ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD {
			params["BackupMethod"] = "Physical"
		} else {
			params["BackupMethod"] = "Snapshot"
		}
	case api.DBINSTANCE_TYPE_PPAS:
		params["BackupMethod"] = "Physical"
	}
	body, err := self.rdsRequest("CreateBackup", params)
	if err != nil {
		return "", errors.Wrap(err, "CreateBackup")
	}
	jobId, err := body.GetString("BackupJobId")
	if err != nil {
		return "", errors.Wrap(err, "body.BackupJobId")
	}
	return self.waitBackupCreateComplete(rds.DBInstanceId, jobId)
}

func (rds *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	return rds.region.CreateDBInstanceBackup(rds.DBInstanceId, conf.Databases)
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
	BackupId             string
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

func (region *SRegion) waitBackupCreateComplete(instanceId, jobId string) (string, error) {
	err := cloudprovider.Wait(time.Second*10, time.Minute*40, func() (bool, error) {
		jobs, err := region.GetDBInstanceBackupJobs(instanceId, jobId)
		if err != nil {
			return false, errors.Wrapf(err, "region.GetDBInstanceBackupJobs(%s, %s)", instanceId, jobId)
		}
		if len(jobs.BackupJob) == 0 {
			return true, nil
		}
		for _, job := range jobs.BackupJob {
			log.Infof("instance %s backup job %s status: %s(%s)", instanceId, jobId, job.BackupStatus, job.Process)
			if job.BackupStatus == "Finished" && job.BackupJobId == jobId {
				return true, nil
			}
			if job.BackupStatus == "Failed" && job.BackupJobId == jobId {
				return false, fmt.Errorf("instance %s backup job %s failed", instanceId, jobId)
			}
		}
		return false, nil
	})
	if err != nil {
		return "", errors.Wrapf(err, "wait backup create job")
	}
	jobs, err := region.GetDBInstanceBackupJobs(instanceId, jobId)
	if err != nil {
		return "", errors.Wrapf(err, "region.GetDBInstanceBackupJobs(%s, %s)", instanceId, jobId)
	}
	for _, job := range jobs.BackupJob {
		if job.BackupStatus == "Finished" && job.BackupJobId == jobId {
			if len(job.BackupId) == 0 {
				return "", fmt.Errorf("Missing backup id")
			}
			return job.BackupId, nil
		}
	}
	return "", fmt.Errorf("failed to found backup job %s backupid", jobId)
}

func (self *SDBInstanceBackup) GetBackupMethod() cloudprovider.TBackupMethod {
	return cloudprovider.TBackupMethod(self.BackupMethod)
}

func (self *SDBInstanceBackup) CreateICloudDBInstance(opts *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	rdsId, err := self.region.CreateDBInstanceByBackup(self.BackupId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDBInstanceByBackup")
	}
	return self.region.GetDBInstanceDetail(rdsId)
}

func (self *SRegion) CreateDBInstanceByBackup(backupId string, opts *cloudprovider.SManagedDBInstanceCreateConfig) (string, error) {
	params := map[string]string{
		"DBInstanceId":          opts.RdsId,
		"DBInstanceStorageType": opts.StorageType,
		"PayType":               "Postpaid",
		"BackupId":              backupId,
	}
	resp, err := self.rdsRequest("CloneDBInstance", params)
	if err != nil {
		return "", errors.Wrapf(err, "rdsRequest")
	}
	rdsId, err := resp.GetString("DBInstanceId")
	if err != nil {
		return "", fmt.Errorf("missing DBInstanceId after CloneDBInstance")
	}
	return rdsId, nil
}
