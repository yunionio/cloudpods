package modules

var (
	Cloudregions ResourceManager
)

func init() {
	Cloudregions = NewComputeManager("cloudregion", "cloudregions",
		[]string{"ID", "Name", "Enabled", "Status", "Provider",
			"Latitude", "Longitude", "City", "Country_Code",
			"vpc_count", "zone_count", "guest_count", "guest_increment_count",
			"External_Id"},
		[]string{})

	registerCompute(&Cloudregions)
}
