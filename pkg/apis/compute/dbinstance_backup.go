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

package compute

import "yunion.io/x/onecloud/pkg/apis"

type DBInstanceBackupCreateInput struct {
	apis.VirtualResourceCreateInput

	// Rds实例名称或Id, 建议使用Id
	// required: true
	DBInstance string `json:"dbinstance"`
	// swagger:ignore
	DBInstanceId string `json:"dbinstance_id"`

	// 需要备份的Rds数据库列表
	// required: false
	Databases []string `json:"databases"`
	// swagger:ignore
	DBNames string `json:"db_names"`

	// swagger:ignore
	Engine string `json:"engine"`
	// swagger:ignore
	EngineVersion string `json:"engine_version"`
	// swagger:ignore
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	BackupMode string `json:"backup_mode"`
	// swagger:ignore
	ManagerId string `json:"manager_id"`
}

type DBInstanceBackupDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	DBInstanceResourceInfoBase

	SDBInstanceBackup
}
