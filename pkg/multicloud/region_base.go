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
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRegion struct{}

func (r *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDiskById")
}

func (r *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIHostById")
}

func (r *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIHosts")
}

func (r *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshotById")
}

func (r *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshots")
}

func (r *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStorageById")
}

func (r *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStoragecacheById")
}

func (r *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStoragecaches")
}

func (r *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStorages")
}

func (r *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIVMById")
}

func (r *SRegion) CreateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateSnapshotPolicy")
}

func (r *SRegion) UpdateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput, snapshotPolicyId string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "UpdateSnapshotPolicy")
}

func (r *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshotPolicyById")
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshotPolicies")
}

func (self *SRegion) DeleteSnapshotPolicy(string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "DeleteSnapshotPolicy")
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ApplySnapshotPolicyToDisks")
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "CancelSnapshotPolicyToDisks")
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "GetISkus")
}

func (self *SRegion) CreateISku(name string, vCpu int, memoryMb int) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateISku")
}

func (self *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetINetworkInterfaces")
}

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstances")
}

func (self *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceById")
}

func (self *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceBackups")
}

func (self *SRegion) GetIDBInstanceBackupById(backupId string) (cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceBackupById")
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticcaches")
}

func (self *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIDBInstance")
}

func (self *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIElasticcaches")
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticcacheById")
}

func (self *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudEvents")
}

func (self *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudQuotas")
}

func (self *SRegion) CreateInternetGateway() (cloudprovider.ICloudInternetGateway, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "CreateInternetGateway")
}
