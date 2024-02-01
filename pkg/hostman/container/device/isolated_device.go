package device

import (
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	RegisterDriver(newIsolatedDevice())
}

type isolatedDevice struct{}

func newIsolatedDevice() IDeviceDriver {
	return &isolatedDevice{}
}

func (i isolatedDevice) GetType() apis.ContainerDeviceType {
	return apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE
}

func (i isolatedDevice) GetRuntimeDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	man, err := isolated_device.GetContainerDeviceManager(isolated_device.ContainerDeviceType(dev.IsolatedDevice.DeviceType))
	if err != nil {
		return nil, errors.Wrapf(err, "GetContainerDeviceManager by type %q", dev.Type)
	}
	ctrDevs, err := man.NewContainerDevices(input, dev)
	if err != nil {
		return nil, errors.Wrapf(err, "NewContainerDevices with %#v", dev)
	}
	return ctrDevs, nil
}
