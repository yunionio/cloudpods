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
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SOsDisk struct {
	multicloud.SDisk
	AzureTags
	region *SRegion

	OsType       string `json:"osType,omitempty"`
	Caching      string `json:"caching,omitempty"`
	Name         string
	DiskSizeGB   TAzureInt32            `json:"diskSizeGB,omitempty"`
	ManagedDisk  *ManagedDiskParameters `json:"managedDisk,omitempty"`
	CreateOption string                 `json:"createOption,omitempty"`
	Vhd          *VirtualHardDisk       `json:"vhd,omitempty"`
}

func (self *SOsDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	if self.ManagedDisk != nil {
		snapshot, err := self.region.CreateSnapshot(self.ManagedDisk.ID, name, desc)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateSnapshot")
		}
		return snapshot, nil
	}
	return nil, cloudprovider.ErrNotSupported
}

func (self *SOsDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (self *SOsDisk) Delete(ctx context.Context) error {
	if self.ManagedDisk != nil {
		return self.region.del(self.ManagedDisk.ID)
	}
	return cloudprovider.ErrNotSupported
}

func (self *SOsDisk) GetStatus() string {
	return api.DISK_READY
}

func (self *SOsDisk) GetId() string {
	if self.ManagedDisk != nil {
		return strings.ToLower(self.ManagedDisk.ID)
	}
	return self.Vhd.Uri
}

func (self *SOsDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SOsDisk) GetName() string {
	return self.Name
}

func (self *SOsDisk) Resize(ctx context.Context, sizeMb int64) error {
	if self.ManagedDisk != nil {
		return self.region.ResizeDisk(self.ManagedDisk.ID, int32(sizeMb/1024))
	}
	return cloudprovider.ErrNotSupported
}

func (self *SOsDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	storageType := "Standard_LRS"
	if self.ManagedDisk != nil && len(self.ManagedDisk.StorageAccountType) > 0 {
		storageType = self.ManagedDisk.StorageAccountType
	}
	return &SStorage{storageType: storageType, zone: self.region.getZone()}, nil
}

func (self *SOsDisk) GetFsFormat() string {
	return ""
}

func (self *SOsDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SOsDisk) GetDriver() string {
	return "scsi"
}

func (self *SOsDisk) GetCacheMode() string {
	return "none"
}

func (self *SOsDisk) GetMountpoint() string {
	return ""
}

func (self *SOsDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SOsDisk) GetDiskSizeMB() int {
	if self.ManagedDisk != nil {
		disk, err := self.region.GetDisk(self.ManagedDisk.ID)
		if err != nil {
			return 0
		}
		return disk.GetDiskSizeMB()
	}
	return int(self.DiskSizeGB.Int32()) * 1024
}

func (self *SOsDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SOsDisk) GetTemplateId() string {
	if self.ManagedDisk != nil {
		disk, err := self.region.GetDisk(self.ManagedDisk.ID)
		if err == nil {
			return disk.GetTemplateId()
		}
	}
	return ""
}

func (self *SOsDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SOsDisk) GetDiskType() string {
	return api.DISK_TYPE_SYS
}

func (disk *SOsDisk) GetAccessPath() string {
	return ""
}

func (self *SOsDisk) Rebuild(ctx context.Context) error {
	// TODO
	return cloudprovider.ErrNotSupported
}

func (self *SOsDisk) GetProjectId() string {
	if self.ManagedDisk != nil {
		return getResourceGroup(self.ManagedDisk.ID)
	}
	return ""
}
