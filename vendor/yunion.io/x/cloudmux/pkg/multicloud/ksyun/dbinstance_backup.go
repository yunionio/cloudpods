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

package ksyun

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	SKsyunTags
	region *SRegion

	DBInstanceIdentifier string
	DBBackupIdentifier   string
	Engine               string
	EngineVersion        string
	BackupCreateTime     string
	BackupUpdatedTime    string
	DBBackupName         string
	BackupMode           string
	BackupType           string
	Status               string
	BackupSize           float64
	BackupLocationRef    string
	RemotePath           string
	MD5                  string
}

func (backup *SDBInstanceBackup) GetId() string {
	return backup.DBBackupIdentifier
}

func (backup *SDBInstanceBackup) GetGlobalId() string {
	return backup.DBBackupIdentifier
}

func (backup *SDBInstanceBackup) GetName() string {
	return backup.DBBackupName
}

func (backup *SDBInstanceBackup) GetStartTime() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05-0700", backup.BackupCreateTime)
	return t
}

func (backup *SDBInstanceBackup) GetEndTime() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05-0700", backup.BackupUpdatedTime)
	return t
}

func (backup *SDBInstanceBackup) GetBackupMode() string {
	switch backup.BackupType {
	case "AutoBackup":
		return api.BACKUP_MODE_AUTOMATED
	default:
		return api.BACKUP_MODE_MANUAL
	}
}

func (backup *SDBInstanceBackup) GetStatus() string {
	switch backup.Status {
	case "COMPLETED":
		return api.DBINSTANCE_BACKUP_READY
	case "Failed":
		return api.DBINSTANCE_BACKUP_FAILED
	default:
		return api.DBINSTANCE_BACKUP_UNKNOWN
	}
}

func (backup *SDBInstanceBackup) GetBackupSizeMb() int {
	return int(backup.BackupSize * 1024)
}

func (backup *SDBInstanceBackup) GetDBNames() string {
	return ""
}

func (backup *SDBInstanceBackup) GetEngine() string {
	return backup.Engine
}

func (backup *SDBInstanceBackup) GetEngineVersion() string {
	return backup.EngineVersion
}

func (backup *SDBInstanceBackup) GetDBInstanceId() string {
	return backup.DBInstanceIdentifier
}

func (region *SRegion) GetDBInstanceBackups(id string) ([]SDBInstanceBackup, error) {
	params := map[string]interface{}{
		"DBInstanceIdentifier": id,
	}
	ret := []SDBInstanceBackup{}
	for {
		resp, err := region.rdsRequest("DescribeDBBackups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			DBBackup   []SDBInstanceBackup
			TotalCount int
		}{}
		err = resp.Unmarshal(&part, "Data")
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.DBBackup...)
		if len(ret) >= part.TotalCount {
			break
		}
		params["Marker"] = fmt.Sprintf("%d", len(ret))
	}
	return ret, nil
}

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := rds.region.GetDBInstanceBackups(rds.DBInstanceIdentifier)
	if err != nil {
		return nil, err
	}

	ret := []cloudprovider.ICloudDBInstanceBackup{}
	for i := 0; i < len(backups); i++ {
		backups[i].region = rds.region
		ret = append(ret, &backups[i])
	}
	return ret, nil
}

func (self *SDBInstanceBackup) GetBackupMethod() cloudprovider.TBackupMethod {
	switch self.BackupMode {
	case "FULL_AMOUNT_BACKUP":
		return cloudprovider.BackupMethodPhysical
	default:
		return cloudprovider.BackupMethodLogical
	}
}
