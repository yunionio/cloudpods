package modules

var (
	Cloudproviders ResourceManager
)

func init() {
	Cloudproviders = NewComputeManager("cloudprovider", "cloudproviders",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url", "Account", "Last_sync", "Provider"},
		[]string{})

	registerCompute(&Cloudproviders)
}
