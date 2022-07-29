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

package jdcloud

import (
	"fmt"
	"time"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/models"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	multicloud.JdcloudTags

	rds *SDBInstance
	models.Backup
}

func (self *SDBInstanceBackup) GetGlobalId() string {
	return self.BackupId
}

func (self *SDBInstanceBackup) GetId() string {
	return self.BackupId
}

func (self *SDBInstanceBackup) GetName() string {
	return self.BackupName
}

func (self *SDBInstanceBackup) GetBackupMethod() cloudprovider.TBackupMethod {
	return cloudprovider.TBackupMethod(self.BackupMethod)
}

func (self *SDBInstanceBackup) GetBackupMode() string {
	switch self.BackupMode {
	case "auto":
		return api.BACKUP_MODE_AUTOMATED
	default:
		return self.BackupMode
	}
}

func (self *SDBInstanceBackup) GetBackupSizeMb() int {
	return int(self.BackupSizeByte / 1024 / 1024)
}

func (self *SDBInstanceBackup) GetDBInstanceId() string {
	return self.rds.GetGlobalId()
}

func (self *SDBInstanceBackup) GetEngine() string {
	return self.rds.GetEngine()
}

func (self *SDBInstanceBackup) GetEngineVersion() string {
	return self.rds.GetEngineVersion()
}

func (self *SDBInstanceBackup) GetEndTime() time.Time {
	return parseTime(self.BackupEndTime)
}

func (self *SDBInstanceBackup) GetStartTime() time.Time {
	return parseTime(self.BackupStartTime)
}

func (self *SDBInstanceBackup) GetDBNames() string {
	return ""
}

func (self *SDBInstanceBackup) GetStatus() string {
	switch self.BackupStatus {
	case "ERROR":
		return api.DBINSTANCE_BACKUP_CREATE_FAILED
	case "COMPLETED":
		return api.DBINSTANCE_BACKUP_READY
	case "BUILDING":
		return api.DBINSTANCE_BACKUP_CREATING
	case "DELETING":
		return api.DBINSTANCE_BACKUP_DELETING
	default:
		return api.DBINSTANCE_BACKUP_READY
	}
}

func (self *SRegion) GetDBInstanceBackups(id string, pageNumber, pageSize int) ([]SDBInstanceBackup, int, error) {
	req := apis.NewDescribeBackupsRequest(self.ID, id, pageNumber, pageSize)
	client := client.NewRdsClient(self.getCredential())
	client.Logger = Logger{debug: self.client.debug}
	resp, err := client.DescribeBackups(req)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeBackups")
	}
	if resp.Error.Code >= 400 {
		err = fmt.Errorf(resp.Error.Message)
		return nil, 0, err
	}
	total := resp.Result.TotalCount
	ret := []SDBInstanceBackup{}
	for i := range resp.Result.Backup {
		ret = append(ret, SDBInstanceBackup{
			Backup: resp.Result.Backup[i],
		})
	}
	return ret, total, nil
}

func (self *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	n := 1
	for {
		part, total, err := self.region.GetDBInstanceBackups(self.InstanceId, n, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDBInstanceBackups")
		}
		backups = append(backups, part...)
		if len(backups) >= total {
			break
		}
		n++
	}
	ret := []cloudprovider.ICloudDBInstanceBackup{}
	for i := range backups {
		backups[i].rds = self
		ret = append(ret, &backups[i])
	}
	return ret, nil
}
