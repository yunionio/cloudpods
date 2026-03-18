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
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

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
	AttachServerTypes []string `json:"attachServerTypes,omitempty"`
	AvailabilityZone  string   `json:"region"`
	BackupId          string   `json:"backupId,omitempty"`
	Description       string   `json:"description,omitempty"`
	ID                string   `json:"id"`
	IsDelete          bool     `json:"isDelete,omitempty"`
	IsShare           bool     `json:"isShare,omitempty"`
	// 磁盘所在集群的ID
	Metadata      string `json:"metadata,omitempty"`
	Name          string `json:"name,omitempty"`
	OperationFlag string `json:"operationFlag,omitempty"`
	// 硬盘挂载主机ID列表
	ServerId       []string `json:"serverIds,omitempty"`
	SizeGB         int      `json:"size"`
	SourceVolumeId string   `json:"sourceVolumeId,omitempty"`
	Status         string   `json:"status,omitempty"`
	Type           string   `json:"type,omitempty"`
	VolumeType     string   `json:"volumeType,omitempty"`
	Iscsi          bool     `json:"iscsi,omitempty"`
	ProductType    string   `json:"productType,omitempty"`
}

type SDiskManualAttr struct {
	IsVirtual  bool
	TemplateId string
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
	switch strings.ToLower(d.Status) {
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
	if s.ManualAttr.TemplateId != "" {
		return s.ManualAttr.TemplateId
	}
	return ""
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
	if s.storage == nil {
		return fmt.Errorf("disk not attached to storage")
	}
	return s.storage.zone.region.PreDeleteVolume(s.ID)
}

func (s *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("disk not attached to storage")
	}
	r := s.storage.zone.region
	snapshotId, err := r.CreateEbsSnapshot(s.ID, name, desc)
	if err != nil {
		return nil, err
	}
	snapshots, err := r.GetSnapshots(snapshotId, s.ID, false)
	if err != nil || len(snapshots) == 0 {
		return nil, errors.Wrapf(err, "GetSnapshots after create")
	}
	for i := range snapshots {
		if snapshots[i].Id == snapshotId {
			return &snapshots[i], nil
		}
	}
	return &snapshots[0], nil
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
	if s.storage == nil {
		return fmt.Errorf("disk not attached to storage")
	}
	newSizeGB := int64(newSizeMB / 1024)
	return s.storage.zone.region.ResizeDisk(ctx, s.ID, newSizeGB)
}

func (s *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (s *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (s *SRegion) GetDisks() ([]SDisk, error) {
	req := NewOpenApiEbsRequest(s.RegionId, "/api/v2/volume/volume/volume/list/with/server", nil, nil)
	disks := make([]SDisk, 0, 5)
	err := s.client.doList(context.Background(), req.Base(), &disks)
	if err != nil {
		return nil, err
	}
	return disks, nil
}

func (s *SRegion) GetDisk(id string) (*SDisk, error) {
	req := NewOpenApiEbsRequest(s.RegionId, fmt.Sprintf("/api/v2/volume/volume/volumeDetail/%s", id), nil, nil)
	var disk SDisk
	err := s.client.doGet(context.Background(), req.Base(), &disk)
	if err != nil {
		return nil, err
	}
	return &disk, nil
}
