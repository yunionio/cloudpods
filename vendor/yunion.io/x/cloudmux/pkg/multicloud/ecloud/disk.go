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

package ecloud

import (
	"context"
	"fmt"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	storage *SStorage
	// TODO instance

	multicloud.SDisk
	EcloudTags
	multicloud.SBillingBase
	SZoneRegionBase
	SCreateTime

	ManualAttr SDiskManualAttr

	// 硬盘可挂载主机类型
	AttachServerTypes []string
	AvailabilityZone  string
	BackupId          string
	Description       string
	ID                string
	IsDelete          bool
	IsShare           bool
	// 磁盘所在集群的ID
	Metadata      string
	Name          string
	OperationFlag string
	// 硬盘挂在主机ID列表
	ServerId       []string
	SizeGB         int `json:"size"`
	SourceVolumeId string
	Status         string
	Type           string
	VolumeType     string
	Iscsi          bool
	ProductType    string
}

type SDiskManualAttr struct {
	IsVirtual  bool
	TempalteId string
	ServerId   string
}

func (d *SDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (d *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (d *SDisk) GetId() string {
	return d.ID
}

func (d *SDisk) GetName() string {
	return d.Name
}

func (d *SDisk) GetGlobalId() string {
	return d.ID
}

func (d *SDisk) GetCreatedAt() time.Time {
	return d.SCreateTime.GetCreatedAt()
}

func (d *SDisk) GetStatus() string {
	if d.IsDelete {
		// TODO
		return ""
	}
	switch d.Status {
	case "available", "in-use":
		return api.DISK_READY
	case "attaching":
		return api.DISK_ATTACHING
	case "backing_up":
		return api.DISK_BACKUP_STARTALLOC
	case "creating", "downloading":
		return api.DISK_ALLOCATING
	case "deleting":
		return api.DISK_DEALLOC
	case "uploading":
		return api.DISK_SAVING
	case "error":
		return api.DISK_ALLOC_FAILED
	case "error_deleting":
		return api.DISK_DEALLOC_FAILED
	case "restoring_backup":
		return api.DISK_REBUILD
	case "detaching":
		return api.DISK_DETACHING
	case "extending":
		return api.DISK_RESIZING
	case "error_extending":
		return api.DISK_RESIZE_FAILED
	case "error_restoring", "unrecognized":
		return api.DISK_UNKNOWN
	default:
		return api.DISK_UNKNOWN
	}
}

func (d *SDisk) Refresh() error {
	return nil
}

func (d *SDisk) IsEmulated() bool {
	return false
}

func (s *SDisk) GetSysTags() map[string]string {
	data := map[string]string{}
	data["hypervisor"] = api.HYPERVISOR_ECLOUD
	return data
}

func (s *SDisk) GetProjectId() string {
	return ""
}

func (s *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return s.storage, nil
}

func (s *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (s *SDisk) GetDiskSizeMB() int {
	return s.SizeGB * 1024
}

func (s *SDisk) GetIsAutoDelete() bool {
	if s.GetDiskType() == api.DISK_TYPE_SYS {
		return true
	}
	return false
}

func (s *SDisk) GetTemplateId() string {
	return s.ManualAttr.TempalteId
}

func (s *SDisk) GetDiskType() string {
	if s.ManualAttr.IsVirtual {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (s *SDisk) GetFsFormat() string {
	return ""
}

func (s *SDisk) GetIsNonPersistent() bool {
	return false
}

func (s *SDisk) GetDriver() string {
	return "scsi"
}

func (s *SDisk) GetCacheMode() string {
	return "none"
}

func (s *SDisk) GetMountpoint() string {
	return ""
}

func (s *SDisk) GetAccessPath() string {
	return ""
}

func (s *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (s *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (s *SDisk) GetISnapshot(id string) (cloudprovider.ICloudSnapshot, error) {
	parentId, isSystem := s.ID, false
	if s.ManualAttr.IsVirtual {
		parentId, isSystem = s.ManualAttr.ServerId, true
	}
	snapshots, err := s.storage.zone.region.GetSnapshots(id, parentId, isSystem)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &snapshots[0], nil
}

func (s *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	parentId, isSystem := s.ID, false
	if s.ManualAttr.IsVirtual {
		parentId, isSystem = s.ManualAttr.ServerId, true
	}
	snapshots, err := s.storage.zone.region.GetSnapshots("", parentId, isSystem)
	if err != nil {
		return nil, err
	}
	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := range snapshots {
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (s *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotImplemented
}

func (s *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (s *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (s *SRegion) GetDisks() ([]SDisk, error) {
	request := NewNovaRequest(NewApiRequest(s.ID, "/api/v2/volume/volume/volume/list/with/server", nil, nil))
	disks := make([]SDisk, 0, 5)
	err := s.client.doList(context.Background(), request, &disks)
	if err != nil {
		return nil, err
	}
	return disks, nil
}

func (s *SRegion) GetDisk(id string) (*SDisk, error) {
	// TODO
	request := NewNovaRequest(NewApiRequest(s.ID, fmt.Sprintf("/api/v2/volume/volume/volumeDetail/%s", id), nil, nil))
	var disk SDisk
	err := s.client.doGet(context.Background(), request, &disk)
	if err != nil {
		return nil, err
	}
	return &disk, nil
}
