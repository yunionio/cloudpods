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

package huawei

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	HuaweiTags
	region *SRegion

	BeginTime  string
	Datastore  SDatastore
	EndTime    string
	Id         string
	InstanceId string
	Name       string
	Size       int
	Status     string
	Type       string
}

func (backup *SDBInstanceBackup) GetId() string {
	return backup.Id
}

func (backup *SDBInstanceBackup) GetGlobalId() string {
	return backup.Id
}

func (backup *SDBInstanceBackup) GetName() string {
	return backup.Name
}

func (backup *SDBInstanceBackup) GetEngine() string {
	return backup.Datastore.Type
}

func (backup *SDBInstanceBackup) GetEngineVersion() string {
	return backup.Datastore.Version
}

func (backup *SDBInstanceBackup) GetStartTime() time.Time {
	//2019-08-05T08:00:02+0000
	t, err := time.Parse("2006-01-02T15:04:05Z0700", backup.BeginTime)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (backup *SDBInstanceBackup) GetEndTime() time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z0700", backup.EndTime)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (backup *SDBInstanceBackup) GetBackupMode() string {
	switch backup.Type {
	case "manual":
		return api.BACKUP_MODE_MANUAL
	default:
		return api.BACKUP_MODE_AUTOMATED
	}
}

func (backup *SDBInstanceBackup) GetStatus() string {
	switch backup.Status {
	case "COMPLETED":
		return api.DBINSTANCE_BACKUP_READY
	case "FAILED":
		return api.DBINSTANCE_BACKUP_FAILED
	case "BUILDING":
		return api.DBINSTANCE_BACKUP_CREATING
	case "DELETING":
		return api.DBINSTANCE_BACKUP_DELETING
	default:
		return api.DBINSTANCE_BACKUP_UNKNOWN
	}
}

func (backup *SDBInstanceBackup) GetBackupSizeMb() int {
	return backup.Size / 1024
}

func (backup *SDBInstanceBackup) GetDBNames() string {
	return ""
}

func (backup *SDBInstanceBackup) Delete() error {
	return backup.region.DeleteDBInstanceBackup(backup.Id)
}

func (region *SRegion) DeleteDBInstanceBackup(backupId string) error {
	_, err := region.delete(SERVICE_RDS, "backups/"+backupId)
	return err
}

func (backup *SDBInstanceBackup) GetDBInstanceId() string {
	return backup.InstanceId
}

func (region *SRegion) GetDBInstanceBackups(instanceId, backupId string) ([]SDBInstanceBackup, error) {
	params := url.Values{}
	params.Set("instance_id", instanceId)
	if len(backupId) > 0 {
		params.Set("backup_id", backupId)
	}
	backups := []SDBInstanceBackup{}
	for {
		resp, err := region.list(SERVICE_RDS, "backups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Backups    []SDBInstanceBackup
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		backups = append(backups, part.Backups...)
		if len(backups) >= part.TotalCount || len(part.Backups) == 0 {
			break
		}
		params.Set("offset", fmt.Sprintf("%d", len(backups)))
	}
	return backups, nil
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

func (region *SRegion) GetIDBInstanceBackupById(backupId string) (cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := region.GetIDBInstanceBackups()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIDBInstanceBackups")
	}
	for _, backup := range backups {
		if backup.GetGlobalId() == backupId {
			return backup, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := rds.region.GetDBInstanceBackups(rds.Id, "")
	if err != nil {
		return nil, err
	}

	ibackups := []cloudprovider.ICloudDBInstanceBackup{}
	for i := 0; i < len(backups); i++ {
		backups[i].region = rds.region
		ibackups = append(ibackups, &backups[i])
	}
	return ibackups, nil
}

func (backup *SDBInstanceBackup) Refresh() error {
	backups, err := backup.region.GetDBInstanceBackups(backup.InstanceId, backup.Id)
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		return cloudprovider.ErrNotFound
	}
	return jsonutils.Update(backup, backups[0])
}

func (rds *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	backup, err := rds.region.CreateDBInstanceBackup(rds.Id, conf.Name, conf.Description, conf.Databases)
	if err != nil {
		return "", err
	}
	cloudprovider.WaitStatus(backup, api.DBINSTANCE_BACKUP_READY, time.Second*3, time.Minute*30)
	return backup.GetGlobalId(), nil
}

func (region *SRegion) CreateDBInstanceBackup(instanceId string, name string, descrition string, databases []string) (*SDBInstanceBackup, error) {
	params := map[string]interface{}{
		"instance_id": instanceId,
		"name":        name,
		"description": descrition,
	}
	if len(databases) > 0 {
		dbs := []map[string]string{}
		for _, database := range databases {
			dbs = append(dbs, map[string]string{"name": database})
		}
		params["databases"] = dbs
	}
	resp, err := region.post(SERVICE_RDS, "backups", params)
	if err != nil {
		return nil, errors.Wrap(err, "DBInstanceBackup.Create")
	}
	ret := &SDBInstanceBackup{region: region}
	err = resp.Unmarshal(ret, "backup")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SDBInstanceBackup) CreateICloudDBInstance(opts *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	opts.BackupId = self.Id
	return self.region.CreateIDBInstance(opts)
}
