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

package baidu

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type Attachment struct {
	InstanceId         string
	MountPoint         string
	DeleteWithInstance bool
}

type Attachments struct {
	Id         string
	InstanceId string
	Device     string
	Serial     string
}

type AutoSnapshotPolicy struct {
	Id             string
	Name           string
	TimePoints     []int
	RepeatWeekdays []int
	RetentionDays  int
	Status         string
}

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	SBaiduTag

	Id                 string
	CreateTime         time.Time
	ExpireTime         string
	Name               string
	DiskSizeInGB       int
	Status             string
	Type               string
	StorageType        string
	Desc               string
	PaymentTiming      string
	Attachments        []Attachments
	RegionId           string
	SourceSnapshotId   string
	SnapshotNum        string
	AutoSnapshotPolicy AutoSnapshotPolicy
	ZoneName           string
	IsSystemVolume     bool
}

func (region *SRegion) GetDisks(storageType, zoneName, instanceId string) ([]SDisk, error) {
	params := url.Values{}
	if len(zoneName) > 0 {
		params.Set("zoneName", zoneName)
	}
	if len(instanceId) > 0 {
		params.Set("instanceId", instanceId)
	}
	disks := []SDisk{}
	for {
		resp, err := region.bccList("v2/volume", params)
		if err != nil {
			return nil, errors.Wrap(err, "list disks")
		}
		part := struct {
			NextMarker string
			Volumes    []SDisk
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		for i := range part.Volumes {
			if len(storageType) == 0 || isMatchStorageType(storageType, part.Volumes[i].StorageType) {
				disks = append(disks, part.Volumes[i])
			}
		}
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return disks, nil
}

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	resp, err := region.bccList(fmt.Sprintf("v2/volume/%s", diskId), nil)
	if err != nil {
		return nil, errors.Wrap(err, "list disks")
	}
	ret := &SDisk{}
	err = resp.Unmarshal(ret, "volume")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal disks")
	}
	return ret, nil
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

func (disk *SDisk) GetId() string {
	return disk.Id
}

func (disk *SDisk) GetGlobalId() string {
	return disk.Id
}

func (disk *SDisk) GetName() string {
	return disk.Name
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case "Available", "InUse", "Recharging":
		return api.DISK_READY
	case "Detaching":
		return api.DISK_DETACHING
	case "Error", "NotAvailable", "Expired":
		return api.DISK_UNKNOWN
	case "Creating":
		return api.DISK_ALLOCATING
	case "Attaching":
		return api.DISK_ATTACHING
	case "Deleting":
		return api.DISK_DETACHING
	case "Scaling":
		return api.DISK_RESIZING
	case "SnapshotProcessing", "ImageProcessing":
		return api.DISK_SAVING
	default:
		return strings.ToLower(disk.Status)
	}
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.DiskSizeInGB * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return false
}

func (disk *SDisk) GetTemplateId() string {
	return ""
}

func (disk *SDisk) GetDiskType() string {
	if disk.IsSystemVolume {
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

func (disk *SDisk) GetIops() int {
	return 0
}

func (disk *SDisk) GetDriver() string {
	return ""
}

func (disk *SDisk) GetCacheMode() string {
	return ""
}

func (disk *SDisk) GetMountpoint() string {
	return ""
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (disk *SDisk) Refresh() error {
	res, err := disk.storage.zone.region.GetDisk(disk.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, res)
}

func (disk *SDisk) Delete(ctx context.Context) error {
	return disk.storage.zone.region.DeleteDisk(disk.Id)
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := disk.storage.zone.region.CreateSnapshot(name, desc, disk.Id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots(disk.Id)
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

func (disk *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	ret := []string{}
	if len(disk.AutoSnapshotPolicy.Id) > 0 {
		ret = append(ret, disk.AutoSnapshotPolicy.Id)
	}
	return ret, nil
}

func (disk *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return disk.storage.zone.region.ResizeDisk(disk.Id, newSizeMB/1024)
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return disk.Id, disk.storage.zone.region.ResetDisk(disk.Id, snapshotId)
}

func (self *SRegion) ResetDisk(diskId, snapshotId string) error {
	params := url.Values{}
	params.Set("rollback", "")
	body := map[string]interface{}{
		"snapshotId": snapshotId,
	}
	_, err := self.bccUpdate(fmt.Sprintf("v2/volume/%s", diskId), params, body)
	return err
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) DeleteDisk(id string) error {
	_, err := region.bccDelete(fmt.Sprintf("v2/volume/%s", id), nil)
	return err
}

func (region *SRegion) CreateDisk(storageType, zoneName string, opts *cloudprovider.DiskCreateConfig) (*SDisk, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	body := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Desc,
		"cdsSizeInGB": opts.SizeGb,
		"storageType": storageType,
		"zoneName":    zoneName,
	}
	resp, err := region.bccPost("v2/volume", params, body)
	if err != nil {
		return nil, err
	}
	ret := struct {
		VolumeIds []string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	for _, id := range ret.VolumeIds {
		return region.GetDisk(id)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, resp.String())
}

func (region *SRegion) ResizeDisk(diskId string, sizeGb int64) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	params.Set("resize", "")
	body := map[string]interface{}{
		"newCdsSizeInGB": sizeGb,
	}
	_, err := region.bccUpdate(fmt.Sprintf("v2/volume/%s", diskId), params, body)
	return err
}
