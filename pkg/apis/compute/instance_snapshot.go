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

	"yunion.io/x/onecloud/pkg/apis"
)

type SimpleSnapshot struct {
	// 快照Id
	Id string `json:"id"`
	// 快照名称
	Name string `json:"name"`
	// 存储Id
	StorageId string `json:"storage_id"`
	// 磁盘类型
	DiskType string `json:"disk_type"`
	// 区域Id
	CloudregionId string `json:"cloudregion_id"`
	// 快照大小
	Size int `json:"size"`
	// 快照状态
	Status string `json:"status"`
	// 存储类型
	StorageType string `json:"storage_type"`
	// 密钥
	EncryptKeyId string `json:"encrypt_key_id"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
}

type InstanceSnapshotDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	SInstanceSnapshot
	apis.EncryptedResourceDetails

	// 云主机状态
	GuestStatus string `json:"guest_status"`
	// 云主机名称
	Guest string `json:"guest"`

	// 存储类型
	StorageType string `json:"storage_type"`

	// 快照列表
	Snapshots  []SimpleSnapshot  `json:"snapshots"`
	Properties map[string]string `json:"properties"`

	// 主机快照大小
	Size int `json:"size"`
}

type InstanceSnapshotListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.MultiArchResourceBaseListInput

	ManagedResourceListInput

	ServerFilterListInput

	// 操作系统类型
	OsType []string `json:"os_type"`

	// 包含内存快照
	WithMemory *bool `json:"with_memory"`
	// 按磁盘快照数量排序
	// pattern:asc|desc
	OrderByDiskSnapshotCount string `json:"order_by_disk_snapshot_count"`
	// 按虚拟机名称排序
	// pattern:asc|desc
	OrderByGuest string `json:"order_by_guest"`
}
