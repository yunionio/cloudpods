package modules

var (
	Baremetalstorages JointResourceManager
)

func init() {
	Baremetalstorages = NewJointComputeManager(
		"baremetalstorage",
		"baremetalstorages",
		[]string{"Baremetal_ID", "Baremetal",
			"Storage_ID", "Storage", "Config",
			"Real_capacity"},
		[]string{},
		&Hosts,
		&Storages)
	// register(&Baremetalstorages)
}
