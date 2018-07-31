package modules

var (
	Cloudregions ResourceManager
)

func init() {
	Cloudregions = NewComputeManager("cloudregion", "cloudregions",
		[]string{"ID", "Name", "Enabled", "Status", "Provider", "Latitude", "Longitude"},
		[]string{})

	registerCompute(&Cloudregions)
}
