package modules

var (
	Hoststorages JointResourceManager
)

func init() {
	Hoststorages = NewJointComputeManager("hoststorage", "hoststorages",
		[]string{"Host_ID", "Host", "Storage_ID",
			"Storage", "Mount_point", "Capacity",
			"Used_capacity", "Waste_capacity",
			"Free_capacity", "cmtbound"},
		[]string{},
		&Hosts,
		&Storages)
	registerCompute(&Hoststorages)
}
