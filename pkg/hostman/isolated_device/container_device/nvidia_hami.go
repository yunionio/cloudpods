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
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaHAMIManager())
}

type nvidiaHAMIManager struct {
}

func (m *nvidiaHAMIManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *nvidiaHAMIManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return nil, nil, nil
}

func newNvidiaHAMIManager() *nvidiaHAMIManager {
	return &nvidiaHAMIManager{}
}

func (m *nvidiaHAMIManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNvidiaHAMI
}

func (m *nvidiaHAMIManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return probeNvidiaGpus(computeapi.DEVICE_SHARING_MODE_HAMI, m)
}

func (m *nvidiaHAMIManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	gpuIds := []string{}
	memoryLimit := ""
	smLimit := ""
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}

		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		devMan := iDev.GetContainerDeviceManager()
		if _, ok := devMan.(*nvidiaHAMIManager); !ok {
			continue
		}
		gpuIds = append(gpuIds, dev.IsolatedDevice.Path)
		if memoryLimit == "" {
			memoryLimit = fmt.Sprintf("%dM", dev.IsolatedDevice.MemoryLimit)
		}
		if smLimit == "" && dev.IsolatedDevice.SmUtilLimit > 0 {
			smLimit = fmt.Sprintf("%d", dev.IsolatedDevice.SmUtilLimit)
		}
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
			{
				Key:   "LD_PRELOAD",
				Value: options.HostOptions.HAMICoreLibvgpuPath,
			},
			{
				Key:   "CUDA_DEVICE_MEMORY_LIMIT",
				Value: memoryLimit,
			},
		}...)
		if len(smLimit) > 0 {
			retEnvs = append(retEnvs, &runtimeapi.KeyValue{
				Key:   "CUDA_DEVICE_SM_LIMIT",
				Value: smLimit,
			})
		}
	}
	return retEnvs, []*runtimeapi.Mount{
		{
			ContainerPath: options.HostOptions.HAMICoreLibvgpuPath,
			HostPath:      options.HostOptions.HAMICoreLibvgpuPath,
			Readonly:      true,
		},
	}
}
