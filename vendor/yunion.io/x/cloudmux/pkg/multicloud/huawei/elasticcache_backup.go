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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423035.html
type SElasticcacheBackup struct {
	multicloud.SElasticcacheBackupBase
	HuaweiTags

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

func (self *SElasticcacheBackup) Refresh() error {
	cache, err := self.cacheDB.GetICloudElasticcacheBackup(self.GetId())
	if err != nil {
		return errors.Wrap(err, "ElasticcacheBackup.Refresh.GetICloudElasticcacheBackup")
	}

	err = jsonutils.Update(self, cache)
	if err != nil {
		return errors.Wrap(err, "ElasticcacheBackup.Refresh.Update")
	}

	return nil
}

func (self *SElasticcacheBackup) GetStatus() string {
	switch self.Status {
	case "waiting", "backuping":
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATING
	case "succeed":
		return api.ELASTIC_CACHE_BACKUP_STATUS_SUCCESS
	case "failed":
		return api.ELASTIC_CACHE_BACKUP_STATUS_FAILED
	case "expired":
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATE_EXPIRED
	case "deleted":
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATE_DELETED
	default:
		return self.Status
	}
}

func (self *SElasticcacheBackup) GetBackupSizeMb() int {
	return int(self.SizeByte / 1024 / 1024)
}

func (self *SElasticcacheBackup) GetBackupType() string {
	switch self.BackupType {
	case "manual":
		return api.ELASTIC_CACHE_BACKUP_MODE_MANUAL
	case "auto":
		return api.ELASTIC_CACHE_BACKUP_MODE_AUTOMATED
	default:
		return self.BackupType
	}

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

func (self *SElasticcacheBackup) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticcacheBackup) RestoreInstance(instanceId string) error {
	params := map[string]interface{}{
		"backup_id": self.BackupID,
	}
	_, err := self.cacheDB.region.post(SERVICE_DCS, fmt.Sprintf("instances/%s/restores", self.cacheDB.InstanceID), params)
	return err
}
