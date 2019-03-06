package modules

var (
	CloudSkuRates ResourceManager
)

func init() {
	CloudSkuRates = NewMeterManager("cloud_sku_rate", "cloud_sku_rates",
		[]string{"id", "data_id", "data_key", "hour_price", "month_price", "year_price"},
		[]string{},
	)
	register(&CloudSkuRates)
}
