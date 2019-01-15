package modules

var (
	Zones ResourceManager
)

func init() {
	Zones = NewComputeManager("zone", "zones",
		[]string{"ID", "Name", "Name_CN", "Status", "Cloudregion_ID", "CloudRegion", "Location"},
		[]string{"Manager_URI"})

	registerCompute(&Zones)
}
