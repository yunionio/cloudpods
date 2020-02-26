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
	DBNames string

	// swagger:ignore
	Engine string
	// swagger:ignore
	EngineVersion string
	// swagger:ignore
	CloudregionId string
	// swagger:ignore
	BackupMode string
	// swagger:ignore
	ManagerId string
}

type DBInstanceBackupDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	DBInstanceResourceInfoBase

	SDBInstanceBackup
}
