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

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
)

type sGeneralPCIDevice struct {
	*sBaseDevice

	sizeMB int
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

func (dev *sGeneralPCIDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error) {
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

func (dev *sGeneralPCIDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	if len(isolatedDev.VfioDevs) == 0 {
		return nil, errors.Errorf("device %s no pci ids", isolatedDev.Id)
	}

	return []*HotUnplugOption{
		{
			Id: isolatedDev.VfioDevs[0].Id,
		},
	}, nil
}

func newGeneralPCIDevice(dev *PCIDevice, devType string) *sGeneralPCIDevice {
	return &sGeneralPCIDevice{
		sBaseDevice: newBaseDevice(dev, devType),
	}
}

func getPassthroughPCIDevs(pciDevs []string) ([]*sGeneralPCIDevice, error) {
	devs := make([]*sGeneralPCIDevice, 0)
	for i, pciDev := range pciDevs {
		segs := strings.SplitN(pciDev, "/", 3)
		pciAddr := segs[0]
		dev, err := detectPCIDevByAddrWithoutIOMMUGroup(pciAddr)
		if err != nil {
			return nil, errors.Wrap(err, "detectPCIDevByAddrWithoutIOMMUGroup")
		}

		if dev.ModelName == "" {
			if len(segs) != 3 {
				return nil, errors.Errorf("failed get device %s model name", pciDev[i])
			}
			dev.ModelName = segs[2]
		}
		devs = append(devs, newGeneralPCIDevice(dev, segs[1]))
	}
	return devs, nil
}
