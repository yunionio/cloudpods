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
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func getSRIOVNics(hostNics [][2]string) ([]*sSRIOVNicDevice, error) {
	sysfsNetDir := "/sys/class/net"
	files, err := ioutil.ReadDir(sysfsNetDir)
	if err != nil {
		return nil, err
	}
	nics := [][2]string{}
	for i := 0; i < len(files); i++ {
		nicPath, err := filepath.EvalSymlinks(path.Join(sysfsNetDir, files[i].Name()))
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(nicPath, "/sys/devices/pci0000:") {
			continue
		}
		var idx int
		for idx = range hostNics {
			if hostNics[idx][0] == files[i].Name() {
				nics = append(nics, hostNics[idx])
			}
		}
	}
	log.Infof("host nics %s detected support sriov nics %v", hostNics, nics)
	sriovNics := make([]*sSRIOVNicDevice, 0)
	for i := 0; i < len(nics); i++ {
		nicDir := path.Join(sysfsNetDir, nics[i][0], "device")
		vfs, err := ioutil.ReadDir(nicDir)
		if err != nil {
			return nil, err
		}
		for j := 0; j < len(vfs); j++ {
			if strings.HasPrefix(vfs[j].Name(), "virtfn") {
				virtfn, err := strconv.Atoi(vfs[j].Name()[len("virtfn"):])
				if err != nil {
					return nil, err
				}
				vfPath, err := filepath.EvalSymlinks(path.Join(nicDir, vfs[j].Name()))
				if err != nil {
					return nil, err
				}
				vfBDF := path.Base(vfPath)
				vfDev, err := detectSRIOVNicDevice(vfBDF)
				if err != nil {
					return nil, err
				}
				sriovNics = append(sriovNics, NewSRIOVNicDevice(vfDev, api.NIC_TYPE, nics[i][1], nics[i][0], virtfn))
			}
		}
	}
	return sriovNics, nil
}

func detectSRIOVNicDevice(vfBDF string) (*PCIDevice, error) {
	dev, err := detectPCIDevByAddrWithoutIOMMUGroup(vfBDF)
	if err != nil {
		return nil, err
	}
	driver, err := dev.getKernelDriver()
	if err != nil {
		return nil, err
	}
	if driver == VFIO_PCI_KERNEL_DRIVER {
		return dev, nil
	}
	if driver != "" {
		if err = dev.unbindDriver(); err != nil {
			return nil, err
		}
	}
	if err = dev.bindDriver(); err != nil {
		return nil, err
	}
	return dev, nil
}

func NewSRIOVNicDevice(dev *PCIDevice, devType, wireId, pfName string, virtfn int) *sSRIOVNicDevice {
	return &sSRIOVNicDevice{
		sBaseDevice: newBaseDevice(dev, devType),
		WireId:      wireId,
		pfName:      pfName,
		virtfn:      virtfn,
	}
}

type sSRIOVNicDevice struct {
	*sBaseDevice

	WireId string
	pfName string
	virtfn int
}

func (dev *sSRIOVNicDevice) GetPfName() string {
	return dev.pfName
}

func (dev *sSRIOVNicDevice) GetVirtfn() int {
	return dev.virtfn
}

func (dev *sSRIOVNicDevice) GetVGACmd() string {
	return ""
}

func (dev *sSRIOVNicDevice) GetCPUCmd() string {
	return ""
}

func (dev *sSRIOVNicDevice) GetQemuId() string {
	return fmt.Sprintf("dev_%s", strings.ReplaceAll(dev.GetAddr(), ":", "_"))
}

func (dev *sSRIOVNicDevice) GetWireId() string {
	return dev.WireId
}

func (dev *sSRIOVNicDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error) {
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

func (dev *sSRIOVNicDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	if len(isolatedDev.VfioDevs) == 0 {
		return nil, errors.Errorf("device %s no pci ids", isolatedDev.Id)
	}

	return []*HotUnplugOption{
		{
			Id: isolatedDev.VfioDevs[0].Id,
		},
	}, nil
}

func (dev *sSRIOVNicDevice) CustomProbe(idx int) error {
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
