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

package isolated_device

import (
	"fmt"
	"strings"
)

type sGeneralPCIDevice struct {
	*sBaseDevice
}

func (dev *sGeneralPCIDevice) GetVGACmd() string {
	return ""
}

func (dev *sGeneralPCIDevice) GetCPUCmd() string {
	return ""
}

func (dev *sGeneralPCIDevice) GetQemuId() string {
	return fmt.Sprintf("dev_%s", strings.ReplaceAll(dev.GetAddr(), ":", "_"))
}

func newGeneralPCIDevice(dev *PCIDevice, devType string) *sGeneralPCIDevice {
	return &sGeneralPCIDevice{
		sBaseDevice: newBaseDevice(dev, devType),
	}
}

func getPassthroughPCIDevs(devModel IsolatedDeviceModel) ([]*sGeneralPCIDevice, error) {
	ret, err := bashOutput(fmt.Sprintf("lspci -d %s:%s -nnmm", devModel.VendorId, devModel.DeviceId))
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, l := range ret {
		if len(l) != 0 {
			lines = append(lines, l)
		}
	}

	devs := []*sGeneralPCIDevice{}
	for _, line := range lines {
		dev := NewPCIDevice2(line)
		if dev.ModelName == "" {
			dev.ModelName = devModel.Model
		}
		devs = append(devs, newGeneralPCIDevice(dev, devModel.DevType))
	}
	return devs, nil
}
