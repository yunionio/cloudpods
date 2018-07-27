package modules

var (
	ResourceFees ResourceManager
)

func init() {
	ResourceFees = NewMeterManager("resource_fee", "resource_fees",
		[]string{"baremetal_fee", "server_fee", "gpu_fee", "disk_fee", "res_fee", "item_name", "stat_type", "stat_month", "month_total"},

		[]string{},
	)
	register(&ResourceFees)
}
