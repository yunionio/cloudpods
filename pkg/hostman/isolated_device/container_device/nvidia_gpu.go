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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaGPUManager())
}

type nvidiaGPUManager struct{}

func newNvidiaGPUManager() *nvidiaGPUManager {
	return &nvidiaGPUManager{}
}

func (m *nvidiaGPUManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNvidiaGpu
}

func (m *nvidiaGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getNvidiaGPUs()
}

func (m *nvidiaGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *nvidiaGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return nil, nil, nil
}

func (m *nvidiaGPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	gpuIds := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		if isolated_device.ContainerDeviceType(dev.IsolatedDevice.DeviceType) != isolated_device.ContainerDeviceTypeNvidiaGpu {
			continue
		}
		gpuIds = append(gpuIds, dev.IsolatedDevice.Path)
	}
	if len(gpuIds) == 0 {
		return nil, nil
	}

	return []*runtimeapi.KeyValue{
		{
			Key:   "NVIDIA_VISIBLE_DEVICES",
			Value: strings.Join(gpuIds, ","),
		},
		{
			Key:   "NVIDIA_DRIVER_CAPABILITIES",
			Value: "all",
		},
	}, nil
}

type nvidiaGPU struct {
	*BaseDevice
}

func getNvidiaGPUs() ([]isolated_device.IDevice, error) {
	devs := make([]isolated_device.IDevice, 0)
	// nvidia-smi --query-gpu=gpu_uuid,gpu_name,gpu_bus_id --format=csv
	// uuid, name, pci.bus_id
	// GPU-bc1a3bb9-55cb-8c52-c374-4f8b4f388a20, NVIDIA A800-SXM4-80GB, 00000000:10:00.0

	// nvidia-smi  --query-gpu=gpu_uuid,gpu_name,gpu_bus_id,memory.total,compute_mode --format=csv
	out, err := procutils.NewRemoteCommandAsFarAsPossible("nvidia-smi", "--query-gpu=gpu_uuid,gpu_name,gpu_bus_id,compute_mode", "--format=csv").Output()
	if err != nil {
		return nil, errors.Wrap(err, "nvidia-smi")
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "uuid") {
			continue
		}
		segs := strings.Split(line, ",")
		if len(segs) != 4 {
			log.Errorf("unknown nvidia-smi out line %s", line)
			continue
		}
		gpuId, gpuName, gpuPciAddr, computeMode := strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1]), strings.TrimSpace(segs[2]), strings.TrimSpace(segs[3])
		if computeMode != "Default" {
			log.Warningf("gpu device %s compute mode %s, skip.", gpuId, computeMode)
			continue
		}

		pciOutput, err := isolated_device.GetPCIStrByAddr(gpuPciAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", gpuPciAddr)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])
		gpuDev := &nvidiaGPU{
			BaseDevice: NewBaseDevice(dev, isolated_device.ContainerDeviceTypeNvidiaGpu, gpuId),
		}
		gpuDev.SetModelName(gpuName)

		devs = append(devs, gpuDev)
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}
