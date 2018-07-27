package modules

var (
	Storagecaches ResourceManager
)

func init() {
	Storagecaches = NewComputeManager("storagecache", "storagecaches",
		[]string{"ID", "Name", "Path", "Storages", "size", "count"},
		[]string{})

	registerCompute(&Storagecaches)
}
