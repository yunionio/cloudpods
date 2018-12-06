package modules

var (
	BillAnalysises ResourceManager
)

func init() {
	BillAnalysises = NewMeterManager("bill_analysis", "bill_analysises",
		[]string{"stat_date", "stat_value", "res_name", "res_type", "project_name", "res_fee"},
		[]string{},
	)
	register(&BillAnalysises)
}
