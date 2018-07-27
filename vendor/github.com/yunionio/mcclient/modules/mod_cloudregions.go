package modules

var (
	Cloudregions ResourceManager
)

func init() {
	Cloudregions = NewComputeManager("cloudregion", "cloudregions",
		[]string{"ID", "Name", "Enabled", "Status"},
		[]string{})

	registerCompute(&Cloudregions)
}
