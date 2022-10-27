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

package azure

import (
	"context"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ClassicProperties struct {
	DiskName        string
	Caching         string
	OperatingSystem string
	IoType          string
	DiskSizeGB      int32
	DiskSize        int32
	diskSizeMB      int32
	CreatedTime     string
	SourceImageName string
	VhdUri          string
	StorageAccount  SubResource
}

type SClassicDisk struct {
	multicloud.SDisk
	AzureTags
	region *SRegion

	Id         string
	Name       string
	Type       string
	Properties ClassicProperties
}

func (self *SClassicDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetClassicDisk(id string) (*SClassicDisk, error) {
	disk := &SClassicDisk{region: self}
	err := self.get(id, url.Values{}, disk)
	if err != nil {
		return nil, errors.Wrapf(err, "get(%s)", id)
	}
	return disk, nil
}

func (self *SClassicDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SClassicDisk) GetFsFormat() string {
	return ""
}

func (self *SClassicDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SClassicDisk) GetDriver() string {
	return "scsi"
}

func (self *SClassicDisk) GetCacheMode() string {
	return "none"
}

func (self *SClassicDisk) GetMountpoint() string {
	return ""
}

func (self *SClassicDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SClassicDisk) GetDiskSizeMB() int {
	if self.Properties.DiskSizeGB > 0 {
		return int(self.Properties.DiskSizeGB * 1024)
	}
	if self.Properties.DiskSize > 0 {
		return int(self.Properties.DiskSize * 1024)
	}
	return 0
}

func (self *SClassicDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SClassicDisk) GetTemplateId() string {
	return ""
}

func (self *SClassicDisk) GetDiskType() string {
	return self.Properties.OperatingSystem
}

func (self *SClassicDisk) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SClassicDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SClassicDisk) GetGlobalId() string {
	return strings.ToLower(self.Id)
}

func (self *SClassicDisk) GetId() string {
	return self.GetGlobalId()
}

func (self *SClassicDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (self *SClassicDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	storage := struct {
		Properties struct {
			AccountType string
		}
	}{}
	err := self.region.get(self.Properties.StorageAccount.ID, url.Values{}, &storage)
	if err != nil {
		return nil, errors.Wrapf(err, "get(%s)", self.Properties.StorageAccount.ID)
	}
	return &SClassicStorage{region: self.region, AccountType: storage.Properties.AccountType}, nil
}

func (self *SClassicDisk) GetName() string {
	return self.Properties.DiskName
}

func (self *SClassicDisk) GetStatus() string {
	return api.DISK_READY
}

func (self *SClassicDisk) IsEmulated() bool {
	return false
}

func (self *SClassicDisk) Refresh() error {
	return nil
}

func (self *SClassicDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) Resize(ctx context.Context, sizeMb int64) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SClassicDisk) GetAccessPath() string {
	return ""
}

func (self *SClassicDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) GetProjectId() string {
	return ""
}
