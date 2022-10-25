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

package hcs

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

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
	ServerId     string `json:"server_id"`
	AttachmentId string `json:"attachment_id"`
	AttachedAt   string `json:"attached_at"`
	HostName     string `json:"host_name"`
	VolumeId     string `json:"volume_id"`
	Device       string `json:"device"`
	Id           string `json:"id"`
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
	ImageId                string `json:"image_id"`
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
	multicloud.SDisk
	multicloud.HuaweiDiskTags
	//details *SResourceDetail

	region *SRegion

	Id                  string              `json:"id"`
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
	SnapshotId          string              `json:"snapshot_id"`
	CreatedAt           time.Time           `json:"created_at"`
	VolumeType          string              `json:"volume_type"`
	VolumeImageMetadata VolumeImageMetadata `json:"volume_image_metadata"`
	ReplicationStatus   string              `json:"replication_status"`
	UserId              string              `json:"user_id"`
	ConsistencygroupId  string              `json:"consistencygroup_id"`
	UpdatedAt           string              `json:"updated_at"`
	EnterpriseProjectId string

	ExpiredTime time.Time
}

func (self *SDisk) GetId() string {
	return self.Id
}

func (self *SDisk) GetName() string {
	if len(self.Name) == 0 {
		return self.Id
	}

	return self.Name
}

func (self *SDisk) GetGlobalId() string {
	return self.Id
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
	ret, err := self.region.GetDisk(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
	//	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.SizeGB * 1024)
}

func (self *SDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SDisk) GetTemplateId() string {
	return self.VolumeImageMetadata.ImageId
}

// Bootable 表示硬盘是否为启动盘。
// 启动盘 != 系统盘(必须是启动盘且挂载在root device上)
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
		return self.Attachments[0].ServerId
	}
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.region.DeleteDisk(self.Id)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	snap, err := self.region.CreateSnapshot(self.Id, name, desc)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (self *SDisk) GetISnapshot(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.region.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.region.GetSnapshots(self.Id, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self.region
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SDisk) Resize(ctx context.Context, sizeMB int64) error {
	return self.region.ResizeDisk(self.GetId(), sizeMB/1024)
}

func (self *SDisk) Detach() error {
	return self.region.DetachDisk(self.GetMountServerId(), self.GetId())
}

func (self *SDisk) Attach(device string) error {
	return self.region.AttachDisk(self.GetMountServerId(), self.GetId(), device)
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

	err := self.region.ResetDisk(self.Id, snapshotId)
	if err != nil {
		return self.Id, err
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
	return self.Id, nil
}

// 华为云不支持重置
func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetDisk(id string) (*SDisk, error) {
	ret := &SDisk{region: self}
	res := fmt.Sprintf("volumes/%s", id)
	return ret, self.evsGet(res, ret)
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762430.html
func (self *SRegion) GetDisks(zoneId string) ([]SDisk, error) {
	params := url.Values{}
	if len(zoneId) > 0 {
		params.Set("availability_zone", zoneId)
	}
	disks := []SDisk{}
	return disks, self.evsList("volumes/detail", params, &disks)
}

func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, snapshotId string, desc string, projectId string) (*SDisk, error) {
	volume := map[string]interface{}{
		"name":              name,
		"availability_zone": zoneId,
		"description":       desc,
		"volume_type":       category,
		"size":              sizeGb,
	}
	if len(snapshotId) > 0 {
		volume["snapshot_id"] = snapshotId
	}
	if len(projectId) > 0 {
		volume["enterprise_project_id"] = projectId
	}
	params := map[string]interface{}{
		"volume": volume,
	}
	ret := &SDisk{region: self}
	return ret, self.evsCreate("volumes", params, ret)
}

func (self *SRegion) DeleteDisk(id string) error {
	res := fmt.Sprintf("volumes/%s", id)
	return self.evsDelete(res)
}

func (self *SRegion) ResizeDisk(id string, sizeGb int64) error {
	params := map[string]interface{}{
		"os_extend": map[string]interface{}{
			"new_size": sizeGb,
		},
	}
	res := fmt.Sprintf("volumes/%s", id)
	return self.evsPerform(res, "action", params)
}

func (self *SRegion) ResetDisk(diskId, snapshotId string) error {
	params := map[string]interface{}{
		"volume_id": diskId,
	}
	res := fmt.Sprintf("os-vendor-snapshots/%s", snapshotId)
	return self.evsPerform(res, "rollback", params)
}

func (self *SDisk) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	res := fmt.Sprintf("servers/%s/os-volume_attachments/%s", instanceId, diskId)
	return self.delete("ecs", "v2", res)
}

func (self *SRegion) AttachDisk(instanceId string, diskId string, device string) error {
	params := map[string]interface{}{
		"volumeAttachment": map[string]interface{}{
			"volumeId": instanceId,
			"device":   device,
		},
	}
	res := fmt.Sprintf("servers/%s/os-volume_attachments", instanceId)
	return self.perform("ecs", "v2", res, "action", params, nil)
}

type SDiskType struct {
	ExtraSpecs ExtraSpecs `json:"extra_specs"`
	Name       string     `json:"name"`
	QosSpecsId string     `json:"qos_specs_id"`
	Id         string     `json:"id"`
	IsPublic   bool       `json:"is_public"`
}

func (self *SDiskType) IsAvaliableInZone(zoneId string) bool {
	if strings.Contains(self.ExtraSpecs.HwAvailabilityZone, zoneId) {
		return true
	}
	return false
}

type ExtraSpecs struct {
	VolumeBackendName  string `json:"volume_backend_name"`
	HwAvailabilityZone string `json:"HW:availability-zone"`
}

func (self *SRegion) GetDiskTypes() ([]SDiskType, error) {
	ret := []SDiskType{}
	return ret, self.get("evs", "v3", "types", &ret)
}
