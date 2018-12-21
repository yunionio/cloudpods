package modules

var (
	PriceInfos ResourceManager
)

func init() {
	PriceInfos = NewMeterManager("price_info", "price_infos",
		[]string{"provider", "currency", "sum_price", "spec", "quantity", "period", "price_key", "region_id"},
		[]string{},
	)
	register(&PriceInfos)
}
