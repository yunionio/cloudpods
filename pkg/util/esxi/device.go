package esxi

import (
	"reflect"
	"strings"

	"github.com/vmware/govmomi/vim25/types"
	// "yunion.io/x/log"
)

type SVirtualDevice struct {
	vm    *SVirtualMachine
	dev   types.BaseVirtualDevice
	index int
}

func NewVirtualDevice(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualDevice {
	return SVirtualDevice{
		vm:    vm,
		dev:   dev,
		index: index,
	}
}

func (dev *SVirtualDevice) getKey() int32 {
	return dev.dev.GetVirtualDevice().Key
}

func (dev *SVirtualDevice) getControllerKey() int32 {
	return dev.dev.GetVirtualDevice().ControllerKey
}

func (dev *SVirtualDevice) getIndex() int {
	return dev.index
}

func (dev *SVirtualDevice) getLabel() string {
	return dev.dev.GetVirtualDevice().DeviceInfo.GetDescription().Label
}

func (dev *SVirtualDevice) GetDriver() string {
	val := reflect.Indirect(reflect.ValueOf(dev.dev))
	driver := strings.ToLower(val.Type().Name())
	if strings.Contains(driver, "virtualmachine") {
		return strings.Replace(driver, "virtualmachine", "", -1)
	} else if strings.Contains(driver, "virtual") {
		return strings.Replace(driver, "virtual", "", -1)
	} else {
		return driver
	}
}

type SVirtualVGA struct {
	SVirtualDevice
}

func NewVirtualVGA(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualVGA {
	return SVirtualVGA{
		NewVirtualDevice(vm, dev, index),
	}
}

func (vga *SVirtualVGA) getVirtualMachineVideoCard() *types.VirtualMachineVideoCard {
	return vga.dev.(*types.VirtualMachineVideoCard)
}

func (vga *SVirtualVGA) GetEnable3D() bool {
	p3d := vga.getVirtualMachineVideoCard().Enable3DSupport
	return p3d != nil && *p3d
}

func (vga *SVirtualVGA) GetRamSizeMB() int {
	return int(vga.getVirtualMachineVideoCard().VideoRamSizeInKB / 1024)
}

func (vga *SVirtualVGA) String() string {
	return vga.getVirtualMachineVideoCard().DeviceInfo.GetDescription().Summary
}

type SVirtualCdrom struct {
	SVirtualDevice
}

func NewVirtualCdrom(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualCdrom {
	return SVirtualCdrom{
		NewVirtualDevice(vm, dev, index),
	}
}

func (cdrom *SVirtualCdrom) getVirtualCdrom() *types.VirtualCdrom {
	return cdrom.dev.(*types.VirtualCdrom)
}
