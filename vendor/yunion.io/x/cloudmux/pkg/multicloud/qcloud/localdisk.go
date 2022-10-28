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

package qcloud

import (
	"context"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLocalDisk struct {
	multicloud.SDisk
	QcloudTags

	storage   *SLocalStorage
	DiskId    string
	DiskSize  float32
	DisktType string
	DiskUsage string
	imageId   string
}

func (self *SLocalDisk) GetSysTags() map[string]string {
	data := map[string]string{}
	data["hypervisor"] = api.HYPERVISOR_QCLOUD
	return data
}

func (self *SLocalDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLocalDisk) Delete(ctx context.Context) error {
	return nil
}

func (self *SLocalDisk) GetBillingType() string {
	return ""
}

func (self *SLocalDisk) GetFsFormat() string {
	return ""
}

func (self *SLocalDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SLocalDisk) GetDriver() string {
	return "scsi"
}

func (self *SLocalDisk) GetCacheMode() string {
	return "none"
}

func (self *SLocalDisk) GetMountpoint() string {
	return ""
}

func (self *SLocalDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SLocalDisk) GetDiskSizeMB() int {
	return int(self.DiskSize) * 1024
}

func (self *SLocalDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SLocalDisk) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLocalDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SLocalDisk) GetDiskType() string {
	switch self.DiskUsage {
	case "SYSTEM_DISK":
		return api.DISK_TYPE_SYS
	case "DATA_DISK":
		return api.DISK_TYPE_DATA
	default:
		return api.DISK_TYPE_DATA
	}
}

func (self *SLocalDisk) Refresh() error {
	return nil
}

func (self *SLocalDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SLocalDisk) GetTemplateId() string {
	return self.imageId
}

func (self *SLocalDisk) GetStatus() string {
	return api.DISK_READY
}

func (self *SLocalDisk) GetName() string {
	return self.DiskId
}

func (self *SLocalDisk) GetId() string {
	return self.DiskId
}

func (self *SLocalDisk) GetGlobalId() string {
	return self.DiskId
}

func (self *SLocalDisk) IsEmulated() bool {
	return false
}

func (self *SLocalDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, nil
}

func (self *SLocalDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, nil
}

func (self *SLocalDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SLocalDisk) Resize(ctx context.Context, size int64) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SLocalDisk) GetAccessPath() string {
	return ""
}

func (self *SLocalDisk) Rebuild(ctx context.Context) error {
	// TODO
	return cloudprovider.ErrNotSupported
}

func (disk *SLocalDisk) GetProjectId() string {
	return ""
}
