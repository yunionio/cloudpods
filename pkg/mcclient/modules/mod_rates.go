package modules

var (
	Rates ResourceManager
)

func init() {
	Rates = NewMeterManager("rate", "rates",
		[]string{"id", "brand", "model", "res_type", "sub_res_type", "duration", "unit", "spec", "rate", "effective_date", "platform", "effective_flag"},
		[]string{},
	)
	register(&Rates)
}
