package modules

var (
	Zones ResourceManager
)

func init() {
	Zones = NewComputeManager("zone", "zones",
		[]string{"ID", "Name", "Name_CN", "Status", "Cloudregion_ID", "CloudRegion"},
		[]string{"Manager_URI"})

	registerCompute(&Zones)
}
