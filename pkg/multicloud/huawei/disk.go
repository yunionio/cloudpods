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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	details *SResourceDetail

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
	// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051803385.html
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
	data.Add(jsonutils.NewString(api.HYPERVISOR_HUAWEI), "hypervisor")

	return data
}

func (self *SDisk) getResourceDetails() *SResourceDetail {
	if self.details != nil {
		return self.details
	}

	res, err := self.storage.zone.region.GetOrderResourceDetail(self.GetId())
	if err != nil {
		log.Debugln(err)
		return nil
	}

	self.details = &res
	return self.details
}

func (self *SDisk) GetBillingType() string {
	details := self.getResourceDetails()
	if details == nil {
		return billing_api.BILLING_TYPE_POSTPAID
	} else {
		return billing_api.BILLING_TYPE_PREPAID
	}
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SDisk) GetExpiredAt() time.Time {
	var expiredTime time.Time
	details := self.getResourceDetails()
	if details != nil {
		expiredTime = details.ExpireTime
	}

	return expiredTime
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	// self.volume_type ?
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
			vm, err := self.storage.zone.region.GetInstanceByID(attach.ServerID)
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

// Bootable 表示硬盘是否为启动盘。
// 启动盘 != 系统盘(必须是启动盘且挂载在root device上)
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
	// https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762431.html
	// scsi or vbd?
	// todo: implement me
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
	if disk, err := self.storage.zone.region.GetDisk(self.GetId()); err == cloudprovider.ErrNotFound {
		log.Errorf("Failed to find disk %s when delete", self.GetId())
		return nil
	} else if disk.Status != "deleting" {
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
	if snapshotId, err := self.storage.zone.region.CreateSnapshot(self.GetId(), name, desc); err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	} else if snapshot, err := self.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = self.storage.zone.region
		if err := cloudprovider.WaitStatus(snapshot, api.SNAPSHOT_READY, 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return snapshot, nil
	}
}

func (self *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshotById(snapshotId)
	return &snapshot, err
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.getSnapshot(snapshotId)
	return snapshot, err
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
	err := cloudprovider.WaitStatus(self, api.DISK_READY, 5*time.Second, 60*time.Second)
	if err != nil {
		return err
	}

	sizeGb := newSizeMB / 1024
	err = self.storage.zone.region.resizeDisk(self.GetId(), sizeGb)
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
		log.Debugf("attach server %s disk %s failed: %s", self.GetMountServerId(), self.GetId(), err)
		return err
	}

	return cloudprovider.WaitStatusWithDelay(self, api.DISK_READY, 10*time.Second, 5*time.Second, 60*time.Second)
}

// 在线卸载磁盘 https://support.huaweicloud.com/usermanual-ecs/zh-cn_topic_0036046828.html
// 对于挂载在系统盘盘位（也就是“/dev/sda”或“/dev/vda”挂载点）上的磁盘，当前仅支持离线卸载
func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	mountpoint := self.GetMountpoint()
	if len(mountpoint) > 0 {
		err := self.Detach()
		if err != nil {
			return "", err
		}
	}

	diskId, err := self.storage.zone.region.resetDisk(self.GetId(), snapshotId)
	if err != nil {
		return diskId, err
	}

	err = cloudprovider.WaitStatus(self, api.DISK_READY, 5*time.Second, 300*time.Second)
	if err != nil {
		return "", err
	}

	if len(mountpoint) > 0 {
		err := self.Attach(mountpoint)
		if err != nil {
			return "", err
		}
	}

	return diskId, nil
}

// 华为云不支持重置
func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	var disk SDisk
	err := DoGet(self.ecsClient.Disks.Get, diskId, nil, &disk)
	return &disk, err
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762430.html
func (self *SRegion) GetDisks(zoneId string) ([]SDisk, error) {
	queries := map[string]string{}
	if len(zoneId) > 0 {
		queries["availability_zone"] = zoneId
	}

	disks := make([]SDisk, 0)
	err := doListAllWithOffset(self.ecsClient.Disks.List, queries, &disks)
	return disks, err
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762427.html
func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, snapshotId string, desc string) (string, error) {
	params := jsonutils.NewDict()
	volumeObj := jsonutils.NewDict()
	volumeObj.Add(jsonutils.NewString(name), "name")
	volumeObj.Add(jsonutils.NewString(zoneId), "availability_zone")
	volumeObj.Add(jsonutils.NewString(desc), "description")
	volumeObj.Add(jsonutils.NewString(category), "volume_type")
	volumeObj.Add(jsonutils.NewInt(int64(sizeGb)), "size")
	if len(snapshotId) > 0 {
		volumeObj.Add(jsonutils.NewString(snapshotId), "snapshot_id")
	}

	params.Add(volumeObj, "volume")

	disk := SDisk{}
	err := DoCreate(self.ecsClient.Disks.Create, params, &disk)
	return disk.ID, err
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762428.html
// 默认删除云硬盘关联的所有快照
func (self *SRegion) DeleteDisk(diskId string) error {
	return DoDeleteWithSpec(self.ecsClient.Disks.DeleteInContextWithSpec, nil, diskId, "", nil, nil)
}

/*
扩容状态为available的云硬盘时，没有约束限制。
扩容状态为in-use的云硬盘时，有以下约束：
不支持共享云硬盘，即multiattach参数值必须为false。
云硬盘所挂载的云服务器状态必须为ACTIVE、PAUSED、SUSPENDED、SHUTOFF才支持扩容
*/
func (self *SRegion) resizeDisk(diskId string, sizeGB int64) error {
	params := jsonutils.NewDict()
	osExtendObj := jsonutils.NewDict()
	osExtendObj.Add(jsonutils.NewInt(sizeGB), "new_size") // GB
	params.Add(osExtendObj, "os-extend")
	_, err := self.ecsClient.Disks.PerformAction2("action", diskId, params, "")
	return err
}

/*
https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408629.html
只支持快照回滚到源云硬盘，不支持快照回滚到其它指定云硬盘。
只有云硬盘状态处于“available”或“error_rollbacking”状态才允许快照回滚到源云硬盘。
*/
func (self *SRegion) resetDisk(diskId, snapshotId string) (string, error) {
	params := jsonutils.NewDict()
	rollbackObj := jsonutils.NewDict()
	rollbackObj.Add(jsonutils.NewString(diskId), "volume_id")
	params.Add(rollbackObj, "rollback")
	_, err := self.ecsClient.OsSnapshots.PerformAction2("rollback", snapshotId, params, "")
	return diskId, err
}

func (self *SDisk) GetProjectId() string {
	return ""
}
