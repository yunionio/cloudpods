// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esxi

import (
	"reflect"
	"strings"

	"github.com/vmware/govmomi/vim25/types"
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

func (dev *SVirtualDevice) GetIndex() int {
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
