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

package jdcloud

import (
	"context"
	"fmt"
	"time"

	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/disk/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/disk/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/disk/models"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	storage *SStorage

	multicloud.SDisk
	JdcloudTags
	multicloud.SBillingBase

	models.Disk

	ImageId      string
	IsSystemDisk bool
}

func (d *SDisk) GetBillingType() string {
	return billingType(&d.Charge)
}

func (d *SDisk) GetExpiredAt() time.Time {
	return expireAt(&d.Charge)
}

func (d *SDisk) GetId() string {
	return d.DiskId
}

func (d *SDisk) GetName() string {
	return d.Name
}

func (d *SDisk) GetGlobalId() string {
	return d.GetId()
}

func (d *SDisk) GetIops() int {
	return d.Iops
}

func (d *SDisk) GetStatus() string {
	switch d.Status {
	case "available", "in-use":
		return api.DISK_READY
	case "creating":
		return api.DISK_ALLOCATING
	case "extending":
		return api.DISK_RESIZING
	case "restoring":
		return api.DISK_RESET
	case "deleting":
		return api.DISK_DEALLOC
	case "error_create":
		return api.DISK_ALLOC_FAILED
	case "error_delete":
		return api.DISK_DEALLOC_FAILED
	case "error_restore":
		return api.DISK_RESET_FAILED
	case "error_extend":
		return api.DISK_RESIZE_FAILED
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

func (d *SDisk) GetSysTags() map[string]string {
	return map[string]string{
		"hypervisor": api.HYPERVISOR_JDCLOUD,
	}
}

func (d *SDisk) GetProjectId() string {
	return ""
}

func (d *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return d.storage, nil
}

func (d *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (d *SDisk) GetDiskSizeMB() int {
	return d.DiskSizeGB * 1024
}

func (d *SDisk) GetIsAutoDelete() bool {
	if len(d.Attachments) == 0 {
		return false
	}
	return true
}

func (d *SDisk) GetTemplateId() string {
	// TODO 通过快照创建的盘应该如何
	return d.ImageId
}

func (d *SDisk) GetDiskType() string {
	if d.IsSystemDisk {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (d *SDisk) GetFsFormat() string {
	return ""
}

func (d *SDisk) GetIsNonPersistent() bool {
	return false
}

func (d *SDisk) GetDriver() string {
	return "scsi"
}

func (d *SDisk) GetCacheMode() string {
	return "none"
}

func (d *SDisk) GetMountpoint() string {
	return ""
}

func (d *SDisk) GetAccessPath() string {
	return ""
}

func (d *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (d *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (d *SDisk) GetISnapshot(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := d.storage.zone.region.GetSnapshotById(id)
	if err != nil {
		return nil, err
	}
	snapshot.disk = d
	return snapshot, nil
}

func (s *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	n := 1
	for {
		parts, total, err := s.storage.zone.region.GetSnapshots(s.DiskId, n, 100)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, parts...)
		if len(snapshots) >= total {
			break
		}
		n++
	}
	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := range snapshots {
		snapshots[i].disk = s
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

func (r *SRegion) GetDisks(instanceId, zoneId, diskType string, diskIds []string, pageNumber, pageSize int) ([]SDisk, int, error) {
	filters := []commodels.Filter{}
	if instanceId != "" {
		filters = append(filters, commodels.Filter{
			Name:   "instanceId",
			Values: []string{instanceId},
		}, commodels.Filter{
			Name:   "instanceType",
			Values: []string{"vm"},
		})
	}
	if zoneId != "" {
		filters = append(filters, commodels.Filter{
			Name:   "az",
			Values: []string{zoneId},
		})
	}
	if len(diskIds) > 0 {
		filters = append(filters, commodels.Filter{
			Name:   "diskId",
			Values: diskIds,
		})
	}
	if diskType != "" {
		filters = append(filters, commodels.Filter{
			Name:   "diskType",
			Values: []string{diskType},
		})
	}
	req := apis.NewDescribeDisksRequestWithAllParams(r.ID, &pageNumber, &pageSize, nil, filters)
	client := client.NewDiskClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeDisks(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error.Code >= 400 {
		return nil, 0, fmt.Errorf(resp.Error.Message)
	}
	total := resp.Result.TotalCount
	disks := make([]SDisk, 0, len(resp.Result.Disks))
	for i := range resp.Result.Disks {
		disks = append(disks, SDisk{
			Disk: resp.Result.Disks[i],
		})
	}
	return disks, total, nil
}

func (r *SRegion) GetDiskById(id string) (*SDisk, error) {
	req := apis.NewDescribeDiskRequest(r.ID, id)
	client := client.NewDiskClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeDisk(req)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code >= 400 {
		return nil, fmt.Errorf(resp.Error.Message)
	}
	return &SDisk{
		Disk: resp.Result.Disk,
	}, nil
}
