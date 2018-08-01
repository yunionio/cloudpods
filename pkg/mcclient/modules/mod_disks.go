package modules

var (
	Disks ResourceManager
)

func init() {
	Disks = NewComputeManager("disk", "disks",
		[]string{"ID", "Name", "Billing_type",
			"Disk_size", "Status", "Fs_format",
			"Disk_type", "Disk_format", "Is_public",
			"Guest_count", "Storage_type",
			"Zone", "Device", "Guest",
			"Guest_id", "Created_at"},
		[]string{"Storage", "Tenant"})

	registerCompute(&Disks)
}
