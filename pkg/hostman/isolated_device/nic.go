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
		nicType, err := fileutils2.FileGetContents(path.Join(sysfsNetDir, nics[i].Interface, "type"))
		if err != nil {
			return nil, errors.Wrap(err, "failed get nic type")
		}
		var isInfinibandNic = false
		if strings.TrimSpace(nicType) == "32" {
			// include/uapi/linux/if_arp.h
			// #define ARPHRD_INFINIBAND 32		/* InfiniBand			*/
			isInfinibandNic = true
		}

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
				vfDev, err := detectSRIOVDevice(vfBDF)
				if err != nil {
					return nil, err
				}
				sriovNics = append(sriovNics, NewSRIOVNicDevice(vfDev, api.NIC_TYPE, nics[i].Wire, nics[i].Interface, virtfn, isInfinibandNic))
			}
		}
	}
	return sriovNics, nil
}

func NewSRIOVNicDevice(dev *PCIDevice, devType, wireId, pfName string, virtfn int, isInfinibandNic bool) *sSRIOVNicDevice {
	return &sSRIOVNicDevice{
		sSRIOVBaseDevice: newSRIOVBaseDevice(dev, devType),
		WireId:           wireId,
		pfName:           pfName,
		virtfn:           virtfn,
		isInfinibandNic:  isInfinibandNic,
	}
}

type sSRIOVNicDevice struct {
	*sSRIOVBaseDevice

	WireId          string
	pfName          string
	virtfn          int
	isInfinibandNic bool
}

func (dev *sSRIOVNicDevice) GetPfName() string {
	return dev.pfName
}

func (dev *sSRIOVNicDevice) IsInfinibandNic() bool {
	return dev.isInfinibandNic
}

func (dev *sSRIOVNicDevice) GetVirtfn() int {
	return dev.virtfn
}

func (dev *sSRIOVNicDevice) GetWireId() string {
	return dev.WireId
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
			err = fileutils2.FilePutContents(devlinkPath, "switchdev\n", false)
			if err != nil {
				return nil, errors.Wrap(err, "fileutils2.FilePutContents linkMode")
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
				vfDev, err := detectSRIOVDevice(vfBDF)
				if err != nil {
					return nil, errors.Wrap(err, "detectSRIOVDevice")
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
		sSRIOVNicDevice: NewSRIOVNicDevice(dev, devType, wireId, pfName, virtfn, false),
		interfaceName:   ifname,
	}
}

func (dev *sOvsOffloadNicDevice) GetOvsOffloadInterfaceName() string {
	return dev.interfaceName
}
