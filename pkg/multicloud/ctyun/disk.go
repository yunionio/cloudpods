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

package ctyun

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// http://ctyun-api-url/apiproxy/v3/ondemand/queryVolumes
type SDisk struct {
	storage *SStorage
	multicloud.SDisk

	diskDetails *DiskDetails

	ID                  string       `json:"id"`
	Status              string       `json:"status"`
	Name                string       `json:"name"`
	CreatedAt           int64        `json:"created_at"`
	UpdatedAt           string       `json:"updated_at"`
	Multiattach         bool         `json:"multiattach"`
	ReplicationStatus   string       `json:"replication_status"`
	SizeGB              int64        `json:"size"`
	Metadata            Metadata     `json:"metadata"`
	VolumeType          string       `json:"volume_type"`
	UserID              string       `json:"user_id"`
	Shareable           bool         `json:"shareable"`
	Encrypted           bool         `json:"encrypted"`
	Bootable            string       `json:"bootable"`
	AvailabilityZone    string       `json:"availability_zone"`
	Attachments         []Attachment `json:"attachments"`
	MasterOrderID       string       `json:"masterOrderId"`
	WorkOrderResourceID string       `json:"workOrderResourceId"`
	ExpireTime          int64        `json:"expireTime"`
	IsFreeze            int64        `json:"isFreeze"`
}

type Attachment struct {
	VolumeID     string `json:"volume_id"`
	AttachmentID string `json:"attachment_id"`
	AttachedAt   string `json:"attached_at"`
	ServerID     string `json:"server_id"`
	Device       string `json:"device"`
	ID           string `json:"id"`
}

type Metadata struct {
	OrderID          string `json:"orderID"`
	AttachedMode     string `json:"attached_mode"`
	ResourceSpecCode string `json:"resourceSpecCode"`
	ProductID        string `json:"productID"`
	Readonly         string `json:"readonly"`
}

type DiskDetails struct {
	ID                  string `json:"id"`
	ResEbsID            string `json:"resEbsId"`
	Size                int64  `json:"size"`
	Name                string `json:"name"`
	RegionID            string `json:"regionId"`
	AccountID           string `json:"accountId"`
	UserID              string `json:"userId"`
	HostID              string `json:"hostId"`
	OrderID             string `json:"orderId"`
	Status              int64  `json:"status"`
	Type                string `json:"type"`
	VolumeStatus        int64  `json:"volumeStatus"`
	CreateDate          int64  `json:"createDate"`
	DueDate             int64  `json:"dueDate"`
	ZoneID              string `json:"zoneId"`
	ZoneName            string `json:"zoneName"`
	IsSysVolume         int64  `json:"isSysVolume"`
	IsPackaged          int64  `json:"isPackaged"`
	WorkOrderResourceID string `json:"workOrderResourceId"`
	IsFreeze            int64  `json:"isFreeze"`
}

func (self *SDisk) GetBillingType() string {
	if self.ExpireTime > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	}

	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return time.Unix(self.CreatedAt/1000, 0)
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Unix(self.ExpireTime/1000, 0)
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SDisk) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}

	return self.ID
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) GetStatus() string {
	switch self.Status {
	case "creating", "downloading":
		return api.DISK_ALLOCATING
	case "available", "in-use":
		return api.DISK_READY
	case "error":
		return api.DISK_ALLOC_FAILED
	case "attaching":
		return api.DISK_ATTACHING
	case "detaching":
		return api.DISK_DETACHING
	case "restoring-backup":
		return api.DISK_REBUILD
	case "backing-up":
		return api.DISK_BACKUP_STARTALLOC
	case "error_restoring":
		return api.DISK_BACKUP_ALLOC_FAILED
	case "uploading":
		return api.DISK_SAVING
	case "extending":
		return api.DISK_RESIZING
	case "error_extending":
		return api.DISK_ALLOC_FAILED
	case "deleting":
		return api.DISK_DEALLOC
	case "error_deleting":
		return api.DISK_DEALLOC_FAILED
	case "rollbacking":
		return api.DISK_REBUILD
	case "error_rollbacking":
		return api.DISK_UNKNOWN
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
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(api.HYPERVISOR_CTYUN), "hypervisor")

	return data
}

func (self *SDisk) GetProjectId() string {
	return ""
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetIStorageId() string {
	return self.storage.GetId()
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.SizeGB * 1024)
}

func (self *SDisk) GetIsAutoDelete() bool {
	if len(self.Attachments) == 0 {
		return false
	}

	if self.Bootable == "true" {
		return true
	}

	return false
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetDiskType() string {
	if self.Bootable == "true" {
		return api.DISK_TYPE_SYS
	} else {
		return api.DISK_TYPE_DATA
	}
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
	if len(self.Attachments) > 0 {
		return self.Attachments[0].Device
	}

	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetISnapshot(idStr string) (cloudprovider.ICloudSnapshot, error) {
	return self.storage.zone.region.GetSnapshot(self.GetId(), idStr)
}

// GET http://ctyun-api-url/apiproxy/v3/ondemand/queryVBSs
func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.GetSnapshots(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SDisk.GetISnapshots")
	}

	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		isnapshots[i] = &snapshots[i]
	}

	return isnapshots, nil
}

// POST http://ctyun-api-url/apiproxy/v3/ondemand/updateDiskBackupPolicy
func (self *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	return []string{}, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetDiskDetails() (*DiskDetails, error) {
	if self.diskDetails != nil {
		return self.diskDetails, nil
	}

	details, err := self.storage.zone.region.GetDiskDetailByDiskId(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "SDisk.GetDiskDetails.GetDiskDetailByDiskId")
	}

	self.diskDetails = details
	return self.diskDetails, nil
}

func (self *SRegion) GetDisks() ([]SDisk, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVolumes", params)
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetDisks.DoGet")
	}

	disks := make([]SDisk, 0)
	err = resp.Unmarshal(&disks, "returnObj", "volumes")
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetDisks.Unmarshal")
	}

	for i := range disks {
		izone, err := self.GetIZoneById(disks[i].AvailabilityZone)
		if err != nil {
			return nil, errors.Wrap(err, "SRegion.GetDisk.GetIZoneById")
		}

		disks[i].storage = &SStorage{
			zone:        izone.(*SZone),
			storageType: disks[i].VolumeType,
		}
	}

	return disks, nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"volumeId": diskId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVolumes", params)
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetDisks.DoGet")
	}

	disks := make([]SDisk, 0)
	err = resp.Unmarshal(&disks, "returnObj", "volumes")
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetDisks.Unmarshal")
	}

	if len(disks) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetDisk")
	} else if len(disks) == 1 {
		izone, err := self.GetIZoneById(disks[0].AvailabilityZone)
		if err != nil {
			return nil, errors.Wrap(err, "SRegion.GetDisk.GetIZoneById")
		}

		disks[0].storage = &SStorage{
			zone:        izone.(*SZone),
			storageType: disks[0].VolumeType,
		}

		return &disks[0], nil
	} else {
		return nil, errors.Wrap(cloudprovider.ErrDuplicateId, "SRegion.GetDisk")
	}
}

func (self *SRegion) GetDiskDetailByDiskId(diskId string) (*DiskDetails, error) {
	params := map[string]string{
		"volumeId": diskId,
		"regionId": self.GetId(),
	}

	resp, err := self.client.DoGet("/apiproxy/v3/queryDataDiskDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetDiskDetailByDiskId.DoGet")
	}

	disk := &DiskDetails{}
	err = resp.Unmarshal(disk, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetDiskDetailByDiskId.Unmarshal")
	}

	return disk, nil
}

func (self *SRegion) CreateDisk(zoneId, name, diskType, size string) (*SDisk, error) {
	diskParams := jsonutils.NewDict()
	diskParams.Set("regionId", jsonutils.NewString(self.GetId()))
	diskParams.Set("zoneId", jsonutils.NewString(zoneId))
	diskParams.Set("name", jsonutils.NewString(name))
	diskParams.Set("type", jsonutils.NewString(diskType))
	diskParams.Set("size", jsonutils.NewString(size))
	diskParams.Set("count", jsonutils.NewString("1"))

	params := map[string]jsonutils.JSONObject{
		"createVolumeInfo": diskParams,
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/createVolume", params)
	if err != nil {
		return nil, errors.Wrap(err, "Region.CreateDisk.DoPost")
	}

	disk := &SDisk{}
	err = resp.Unmarshal(disk)
	if err != nil {
		return nil, errors.Wrap(err, "Region.CreateDisk.Unmarshal")
	}

	izone, err := self.GetIZoneById(disk.AvailabilityZone)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateDisk.GetIZoneById")
	}

	disk.storage = &SStorage{
		zone:        izone.(*SZone),
		storageType: disk.VolumeType,
	}

	return disk, nil
}
