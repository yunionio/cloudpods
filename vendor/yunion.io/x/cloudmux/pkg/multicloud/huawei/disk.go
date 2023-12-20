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

package huawei

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

/*
华为云云硬盘
======创建==========
1.磁盘只能挂载到同一可用区的云服务器内，创建后不支持更换可用区
2.计费模式 包年包月/按需计费
3.*支持自动备份


共享盘 和 普通盘：https://support.huaweicloud.com/productdesc-evs/zh-cn_topic_0032860759.html
根据是否支持挂载至多台云服务器可以将云硬盘分为非共享云硬盘和共享云硬盘。
一个非共享云硬盘只能挂载至一台云服务器，而一个共享云硬盘可以同时挂载至多台云服务器。
单个共享云硬盘最多可同时挂载给16个云服务器。目前，共享云硬盘只适用于数据盘，不支持系统盘。
*/

type Attachment struct {
	ServerID     string `json:"server_id"`
	AttachmentID string `json:"attachment_id"`
	AttachedAt   string `json:"attached_at"`
	HostName     string `json:"host_name"`
	VolumeID     string `json:"volume_id"`
	Device       string `json:"device"`
	ID           string `json:"id"`
}

type DiskMeta struct {
	ResourceSpecCode string `json:"resourceSpecCode"`
	Billing          string `json:"billing"`
	ResourceType     string `json:"resourceType"`
	AttachedMode     string `json:"attached_mode"`
	Readonly         string `json:"readonly"`
	OrderId          string `json:"orderID"`
}

type VolumeImageMetadata struct {
	QuickStart             string `json:"__quick_start"`
	ContainerFormat        string `json:"container_format"`
	MinRAM                 string `json:"min_ram"`
	ImageName              string `json:"image_name"`
	ImageID                string `json:"image_id"`
	OSType                 string `json:"__os_type"`
	OSFeatureList          string `json:"__os_feature_list"`
	MinDisk                string `json:"min_disk"`
	SupportKVM             string `json:"__support_kvm"`
	VirtualEnvType         string `json:"virtual_env_type"`
	SizeGB                 string `json:"size"`
	OSVersion              string `json:"__os_version"`
	OSBit                  string `json:"__os_bit"`
	SupportKVMHi1822Hiovs  string `json:"__support_kvm_hi1822_hiovs"`
	SupportXen             string `json:"__support_xen"`
	Description            string `json:"__description"`
	Imagetype              string `json:"__imagetype"`
	DiskFormat             string `json:"disk_format"`
	ImageSourceType        string `json:"__image_source_type"`
	Checksum               string `json:"checksum"`
	Isregistered           string `json:"__isregistered"`
	HwVifMultiqueueEnabled string `json:"hw_vif_multiqueue_enabled"`
	Platform               string `json:"__platform"`
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0124881427.html
type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	HuaweiDiskTags

	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Status              string              `json:"status"`
	Attachments         []Attachment        `json:"attachments"`
	Description         string              `json:"description"`
	SizeGB              int                 `json:"size"`
	Metadata            DiskMeta            `json:"metadata"`
	Encrypted           bool                `json:"encrypted"`
	Bootable            string              `json:"bootable"`
	Multiattach         bool                `json:"multiattach"`
	AvailabilityZone    string              `json:"availability_zone"`
	SourceVolid         string              `json:"source_volid"`
	SnapshotID          string              `json:"snapshot_id"`
	CreatedAt           time.Time           `json:"created_at"`
	VolumeType          string              `json:"volume_type"`
	VolumeImageMetadata VolumeImageMetadata `json:"volume_image_metadata"`
	ReplicationStatus   string              `json:"replication_status"`
	UserID              string              `json:"user_id"`
	ConsistencygroupID  string              `json:"consistencygroup_id"`
	UpdatedAt           string              `json:"updated_at"`
	EnterpriseProjectId string

	ExpiredTime time.Time
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SDisk) GetName() string {
	if len(self.Name) == 0 {
		return self.ID
	}

	return self.Name
}

func (self *SDisk) GetGlobalId() string {
	return self.ID
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
		return api.DISK_BACKUP_STARTALLOC // ?
	case "error_restoring":
		return api.DISK_BACKUP_ALLOC_FAILED
	case "uploading":
		return api.DISK_SAVING //?
	case "extending":
		return api.DISK_RESIZING
	case "error_extending":
		return api.DISK_ALLOC_FAILED // ?
	case "deleting":
		return api.DISK_DEALLOC //?
	case "error_deleting":
		return api.DISK_DEALLOC_FAILED // ?
	case "rollbacking":
		return api.DISK_REBUILD
	case "error_rollbacking":
		return api.DISK_UNKNOWN
	default:
		return api.DISK_UNKNOWN
	}
}

func (self *SDisk) Refresh() error {
	disk, err := self.storage.zone.region.GetDisk(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, disk)
}

func (self *SDisk) GetBillingType() string {
	if len(self.Metadata.OrderId) > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SDisk) GetExpiredAt() time.Time {
	orders, err := self.storage.zone.region.client.GetOrderResources()
	if err != nil {
		return time.Time{}
	}
	order, ok := orders[self.ID]
	if ok {
		return order.ExpireTime
	}
	return time.Time{}
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.SizeGB * 1024)
}

func (self *SDisk) checkAutoDelete(attachments []Attachment) bool {
	autodelete := false
	for _, attach := range attachments {
		if len(attach.ServerID) > 0 {
			// todo : 忽略错误？？
			vm, err := self.storage.zone.region.GetInstance(attach.ServerID)
			if err != nil {
				volumes := vm.OSExtendedVolumesVolumesAttached
				for _, vol := range volumes {
					if vol.ID == self.ID && strings.ToLower(vol.DeleteOnTermination) == "true" {
						autodelete = true
					}
				}
			}

			break
		}
	}

	return autodelete
}

func (self *SDisk) GetIsAutoDelete() bool {
	if len(self.Attachments) > 0 {
		return self.checkAutoDelete(self.Attachments)
	}

	return false
}

func (self *SDisk) GetTemplateId() string {
	return self.VolumeImageMetadata.ImageID
}

func (self *SDisk) GetDiskType() string {
	if self.Bootable == "true" {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
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

func (self *SDisk) GetMountServerId() string {
	if len(self.Attachments) > 0 {
		return self.Attachments[0].ServerID
	}

	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	disk, err := self.storage.zone.region.GetDisk(self.GetId())
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return err
	}
	if disk.Status != "deleting" {
		// 等待硬盘ready
		cloudprovider.WaitStatus(self, api.DISK_READY, 5*time.Second, 60*time.Second)
		err := self.storage.zone.region.DeleteDisk(self.GetId())
		if err != nil {
			return err
		}
	}

	return cloudprovider.WaitDeleted(self, 10*time.Second, 120*time.Second)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.storage.zone.region.CreateSnapshot(self.GetId(), name, desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSnapshot")
	}
	return snapshot, nil
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshot(snapshotId)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.GetSnapshots(self.ID, "")
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
	sizeGb := newSizeMB / 1024
	err := self.storage.zone.region.ResizeDisk(self.GetId(), sizeGb)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatusWithDelay(self, api.DISK_READY, 15*time.Second, 5*time.Second, 60*time.Second)
}

func (self *SDisk) Detach() error {
	err := self.storage.zone.region.DetachDisk(self.GetMountServerId(), self.GetId())
	if err != nil {
		log.Debugf("detach server %s disk %s failed: %s", self.GetMountServerId(), self.GetId(), err)
		return err
	}

	return cloudprovider.WaitCreated(5*time.Second, 60*time.Second, func() bool {
		err := self.Refresh()
		if err != nil {
			log.Debugln(err)
			return false
		}

		if self.Status == "available" {
			return true
		}

		return false
	})
}

func (self *SDisk) Attach(device string) error {
	err := self.storage.zone.region.AttachDisk(self.GetMountServerId(), self.GetId(), device)
	if err != nil {
		return errors.Wrapf(err, "AttachDisk")
	}

	return cloudprovider.WaitStatusWithDelay(self, api.DISK_READY, 10*time.Second, 5*time.Second, 60*time.Second)
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EVS/doc?api=ShowVolume
func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	resp, err := self.list(SERVICE_EVS, "cloudvolumes/"+diskId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "show volume")
	}
	ret := &SDisk{}
	err = resp.Unmarshal(ret, "volume")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EVS/doc?api=ListVolumes
func (self *SRegion) GetDisks(zoneId string) ([]SDisk, error) {
	ret := []SDisk{}
	query := url.Values{}
	if len(zoneId) > 0 {
		query.Set("availability_zone", zoneId)
	}
	for {
		resp, err := self.list(SERVICE_EVS, "cloudvolumes/detail", query)
		if err != nil {
			return nil, errors.Wrapf(err, "list volumes")
		}
		part := struct {
			Volumes []SDisk
			Count   int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.Volumes...)
		if len(ret) >= part.Count || len(part.Volumes) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EVS/doc?api=CreateVolume
func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, snapshotId string, desc string, projectId string) (string, error) {
	params := map[string]interface{}{
		"name":              name,
		"availability_zone": zoneId,
		"description":       desc,
		"volume_type":       category,
		"size":              sizeGb,
	}
	if len(snapshotId) > 0 {
		params["snapshot_id"] = snapshotId
	}
	if len(projectId) > 0 {
		params["enterprise_project_id"] = projectId
	}

	resp, err := self.post(SERVICE_EVS_V2_1, "cloudvolumes", map[string]interface{}{"volume": params})
	if err != nil {
		return "", errors.Wrapf(err, "create volume")
	}

	ret := struct {
		JobId   string
		OrderId string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return "", errors.Wrapf(err, "Unmarshal")
	}

	id := ret.JobId + ret.OrderId

	// 按需计费
	volumeId, err := self.GetTaskEntityID(SERVICE_EVS_V1, id, "volume_id")
	if err != nil {
		return "", errors.Wrap(err, "GetAllSubTaskEntityIDs")
	}

	if len(volumeId) == 0 {
		return "", errors.Errorf("CreateInstance job %s result is emtpy", id)
	}
	return volumeId, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EVS/doc?api=DeleteVolume
func (self *SRegion) DeleteDisk(diskId string) error {
	res := fmt.Sprintf("cloudvolumes/%s", diskId)
	_, err := self.delete(SERVICE_EVS, res)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EVS/doc?api=ResizeVolume
func (self *SRegion) ResizeDisk(diskId string, sizeGB int64) error {
	params := map[string]interface{}{
		"os-extend": map[string]interface{}{
			"new_size": sizeGB,
		},
	}
	_, err := self.post(SERVICE_EVS_V2_1, fmt.Sprintf("cloudvolumes/%s/action", diskId), params)
	return err
}

func (self *SDisk) GetProjectId() string {
	return self.EnterpriseProjectId
}
