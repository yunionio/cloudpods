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

package openstack

import (
	"context"
	"fmt"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNovaDisk struct {
	multicloud.SDisk
	OpenStackTags
	storage *SNovaStorage
	region  *SRegion

	instanceId string
}

func (disk *SNovaDisk) GetId() string {
	return disk.instanceId
}

func (disk *SNovaDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SNovaDisk) Resize(ctx context.Context, sizeMb int64) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SNovaDisk) GetName() string {
	return fmt.Sprintf("Sys disk for instance %s", disk.instanceId)
}

func (disk *SNovaDisk) GetGlobalId() string {
	return disk.instanceId
}

func (disk *SNovaDisk) IsEmulated() bool {
	return false
}

func (disk *SNovaDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SNovaDisk) GetStatus() string {
	return api.DISK_READY
}

func (disk *SNovaDisk) Refresh() error {
	return nil
}

func (disk *SNovaDisk) GetDiskFormat() string {
	return "raw"
}

func (disk *SNovaDisk) GetDiskSizeMB() int {
	instance, err := disk.region.GetInstance(disk.instanceId)
	if err != nil {
		return 0
	}
	if instance.Flavor.Disk != 0 {
		return instance.Flavor.Disk * 1024
	}
	if len(instance.Flavor.Id) > 0 {
		flavor, err := disk.region.GetFlavor(instance.Flavor.Id)
		if err != nil {
			return 0
		}
		return flavor.Disk * 1024
	}
	return 0
}

func (disk *SNovaDisk) GetIsAutoDelete() bool {
	return true
}

func (disk *SNovaDisk) GetTemplateId() string {
	return ""
}

func (disk *SNovaDisk) GetDiskType() string {
	return api.DISK_TYPE_SYS
}

func (disk *SNovaDisk) GetFsFormat() string {
	return ""
}

func (disk *SNovaDisk) GetIsNonPersistent() bool {
	return false
}

func (disk *SNovaDisk) GetDriver() string {
	return "scsi"
}

func (disk *SNovaDisk) GetCacheMode() string {
	return "none"
}

func (disk *SNovaDisk) GetMountpoint() string {
	return ""
}

func (disk *SNovaDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (disk *SNovaDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotFound
}

func (disk *SNovaDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (disk *SNovaDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (disk *SNovaDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (disk *SNovaDisk) GetCreatedAt() time.Time {
	return time.Time{}
}

func (disk *SNovaDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SNovaDisk) GetAccessPath() string {
	return ""
}

func (disk *SNovaDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SNovaDisk) GetProjectId() string {
	instance, err := disk.region.GetInstance(disk.instanceId)
	if err != nil {
		return ""
	}
	return instance.GetProjectId()
}
