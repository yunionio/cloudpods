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
	"fmt"
	"os"
	"path"
	"strings"

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

func (c BaseDevice) GetNumaNode() (int, error) {
	return -1, nil
}

func CheckVirtualNumber(dev *isolated_device.ContainerDevice) error {
	if dev.VirtualNumber <= 0 {
		return errors.Errorf("virtual_number must > 0")
	}
	return nil
}

func getGPUPCIAddr(linkPartName string) (string, error) {
	if !strings.HasPrefix(linkPartName, "pci-") {
		return "", errors.Errorf("wrong link name: %s", linkPartName)
	}
	segs := strings.Split(linkPartName, "-")
	if len(segs) < 3 {
		return "", errors.Errorf("%s: segments length is less than 3 after splited by -", linkPartName)
	}
	fullAddr := segs[1]
	return fullAddr, nil
}

func newPCIGPURenderBaseDevice(devPath string, index int, devType isolated_device.ContainerDeviceType) (*BaseDevice, error) {
	dir := "/dev/dri/by-path/"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "read dir")
	}
	for _, entry := range entries {
		entryName := entry.Name()
		fp := path.Join(dir, entryName)
		linkPath, err := os.Readlink(fp)
		if err != nil {
			return nil, errors.Wrapf(err, "read link of %s", entry.Name())
		}
		linkDevPath := path.Join(dir, linkPath)
		if linkDevPath == devPath {
			// get pci address
			if !strings.HasSuffix(entryName, "-render") {
				return nil, errors.Errorf("%s isn't render device", devPath)
			}
			pciAddr, err := getGPUPCIAddr(entryName)
			if err != nil {
				return nil, errors.Wrapf(err, "get pci address of %s", devPath)
			}
			pciOutput, err := isolated_device.GetPCIStrByAddr(pciAddr)
			if err != nil {
				return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", pciAddr)
			}
			dev := isolated_device.NewPCIDevice2(pciOutput[0])
			dev.Addr = fmt.Sprintf("%s-%d", dev.Addr, index)
			return NewBaseDevice(dev, devType, devPath), nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "%s doesn't exist in %s", devPath, dir)
}
