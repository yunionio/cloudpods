package modules

var (
	BillResources ResourceManager
)

func init() {
	BillResources = NewMeterManager("bill_resource", "bill_resources",
		[]string{"account", "platform", "region", "manager_project", "res_id",
			"res_type", "res_name", "charge_type", "res_fee"},
		[]string{},
	)
	register(&BillResources)
}
