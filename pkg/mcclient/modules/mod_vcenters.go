package modules

var (
	VCenters ResourceManager
)

func init() {
	VCenters = NewComputeManager("vcenter", "vcenters",
		[]string{"ID", "Name", "Hostname", "Status", "Version", "Host_count"},
		[]string{})

	registerCompute(&VCenters)
}
