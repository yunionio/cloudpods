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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SMongoDBBackup struct {
	BackupDBNames             string
	BackupDownloadURL         string
	BackupEndTime             time.Time
	BackupStartTime           time.Time
	BackupId                  string
	BackupIntranetDownloadURL string
	BackupMethod              string
	BackupMode                string
	BackupSize                int
	BackupStatus              string
	BackupType                string
}

func (self *SRegion) GetMongoDBBackups(id string, nodeId string, start time.Time, end time.Time, pageSize, pageNum int) ([]SMongoDBBackup, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	if pageNum < 1 {
		pageNum = 1
	}
	params := map[string]string{
		"StartTime":    start.Format("2006-01-02T15:04Z"),
		"EndTime":      end.Format("2006-01-02T15:04Z"),
		"DBInstanceId": id,
		"PageSize":     fmt.Sprintf("%d", pageSize),
		"PageNumber":   fmt.Sprintf("%d", pageNum),
	}
	if len(nodeId) > 0 {
		params["NodeId"] = nodeId
	}
	resp, err := self.mongodbRequest("DescribeBackups", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeBackups")
	}
	ret := []SMongoDBBackup{}
	err = resp.Unmarshal(&ret, "Backups", "Backup")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Int("TotalCount")
	return ret, int(totalCount), nil
}

func (self *SMongoDB) GetIBackups() ([]cloudprovider.SMongoDBBackup, error) {
	backups := []SMongoDBBackup{}
	err := self.Refresh()
	if err != nil {
		return nil, errors.Wrapf(err, "Refresh")
	}
	ret := []cloudprovider.SMongoDBBackup{}
	if self.DBInstanceType == "sharding" {
		backups, err := self.region.DescribeClusterBackups(self.DBInstanceId)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeClusterBackups")
		}
		for _, res := range backups {
			backup := cloudprovider.SMongoDBBackup{}
			backup.Name = res.ClusterBackupId
			backup.StartTime = res.ClusterBackupStartTime
			backup.EndTime = res.ClusterBackupEndTime
			backup.BackupSizeKb = int(res.ClusterBackupSize / 1024)
			switch res.ClusterBackupStatus {
			case "OK":
				backup.Status = cloudprovider.MongoDBBackupStatusAvailable
			case "Failed":
				backup.Status = cloudprovider.MongoDBBackupStatusFailed
			default:
				backup.Status = cloudprovider.TMongoDBBackupStatus(strings.ToLower(res.ClusterBackupStatus))
			}
			backup.BackupMethod = cloudprovider.TMongoDBBackupMethod(strings.ToLower(res.ClusterBackupMode))
			backup.BackupType = cloudprovider.MongoDBBackupTypeAuto
			if res.ClusterBackupMode == "Manual" {
				backup.BackupType = cloudprovider.MongoDBBackupTypeManual
			}
			ret = append(ret, backup)
		}
		return ret, nil
	}
	now := time.Now().Add(time.Minute * -1)
	for {
		part, total, err := self.region.GetMongoDBBackups(self.DBInstanceId, "", self.CreationTime, now, 100, len(backups)/100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetMongoDBBackups")
		}
		backups = append(backups, part...)
		if len(backups) >= total {
			break
		}
	}

	for _, res := range backups {
		backup := cloudprovider.SMongoDBBackup{}
		backup.Name = res.BackupId
		backup.StartTime = res.BackupStartTime
		backup.EndTime = res.BackupEndTime
		backup.BackupSizeKb = res.BackupSize / 1024
		switch res.BackupStatus {
		case "Success":
			backup.Status = cloudprovider.MongoDBBackupStatusAvailable
		case "Failed":
			backup.Status = cloudprovider.MongoDBBackupStatusFailed
		default:
			backup.Status = cloudprovider.TMongoDBBackupStatus(strings.ToLower(res.BackupStatus))
		}
		backup.BackupMethod = cloudprovider.TMongoDBBackupMethod(strings.ToLower(res.BackupMethod))
		backup.BackupType = cloudprovider.MongoDBBackupTypeAuto
		if res.BackupMode == "Manual" {
			backup.BackupType = cloudprovider.MongoDBBackupTypeManual
		}
		ret = append(ret, backup)
	}
	return ret, nil
}

type SMongoDBClusterBackup struct {
	AttachLogStatus  string
	BackupExpireTime time.Time
	Backups          []struct {
		BackupEndTime   time.Time
		BackupId        string
		BackupName      string
		BackupSize      string
		BackupStartTime time.Time
		BackupStatus    string
		ExtraInfo       struct {
			InstanceClass string
			NodeId        string
			NodeType      string
			StorageSize   string
		}
		InstanceName string
		IsAvail      int
	}
	ClusterBackupEndTime   time.Time
	ClusterBackupId        string
	ClusterBackupMode      string
	ClusterBackupSize      int64
	ClusterBackupStartTime time.Time
	ClusterBackupStatus    string
	EngineVersion          string
	ExtraInfo              struct{}
	IsAvail                int
}

func (region *SRegion) DescribeClusterBackups(id string) ([]SMongoDBClusterBackup, error) {
	backups := []SMongoDBClusterBackup{}
	pageNum := 1
	params := map[string]string{
		"DBInstanceId": id,
		"PageSize":     "50",
		"PageNumber":   fmt.Sprintf("%d", pageNum),
	}
	for {
		resp, err := region.mongodbRequest("DescribeClusterBackups", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeClusterBackups")
		}
		part := struct {
			MaxResults     int
			ClusterBackups []SMongoDBClusterBackup
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		backups = append(backups, part.ClusterBackups...)
		if len(backups) >= part.MaxResults {
			break
		}
		pageNum++
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
	}
	return backups, nil
}
