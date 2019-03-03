package modules

var (
	Cloudproviders ResourceManager
)

func init() {
	Cloudproviders = NewComputeManager("cloudprovider", "cloudproviders",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url", "Account",
			"Sync_Status", "Last_sync", "Last_sync_end_at",
			"Provider", "guest_count", "host_count", "vpc_count",
			"storage_count", "storage_cache_count", "eip_count",
			"tenant_id", "tenant"},
		[]string{})

	registerCompute(&Cloudproviders)
}
