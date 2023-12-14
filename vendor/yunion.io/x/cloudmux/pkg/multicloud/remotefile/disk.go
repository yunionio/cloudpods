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
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SDisk struct {
	SResourceBase

	storage       *SStorage
	ZoneId        string
	StorageId     string
	DiskFormat    string
	DiskSizeMb    int
	IsAutoDelete  bool
	DiskType      string
	FsFormat      string
	Iops          int
	Driver        string
	CacheMode     string
	Mountpoint    string
	Preallocation string
	AccessPath    string
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	if self.storage == nil {
		return nil, fmt.Errorf("disk %s(%s) missing storage", self.Name, self.Id)
	}
	return self.storage, nil
}

func (self *SDisk) GetIStorageId() string {
	return self.StorageId
}

func (self *SDisk) GetDiskFormat() string {
	return self.DiskFormat
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.DiskSizeMb
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.IsAutoDelete
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetDiskType() string {
	if len(self.DiskType) == 0 {
		return api.DISK_TYPE_DATA
	}
	return self.DiskType
}

func (self *SDisk) GetFsFormat() string {
	return self.FsFormat
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetIops() int {
	return self.Iops
}

func (self *SDisk) GetDriver() string {
	return self.Driver
}

func (self *SDisk) GetCacheMode() string {
	return self.CacheMode
}

func (self *SDisk) GetMountpoint() string {
	return self.Mountpoint
}

func (self *SDisk) GetPreallocation() string {
	return self.Preallocation
}

func (self *SDisk) GetAccessPath() string {
	return self.AccessPath
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SDisk) SetStorage(storage SStorage) {
	disk.storage = &storage
}
