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

import "yunion.io/x/onecloud/pkg/apis"

type SnapshotCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EncryptedResourceCreateInput

	// 磁盘Id
	// 目前仅VMware平台不支持创建快照,其余平台磁盘均支持创建快照
	// required: true
	DiskId string `json:"disk_id"`
	// swagger:ignore
	Disk string `json:"disk" yunion-deprecated-by:"disk_id"`
	// swagger:ignore
	StorageId string `json:"storage_id"`
	// swagger:ignore
	CreatedBy string `json:"created_by"`
	// swagger:ignore
	Location string `json:"location"`
	// swagger:ignore
	Size int `json:"size"`
	// swagger:ignore
	DiskType string `json:"disk_type"`
	// swagger:ignore
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	OutOfChain bool `json:"out_of_chain"`
	// swagger:ignore
	ManagerId string `json:"manager_id"`
	// swagger:ignore
	OsArch string `json:"os_arch"`
}

type SSnapshotPolicyCreateInput struct {
	apis.Meta

	Name      string `json:"name"`
	ProjectId string `json:"project_id"`
	DomainId  string `json:"domain_id"`

	RetentionDays  int   `json:"retention_days"`
	RepeatWeekdays []int `json:"repeat_weekdays"`
	TimePoints     []int `json:"time_points"`
}

type SSnapshotPolicyCreateInternalInput struct {
	apis.Meta

	Name      string
	ProjectId string
	DomainId  string

	RetentionDays  int
	RepeatWeekdays uint8
	TimePoints     uint32
}

type SnapshotListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.MultiArchResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	StorageShareFilterListInput

	// filter snapshot that is fake deleted
	FakeDeleted *bool `json:"fake_deleted"`
	// filter by disk type
	DiskType string `json:"disk_type"`
	// filter instance snapshot
	IsInstanceSnapshot *bool `json:"is_instance_snapshot"`

	DiskFilterListInputBase
	StorageFilterListInputBase

	OutOfChain *bool    `json:"out_of_chain"`
	OsType     []string `json:"os_type"`

	// list server snapshots
	ServerId string `json:"server_id"`

	// 按虚拟机名称排序
	// pattern:asc|desc
	OrderByGuest string `json:"order_by_guest"`
	// 按磁盘名称排序
	// pattern:asc|desc
	OrderByDiskName string `json:"order_by_disk_name"`
}

type SnapshotDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	apis.EncryptedResourceDetails

	SSnapshot

	// 存储类型
	StorageType string `json:"storage_type"`
	// 磁盘状态
	DiskStatus string `json:"disk_status"`
	// 云主机名称
	Guest string `json:"guest"`
	// 云主机Id
	GuestId string `json:"guest_id"`
	// 云主机状态
	GuestStatus string `json:"guest_status"`
	// 磁盘名称
	DiskName string `json:"disk_name"`
	// 是否是子快照
	IsSubSnapshot bool `json:"is_sub_snapshot,allowempty"`
}

type SnapshotSyncstatusInput struct {
}
