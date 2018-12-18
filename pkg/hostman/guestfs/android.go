package guestfs

type SAndroidRootFs struct {
	*SGuestRootFsDriver
}

func NewAndroidRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SAndroidRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
}

func init() {
	rootfsDrivers = append(rootfsDrivers, NewAndroidRootFs)
}
