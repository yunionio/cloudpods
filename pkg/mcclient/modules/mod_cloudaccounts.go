package modules

var (
	Cloudaccounts ResourceManager
)

func init() {
	Cloudaccounts = NewComputeManager("cloudaccount", "cloudaccounts",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url", "Account",
			"version", "balance", "error_count",
			"Sync_Status", "Last_sync", "Last_sync_end_at",
			"Provider", "Enable_Auto_Sync", "Sync_Interval_Seconds"},
		[]string{})

	registerCompute(&Cloudaccounts)
}
