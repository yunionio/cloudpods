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

import "yunion.io/x/onecloud/pkg/util/billing"

type TBackupMethod string

const (
	BackupMethodLogical  = TBackupMethod("Logical")
	BackupMethodPhysical = TBackupMethod("Physical")
	BackupMethodUnknown  = TBackupMethod("")
)

type SDBInstanceNetwork struct {
	IP        string
	NetworkId string
}

type SExtraIp struct {
	IP  string
	URL string
}

type SZoneInfo struct {
	Zone1  string
	Zone2  string
	Zone3  string
	ZoneId string
}

type SInstanceType struct {
	InstanceType string
	SZoneInfo
}

type SManagedDBInstanceCreateConfig struct {
	Name        string
	Description string
	StorageType string
	DiskSizeGB  int
	SInstanceType
	VcpuCount        int
	VmemSizeMb       int
	VpcId            string
	SecgroupIds      []string
	NetworkId        string
	Address          string
	Engine           string
	EngineVersion    string
	Category         string
	Port             int
	MasterInstanceId string
	Password         string
	Username         string
	ProjectId        string

	BillingCycle *billing.SBillingCycle
	Tags         map[string]string

	// 仅从备份恢复到新实例用到
	RdsId    string
	BackupId string
}

type SManagedDBInstanceChangeConfig struct {
	DiskSizeGB   int
	StorageType  string
	InstanceType string
	VcpuCount    int
	VmemSizeMb   int
}

type SDBInstanceDatabaseCreateConfig struct {
	Name         string
	CharacterSet string
	Description  string
}

type SDBInstancePrivilege struct {
	Account   string
	Database  string
	Privilege string
}

type SDBInstanceAccountCreateConfig struct {
	Name        string
	Host        string
	Description string
	Password    string
}

type SDBInstanceBackupCreateConfig struct {
	Name        string
	Description string
	Databases   []string
}

type SDBInstanceRecoveryConfig struct {
	BackupId                   string
	Databases                  map[string]string
	OriginDBInstanceExternalId string
}
