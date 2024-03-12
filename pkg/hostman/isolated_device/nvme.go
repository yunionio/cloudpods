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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type sNVMEDevice struct {
	*SBaseDevice

	sizeMB int
}

func (dev *sNVMEDevice) GetVGACmd() string {
	return ""
}

func (dev *sNVMEDevice) GetCPUCmd() string {
	return ""
}

func (dev *sNVMEDevice) GetQemuId() string {
	return fmt.Sprintf("dev_%s", strings.ReplaceAll(dev.GetAddr(), ":", "_"))
}

func (dev *sNVMEDevice) GetNVMESizeMB() int {
	return dev.sizeMB
}

func newNVMEDevice(dev *PCIDevice, devType string, sizeMB int) *sNVMEDevice {
	return &sNVMEDevice{
		SBaseDevice: NewBaseDevice(dev, devType),
		sizeMB:      sizeMB,
	}
}

func getPassthroughNVMEDisks(nvmePciDisks []string) ([]*sNVMEDevice, error) {
	devs := make([]*sNVMEDevice, 0)
	for _, conf := range nvmePciDisks {
		diskConf := strings.Split(conf, "/")
		if len(diskConf) != 2 {
			return nil, fmt.Errorf("bad nvme config %s", conf)
		}
		var pciAddr, size = diskConf[0], diskConf[1]
		sizeMb, err := fileutils.GetSizeMb(size, 'M', 1024)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse pci device %s size %s", pciAddr, size)
		}
		dev, err := detectPCIDevByAddrWithoutIOMMUGroup(pciAddr)
		if err != nil {
			return nil, errors.Wrap(err, "detectPCIDevByAddrWithoutIOMMUGroup")
		}
		devs = append(devs, newNVMEDevice(dev, api.NVME_PT_TYPE, sizeMb))
	}
	return devs, nil
}
