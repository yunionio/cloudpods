package modules

var (
	Cloudaccounts ResourceManager
)

func init() {
	Cloudaccounts = NewComputeManager("cloudaccount", "cloudaccounts",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url",
			"balance", "error_count", "health_status",
			"Sync_Status", "Last_sync",
			"guest_count",
			"Provider", "Enable_Auto_Sync", "Sync_Interval_Seconds"},
		[]string{})

	registerCompute(&Cloudaccounts)
}
