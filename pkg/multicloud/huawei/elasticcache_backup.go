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

	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423035.html
type SElasticcacheBackup struct {
	multicloud.SElasticcacheBackupBase

	cacheDB *SElasticcache

	Status           string    `json:"status"`
	Remark           string    `json:"remark"`
	Period           string    `json:"period"`
	Progress         string    `json:"progress"`
	SizeByte         int64     `json:"size"`
	InstanceID       string    `json:"instance_id"`
	BackupID         string    `json:"backup_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ExecutionAt      time.Time `json:"execution_at"`
	BackupType       string    `json:"backup_type"`
	BackupName       string    `json:"backup_name"`
	ErrorCode        string    `json:"error_code"`
	IsSupportRestore string    `json:"is_support_restore"`
}

func (self *SElasticcacheBackup) GetId() string {
	return self.BackupID
}

func (self *SElasticcacheBackup) GetName() string {
	return self.BackupName
}

func (self *SElasticcacheBackup) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheBackup) GetStatus() string {
	return self.Status
}

func (self *SElasticcacheBackup) GetBackupSizeMb() int {
	return int(self.SizeByte / 1024 / 1024)
}

func (self *SElasticcacheBackup) GetBackupType() string {
	return self.BackupType
}

func (self *SElasticcacheBackup) GetBackupMode() string {
	return ""
}

func (self *SElasticcacheBackup) GetDownloadURL() string {
	return ""
}

func (self *SElasticcacheBackup) GetStartTime() time.Time {
	return self.CreatedAt
}

func (self *SElasticcacheBackup) GetEndTime() time.Time {
	return self.UpdatedAt
}
