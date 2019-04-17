package compute

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
	SkuStatusAvailable = "available"
	SkuStatusSoldout   = "soldout"
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
