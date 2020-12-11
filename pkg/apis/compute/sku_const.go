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
	"yunion.io/x/onecloud/pkg/apis/billing"
)

const (
	SkuCategoryGeneralPurpose      = "general_purpose"      // 通用型
	SkuCategoryBurstable           = "burstable"            // 突发性能型
	SkuCategoryComputeOptimized    = "compute_optimized"    // 计算优化型
	SkuCategoryMemoryOptimized     = "memory_optimized"     // 内存优化型
	SkuCategoryStorageIOOptimized  = "storage_optimized"    // 存储IO优化型
	SkuCategoryHardwareAccelerated = "hardware_accelerated" // 硬件加速型
	SkuCategoryHighStorage         = "high_storage"         // 高存储型
	SkuCategoryHighMemory          = "high_memory"          // 高内存型
)

const (
	SkuStatusAvailable    = "available"
	SkuStatusSoldout      = "soldout"
	SkuStatusCreating     = "creating"
	SkuStatusCreatFailed  = "create_failed"
	SkuStatusDeleting     = "deleting"
	SkuStatusDeleteFailed = "delete_failed"
	SkuStatusUnknown      = "unknown"
	SkuStatusReady        = "ready"
)

var InstanceFamilies = map[string]string{
	SkuCategoryGeneralPurpose:      "g1",
	SkuCategoryBurstable:           "t1",
	SkuCategoryComputeOptimized:    "c1",
	SkuCategoryMemoryOptimized:     "r1",
	SkuCategoryStorageIOOptimized:  "i1",
	SkuCategoryHardwareAccelerated: "",
	SkuCategoryHighStorage:         "hc1",
	SkuCategoryHighMemory:          "hr1",
}

var SKU_FAMILIES = []string{
	SkuCategoryGeneralPurpose,
	SkuCategoryBurstable,
	SkuCategoryComputeOptimized,
	SkuCategoryMemoryOptimized,
	SkuCategoryStorageIOOptimized,
	SkuCategoryHardwareAccelerated,
	SkuCategoryHighStorage,
	SkuCategoryHighMemory,
}

type ServerSkuListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput

	ZonalFilterListInput
	billing.BillingResourceListInput
	UsableResourceListInput

	// filter sku by memory size in MB
	MemorySizeMb []int `json:"memory_size_mb"`
	// filter sku by CPU core count
	CpuCoreCount []int `json:"cpu_core_count"`

	// 后付费状态
	PostpaidStatus string `json:"postpaid_status"`

	// 预付费状态
	PrepaidStatus string `json:"prepaid_status"`
}

type ElasticcacheSkuListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput

	UsableResourceListInput
	billing.BillingResourceListInput

	ZonalFilterListInput

	// filter sku by memory size in MB
	MemorySizeMb int `json:"memory_size_mb"`

	InstanceSpec []string `json:"instance_spec"`

	EngineArch []string `json:"engine_arch"`

	LocalCategory []string `json:"local_category"`

	PrepaidStatus []string `json:"prepaid_status"`

	PostpaidStatus []string `json:"postpaid_sStatus"`

	// 引擎 redis|memcached
	Engine []string `json:"engine"`

	// 引擎版本 3.0
	EngineVersion []string `json:"engine_version"`

	// CPU 架构 x86|ARM
	CpuArch []string `json:"cpu_arch"`

	// 存储类型 DRAM|SCM
	StorageType []string `json:"storage_type"`

	// standrad|enhanced
	PerformanceType []string `json:"performance_type"`

	// single（单副本） | double（双副本) | readone (单可读) | readthree （3可读） | readfive（5只读）
	NodeType []string `json:"node_type"`
}

type DBInstanceSkuListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput

	RegionalFilterListInput
	billing.BillingResourceListInput

	// StorageType
	StorageType []string `json:"storage_type"`

	VcpuCount []int `json:"vcpu_count"`

	VmemSizeMb []int `json:"vmem_size_mb"`

	Category []string `json:"category"`

	Engine []string `json:"engine"`

	EngineVersion []string `json:"engine_version"`

	Zone1 []string `json:"zone1"`
	Zone2 []string `json:"zone2"`
	Zone3 []string `json:"zone3"`
}
