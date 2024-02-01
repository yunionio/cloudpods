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
	"regexp"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	NetintCAASICReg   = regexp.MustCompile("T4\\d\\d-.*")
	NetintCAQuadraReg = regexp.MustCompile("Quadra.*")
)

const (
	NETINT_VENDOR_ID = "0000"
	NETINT_DEVICE_ID = "0000"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNetintDeviceManager(isolated_device.ContainerNetintCAASIC, NetintCAASICReg))
	isolated_device.RegisterContainerDeviceManager(newNetintDeviceManager(isolated_device.ContainerNetintCAQuadra, NetintCAQuadraReg))
}

type NetintDeviceInfo struct {
	Namespace    int    `json:"NameSpace"`
	DevicePath   string `json:"DevicePath"`
	Firmware     string `json:"Firmware"`
	Index        int    `json:"Index"`
	ModelNumber  string `json:"ModelNumber"`
	ProductName  string `json:"ProductName"`
	SerialNumber string `json:"SerialNumber"`
	UsedBytes    int    `json:"UsedBytes"`
	MaximumLBA   int    `json:"MaximumLBA"`
	PhysicalSize int    `json:"PhysicalSize"`
	SectorSize   int    `json:"SectorSize"`
}

type netintDeviceManager struct {
	devType       isolated_device.ContainerDeviceType
	devRegPattern *regexp.Regexp
}

func newNetintDeviceManager(devType isolated_device.ContainerDeviceType, reg *regexp.Regexp) *netintDeviceManager {
	return &netintDeviceManager{
		devType:       devType,
		devRegPattern: reg,
	}
}

func (m *netintDeviceManager) GetType() isolated_device.ContainerDeviceType {
	return m.devType
}

type NVMEListResult struct {
	Devices []*NetintDeviceInfo `json:"devices"`
}

func (m *netintDeviceManager) fetchNVMEDevices() ([]*NetintDeviceInfo, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", "nvme list -o json").Output()
	if err != nil {
		return nil, errors.Wrap(err, "get nvme device json output")
	}
	obj, err := jsonutils.Parse(out)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse %s", string(out))
	}
	output := new(NVMEListResult)
	if err := obj.Unmarshal(&output); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal to NetIntDeviceInfo: %s", obj.String())
	}
	result := make([]*NetintDeviceInfo, 0)
	for _, dev := range output.Devices {
		if !m.devRegPattern.MatchString(dev.ModelNumber) {
			continue
		}
		tmpDev := dev
		result = append(result, tmpDev)
	}
	return result, nil
}

func (m *netintDeviceManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	if err := CheckVirtualNumber(dev); err != nil {
		return nil, err
	}
	nvmeDevs, err := m.fetchNVMEDevices()
	if err != nil {
		return nil, errors.Wrap(err, "fetch nvme devices")
	}
	result := make([]isolated_device.IDevice, 0)
	for _, nvmeDev := range nvmeDevs {
		for i := 0; i < dev.VirtualNumber; i++ {
			newDev, err := m.newDeviceByIndex(nvmeDev, i)
			if err != nil {
				return nil, errors.Wrapf(err, "newDeviceByIndex %#v %d", nvmeDev, i)
			}
			result = append(result, newDev)
		}
	}
	return result, nil
}

func (m *netintDeviceManager) newDeviceByIndex(dev *NetintDeviceInfo, idx int) (*netintDevice, error) {
	devInfo := &isolated_device.PCIDevice{
		Addr:      fmt.Sprintf("%d-%d", dev.Index, idx),
		VendorId:  NETINT_VENDOR_ID,
		DeviceId:  NETINT_DEVICE_ID,
		ModelName: dev.ModelNumber,
	}
	nvmeDev := &netintDevice{
		BaseDevice: NewBaseDevice(devInfo, m.devType, dev.DevicePath),
		info:       dev,
	}
	return nvmeDev, nil
}

func (m *netintDeviceManager) NewContainerDevices(_ *hostapi.ContainerCreateInput, input *hostapi.ContainerDevice) ([]*runtimeapi.Device, error) {
	dev := input.IsolatedDevice
	if !fileutils2.Exists(dev.Path) {
		return nil, errors.Wrapf(httperrors.ErrNotFound, "device path %s doesn't exist", dev.Path)
	}

	charDevReg := regexp.MustCompile("(.*)n\\d+")
	charDevPath := charDevReg.FindAllStringSubmatch(dev.Path, -1)[0][1]

	ctrDevs := []*runtimeapi.Device{
		{
			HostPath:      dev.Path,
			ContainerPath: dev.Path,
			Permissions:   "rwm",
		},
		{
			HostPath:      charDevPath,
			ContainerPath: charDevPath,
			Permissions:   "rwm",
		},
	}
	return ctrDevs, nil
}

type netintDevice struct {
	*BaseDevice
	info *NetintDeviceInfo
}

func (d netintDevice) GetNVMESizeMB() int {
	return d.info.PhysicalSize / 1024 / 1024
}
