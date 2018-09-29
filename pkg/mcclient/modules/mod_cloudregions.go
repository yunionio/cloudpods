package modules

var (
	Cloudregions ResourceManager
)

func init() {
	Cloudregions = NewComputeManager("cloudregion", "cloudregions",
		[]string{"ID", "Name", "Enabled", "Status", "Provider", "Latitude", "Longitude",
			"vpc_count", "zone_count", "guest_count", "guest_increment_count"},
		[]string{})

	registerCompute(&Cloudregions)
}
