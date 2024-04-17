package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	HostIsolatedDeviceModels modulebase.JointResourceManager
)

func init() {
	HostIsolatedDeviceModels = modules.NewJointComputeManager("host_isolated_device_model", "host_isolated_device_models",
		[]string{"Host_ID", "Host", "Isolated_device_model_id", "Isolated_device_model", "Model", "Dev_type", "Device_id", "Vendor_id",
			"Hot_pluggable", "Disable_auto_detect"},
		[]string{},
		&Hosts,
		&IsolatedDeviceModels)
	modules.RegisterCompute(&HostIsolatedDeviceModels)
}
