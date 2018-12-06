package modules

var (
	BillDetails ResourceManager
)

func init() {
	BillDetails = NewMeterManager("bill_detail", "bill_details",
		[]string{"bill_id", "account", "platform", "region", "manager_project", "res_id",
			"res_type", "res_name", "start_time", "end_time", "charge_type", "item_rate", "item_fee"},
		[]string{},
	)
	register(&BillDetails)
}
