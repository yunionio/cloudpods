package compute

type SGMapItem struct {
	SGDeviceName    string
	HostNumber      int
	Bus             int
	SCSIId          int
	Lun             int
	Type            int
	LinuxDeviceName string
}
