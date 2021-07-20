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
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheBackup struct {
	multicloud.SElasticcacheBackupBase
	multicloud.QcloudTags

	cacheDB *SElasticcache

	StartTime  time.Time `json:"StartTime"`
	BackupID   string    `json:"BackupId"`
	BackupType string    `json:"BackupType"`
	Status     int       `json:"Status"`
	Remark     string    `json:"Remark"`
	Locked     int64     `json:"Locked"`
}

func (self *SElasticcacheBackup) GetId() string {
	return self.BackupID
}

func (self *SElasticcacheBackup) GetName() string {
	if len(self.Remark) > 0 {
		segs := strings.Split(self.Remark, "@")
		return segs[0]
	}

	return self.GetId()
}

func (self *SElasticcacheBackup) GetGlobalId() string {
	return self.GetId()
}

/*
	ELASTIC_CACHE_BACKUP_STATUS_CREATING       = "creating" // 备份中
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED = "expired"  //（备份文件已过期）
	ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED = "deleted"  //（备份文件已删除）
	ELASTIC_CACHE_BACKUP_STATUS_DELETING       = "deleting" // 删除中
	ELASTIC_CACHE_BACKUP_STATUS_SUCCESS        = "success"  // 备份成功
	ELASTIC_CACHE_BACKUP_STATUS_FAILED         = "failed"   // 备份失败
*/
func (self *SElasticcacheBackup) GetStatus() string {
	switch self.Status {
	case 1:
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATING
	case 2:
		return api.ELASTIC_CACHE_BACKUP_STATUS_SUCCESS
	case -1:
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED
	case -2:
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED
	case 3:
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATING
	case 4:
		return api.ELASTIC_CACHE_BACKUP_STATUS_SUCCESS
	default:
		return strconv.Itoa(self.Status)
	}
}

func (self *SElasticcacheBackup) Refresh() error {
	backup, err := self.cacheDB.GetICloudElasticcacheBackup(self.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetICloudElasticcacheBackup")
	}

	err = jsonutils.Update(self, backup)
	if err != nil {
		return errors.Wrap(err, "Update")
	}

	return nil
}

func (self *SElasticcacheBackup) GetBackupSizeMb() int {
	return 0
}

func (self *SElasticcacheBackup) GetBackupType() string {
	return api.ELASTIC_CACHE_BACKUP_TYPE_INCREMENTAL
}

func (self *SElasticcacheBackup) GetBackupMode() string {
	switch self.BackupType {
	case "1", "systemBackupInstance":
		return api.ELASTIC_CACHE_BACKUP_MODE_AUTOMATED
	case "0", "manualBackupInstance":
		return api.ELASTIC_CACHE_BACKUP_MODE_MANUAL
	default:
		return self.BackupType
	}
}

// https://cloud.tencent.com/document/api/239/34443
func (self *SElasticcacheBackup) GetDownloadURL() string {
	url, err := self.GetBackupDownloadURL()
	if err != nil {
		log.Debugf("GetBackupDownloadURL %s", err)
		return ""
	}

	return url
}

func (self *SElasticcacheBackup) GetStartTime() time.Time {
	return self.StartTime
}

func (self *SElasticcacheBackup) GetEndTime() time.Time {
	return time.Time{}
}

func (self *SElasticcacheBackup) Delete() error {
	return errors.Wrap(cloudprovider.ErrNotSupported, "Delete")
}

// https://cloud.tencent.com/document/product/239/34435
// todo: password
func (self *SElasticcacheBackup) RestoreInstance(instanceId string) error {
	params := map[string]string{}
	params["InstanceId"] = instanceId
	params["BackupId"] = self.GetId()
	//if len(Password) > 0 {
	//	params["Password"] = Password
	//}

	_, err := self.cacheDB.region.redisRequest("RestoreInstance", params)
	if err != nil {
		return errors.Wrap(err, "RestoreInstance")
	}

	return nil
}

func (self *SElasticcacheBackup) GetBackupDownloadURL() (string, error) {
	params := map[string]string{}
	params["InstanceId"] = self.cacheDB.GetId()
	params["BackupId"] = self.GetId()
	client := self.cacheDB.region.client
	resp, err := client.redisRequest("DescribeBackupUrl", params)
	if err != nil {
		return "", errors.Wrap(err, "DescribeBackupUrl")
	}

	urls := []string{}
	err = resp.Unmarshal(&urls, "DownloadUrl")
	if err != nil {
		return "", errors.Wrap(err, "Unmarshal.DownloadUrl")
	}

	if len(urls) > 0 {
		return urls[0], nil
	}

	return "", nil
}
