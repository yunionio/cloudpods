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

package compute

import (
	"time"

	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/fileutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type DiskCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EncryptedResourceCreateInput

	*DiskConfig

	// 调度使用指定的云账号
	PreferManager string `json:"prefer_manager_id"`

	// 此参数仅适用于未指定storage时进行调度到指定区域创建磁盘
	// required: false
	PreferRegion string `json:"prefer_region_id"`

	// 此参数仅适用于未指定storage时进行调度到指定可用区区创建磁盘
	// required: false
	PreferZone string `json:"prefer_zone_id"`

	// swagger:ignore
	PreferWire string `json:"prefer_wire_id"`

	// 此参数仅适用于未指定storage时进行调度到指定可用区区创建磁盘
	// required: false
	PreferHost string `json:"prefer_host_id"`

	// 此参数仅适用于未指定storage时进行调度到指定平台创建磁盘
	// default: kvm
	// enum: kvm, openstack, esxi, aliyun, aws, qcloud, azure, huawei, openstack, ucloud, zstack google, ctyun
	Hypervisor string `json:"hypervisor"`
}

// ToServerCreateInput used by disk schedule
func (req *DiskCreateInput) ToServerCreateInput() *ServerCreateInput {
	input := ServerCreateInput{
		ServerConfigs: &ServerConfigs{
			PreferManager: req.PreferManager,
			PreferRegion:  req.PreferRegion,
			PreferZone:    req.PreferZone,
			PreferWire:    req.PreferWire,
			PreferHost:    req.PreferHost,
			Hypervisor:    req.Hypervisor,
			Disks:         []*DiskConfig{req.DiskConfig},
			// Project:      req.Project,
			// Domain:       req.Domain,
		},
	}
	input.Name = req.Name
	input.ProjectId = req.ProjectId
	input.ProjectDomainId = req.ProjectDomainId
	return &input
}

func (req *ServerCreateInput) ToDiskCreateInput() *DiskCreateInput {
	input := DiskCreateInput{
		DiskConfig:   req.Disks[0],
		PreferRegion: req.PreferRegion,
		PreferHost:   req.PreferHost,
		PreferZone:   req.PreferZone,
		PreferWire:   req.PreferWire,
		Hypervisor:   req.Hypervisor,
	}
	input.Name = req.Name
	input.ProjectId = req.ProjectId
	input.ProjectDomainId = req.ProjectDomainId
	return &input
}

type SnapshotPolicyResourceInput struct {
	// filter disk by snapshotpolicy
	SnapshotpolicyId string `json:"snapshotpolicy_id"`
	// swagger:ignore
	// Deprecated
	// filter disk by snapshotpolicy_id
	Snapshotpolicy string `json:"snapshotpolicy" yunion-deprecated-by:"snapshotpolicy_id"`
}

type SnapshotPolicyFilterListInput struct {
	SnapshotPolicyResourceInput

	// 以快照策略名称排序
	OrderBySnapshotpolicy string `json:"order_by_snapshotpolicy"`
}

type DiskListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.MultiArchResourceBaseListInput
	apis.AutoDeleteResourceBaseListInput
	billing.BillingResourceListInput
	StorageFilterListInput

	SnapshotPolicyFilterListInput
	ServerFilterListInput

	// filter disk by whether it is being used
	Unused *bool `json:"unused"`

	// swagger:ignore
	// Deprecated
	// filter by disk type
	Type string `json:"type" yunion-deprecated-by:"disk_type"`
	// 过滤指定disk_type的磁盘列表，可能的值为：sys, data, swap. volume
	//
	// | disk_type值 | 说明 |
	// |-------------|----------|
	// | sys         | 虚拟机系统盘    |
	// | data        | 虚拟机数据盘    |
	// | swap        | 虚拟机内存交换盘 |
	// | volume      | 容器volumn盘   |
	//
	DiskType string `json:"disk_type"`

	DiskFormat string `json:"disk_format"`

	DiskSize int `json:"disk_size"`

	FsFormat string `json:"fs_format"`

	OrderByServer string `json:"order_by_server" choices:"asc|desc"`

	OrderByGuestCount string `json:"order_by_guest_count" choices:"asc|desc"`
	// 镜像
	ImageId string `json:"image_id"`
	// swagger:ignore
	// Deprecated
	Template string `json:"template" yunion-deprecated-by:"image_id"`
	// swagger:ignore
	// Deprecated
	TemplateId string `json:"template_id" yunion-deprecated-by:"image_id"`

	// 快照
	SnapshotId string `json:"snapshot_id"`
	// swagger:ignore
	// Deprecated
	Snapshot string `json:"snapshot" yunion-deprecated-by:"snapshot_id"`
}

type DiskResourceInput struct {
	// 虚拟磁盘（ID或Name）
	DiskId string `json:"disk_id"`
	// swagger:ignore
	// Deprecated
	// filter by disk_id
	Disk string `json:"disk" yunion-deprecated-by:"disk_id"`
}

type DiskFilterListInputBase struct {
	DiskResourceInput

	// 以磁盘名称排序
	// pattern:asc|desc
	OrderByDisk string `json:"order_by_disk"`
}

type DiskFilterListInput struct {
	StorageFilterListInput

	DiskFilterListInputBase
}

type SimpleGuest struct {
	// 主机名称
	Name string `json:"name"`
	// 主机ID
	Id string `json:"id"`
	// 主机状态
	Status string `json:"status"`
	// 磁盘序号
	Index int `json:"index"`
	// 磁盘驱动
	Driver string `json:"driver"`
	// 缓存模式
	CacheMode string `json:"cache_mode"`
}

type SimpleSnapshotPolicy struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	RepeatWeekdays []int  `json:"repeat_weekdays"`
	TimePoints     []int  `json:"time_points"`
}

type DiskDetails struct {
	apis.VirtualResourceDetails
	StorageResourceInfo
	apis.EncryptedResourceDetails

	SDisk

	// 所挂载的虚拟机
	Guests []SimpleGuest `json:"guests"`
	// 所挂载的虚拟机
	Guest string `json:"guest"`
	// 所挂载虚拟机的数量
	GuestCount int `json:"guest_count"`
	// 所挂载虚拟机状态
	GuestStatus string `json:"guest_status"`

	// 自动清理时间
	AutoDeleteAt time.Time `json:"auto_delete_at"`
	// 自动快照策略状态
	SnapshotpolicyStatus string `json:"snapshotpolicy_status,allowempty"`

	// 自动快照策略
	Snapshotpolicies []SimpleSnapshotPolicy `json:"snapshotpolicies"`

	// 手动快照数量
	ManualSnapshotCount int `json:"manual_snapshot_count"`
	// 最多可创建手动快照数量
	MaxManualSnapshotCount int `json:"max_manual_snapshot_count"`
}

type DiskResourceInfoBase struct {
	// 磁盘名称
	Disk string `json:"disk"`
}

type DiskResourceInfo struct {
	DiskResourceInfoBase

	// 存储ID
	StorageId string `json:"storage_id"`

	StorageResourceInfo
}

type DiskSyncstatusInput struct {
}

type DiskUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	// 磁盘类型
	DiskType string `json:"disk_type"`
}

type DiskSaveInput struct {
	Name   string
	Format string

	// swagger: ignore
	ImageId string
}

type DiskResizeInput struct {
	// default unit: Mb
	// example: 1024; 40G; 1024M
	Size string `json:"size"`
}

func (self DiskResizeInput) SizeMb() (int, error) {
	if len(self.Size) == 0 {
		return 0, httperrors.NewMissingParameterError("size")
	}
	return fileutils.GetSizeMb(self.Size, 'M', 1024)
}

type DiskAllocateInput struct {
	Format        string
	DiskSizeMb    int
	ImageId       string
	FsFormat      string
	Rebuild       bool
	BackingDiskId string
	SnapshotId    string

	BackupId string
	Backup   *DiskAllocateFromBackupInput

	SnapshotUrl        string
	SnapshotOutOfChain bool
	Protocol           string
	SrcDiskId          string
	SrcPool            string
	ExistingPath       string

	// vmware
	HostIp    string
	Datastore vcenter.SVCenterAccessInfo

	// encryption
	Encryption  bool
	EncryptInfo apis.SEncryptInfo
}

type DiskAllocateFromBackupInput struct {
	BackupId                string
	BackupStorageId         string
	BackupStorageAccessInfo *jsonutils.JSONDict
}

type DiskDeleteInput struct {
	SkipRecycle      *bool
	EsxiFlatFilePath string
}

type DiskResetInput struct {
	SnapshotId string `json:"snapshot_id"`
	AutoStart  bool   `json:"auto_start"`
}
