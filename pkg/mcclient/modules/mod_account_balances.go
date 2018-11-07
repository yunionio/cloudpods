package modules

var (
	AccountBalances ResourceManager
)

func init() {
	AccountBalances = NewMeterManager("account_balance", "account_balances",
		[]string{"account_type", "available_amount", "current_outcome", "current_income"},
		[]string{},
	)
	register(&AccountBalances)
}
