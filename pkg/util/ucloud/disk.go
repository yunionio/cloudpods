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

package ucloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://docs.ucloud.cn/api/udisk-api/describe_udisk
type SDisk struct {
	storage *SStorage
	multicloud.SDisk

	Status        string `json:"Status"`
	DeviceName    string `json:"DeviceName"`
	UHostID       string `json:"UHostId"`
	Tag           string `json:"Tag"`
	Version       string `json:"Version"`
	Name          string `json:"Name"`
	Zone          string `json:"Zone"`
	UHostIP       string `json:"UHostIP"`
	DiskType      string `json:"DiskType"`
	UDataArkMode  string `json:"UDataArkMode"`
	SnapshotLimit int    `json:"SnapshotLimit"`
	ExpiredTime   int64  `json:"ExpiredTime"`
	SnapshotCount int    `json:"SnapshotCount"`
	IsExpire      string `json:"IsExpire"`
	UDiskID       string `json:"UDiskId"`
	ChargeType    string `json:"ChargeType"`
	UHostName     string `json:"UHostName"`
	CreateTime    int64  `json:"CreateTime"`
	SizeGB        int    `json:"Size"`
}

func (self *SDisk) GetProjectId() string {
	return self.storage.zone.region.client.projectId
}

func (self *SDisk) GetId() string {
	return self.UDiskID
}

func (self *SDisk) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) GetStatus() string {
	switch self.Status {
	case "Available":
		return api.DISK_READY
	case "Attaching":
		return api.DISK_ATTACHING
	case "InUse":
		return api.DISK_READY
	case "Detaching":
		return api.DISK_DETACHING
	case "Initializating":
		return api.DISK_ALLOCATING
	case "Failed":
		return api.DISK_ALLOC_FAILED
	case "Cloning":
		return api.DISK_CLONING
	case "Restoring":
		return api.DISK_RESET
	case "RestoreFailed":
		return api.DISK_RESET_FAILED
	default:
		return api.DISK_UNKNOWN
	}
}

func (self *SDisk) Refresh() error {
	new, err := self.storage.zone.region.GetDisk(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	// todo: add price key
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(api.HYPERVISOR_UCLOUD), "hypervisor")

	return data
}

// Year,Month,Dynamic,Trial
func (self *SDisk) GetBillingType() string {
	switch self.ChargeType {
	case "Year", "Month":
		return billing_api.BILLING_TYPE_PREPAID
	default:
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SDisk) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime, 0)
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Unix(self.ExpiredTime, 0)
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.SizeGB * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	if self.DiskType == "SystemDisk" {
		return true
	}

	return false
}

func (self *SDisk) GetTemplateId() string {
	if strings.Contains(self.DiskType, "SystemDisk") && len(self.UHostID) > 0 {
		ins, err := self.storage.zone.region.GetInstanceByID(self.UHostID)
		if err != nil {
			log.Errorln(err)
		}

		return ins.ImageID
	}

	return ""
}

func (self *SDisk) GetDiskType() string {
	if strings.Contains(self.DiskType, "SystemDisk") {
		return api.DISK_TYPE_SYS
	}

	return api.DISK_TYPE_DATA
}

func (self *SDisk) GetStorageType() string {
	if self.storage == nil {
		if strings.Contains(self.DiskType, "SSD") {
			return api.STORAGE_UCLOUD_CLOUD_SSD
		} else {
			return api.STORAGE_UCLOUD_CLOUD_NORMAL
		}
	}

	return self.storage.storageType
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

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.DeleteDisk(self.Zone, self.GetId())
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.storage.zone.region.CreateSnapshot(self.Zone, self.GetId(), name, desc)
	if err != nil {
		return nil, err
	}

	isnapshot, err := self.GetISnapshot(snapshot)
	if err != nil {
		return nil, err
	}

	err = cloudprovider.WaitStatus(isnapshot, api.SNAPSHOT_READY, time.Second*10, time.Second*300)
	if err != nil {
		return nil, errors.Wrap(err, "CreateISnapshot")
	}

	return isnapshot, nil
}

func (self *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshotById(self.Zone, snapshotId)
	return &snapshot, err
}

func (self *SDisk) GetISnapshot(idStr string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.getSnapshot(idStr)
	return snapshot, err
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.GetSnapshots("", self.GetId(), "")
	if err != nil {
		return nil, err
	}

	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	var sizeGB int64
	// 向上取整
	if (newSizeMB % 1024) > 0 {
		sizeGB = newSizeMB/1024 + 1
	} else {
		sizeGB = newSizeMB / 1024
	}

	if self.Status == "InUse" {
		err := self.storage.zone.region.DetachDisk(self.Zone, self.UHostID, self.UDiskID)
		if err != nil {
			return err
		}

		defer self.storage.zone.region.AttachDisk(self.Zone, self.UHostID, self.UDiskID)
		err = cloudprovider.WaitStatusWithDelay(self, api.DISK_READY, 10*time.Second, 5*time.Second, 60*time.Second)
		if err != nil {
			return errors.Wrap(err, "DiskResize")
		}

	}
	return self.storage.zone.region.resizeDisk(self.Zone, self.GetId(), sizeGB)
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	err := self.storage.zone.region.resetDisk(self.Zone, self.GetId(), snapshotId)
	if err != nil {
		return "", err
	}

	return self.GetId(), nil
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return self.storage.zone.region.resetDisk(self.Zone, self.GetId(), "")
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		return nil, fmt.Errorf("GetDisk id should not empty")
	}

	disks, err := self.GetDisks("", "", []string{diskId})
	if err != nil {
		return nil, err
	}

	if len(disks) == 1 {
		return &disks[0], nil
	} else if len(disks) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, fmt.Errorf("GetDisk %s %d found", diskId, len(disks))
	}
}

// https://docs.ucloud.cn/api/udisk-api/describe_udisk
// diskType DataDisk|SystemDisk (DataDisk表示数据盘，SystemDisk表示系统盘)
func (self *SRegion) GetDisks(zoneId string, diskType string, diskIds []string) ([]SDisk, error) {
	disks := make([]SDisk, 0)
	params := NewUcloudParams()
	if len(zoneId) > 0 {
		params.Set("Zone", zoneId)
	}

	if len(diskType) > 0 {
		params.Set("DiskType", diskType)
	}

	err := self.DoListAll("DescribeUDisk", params, &disks)
	if err != nil {
		return nil, err
	}

	if len(diskIds) > 0 {
		filtedDisks := make([]SDisk, 0)
		for i := range disks {
			if utils.IsInStringArray(disks[i].UDiskID, diskIds) {
				filtedDisks = append(filtedDisks, disks[i])
			}
		}

		return filtedDisks, nil
	}

	return disks, nil
}

// https://docs.ucloud.cn/api/udisk-api/delete_udisk
func (self *SRegion) DeleteDisk(zoneId string, diskId string) error {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("UDiskId", diskId)

	return self.DoAction("DeleteUDisk", params, nil)
}

// https://docs.ucloud.cn/api/udisk-api/create_udisk
func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int) (string, error) {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("Size", sizeGb)
	params.Set("Name", name)
	params.Set("DiskType", category)

	diskIds := make([]string, 0)
	err := self.DoAction("CreateUDisk", params, &diskIds)
	if err != nil {
		return "", err
	}

	if len(diskIds) == 0 {
		return "", fmt.Errorf("CreateDisk with empty response")
	}

	return diskIds[0], nil
}

// https://docs.ucloud.cn/api/udisk-api/create_udisk_snapshot
func (self *SRegion) CreateSnapshot(zoneId, diskId, name, desc string) (string, error) {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("UDiskId", diskId)
	params.Set("Name", name)
	params.Set("Comment", desc)

	snapshotIds := make([]string, 0)
	err := self.DoAction("CreateUDiskSnapshot", params, &snapshotIds)
	if err != nil {
		return "", err
	}

	if len(snapshotIds) == 0 {
		return "", fmt.Errorf("CreateSnapshot with empty response")
	}

	return snapshotIds[0], nil
}

// https://docs.ucloud.cn/api/udisk-api/resize_udisk
func (self *SRegion) resizeDisk(zoneId string, diskId string, sizeGB int64) error {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("Size", sizeGB)
	params.Set("UDiskId", diskId)

	return self.DoAction("ResizeUDisk", params, nil)
}

// https://docs.ucloud.cn/api/udisk-api/restore_u_disk
func (self *SRegion) resetDisk(zoneId, diskId, snapshotId string) error {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("UDiskId", diskId)
	if len(snapshotId) > 0 {
		params.Set("SnapshotId", snapshotId)
	}

	return self.DoAction("RestoreUDisk", params, nil)
}

// https://docs.ucloud.cn/api/udisk-api/attach_udisk
func (self *SRegion) AttachDisk(zoneId string, instanceId string, diskId string) error {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("UHostId", instanceId)
	params.Set("UDiskId", diskId)

	return self.DoAction("AttachUDisk", params, nil)
}

// https://docs.ucloud.cn/api/udisk-api/detach_udisk
func (self *SRegion) DetachDisk(zoneId string, instanceId string, diskId string) error {
	idisks, err := self.GetDisk(diskId)
	if err != nil {
		return err
	}

	if idisks.Status == "Available" {
		return nil
	}

	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("UHostId", instanceId)
	params.Set("UDiskId", diskId)

	return self.DoAction("DetachUDisk", params, nil)
}
