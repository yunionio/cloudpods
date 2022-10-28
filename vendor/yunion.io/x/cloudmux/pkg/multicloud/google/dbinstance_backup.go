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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type OperationError struct {
	Kind    string
	Code    string
	Message string
}

type SDBInstanceBackup struct {
	multicloud.SDBInstanceBackupBase
	GoogleTags
	rds *SDBInstance

	Kind            string
	Status          string
	EnqueuedTime    string
	Id              string
	StartTime       time.Time
	EndTime         time.Time
	Type            string
	Description     string
	WindowStartTime time.Time
	Instance        string
	SelfLink        string
	Location        string
	Error           OperationError
}

func (region *SRegion) GetDBInstanceBackups(instanceId string) ([]SDBInstanceBackup, error) {
	backups := []SDBInstanceBackup{}
	params := map[string]string{}
	resource := fmt.Sprintf("instances/%s/backupRuns", instanceId)
	err := region.RdsListAll(resource, params, &backups)
	if err != nil {
		return nil, errors.Wrap(err, "RdsListAll")
	}
	return backups, nil
}

func (region *SRegion) GetDBInstanceBackup(backupId string) (*SDBInstanceBackup, error) {
	backup := SDBInstanceBackup{}
	err := region.rdsGet(backupId, &backup)
	if err != nil {
		return nil, errors.Wrap(err, "RdsGet")
	}
	rds, err := region.GetDBInstance(strings.TrimSuffix(backup.SelfLink, fmt.Sprintf("/%s", backup.Id)))
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstance")
	}
	backup.rds = rds
	return &backup, nil
}

func (backup *SDBInstanceBackup) GetName() string {
	return backup.Id
}

func (backup *SDBInstanceBackup) GetId() string {
	return backup.SelfLink
}

func (backup *SDBInstanceBackup) GetGlobalId() string {
	return strings.TrimPrefix(backup.SelfLink, fmt.Sprintf("%s/%s/", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION))
}

func (backup *SDBInstanceBackup) GetProjectId() string {
	return backup.rds.GetProjectId()
}

func (backup *SDBInstanceBackup) Refresh() error {
	_backup, err := backup.rds.region.GetDBInstanceBackup(backup.SelfLink)
	if err != nil {
		return errors.Wrap(err, "GetDBInstanceBackup")
	}
	return jsonutils.Update(backup, _backup)
}

func (backup *SDBInstanceBackup) GetStatus() string {
	switch backup.Status {
	case "SQL_BACKUP_RUN_STATUS_UNSPECIFIED":
		return api.DBINSTANCE_BACKUP_UNKNOWN
	case "ENQUEUED":
		return api.DBINSTANCE_BACKUP_CREATING
	case "FAILED":
		return api.DBINSTANCE_BACKUP_FAILED
	case "SUCCESSFUL", "OVERDUE", "RUNNING", "SKIPPED", "DELETION_PENDING":
		return api.DBINSTANCE_BACKUP_READY
	case "DELETION_FAILED", "DELETED":
		return api.DBINSTANCE_BACKUP_DELETING
	}
	return backup.Status
}

func (backup *SDBInstanceBackup) IsEmulated() bool {
	return false
}

func (backup *SDBInstanceBackup) GetEngine() string {
	return backup.rds.GetEngine()
}

func (backup *SDBInstanceBackup) GetEngineVersion() string {
	return backup.rds.GetEngineVersion()
}

func (backup *SDBInstanceBackup) GetDBInstanceId() string {
	return backup.rds.GetGlobalId()
}

func (backup *SDBInstanceBackup) GetStartTime() time.Time {
	return backup.StartTime
}

func (backup *SDBInstanceBackup) GetEndTime() time.Time {
	return backup.EndTime
}

func (backup *SDBInstanceBackup) GetBackupSizeMb() int {
	return 0
}

func (backup *SDBInstanceBackup) GetDBNames() string {
	return ""
}

func (backup *SDBInstanceBackup) GetBackupMode() string {
	switch backup.Type {
	case "AUTOMATED":
		return api.BACKUP_MODE_AUTOMATED
	default:
		return api.BACKUP_MODE_MANUAL
	}
}

func (backup *SDBInstanceBackup) Delete() error {
	return backup.rds.region.rdsDelete(backup.SelfLink)
}

func (region *SRegion) CreateDBInstanceBackup(instanceId string, name, desc string) error {
	body := map[string]interface{}{
		"name":        name,
		"description": desc,
	}
	return region.rdsDo(instanceId, "backupRuns", nil, jsonutils.Marshal(body))
}
