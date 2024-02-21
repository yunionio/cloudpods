// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	VolcEngineTags

	ZoneId             string
	VolumeId           string
	VolumeName         string
	VolumeType         string
	Description        string
	InstanceId         string
	ImageId            string
	Size               int
	Status             string
	Kind               string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	BillingType        string
	PayType            string
	TradeStatus        int
	ExpiredTime        time.Time
	ProjectName        string
	DeleteWithInstance bool
}

func (disk *SDisk) GetId() string {
	return disk.VolumeId
}

func (disk *SDisk) Delete(ctx context.Context) error {
	return disk.storage.zone.region.DeleteDisk(disk.VolumeId)
}

func (disk *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return disk.storage.zone.region.ResizeDisk(disk.VolumeId, sizeMb/1024)
}

func (disk *SDisk) GetName() string {
	if len(disk.VolumeName) > 0 {
		return disk.VolumeName
	}
	return disk.VolumeId
}

func (disk *SDisk) GetGlobalId() string {
	return disk.VolumeId
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case "creating":
		return api.DISK_ALLOCATING
	default:
		return api.DISK_READY
	}
}

func (disk *SDisk) Refresh() error {
	_disk, err := disk.storage.zone.region.GetDisk(disk.VolumeId)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, _disk)
}

func (disk *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.Size * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return disk.DeleteWithInstance
}

func (disk *SDisk) GetTemplateId() string {
	return disk.ImageId
}

func (disk *SDisk) GetDiskType() string {
	switch disk.Kind {
	case "system":
		return api.DISK_TYPE_SYS
	case "data":
		return api.DISK_TYPE_DATA
	default:
		return api.DISK_TYPE_DATA
	}
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

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots(disk.VolumeId, "", nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = disk.storage.zone.region
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (disk *SDisk) GetBillingType() string {
	if disk.PayType != "post" {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreatedAt
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return convertExpiredAt(disk.ExpiredTime)
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "Rebuild")

}

func (disk *SDisk) GetProjectId() string {
	return disk.ProjectName
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := disk.storage.zone.region.CreateSnapshot(disk.VolumeId, name, desc)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// region
func (region *SRegion) GetDisks(instanceId string, zoneId string, category string, diskIds []string) ([]SDisk, error) {
	params := make(map[string]string)
	params["PageSize"] = "100"
	pageNum := 1
	params["PageNumber"] = fmt.Sprintf("%d", pageNum)

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}
	if len(category) > 0 {
		params["VolumeType"] = category
	}
	for index, id := range diskIds {
		key := fmt.Sprintf("VolumeIds.%d", index+1)
		params[key] = id
	}

	ret := []SDisk{}
	for {
		resp, err := region.storageRequest("DescribeVolumes", params)
		if err != nil {
			return nil, errors.Wrap(err, "GetDisks fail")
		}
		part := struct {
			Volumes    []SDisk
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Volumes...)
		if len(ret) >= part.TotalCount || len(part.Volumes) == 0 {
			break
		}
		pageNum++
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
	}
	return ret, nil
}

func (region *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, desc string, projectId string) (string, error) {
	params := make(map[string]string)
	params["ZoneId"] = zoneId
	params["VolumeName"] = name
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["VolumeType"] = category
	// only data disk is supported
	params["Kind"] = "data"

	params["Size"] = fmt.Sprintf("%d", sizeGb)
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := region.storageRequest("CreateVolume", params)
	if err != nil {
		return "", err
	}
	return body.GetString("VolumeId")
}

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, err := region.GetDisks("", "", "", []string{diskId})
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("%s not found", diskId))
	}
	for _, disk := range disks {
		if disk.VolumeId == diskId {
			return &disk, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, fmt.Sprintf("%s not found", diskId))
}

func (region *SRegion) DeleteDisk(diskId string) error {
	params := make(map[string]string)
	params["VolumeId"] = diskId
	_, err := region.storageRequest("DeleteVolume", params)
	return err
}

func (region *SRegion) ResizeDisk(diskId string, sizeGb int64) error {
	params := make(map[string]string)
	params["VolumeId"] = diskId
	params["NewSize"] = fmt.Sprintf("%d", sizeGb)

	_, err := region.storageRequest("ExtendVolume", params)
	if err != nil {
		return errors.Wrapf(err, "Resizing disk (%s) to %d GiB failed", diskId, sizeGb)
	}

	return nil
}
