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

package remotefile

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SDBInstance struct {
	SResourceBase

	region                *SRegion
	RegionId              string
	SecurityGroupIds      []string
	Port                  int
	Engine                string
	EngineVersion         string
	InstanceType          string
	VcpuCount             int
	VmemSizeMb            int
	DiskSizeGb            int
	DiskSizeUsedGb        int
	Category              string
	StorageType           string
	MaintainTime          string
	ConnectionStr         string
	InternalConnectionStr string
	Zone1Id               string
	Zone2Id               string
	Zone3Id               string
	VpcId                 string
	Iops                  int
}

func (self *SDBInstance) Reboot() error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) GetMasterInstanceId() string {
	return ""
}

func (self *SDBInstance) GetSecurityGroupIds() ([]string, error) {
	return self.SecurityGroupIds, nil
}

func (self *SDBInstance) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) GetPort() int {
	return self.Port
}

func (self *SDBInstance) GetEngine() string {
	return self.Engine
}

func (self *SDBInstance) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SDBInstance) GetInstanceType() string {
	return self.InstanceType
}

func (self *SDBInstance) GetVcpuCount() int {
	return self.VcpuCount
}

func (self *SDBInstance) GetVmemSizeMB() int {
	return self.VmemSizeMb
}

func (self *SDBInstance) GetDiskSizeGB() int {
	return self.DiskSizeGb
}

func (self *SDBInstance) GetDiskSizeUsedMB() int {
	return self.DiskSizeUsedGb
}

func (self *SDBInstance) GetCategory() string {
	return self.Category
}

func (self *SDBInstance) GetStorageType() string {
	return self.StorageType
}

func (self *SDBInstance) GetMaintainTime() string {
	return self.MaintainTime
}

func (self *SDBInstance) GetConnectionStr() string {
	return self.ConnectionStr
}

func (self *SDBInstance) GetInternalConnectionStr() string {
	return self.InternalConnectionStr
}

func (self *SDBInstance) GetZone1Id() string {
	return self.Zone1Id
}

func (self *SDBInstance) GetZone2Id() string {
	return self.Zone2Id
}

func (self *SDBInstance) GetZone3Id() string {
	return self.Zone3Id
}

func (self *SDBInstance) GetIVpcId() string {
	return self.VpcId
}

func (self *SDBInstance) GetIops() int {
	return self.Iops
}

func (self *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDBInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) OpenPublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) ClosePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDBInstance) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDBInstance) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	res, err := self.client.GetDBInstances()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstance{}
	for i := range res {
		if res[i].RegionId != self.GetId() {
			continue
		}
		res[i].region = self
		ret = append(ret, &res[i])
	}
	return ret, nil
}

func (self *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	res, err := self.GetIDBInstances()
	if err != nil {
		return nil, err
	}
	for i := range res {
		if res[i].GetGlobalId() == id {
			return res[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (instance *SDBInstance) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	return errors.ErrNotImplemented
}
