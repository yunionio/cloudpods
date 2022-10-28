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

package qcloud

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMySQLInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	QcloudTags
	rds *SMySQLInstance

	Name        string
	Size        int
	Date        string
	IntranetUrl string
	InternetUrl string
	Type        string
	BackupId    int
	Status      string
	FinishTime  string
	Creator     string
	StartTime   string
	Method      string
	Way         string
}

func (self *SMySQLInstanceBackup) GetId() string {
	return fmt.Sprintf("%d", self.BackupId)
}

func (self *SMySQLInstanceBackup) GetGlobalId() string {
	return self.GetId()
}

func (self *SMySQLInstanceBackup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.GetId()
}

func (self *SMySQLInstanceBackup) GetEngine() string {
	return api.DBINSTANCE_TYPE_MYSQL
}

func (self *SMySQLInstanceBackup) GetStatus() string {
	switch self.Status {
	case "SUCCESS":
		return api.DBINSTANCE_BACKUP_READY
	case "FAILED":
		return api.DBINSTANCE_BACKUP_CREATE_FAILED
	case "RUNNING":
		return api.DBINSTANCE_BACKUP_CREATING
	default:
		return api.DBINSTANCE_BACKUP_UNKNOWN
	}
}

func (self *SMySQLInstanceBackup) GetEngineVersion() string {
	return self.rds.EngineVersion
}

func (self *SMySQLInstanceBackup) GetDBInstanceId() string {
	return self.rds.InstanceId
}

func (self *SMySQLInstanceBackup) GetStartTime() time.Time {
	start, err := timeutils.ParseTimeStr(self.StartTime)
	if err != nil {
		return time.Time{}
	}
	return start.Add(time.Hour * -8)
}

func (self *SMySQLInstanceBackup) GetEndTime() time.Time {
	end, err := timeutils.ParseTimeStr(self.FinishTime)
	if err != nil {
		return time.Time{}
	}
	return end.Add(time.Hour * -8)
}

func (self *SMySQLInstanceBackup) GetBackupSizeMb() int {
	return self.Size / 1024 / 1024
}

func (self *SMySQLInstanceBackup) GetDBNames() string {
	return ""
}

func (self *SMySQLInstanceBackup) GetBackupMode() string {
	if self.Way == "manual" {
		return api.BACKUP_MODE_MANUAL
	}
	return api.BACKUP_MODE_AUTOMATED
}

func (self *SMySQLInstanceBackup) Delete() error {
	return self.rds.region.DeleteBackup(self.rds.InstanceId, fmt.Sprintf("%d", self.BackupId))
}

func (self *SMySQLInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups := []cloudprovider.ICloudDBInstanceBackup{}
	for {
		part, total, err := self.region.DescribeMySQLBackups(self.InstanceId, len(backups), 100)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeMySQLBackups")
		}
		for i := range part {
			part[i].rds = self
			backups = append(backups, &part[i])
		}
		if len(backups) >= total {
			break
		}
	}
	return backups, nil
}

func (self *SRegion) DescribeMySQLBackups(instanceId string, offset, limit int) ([]SMySQLInstanceBackup, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Offset": fmt.Sprintf("%d", offset),
		"Limit":  fmt.Sprintf("%d", limit),
	}
	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	resp, err := self.cdbRequest("DescribeBackups", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeBackups")
	}
	backups := []SMySQLInstanceBackup{}
	err = resp.Unmarshal(&backups, "Items")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return backups, int(totalCount), nil
}

func (self *SRegion) DeleteBackup(instanceId, id string) error {
	params := map[string]string{
		"InstanceId": instanceId,
		"BackupId":   id,
	}
	_, err := self.cdbRequest("DeleteBackup", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteBackup")
	}
	return nil
}

func (self *SRegion) GetMySQLInstanceBackup(instanceId, backupId string) (*SMySQLInstanceBackup, error) {
	backups := []SMySQLInstanceBackup{}
	for {
		part, total, err := self.DescribeMySQLBackups(instanceId, len(backups), 100)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeMySQLBackups")
		}
		for i := range part {
			if fmt.Sprintf("%d", part[i].BackupId) == backupId {
				return &part[i], nil
			}
		}
		backups = append(backups, part...)
		if len(backups) >= total {
			break
		}
	}
	return nil, fmt.Errorf("failed to found rds %s backup %s", instanceId, backupId)
}

func (self *SRegion) waitMySQLBackupReady(instanceId, backupId string) error {
	return cloudprovider.Wait(time.Second*20, time.Minute*30, func() (bool, error) {
		backup, err := self.GetMySQLInstanceBackup(instanceId, backupId)
		if err != nil {
			return false, errors.Wrapf(err, "GetMySQLInstanceBackup")
		}
		log.Infof("backup %s for instance %s status %s", backup.GetName(), instanceId, backup.Status)
		if utils.IsInStringArray(backup.Status, []string{"FAILED", "SUCCESS"}) {
			return true, nil
		}
		return false, nil
	})
}

func (self *SRegion) CreateMySQLBackup(instanceId string, tables map[string]string) (string, error) {
	params := map[string]string{
		"InstanceId":   instanceId,
		"BackupMethod": "physical",
	}
	if len(tables) > 0 {
		params["BackupMethod"] = "logical"
		idx := 0
		for db, table := range tables {
			params[fmt.Sprintf("BackupDBTableList.%d.Db", idx)] = db
			if len(table) > 0 {
				params[fmt.Sprintf("BackupDBTableList.%d.Table", idx)] = table
			}
			idx++
		}
	}
	resp, err := self.cdbRequest("CreateBackup", params)
	if err != nil {
		return "", errors.Wrapf(err, "CreateBackup")
	}
	_backupId, _ := resp.Float("BackupId")
	backupId := fmt.Sprintf("%d", int(_backupId))
	err = self.waitMySQLBackupReady(instanceId, backupId)
	if err != nil {
		return "", errors.Wrapf(err, "waitBackupReady")
	}
	return backupId, nil
}
