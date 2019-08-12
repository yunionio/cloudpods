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
	"time"

	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheBackup struct {
	multicloud.SElasticcacheBackupBase

	cacheDB *SElasticcache

	BackupIntranetDownloadURL string    `json:"BackupIntranetDownloadURL"`
	BackupType                string    `json:"BackupType"`
	BackupEndTime             time.Time `json:"BackupEndTime"`
	BackupMethod              string    `json:"BackupMethod"`
	BackupID                  int64     `json:"BackupId"`
	BackupStartTime           time.Time `json:"BackupStartTime"`
	BackupDownloadURL         string    `json:"BackupDownloadURL"`
	BackupDBNames             string    `json:"BackupDBNames"`
	NodeInstanceID            string    `json:"NodeInstanceId"`
	BackupMode                string    `json:"BackupMode"`
	BackupStatus              string    `json:"BackupStatus"`
	BackupSizeByte            int64     `json:"BackupSize"`
	EngineVersion             string    `json:"EngineVersion"`
}

func (self *SElasticcacheBackup) GetId() string {
	return fmt.Sprintf("%d", self.BackupID)
}

func (self *SElasticcacheBackup) GetName() string {
	return self.GetId()
}

func (self *SElasticcacheBackup) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheBackup) GetStatus() string {
	return ""
}

func (self *SElasticcacheBackup) GetBackupSizeMb() int {
	return int(self.BackupSizeByte / 1024 / 1024)
}

func (self *SElasticcacheBackup) GetBackupType() string {
	return self.BackupType
}

func (self *SElasticcacheBackup) GetBackupMode() string {
	return self.BackupMode
}

func (self *SElasticcacheBackup) GetDownloadURL() string {
	return self.BackupDownloadURL
}

func (self *SElasticcacheBackup) GetStartTime() time.Time {
	return self.BackupStartTime
}

func (self *SElasticcacheBackup) GetEndTime() time.Time {
	return self.BackupEndTime
}
