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

package google

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	billing "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SDisk struct {
	storage *SStorage

	Id                     string
	CreationTimestamp      time.Time
	Name                   string
	SizeGB                 int
	Zone                   string
	Status                 string
	SelfLink               string
	Type                   string
	SourceImage            string
	LastAttachTimestamp    time.Time
	LastDetachTimestamp    time.Time
	LabelFingerprint       string
	PhysicalBlockSizeBytes string
	ResourcePolicies       []string
	Kind                   string
	autoDelete             bool
	boot                   bool
	index                  int
}

func (region *SRegion) GetDisks(zone string, storageType string, maxResults int, pageToken string) ([]SDisk, error) {
	disks := []SDisk{}
	if len(zone) == 0 {
		return nil, fmt.Errorf("zone params can not be empty")
	}
	params := map[string]string{}
	if len(storageType) > 0 {
		params["filter"] = fmt.Sprintf(`type="%s/zones/%s/diskTypes/%s"`, region.GetUrlPrefixWithProjectId(), zone, storageType)
	}
	return disks, region.List(fmt.Sprintf("zones/%s/disks", zone), params, maxResults, pageToken, &disks)
}

func (region *SRegion) GetDisk(id string) (*SDisk, error) {
	disk := &SDisk{}
	return disk, region.Get(id, disk)
}

func (disk *SDisk) GetId() string {
	return disk.SelfLink
}

func (disk *SDisk) GetGlobalId() string {
	return getGlobalId(disk.SelfLink)
}

func (disk *SDisk) GetName() string {
	return disk.Name
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case "READY":
		return api.DISK_READY
	case "CREATING":
		return api.DISK_ALLOCATING
	case "RESTORING":
		return api.DISK_RESET
	case "FAILED":
		return api.DISK_ALLOC_FAILED
	case "DELETING":
		return api.DISK_DEALLOC
	default:
		return api.DISK_UNKNOWN
	}
}

func (disk *SDisk) IsEmulated() bool {
	return false
}

func (disk *SDisk) Refresh() error {
	_disk, err := disk.storage.zone.region.GetDisk(disk.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, _disk)
}

func (disk *SDisk) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SDisk) GetIStorageId() string {
	return disk.storage.GetGlobalId()
}

func (disk *SDisk) GetDiskFormat() string {
	return ""
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.SizeGB * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return disk.autoDelete
}

func (disk *SDisk) GetTemplateId() string {
	return disk.SourceImage
}

func (disk *SDisk) GetDiskType() string {
	if disk.index == 0 || disk.boot {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (disk *SDisk) GetFsFormat() string {
	return ""
}

func (disk *SDisk) GetIsNonPersistent() bool {
	return false
}

func (disk *SDisk) GetDriver() string {
	return "scsi"
}

func (disk *SDisk) GetCacheMode() string {
	return "none"
}

func (disk *SDisk) GetMountpoint() string {
	return ""
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (disk *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots(disk.SelfLink, 0, "")
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = disk.storage.zone.region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (disk *SDisk) GetISnapshot(id string) (cloudprovider.ICloudSnapshot, error) {
	return disk.storage.zone.region.GetSnapshot(id)
}

func (disk *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	result := []string{}
	for _, policy := range disk.ResourcePolicies {
		result = append(result, getGlobalId(policy))
	}
	return result, nil
}

func (disk *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotImplemented
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (disk *SDisk) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreationTimestamp
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SDisk) GetProjectId() string {
	return disk.storage.zone.region.GetProjectId()
}
