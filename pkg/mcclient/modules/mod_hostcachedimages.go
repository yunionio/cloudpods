package modules

var (
	Hostcachedimages JointResourceManager
)

func init() {
	Hostcachedimages = NewJointComputeManager("hostcachedimage",
		"hostcachedimages",
		[]string{"Host_ID", "Host",
			"Cachedimage_ID", "Image", "Size", "Status", "Path", "Reference"},
		[]string{},
		&Hosts,
		&Cachedimages)
	registerCompute(&Hostcachedimages)
}
