package modules

var (
	BillConditions ResourceManager
)

func init() {
	BillConditions = NewMeterManager("bill_condition", "bill_conditions",
		[]string{"item_id", "item_name"},
		[]string{},
	)
	register(&BillConditions)
}
