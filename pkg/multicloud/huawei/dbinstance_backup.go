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
	"time"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
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

func (backup *SDBInstanceBackup) GetDBInstanceId() string {
	return backup.InstanceId
}

func (region *SRegion) GetDBInstanceBackups(instanceId string) ([]SDBInstanceBackup, error) {
	params := map[string]string{
		"instance_id": instanceId,
	}
	backups := []SDBInstanceBackup{}
	err := doListAllWithOffset(region.ecsClient.DBInstanceBackup.List, params, &backups)
	if err != nil {
		return nil, err
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

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := rds.region.GetDBInstanceBackups(rds.Id)
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
