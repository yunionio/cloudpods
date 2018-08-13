package modules

var (
	VCenters ResourceManager
)

func init() {
	VCenters = NewComputeManager("vcenter", "vcenters",
		[]string{"ID", "Name", "access_url", "Status", "Version", "Host_count", "Provider"},
		[]string{})

	registerCompute(&VCenters)
}
