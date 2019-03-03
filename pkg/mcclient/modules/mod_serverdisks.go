package modules

var (
	Serverdisks JointResourceManager
)

func init() {
	Serverdisks = NewJointComputeManager(
		"guestdisk",
		"guestdisks",
		[]string{"Guest_ID", "Guest",
			"Disk_ID", "Disk", "Disk_size",
			"Driver", "Cache_mode", "Index", "Status", "Disk_type", "Storage_type"},
		[]string{},
		&Servers,
		&Disks)
	registerCompute(&Serverdisks)
}
