package guestfs

type SMacOSRootFs struct {
	*SGuestRootFsDriver
}

func NewMacOSRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SMacOSRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
}

func init() {
	rootfsDrivers = append(rootfsDrivers, NewMacOSRootFs)
}
