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
	"yunion.io/x/onecloud/pkg/apis"
)

type ServerSkuCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	// 区域名称或Id,建议使用Id
	// default: default
	CloudregionId string `json:"cloudregion_id"`

	// 可用区名称或Id, 建议使用Id
	// required: false
	ZoneId string `json:"zone_id"`

	// swagger:ignore
	InstanceTypeFamily string

	// 套餐类型
	//
	//
	//
	// | instance_type_category	|	说明	|
	// |	-----				|	---		|
	// |general_purpose			|通用型		|
	// |burstable				|突发性能型		|
	// |compute_optimized		|计算优化型		|
	// |memory_optimized		|内存优化型		|
	// |storage_optimized		|存储IO优化型		|
	// |hardware_accelerated	|硬件加速型		|
	// |high_storage			|高存储型		|
	// |high_memory				|高内存型		|
	// default: general_purpose
	InstanceTypeCategory string `json:"instance_type_category"`

	// swagger:ignore
	LocalCategory string

	// 预付费状态
	// default: available
	PrepaidStatus string `json:"prepaid_status"`
	// 后付费状态
	// default: available
	PostpaidStatus string `json:"postpaid_status"`

	// Cpu核数
	// minimum: 1
	// maximum: 256
	// required: true
	CpuCoreCount int64 `json:"cpu_core_count"`

	// 内存大小
	// minimum: 512
	// maximum: 524288
	// required: true
	MemorySizeMB int64 `json:"memory_size_mb"`

	// swagger:ignore
	OsName string

	// swagger:ignore
	SysDiskResizable *bool

	// swagger:ignore
	SysDiskType string

	// swagger:ignore
	SysDiskMinSizeGB int

	// swagger:ignore
	SysDiskMaxSizeGB int

	// swagger:ignore
	AttachedDiskType string

	// swagger:ignore
	AttachedDiskSizeGB int

	// swagger:ignore
	AttachedDiskCount int

	// swagger:ignore
	DataDiskTypes string

	// swagger:ignore
	DataDiskMaxCount int

	// swagger:ignore
	NicType string

	// swagger:ignore
	NicMaxCount int

	// swagger:ignore
	GpuAttachable *bool

	// swagger:ignore
	GpuSpec string

	// swagger:ignore
	GpuCount int

	// swagger:ignore
	GpuMaxCount int

	// swagger:ignore
	Provider string
}

type ServerSkuDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	ZoneResourceInfo

	SServerSku

	// 云环境
	CloudEnv string `json:"cloud_env"`

	// 绑定云主机数量
	TotalGuestCount int `json:"total_guest_count"`
}

type ServerSkuUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	// 预付费状态
	// default: available
	PrepaidStatus string `json:"prepaid_status"`
	// 后付费状态
	// default: available
	PostpaidStatus string `json:"postpaid_status"`

	InstanceTypeFamily string `json:"instance_type_family"`

	InstanceTypeCategory string `json:"instance_type_category"`

	LocalCategory string `json:"local_category"` // 记录本地分类

	OsName string `json:"os_name"` // Windows|Linux|Any

	SysDiskResizable *bool `json:"sys_disk_resizable"`

	SysDiskType string `json:"sys_disk_type"`

	SysDiskMinSizeGB *int `json:"sys_disk_min_size_gb"` // not required。 windows比较新的版本都是50G左右。

	SysDiskMaxSizeGB *int `json:"sys_disk_max_size_gb"` // not required

	AttachedDiskType string `json:"attached_disk_type"`

	AttachedDiskSizeGB *int `json:"attached_disk_size_gb"`

	AttachedDiskCount *int `json:"attached_disk_count"`

	DataDiskTypes string `json:"data_disk_types"`

	DataDiskMaxCount *int `json:"data_disk_max_count"`

	NicType string `json:"nic_type"`

	NicMaxCount *int `json:"nic_max_count"`

	GpuAttachable *bool `json:"gpu_attachable"`

	GpuSpec string `json:"gpu_spec"`

	GpuCount *int `json:"gpu_count"`

	GpuMaxCount *int `json:"gpu_max_count"`
}
