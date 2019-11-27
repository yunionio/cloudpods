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
	"time"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRegion struct{}

func (r *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return nil, fmt.Errorf("Not Implement GetIDiskById")
}

func (r *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, fmt.Errorf("Not Implement GetIHostById")
}

func (r *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return nil, fmt.Errorf("Not Implement GetIHosts")
}

func (r *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, fmt.Errorf("Not Implement GetISnapshotById")
}

func (r *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, fmt.Errorf("Not Implement GetISnapshots")
}

func (r *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, fmt.Errorf("Not Implement GetIStorageById")
}

func (r *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, fmt.Errorf("Not Implement GetIStoragecacheById")
}

func (r *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return nil, fmt.Errorf("Not Implement GetIStoragecaches")
}

func (r *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return nil, fmt.Errorf("Not Implement GetIStorages")
}

func (r *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, fmt.Errorf("Not Implement GetIVMById")
}

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

func (self *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	return nil, fmt.Errorf("Not Implemented GetICloudEvnets")
}
