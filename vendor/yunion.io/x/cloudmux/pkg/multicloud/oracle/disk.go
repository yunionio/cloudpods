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

package oracle

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	SOracleTag

	AvailabilityDomain string
	DisplayName        string
	Id                 string
	IsHydrated         bool
	LifecycleState     string
	VpusPerGB          int
	SizeInGBs          int
	SizeInMBs          int
	TimeCreated        time.Time
	VolumeType         string
}

func (self *SRegion) GetDisk(id string) (*SDisk, error) {
	resp, err := self.get(SERVICE_IAAS, "volumes", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SDisk{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil

	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetId() string {
	return self.Id
}

func (self *SRegion) DeleteDisk(diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.DeleteDisk(self.Id)
}

func (self *SRegion) ResizeDisk(ctx context.Context, diskId string, sizeGb int64) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return self.storage.zone.region.ResizeDisk(ctx, self.Id, sizeMb/1024)
}

func (self *SDisk) GetName() string {
	return self.DisplayName
}

func (self *SDisk) GetGlobalId() string {
	return self.Id
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetStatus() string {
	// AVAILABLE, FAULTY, PROVISIONING, RESTORING, TERMINATED, TERMINATING
	switch self.LifecycleState {
	case "ATTACHING", "DETACHING", "EXPANDING", "ROLLBACKING":
		return api.DISK_ALLOCATING
	default:
		return api.DISK_READY
	}
}

func (self *SDisk) Refresh() error {
	disk, err := self.storage.zone.region.GetDisk(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, disk)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetDiskType() string {
	return api.DISK_TYPE_DATA
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return "scsi"
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.SizeInMBs
}

func (self *SDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.TimeCreated
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDisk) GetProjectId() string {
	return ""
}

func (self *SRegion) GetDisks(zoneId string) ([]SDisk, error) {
	params := map[string]interface{}{}
	if len(zoneId) > 0 {
		params["availabilityDomain"] = zoneId
	}
	resp, err := self.list(SERVICE_IAAS, "volumes", params)
	if err != nil {
		return nil, err
	}
	ret := []SDisk{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
