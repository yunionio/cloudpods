package guestfs

type SWindowsRootFs struct {
	*SGuestRootFsDriver
}

func NewWindowsRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SWindowsRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
}

func init() {
	rootfsDrivers = append(rootfsDrivers, NewWindowsRootFs)
}
