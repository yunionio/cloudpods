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

package ksyun

import (
	"context"
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type Attachment struct {
	InstanceID         string `json:"InstanceId"`
	MountPoint         string `json:"MountPoint"`
	DeleteWithInstance bool   `json:"DeleteWithInstance"`
}

type HistoryAttachment struct {
	InstanceID string `json:"InstanceId"`
	AttachTime string `json:"AttachTime"`
	DetachTime string `json:"DetachTime"`
	MountPoint string `json:"MountPoint"`
}

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	SKsyunTags

	VolumeId           string              `json:"VolumeId"`
	VolumeName         string              `json:"VolumeName"`
	VolumeDesc         string              `json:"VolumeDesc,omitempty"`
	Size               int                 `json:"Size"`
	VolumeStatus       string              `json:"VolumeStatus"`
	VolumeType         string              `json:"VolumeType"`
	VolumeCategory     string              `json:"VolumeCategory"`
	InstanceId         string              `json:"InstanceId"`
	AvailabilityZone   string              `json:"AvailabilityZone"`
	ChargeType         string              `json:"ChargeType"`
	InstanceTradeType  int                 `json:"InstanceTradeType"`
	CreateTime         string              `json:"CreateTime"`
	Attachment         []Attachment        `json:"Attachment"`
	ProjectId          string              `json:"ProjectId"`
	ExpireTime         string              `json:"ExpireTime,omitempty"`
	HistoryAttachment  []HistoryAttachment `json:"HistoryAttachment,omitempty"`
	DeleteWithInstance bool                `json:"DeleteWithInstance"`
}

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, err := region.GetDisks([]string{diskId}, "", "")
	if err != nil {
		return nil, err
	}
	for i := range disks {
		if disks[i].VolumeId == diskId {
			return &disks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "disk %s", diskId)
}

func (region *SRegion) GetDisks(diskIds []string, storageType, zoneId string) ([]SDisk, error) {
	disks := []SDisk{}
	params := map[string]interface{}{
		"MaxResults": "1000",
	}
	for i, v := range diskIds {
		params[fmt.Sprintf("VolumeId.%d", i+1)] = v
	}
	if len(storageType) > 0 {
		params["VolumeType"] = storageType
	}
	for {
		resp, err := region.ebsRequest("DescribeVolumes", params)
		if err != nil {
			return nil, errors.Wrap(err, "list instance")
		}
		part := struct {
			RequestID  string  `json:"RequestId"`
			Volumes    []SDisk `json:"Volumes"`
			TotalCount int     `json:"TotalCount"`
			Marker     int     `json:"Marker"`
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal instances")
		}
		disks = append(disks, part.Volumes...)
		if len(disks) >= part.TotalCount {
			break
		}
		params["Marker"] = fmt.Sprintf("%d", part.Marker)
	}
	if len(zoneId) > 0 {
		res := []SDisk{}
		for _, disk := range disks {
			if disk.AvailabilityZone == zoneId {
				res = append(res, disk)
			}
		}
		return res, nil
	}
	return disks, nil
}

func (region *SRegion) GetDiskByInstanceId(instanceId string) ([]SDisk, error) {
	params := map[string]interface{}{
		"MaxResults": "1000",
	}
	params["InstanceId"] = instanceId
	resp, err := region.ebsRequest("DescribeInstanceVolumes", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeInstanceVolumes")
	}
	ret := struct {
		Attachments []SDisk `json:"Attachments"`
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal Attachments")
	}
	return ret.Attachments, nil
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	if disk.storage == nil {
		return nil, fmt.Errorf("disk %s(%s) missing storage", disk.VolumeName, disk.VolumeId)
	}
	return disk.storage, nil
}

func (disk *SDisk) GetIStorageId() string {
	if disk.storage == nil {
		return ""
	}
	return disk.storage.GetGlobalId()
}

func (disk *SDisk) GetDiskFormat() string {
	return ""
}

func (disk *SDisk) GetId() string {
	return disk.VolumeId
}

func (disk *SDisk) GetTags() (map[string]string, error) {
	tags, err := disk.storage.zone.region.ListTags("volume", disk.VolumeId)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (disk *SDisk) GetGlobalId() string {
	return disk.VolumeId
}

func (disk *SDisk) GetName() string {
	return disk.VolumeName
}

func (disk *SDisk) GetProjectId() string {
	return disk.ProjectId
}

func (disk *SDisk) Refresh() error {
	if disk.VolumeType == api.STORAGE_KSYUN_LOCAL_SSD {
		return nil
	}
	disk, err := disk.storage.zone.region.GetDisk(disk.VolumeId)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, disk)
}

func (disk *SDisk) GetStatus() string {
	if disk.VolumeType == api.STORAGE_KSYUN_LOCAL_SSD {
		return api.DISK_READY
	}
	// creating、available、attaching、inuse、detaching、extending、deleting、error
	switch disk.VolumeStatus {
	case "available", "inuse", "in-use":
		return api.DISK_READY
	case "detaching":
		return api.DISK_DETACHING
	case "error":
		return api.DISK_UNKNOWN
	case "creating":
		return api.DISK_ALLOCATING
	default:
		return disk.VolumeStatus
	}
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.Size * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return disk.DeleteWithInstance
}

func (disk *SDisk) GetTemplateId() string {
	return ""
}

func (disk *SDisk) GetDiskType() string {
	if disk.VolumeCategory == "system" {
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

func (disk *SDisk) Delete(ctx context.Context) error {
	return disk.storage.zone.region.DeleteDisk(disk.VolumeId)
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	ret, err := disk.storage.zone.region.CreateSnapshot(disk.VolumeId, name, desc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots("", disk.VolumeId)
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
	return nil, cloudprovider.ErrNotSupported
}

func (disk *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return disk.storage.zone.region.ResizeDisk(disk.VolumeId, newSizeMB)
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return disk.VolumeId, disk.storage.zone.region.ResetDisk(disk.VolumeId, snapshotId)
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) DeleteDisk(id string) error {
	_, err := region.ebsRequest("DeleteVolume", map[string]interface{}{
		"VolumeId":    id,
		"ForceDelete": "true",
	})
	return err
}

func (region *SRegion) ResizeDisk(id string, newSizeMB int64) error {
	_, err := region.ebsRequest("ResizeVolume", map[string]interface{}{
		"VolumeId": id,
		"Size":     fmt.Sprintf("%d", newSizeMB/1024),
	})
	return err
}

func (region *SRegion) ResetDisk(id string, snapshotId string) error {
	_, err := region.ebsRequest("ResetVolume", map[string]interface{}{
		"VolumeId":   id,
		"SnapshotId": snapshotId,
	})
	return err
}

func (region *SRegion) CreateDisk(storageType, zoneId string, opts *cloudprovider.DiskCreateConfig) (*SDisk, error) {
	params := map[string]interface{}{
		"VolumeName":       opts.Name,
		"VolumeDesc":       opts.Desc,
		"Size":             fmt.Sprintf("%d", opts.SizeGb),
		"VolumeType":       storageType,
		"AvailabilityZone": zoneId,
		"ChargeType":       "HourlyInstantSettlement",
	}
	if len(opts.ProjectId) > 0 {
		params["ProjectId"] = opts.ProjectId
	}
	if len(opts.SnapshotId) > 0 {
		params["SnapshotId"] = opts.SnapshotId
	}
	resp, err := region.ebsRequest("CreateVolume", params)
	if err != nil {
		return nil, err
	}
	diskId, err := resp.GetString("VolumeId")
	if err != nil {
		return nil, err
	}
	return region.GetDisk(diskId)
}
