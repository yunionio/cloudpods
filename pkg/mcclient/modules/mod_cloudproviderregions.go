package modules

var (
	CloudproviderregionManager JointResourceManager
)

func init() {
	CloudproviderregionManager = NewJointComputeManager("cloudproviderregion",
		"cloudproviderregions",
		[]string{"Cloudaccount_ID", "Cloudaccount",
			"Cloudprovider_ID", "CloudProvider",
			"Cloudregion_ID", "CloudRegion",
			"Enabled", "Sync_Status",
			"Last_Sync", "Last_Sync_End_At", "Auto_Sync"},
		[]string{},
		&Cloudproviders,
		&Cloudregions)

	registerCompute(&CloudproviderregionManager)
}
