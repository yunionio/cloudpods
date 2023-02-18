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
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func getSRIOVNics(hostNics []HostNic) ([]*sSRIOVNicDevice, error) {
	sysfsNetDir := "/sys/class/net"
	files, err := ioutil.ReadDir(sysfsNetDir)
	if err != nil {
		return nil, err
	}
	nics := []HostNic{}
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
			if hostNics[idx].Interface == files[i].Name() {
				nics = append(nics, hostNics[idx])
			}
		}
	}

	log.Infof("host nics %s detected support sriov nics %v", hostNics, nics)
	sriovNics := make([]*sSRIOVNicDevice, 0)
	for i := 0; i < len(nics); i++ {
		nicDir := path.Join(sysfsNetDir, nics[i].Interface, "device")
		err = ensureNumvfsEqualTotalvfs(nicDir)
		if err != nil {
			return nil, err
		}
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
				sriovNics = append(sriovNics, NewSRIOVNicDevice(vfDev, api.NIC_TYPE, nics[i].Wire, nics[i].Interface, virtfn))
			}
		}
	}
	return sriovNics, nil
}

func ensureNumvfsEqualTotalvfs(nicDir string) error {
	sriovNumvfs := path.Join(nicDir, "sriov_numvfs")
	sriovTotalvfs := path.Join(nicDir, "sriov_totalvfs")
	numvfs, err := fileutils2.FileGetContents(sriovNumvfs)
	if err != nil {
		return err
	}
	totalvfs, err := fileutils2.FileGetContents(sriovTotalvfs)
	if err != nil {
		return err
	}
	log.Errorf("numvfs %s total vfs %s", numvfs, totalvfs)
	if numvfs != totalvfs {
		return fileutils2.FilePutContents(sriovNumvfs, fmt.Sprintf("%s", totalvfs), false)
	}
	return nil
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

func getOvsOffloadNics(hostNics []HostNic) ([]*sOvsOffloadNicDevice, error) {
	sysfsNetDir := "/sys/class/net"
	files, err := ioutil.ReadDir(sysfsNetDir)
	if err != nil {
		return nil, errors.Wrap(err, "ioutil.ReadDir(sysfsNetDir)")
	}
	nics := []HostNic{}
	for i := 0; i < len(files); i++ {
		nicPath, err := filepath.EvalSymlinks(path.Join(sysfsNetDir, files[i].Name()))
		if err != nil {
			return nil, errors.Wrap(err, "filepath.EvalSymlinks nicPath")
		}
		if !strings.HasPrefix(nicPath, "/sys/devices/pci0000:") {
			continue
		}
		var idx int
		for idx = range hostNics {
			if hostNics[idx].Interface == files[i].Name() {
				nics = append(nics, hostNics[idx])
			}
		}
	}

	sriovNics := make([]*sOvsOffloadNicDevice, 0)
	for i := range nics {
		nicDir := path.Join(sysfsNetDir, nics[i].Interface, "device")
		err = ensureNumvfsEqualTotalvfs(nicDir)
		if err != nil {
			return nil, errors.Wrap(err, "ensureNumvfsEqualTotalvfs")
		}

		vfs, err := ioutil.ReadDir(nicDir)
		if err != nil {
			return nil, errors.Wrap(err, "ioutil.ReadDir")
		}

		pfPath, err := filepath.EvalSymlinks(nicDir)
		if err != nil {
			return nil, errors.Wrap(err, "filepath.EvalSymlinks pfPath")
		}
		pfBDF := path.Base(pfPath)

		// /sys/class/net/ens1f0/compat/devlink/mode
		devlinkPath := path.Join(sysfsNetDir, nics[i].Interface, "compat/devlink/mode")
		linkMode, err := fileutils2.FileGetContents(devlinkPath)
		if err != nil {
			return nil, errors.Wrap(err, "fileutils2.FileGetContents(devlinkPath)")
		}
		log.Infof("nic %s link mode %s", nics[i].Interface, linkMode)
		if strings.TrimSpace(linkMode) != "switchdev" {
			err = fileutils2.FilePutContents(devlinkPath, "switchdev\n", false)
			if err != nil {
				return nil, errors.Wrap(err, "fileutils2.FilePutContents linkMode")
			}
			for j := 0; j < len(vfs); j++ {
				if strings.HasPrefix(vfs[j].Name(), "virtfn") {
					vfPath, err := filepath.EvalSymlinks(path.Join(nicDir, vfs[j].Name()))
					if err != nil {
						return nil, errors.Wrap(err, "filepath.EvalSymlinks")
					}
					vfBDF := path.Base(vfPath)
					dev, err := detectPCIDevByAddrWithoutIOMMUGroup(vfBDF)
					if err != nil {
						return nil, errors.Wrap(err, "detectPCIDevByAddrWithoutIOMMUGroup")
					}
					err = dev.unbindDriver()
					if err != nil {
						return nil, errors.Wrap(err, "unbindDriver")
					}
				}
			}
		}

		// get interfaces
		// grep -e 'PCI_SLOT_NAME=.*04:00.1'  /sys/class/net/*/device/uevent
		outs, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c",
			fmt.Sprintf("grep -e PCI_SLOT_NAME=.*%s /sys/class/net/*/device/uevent", pfBDF)).Output()
		if err != nil {
			return nil, errors.Wrap(err, "procutils.NewRemoteCommandAsFarAsPossible grep PCI_SLOT_NAME")
		}
		vfNames := map[int]string{}
		for _, line := range strings.Split(string(outs), "\n") {
			line = strings.TrimSpace(line)
			segs := strings.Split(line, "/")
			if len(segs) < 5 {
				continue
			}
			vfName := segs[4]
			if vfName == nics[i].Interface {
				continue
			}
			// eg: pf0vf1
			portname, err := fileutils2.FileGetContents(fmt.Sprintf("/sys/class/net/%s/phys_port_name", vfName))
			if err != nil {
				return nil, errors.Wrap(err, "fileutils2.FileGetContents portname")
			}
			sp := strings.Split(portname, "vf")
			if len(sp) != 2 {
				return nil, errors.Errorf("%s bad portname %s", vfName, portname)
			}
			virtfn, err := strconv.Atoi(strings.TrimSpace(sp[1]))
			if err != nil {
				return nil, errors.Wrapf(err, "failed parse %s portname %s", vfName, portname)
			}
			vfNames[virtfn] = vfName
		}
		log.Infof("vfnames: %v", vfNames)

		for j := 0; j < len(vfs); j++ {
			if strings.HasPrefix(vfs[j].Name(), "virtfn") {
				virtfn, err := strconv.Atoi(vfs[j].Name()[len("virtfn"):])
				if err != nil {
					return nil, errors.Wrap(err, "strconv.Atoi virtfn")
				}
				vfPath, err := filepath.EvalSymlinks(path.Join(nicDir, vfs[j].Name()))
				if err != nil {
					return nil, errors.Wrap(err, "filepath.EvalSymlinks vfpath")
				}
				vfBDF := path.Base(vfPath)
				vfDev, err := detectSRIOVNicDevice(vfBDF)
				if err != nil {
					return nil, errors.Wrap(err, "detectSRIOVNicDevice")
				}
				// out, err := procutils.NewRemoteCommandAsFarAsPossible(
				// 	"ovs-vsctl", "--may-exist", "add-port", nics[i].Bridge, vfNames[virtfn]).Output()
				// if err != nil {
				// 	return nil, errors.Wrapf(err, " ovs-vsctl add-port %s %s failed: %s", nics[i].Bridge, vfNames[virtfn], out)
				// }

				sriovNics = append(sriovNics, NewSRIOVOffloadNicDevice(
					vfDev, api.NIC_TYPE, nics[i].Wire, nics[i].Interface, virtfn, vfNames[virtfn]),
				)
			}
		}
	}

	// ovs-vsctl set Open_vSwitch . other_config:hw-offload=true
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"ovs-vsctl", "set", "Open_vSwitch", ".", "other_config:hw-offload=true").Output()
	if err != nil {
		return nil, errors.Wrapf(err, "ovs enable hw offload failed: %s", out)
	}

	return sriovNics, nil
}

type sOvsOffloadNicDevice struct {
	*sSRIOVNicDevice

	interfaceName string
}

func NewSRIOVOffloadNicDevice(dev *PCIDevice, devType, wireId, pfName string, virtfn int, ifname string) *sOvsOffloadNicDevice {
	return &sOvsOffloadNicDevice{
		sSRIOVNicDevice: NewSRIOVNicDevice(dev, devType, wireId, pfName, virtfn),
		interfaceName:   ifname,
	}
}

func (dev *sOvsOffloadNicDevice) setInterfaceName(ifname string) {
	dev.interfaceName = ifname
}

func (dev *sOvsOffloadNicDevice) GetOvsOffloadInterfaceName() string {
	return dev.interfaceName
}
