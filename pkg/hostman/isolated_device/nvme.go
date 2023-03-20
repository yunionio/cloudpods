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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/util/procutils"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
)

type sNVMEDevice struct {
	*sBaseDevice

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

func (dev *sNVMEDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error) {
	ret := make([]*HotPlugOption, 0)

	var masterDevOpt *HotPlugOption
	for i := 0; i < len(isolatedDev.VfioDevs); i++ {
		cmd := isolatedDev.VfioDevs[i].HostAddr
		if optCmd := isolatedDev.VfioDevs[i].OptionsStr(); len(optCmd) > 0 {
			cmd += fmt.Sprintf(",%s", optCmd)
		}
		opts := map[string]string{
			"host": cmd,
			"id":   isolatedDev.VfioDevs[i].Id,
		}
		devOpt := &HotPlugOption{
			Device:  isolatedDev.VfioDevs[i].DevType,
			Options: opts,
		}
		if isolatedDev.VfioDevs[i].Function == 0 {
			masterDevOpt = devOpt
		} else {
			ret = append(ret, devOpt)
		}
	}
	// if PCI slot function 0 already assigned, qemu will reject hotplug function
	// so put function 0 at the enda
	if masterDevOpt == nil {
		return nil, errors.Errorf("Device no function 0 found")
	}
	ret = append(ret, masterDevOpt)
	return ret, nil
}

func (dev *sNVMEDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	if len(isolatedDev.VfioDevs) == 0 {
		return nil, errors.Errorf("device %s no pci ids", isolatedDev.Id)
	}

	return []*HotUnplugOption{
		{
			Id: isolatedDev.VfioDevs[0].Id,
		},
	}, nil
}

func (dev *sNVMEDevice) GetNVMESizeMB() int {
	return dev.sizeMB
}

func newNVMEDevice(dev *PCIDevice, devType string, sizeMB int) *sNVMEDevice {
	return &sNVMEDevice{
		sBaseDevice: newBaseDevice(dev, devType),
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
		driver, err := dev.getKernelDriver()
		if err != nil {
			return nil, err
		}
		if driver != VFIO_PCI_KERNEL_DRIVER {
			if driver != "" {
				if err = dev.unbindDriver(); err != nil {
					return nil, err
				}
			}
			if err = dev.bindDriver(); err != nil {
				return nil, err
			}
		}
		devs = append(devs, newNVMEDevice(dev, api.NVME_PT_TYPE, sizeMb))
	}
	return devs, nil
}

func (dev *sNVMEDevice) CustomProbe(idx int) error {
	// check environments on first probe
	if idx == 0 {
		for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
			if err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Run(); err != nil {
				return fmt.Errorf("modprobe %s: %v", driver, err)
			}
		}
	}

	driver, err := dev.GetKernelDriver()
	if err != nil {
		return fmt.Errorf("Nic %s is occupied by another driver: %s", dev.GetAddr(), driver)
	}
	return nil
}
