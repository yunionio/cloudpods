package modules

var (
	ServerSkus ResourceManager
)

func init() {
	ServerSkus = NewComputeManager("serversku",
		"serverskus",
		[]string{"id", "name", "sku_id",
			"sku_family", "cpu_core_count", "memory_size_mb",
			"cloudregion_id", "zone_id",
		},
		[]string{})

	registerComputeV2(&ServerSkus)
}
