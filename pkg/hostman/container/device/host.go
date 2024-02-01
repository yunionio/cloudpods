package device

import (
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

func init() {
	RegisterDriver(newHostDevice())
}

type hostDevice struct {
}

func newHostDevice() IDeviceDriver {
	return &hostDevice{}
}

func (h hostDevice) GetType() apis.ContainerDeviceType {
	return apis.CONTAINER_DEVICE_TYPE_HOST
}

func (h hostDevice) GetRuntimeDevices(_ *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	return []*runtimeapi.Device{
		{
			ContainerPath: dev.ContainerPath,
			HostPath:      dev.Host.HostPath,
			Permissions:   dev.Permissions,
		},
	}, nil
}
