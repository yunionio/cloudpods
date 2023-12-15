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

type Placement struct {
	ProjectId int
	Zone      string
}

type SBootDisk struct {
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
	ImageId            string
}

func (self *SRegion) GetBootDisk(id string) (*SBootDisk, error) {
	resp, err := self.get(SERVICE_IAAS, "bootVolumes", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SBootDisk{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil

	return nil, cloudprovider.ErrNotImplemented
}

func (self *SBootDisk) GetId() string {
	return self.Id
}

func (self *SBootDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBootDisk) Resize(ctx context.Context, sizeMb int64) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SBootDisk) GetName() string {
	return self.DisplayName
}

func (self *SBootDisk) GetGlobalId() string {
	return self.Id
}

func (self *SBootDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SBootDisk) GetStatus() string {
	// AVAILABLE, FAULTY, PROVISIONING, RESTORING, TERMINATED, TERMINATING
	switch self.LifecycleState {
	case "ATTACHING", "DETACHING", "EXPANDING", "ROLLBACKING":
		return api.DISK_ALLOCATING
	default:
		return api.DISK_READY
	}
}

func (self *SBootDisk) Refresh() error {
	disk, err := self.storage.zone.region.GetBootDisk(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, disk)
}

func (self *SBootDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SBootDisk) GetDiskType() string {
	return api.DISK_TYPE_SYS
}

func (self *SBootDisk) GetFsFormat() string {
	return ""
}

func (self *SBootDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SBootDisk) GetDriver() string {
	return "scsi"
}

func (self *SBootDisk) GetCacheMode() string {
	return "none"
}

func (self *SBootDisk) GetMountpoint() string {
	return ""
}

func (self *SBootDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SBootDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SBootDisk) GetDiskSizeMB() int {
	return self.SizeInMBs
}

func (self *SBootDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SBootDisk) GetCreatedAt() time.Time {
	return self.TimeCreated
}

func (self *SBootDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SBootDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SBootDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SBootDisk) GetTemplateId() string {
	return self.ImageId
}

func (disk *SBootDisk) GetAccessPath() string {
	return ""
}

func (self *SBootDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SBootDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SBootDisk) GetProjectId() string {
	return ""
}

func (self *SRegion) GetBootDisks(zoneId string) ([]SBootDisk, error) {
	params := map[string]interface{}{}
	if len(zoneId) > 0 {
		params["availabilityDomain"] = zoneId
	}
	resp, err := self.list(SERVICE_IAAS, "bootVolumes", params)
	if err != nil {
		return nil, err
	}
	ret := []SBootDisk{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
