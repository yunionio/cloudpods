package modules

var (
	Storagecachedimages JointResourceManager
)

func init() {
	Storagecachedimages = NewJointComputeManager("storagecachedimage",
		"storagecachedimages",
		[]string{"Storagecache_ID", "Storagecache",
			"Cachedimage_ID", "Image", "Size", "Status", "Path", "Reference", "Storages"},
		[]string{},
		&Storagecaches,
		&Cachedimages)
	registerCompute(&Storagecachedimages)
}
