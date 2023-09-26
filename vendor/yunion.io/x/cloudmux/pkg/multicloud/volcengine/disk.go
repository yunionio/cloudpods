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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

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
	BillingType        TChargeType
	PayType            string
	TradeStatus        int
	ExpiredTime        time.Time
	DeleteWithInstance bool
}

func (disk *SDisk) GetId() string {
	return disk.VolumeId
}

func (disk *SDisk) Delete(ctx context.Context) error {
	_, err := disk.storage.zone.region.getDisk(disk.VolumeId)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "Failed to find disk %s when delete", disk.VolumeId)
	}

	for {
		err := disk.storage.zone.region.DeleteDisk(disk.VolumeId)
		if err != nil {
			if isError(err, "IncorrectDiskStatus") {
				log.Infof("The disk is initializing, try later ...")
				time.Sleep(10 * time.Second)
			} else {
				return errors.Wrapf(err, "DeleteDisk fail")
			}
		} else {
			break
		}
	}
	return cloudprovider.WaitDeleted(disk, 10*time.Second, 300*time.Second) // 5minutes
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
	new, err := disk.storage.zone.region.getDisk(disk.VolumeId)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, new)
}

func (disk *SDisk) ResizeDisk(newSize int64) error {
	// newSize 单位为 GB. 只能扩容，不能缩减。范围参考下面链接。
	// https://www.volcengine.com/docs/6396/76561
	return disk.storage.zone.region.ResizeDisk(disk.VolumeId, newSize)
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

func (disk *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshot")
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshots")
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", disk.storage.zone.region.resetDisk(disk.VolumeId, snapshotId)
}

func (disk *SDisk) GetBillingType() string {
	return convertChargeType(disk.BillingType)
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreatedAt
}

func (disk *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	return nil, errors.ErrNotImplemented
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
	return ""
}

// Snapshot API is not supported, refer to
// https://www.volcengine.com/docs/6460/195549
func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshotId, err := disk.storage.zone.region.CreateSnapshot(disk.VolumeId, name, desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSnapshot")
	}
	snapshot, err := disk.storage.zone.region.GetISnapshotById(snapshotId)
	if err != nil {
		return nil, errors.Wrapf(err, "getSnapshot(%s)", snapshotId)
	}
	err = cloudprovider.WaitStatus(snapshot, api.SNAPSHOT_READY, 15*time.Second, 3600*time.Second)
	if err != nil {
		return nil, errors.Wrapf(err, "cloudprovider.WaitStatus")
	}
	return snapshot, nil
}

// region
func (region *SRegion) GetDisks(instanceId string, zoneId string, category string, diskIds []string, pageNumber int, pageSize int) ([]SDisk, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}
	if len(category) > 0 {
		params["VolumeType"] = category
	}
	if len(diskIds) > 0 {
		for index, id := range diskIds {
			key := fmt.Sprintf("VolumeIds.%d", index+1)
			params[key] = id
		}
	}

	body, err := region.storageRequest("DescribeVolumes", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "GetDisks fail")
	}

	disks := make([]SDisk, 0)
	err = body.Unmarshal(&disks, "Result", "Volumes")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal disk details fail")
	}
	total, _ := body.Int("Result", "TotalCount")
	return disks, int(total), nil
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
	return body.GetString("Result", "VolumeId")
}

func (region *SRegion) getDisk(diskId string) (*SDisk, error) {
	disks, _, err := region.GetDisks("", "", "", []string{diskId}, 1, 50)
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

func (region *SRegion) resetDisk(diskId, snapshotId string) error {
	// not supported API
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "resetDisk")
}

func (region *SRegion) CreateSnapshot(diskId, name, desc string) (string, error) {
	return "", errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateSnapshot")
}
