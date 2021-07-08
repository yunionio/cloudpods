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

package cloudprovider

import "time"

// 备份状态
type TMongoDBBackupStatus string

// 备份方法
type TMongoDBBackupMethod string

// 备份方式
type TMongoDBBackupType string

const (
	MongoDBBackupStatusCreating  = TMongoDBBackupStatus("creating")
	MongoDBBackupStatusAvailable = TMongoDBBackupStatus("available")
	MongoDBBackupStatusFailed    = TMongoDBBackupStatus("failed")
	MongoDBBackupStatusUnknown   = TMongoDBBackupStatus("unknown")

	MongoDBBackupMethodPhysical = TMongoDBBackupMethod("physical")
	MongoDBBackupMethodLogical  = TMongoDBBackupMethod("logical")

	MongoDBBackupTypeAuto   = TMongoDBBackupType("auto")
	MongoDBBackupTypeManual = TMongoDBBackupType("manual")
)

type SMongoDBBackup struct {
	Name         string
	Description  string
	StartTime    time.Time
	EndTime      time.Time
	Status       TMongoDBBackupStatus
	BackupMethod TMongoDBBackupMethod
	BackupType   TMongoDBBackupType
	BackupSizeKb int
}

type SMongoDBBackups struct {
	Data  []SMongoDBBackup
	Total int
}
