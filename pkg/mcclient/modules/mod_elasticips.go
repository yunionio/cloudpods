package modules

var (
	Elasticips ResourceManager
)

func init() {
	Elasticips = NewComputeManager("eip", "eips",
		[]string{"ID", "Name", "IP_Addr", "Status",
			"Associate_Type", "Associate_ID",
			"Associate_Name",
			"Bandwidth", "Charge_Type",
		},
		[]string{})

	registerCompute(&Elasticips)
}
