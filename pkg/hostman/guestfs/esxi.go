package guestfs

type SEsxiRootFs struct {
	*SGuestRootFsDriver
}

func NewEsxiRootFs(part *SKVMGuestDiskPartition) IRootFsDriver {
	return &SEsxiRootFs{SGuestRootFsDriver: NewGuestRootFsDriver(part).(*SGuestRootFsDriver)}
}

func init() {
	rootfsDrivers = append(rootfsDrivers, NewEsxiRootFs)
}
