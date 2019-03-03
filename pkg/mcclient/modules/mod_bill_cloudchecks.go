package modules

var (
	BillCloudChecks ResourceManager
)

func init() {
	BillCloudChecks = NewMeterManager("bill_cloudcheck", "bill_cloudchecks",
		[]string{"provider", "account_id", "sum_month", "res_type", "res_id", "res_name", "external_id", "cloud_fee", "kvm_fee", "diff_fee", "diff_percent"},
		[]string{},
	)
	register(&BillCloudChecks)
}
