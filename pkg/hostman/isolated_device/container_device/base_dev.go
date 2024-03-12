package container_device

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

type BaseDevice struct {
	*isolated_device.SBaseDevice
	Path string
}

func NewBaseDevice(dev *isolated_device.PCIDevice, devType isolated_device.ContainerDeviceType, devPath string) *BaseDevice {
	return &BaseDevice{
		SBaseDevice: isolated_device.NewBaseDevice(dev, string(devType)),
		Path:        devPath,
	}
}

func (b BaseDevice) GetVGACmd() string {
	return ""
}

func (b BaseDevice) GetCPUCmd() string {
	return ""
}

func (b BaseDevice) GetQemuId() string {
	return ""
}

func (c BaseDevice) CustomProbe(idx int) error {
	return nil
}

func (c BaseDevice) GetDevicePath() string {
	return c.Path
}

func CheckVirtualNumber(dev *isolated_device.ContainerDevice) error {
	if dev.VirtualNumber <= 0 {
		return errors.Errorf("virtual_number must > 0")
	}
	return nil
}
