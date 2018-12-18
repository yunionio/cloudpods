package modules

var (
	BillBalances ResourceManager
)

func init() {
	BillBalances = NewMeterManager("bill_balance", "bill_balances",
		[]string{"provider", "account", "account_name", "query_date", "balance", "currency", "today_fee", "month_fee"},
		[]string{},
	)
	register(&BillBalances)
}
