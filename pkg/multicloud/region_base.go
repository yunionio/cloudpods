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

package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRegion struct{}

func (r *SRegion) CreateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", fmt.Errorf("CreateSnapshotPolicy not implement")
}

func (r *SRegion) UpdateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput, snapshotPolicyId string) error {
	return fmt.Errorf("UpdateSnapshotPolicy not implement")
}

func (r *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, fmt.Errorf("GetISnapshotPolicyById not implement")
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, fmt.Errorf("GetISnapshotPolicies not implement")
}

func (self *SRegion) DeleteSnapshotPolicy(string) error {
	return fmt.Errorf("DeleteSnapshotPolicy not implement")
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return fmt.Errorf("ApplySnapshotPolicyToDisks not implement")
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return fmt.Errorf("ApplySnapshotPolicyToDisks not implement")
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateISku(name string, vCpu int, memoryMb int) error {
	return fmt.Errorf("Not Implement CreateISku")
}

func (self *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	return nil, fmt.Errorf("Not Implement GetINetworkInterfaces")
}

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	return nil, fmt.Errorf("GetIDBInstances not implement")
}

func (self *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	return nil, fmt.Errorf("GetIDBInstanceById not implement")
}

func (self *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, fmt.Errorf("Not Implemented GetIDBInstanceBackups")
}

func (self *SRegion) GetIDBInstanceBackupById(backupId string) (cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, fmt.Errorf("Not Implemented GetIDBInstanceBackupById")
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	return nil, fmt.Errorf("Not Implemented GetIElasticcaches")
}

func (self *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	return nil, fmt.Errorf("Not Implemented CreateIDBInstance")
}

func (self *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	return nil, fmt.Errorf("Not Implemented CreateIElasticcaches")
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	return nil, fmt.Errorf("Not Implemented GetIElasticcacheById")
}
