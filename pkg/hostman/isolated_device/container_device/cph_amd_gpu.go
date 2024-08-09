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
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newCphAMDGPUManager())
}

type cphAMDGPUManager struct{}

func newCphAMDGPUManager() *cphAMDGPUManager {
	return &cphAMDGPUManager{}
}

func (m *cphAMDGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *cphAMDGPUManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeCphAMDGPU
}

func (m *cphAMDGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	if !strings.HasPrefix(dev.Path, "/dev/dri/renderD") {
		return nil, errors.Errorf("device path %q doesn't start with /dev/dri/renderD", dev.Path)
	}
	if err := CheckVirtualNumber(dev); err != nil {
		return nil, err
	}
	gpuDevs := make([]isolated_device.IDevice, 0)
	for i := 0; i < dev.VirtualNumber; i++ {
		gpuDev, err := newCphAMDGPU(dev.Path, i)
		if err != nil {
			return nil, errors.Wrapf(err, "new CPH AMD GPU with index %d", i)
		}
		gpuDevs = append(gpuDevs, gpuDev)
	}
	return gpuDevs, nil
}

func (m *cphAMDGPUManager) getDeviceHostPathByAddr(dev *hostapi.ContainerDevice) (string, error) {
	return dev.IsolatedDevice.Path, nil
}

func (m *cphAMDGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	hostPath, err := m.getDeviceHostPathByAddr(dev)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get device host path")
	}
	cDev := &runtimeapi.Device{
		ContainerPath: "/dev/dri/renderD128",
		HostPath:      hostPath,
		Permissions:   "rwm",
	}
	return []*runtimeapi.Device{cDev}, nil, nil
}

func (m *cphAMDGPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return nil, nil
}

type cphAMDGPU struct {
	*BaseDevice
}

func newCphAMDGPU(devPath string, index int) (*cphAMDGPU, error) {
	dev, err := newPCIGPURenderBaseDevice(devPath, index, isolated_device.ContainerDeviceTypeCphAMDGPU)
	if err != nil {
		return nil, errors.Wrap(err, "new PCIGPURenderBaseDevice")
	}
	return &cphAMDGPU{
		BaseDevice: dev,
	}, nil
}
