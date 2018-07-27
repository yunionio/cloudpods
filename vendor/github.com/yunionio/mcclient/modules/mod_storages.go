package modules

var (
	Storages ResourceManager
)

func init() {
	Storages = NewComputeManager("storage", "storages",
		[]string{"ID", "Name", "Capacity", "Status", "Used_capacity", "Waste_capacity", "Free_capacity", "Storage_type", "Medium_type", "Virtual_capacity", "commit_bound", "commit_rate", "Enabled"},
		[]string{})

	registerCompute(&Storages)
}
