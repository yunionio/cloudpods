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
	"path"
	"path/filepath"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaGPUShareManager())
}

type nvidiaGPUShareManager struct {
}

func (m *nvidiaGPUShareManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return nil, nil, nil
}

func (m *nvidiaGPUShareManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	gpuIds := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		devMan := iDev.GetContainerDeviceManager()
		if _, ok := devMan.(*nvidiaGPUShareManager); !ok {
			continue
		}

		gpuIds = append(gpuIds, dev.IsolatedDevice.Path)
	}
	if len(gpuIds) == 0 {
		return nil, nil
	}
	retEnvs := []*runtimeapi.KeyValue{}
	if len(gpuIds) > 0 {
		retEnvs = append(retEnvs, []*runtimeapi.KeyValue{
			{
				Key:   "NVIDIA_VISIBLE_DEVICES",
				Value: strings.Join(gpuIds, ","),
			},
			{
				Key:   "NVIDIA_DRIVER_CAPABILITIES",
				Value: "all",
			},
		}...)
	}
	return retEnvs, nil
}

func newNvidiaGPUShareManager() *nvidiaGPUShareManager {
	return &nvidiaGPUShareManager{}
}

func (m *nvidiaGPUShareManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNvidiaGpuShare
}

func (m *nvidiaGPUShareManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *nvidiaGPUShareManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	if !strings.HasPrefix(dev.Path, "/dev/dri/renderD") {
		return nil, errors.Errorf("device path %q doesn't start with /dev/dri/renderD", dev.Path)
	}
	if err := CheckVirtualNumber(dev); err != nil {
		return nil, err
	}

	gpuDev, err := m.newNvidiaGpuShare(dev.Path, dev.VirtualNumber)
	if err != nil {
		return nil, errors.Wrap(err, "new CPH AMD GPU")
	}
	return []isolated_device.IDevice{gpuDev}, nil
}

type nvidiaGpuShareDev struct {
	nvidiaGPU
	manager *nvidiaGPUShareManager

	CardPath   string
	RenderPath string
}

func (dev *nvidiaGpuShareDev) GetCardPath() string {
	return dev.CardPath
}

func (dev *nvidiaGpuShareDev) GetRenderPath() string {
	return dev.RenderPath
}

func (dev *nvidiaGpuShareDev) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

type nvidiaGpuUsage struct {
	*nvidiaGPU

	Used bool
}

var nvidiaGpuUsages map[string]*nvidiaGpuUsage = nil

func getNvidiaGpuUsage() (map[string]*nvidiaGpuUsage, error) {
	if nvidiaGpuUsages != nil {
		return nvidiaGpuUsages, nil
	}
	devs, err := getNvidiaGPUs(api.DEVICE_SHARING_MODE_UNLIMITED, nil)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	gpuUsages := map[string]*nvidiaGpuUsage{}
	for i := range devs {
		gpuUsages[devs[i].GetAddr()] = &nvidiaGpuUsage{
			nvidiaGPU: devs[i],
			Used:      false,
		}
	}
	nvidiaGpuUsages = gpuUsages
	return nvidiaGpuUsages, nil
}

func (m *nvidiaGPUShareManager) newNvidiaGpuShare(devPath string, virtualNumber int) (*nvidiaGpuShareDev, error) {
	devUsages, err := getNvidiaGpuUsage()
	if err != nil {
		return nil, errors.Wrap(err, "getNvidiaGpuUsage")
	}

	dev, err := NewPCIGPURenderBaseDevice(devPath, virtualNumber, api.GPU_TYPE, api.DEVICE_SHARING_MODE_UNLIMITED)
	if err != nil {
		return nil, errors.Wrap(err, "NewPCIGPURenderBaseDevice")
	}
	devAddr := dev.GetOriginAddr()
	cardPath := path.Join("/dev/dri/by-path", fmt.Sprintf("pci-0000:%s-card", devAddr))
	cardLinkPath, err := filepath.EvalSymlinks(cardPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read link of %s", cardPath)
	}
	nvidiaGPUDev, ok := devUsages[devAddr]
	if !ok {
		return nil, errors.Errorf("newNvidiaGpuShare dev addr not found %s", devAddr)
	}
	devUsages[devAddr].Used = true
	dev.SetDevicePath(nvidiaGPUDev.Path)

	return &nvidiaGpuShareDev{
		manager: m,
		nvidiaGPU: nvidiaGPU{
			BaseDevice:  dev,
			memSize:     devUsages[devAddr].memSize,
			gpuIndex:    devUsages[devAddr].gpuIndex,
			deviceMinor: devUsages[devAddr].deviceMinor,
		},
		CardPath:   cardLinkPath,
		RenderPath: devPath,
	}, nil
}
