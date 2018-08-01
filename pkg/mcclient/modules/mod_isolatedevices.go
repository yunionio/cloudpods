package modules

var (
	IsolatedDevices ResourceManager
)

func init() {
	IsolatedDevices = NewComputeManager("isolated_device", "isolated_devices",
		[]string{"ID", "Dev_type",
			"Model", "Addr", "Vendor_device_id",
			"Host_id", "Host",
			"Guest_id", "Guest", "Guest_status"},
		[]string{})
	registerCompute(&IsolatedDevices)
}
