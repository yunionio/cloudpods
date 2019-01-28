package fsdriver

type SWindowsRootFs struct {
	*sGuestRootFsDriver
}

//func NewWindowsRootFs(part IDiskPartition) IRootFsDriver {
//return &SWindowsRootFs{sGuestRootFsDriver: newGuestRootFsDriver(part)}
//}
