package modules

var (
	Cloudaccounts ResourceManager
)

func init() {
	Cloudaccounts = NewComputeManager("cloudaccount", "cloudaccounts",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url", "Account",
			"Last_sync", "Provider"},
		[]string{})

	registerCompute(&Cloudaccounts)
}
