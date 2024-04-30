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

func (c BaseDevice) GetNvidiaMpsMemoryLimit() int {
	return -1
}

func (c BaseDevice) GetNvidiaMpsMemoryTotal() int {
	return -1
}

func (c BaseDevice) GetNvidiaMpsThreadPercentage() int {
	return -1
}

func CheckVirtualNumber(dev *isolated_device.ContainerDevice) error {
	if dev.VirtualNumber <= 0 {
		return errors.Errorf("virtual_number must > 0")
	}
	return nil
}
