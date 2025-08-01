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
	"strconv"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

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
	return probeNvidiaGpus()
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
		types := sets.NewString(
			string(isolated_device.ContainerDeviceTypeNvidiaGpu),
			string(isolated_device.ContainerDeviceTypeNvidiaGpuShare),
		)
		if !types.Has(dev.IsolatedDevice.DeviceType) {
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

type nvidiaGPU struct {
	*BaseDevice

	memSize     int
	gpuIndex    int
	deviceMinor int
}

func (dev *nvidiaGPU) GetNvidiaDevMemSize() int {
	return dev.memSize
}

func (dev *nvidiaGPU) GetNvidiaDevIndex() string {
	return fmt.Sprintf("%d", dev.gpuIndex)
}

func (dev *nvidiaGPU) GetIndex() int {
	return dev.gpuIndex
}

func (dev *nvidiaGPU) GetDeviceMinor() int {
	return dev.deviceMinor
}

func probeNvidiaGpus() ([]isolated_device.IDevice, error) {
	if nvidiaGpuUsages != nil {
		res := make([]isolated_device.IDevice, 0)
		for pciAddr, dev := range nvidiaGpuUsages {
			if dev.Used {
				continue
			}
			res = append(res, nvidiaGpuUsages[pciAddr].nvidiaGPU)
		}
		nvidiaGpuUsages = nil
		return res, nil
	}

	devs, err := getNvidiaGPUs()
	if err != nil {
		return nil, err
	}
	res := make([]isolated_device.IDevice, 0)
	for i := range devs {
		res = append(res, devs[i])
	}
	return res, nil
}

func getNvidiaGPUs() ([]*nvidiaGPU, error) {
	devs := make([]*nvidiaGPU, 0)
	// nvidia-smi --query-gpu=gpu_uuid,gpu_name,gpu_bus_id --format=csv
	// uuid, name, pci.bus_id
	// GPU-bc1a3bb9-55cb-8c52-c374-4f8b4f388a20, NVIDIA A800-SXM4-80GB, 00000000:10:00.0

	// nvidia-smi  --query-gpu=gpu_uuid,gpu_name,gpu_bus_id,memory.total,compute_mode --format=csv
	out, err := procutils.NewRemoteCommandAsFarAsPossible("nvidia-smi", "--query-gpu=gpu_uuid,gpu_name,gpu_bus_id,compute_mode,memory.total,index", "--format=csv").Output()
	if err != nil {
		return nil, errors.Wrap(err, "nvidia-smi")
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "uuid") {
			continue
		}
		segs := strings.Split(line, ",")
		if len(segs) != 6 {
			log.Errorf("unknown nvidia-smi out line %s", line)
			continue
		}
		gpuId, gpuName, gpuPciAddr, computeMode, memTotal, index := strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1]), strings.TrimSpace(segs[2]), strings.TrimSpace(segs[3]), strings.TrimSpace(segs[4]), strings.TrimSpace(segs[5])
		if computeMode != "Default" {
			log.Warningf("gpu device %s compute mode %s, skip.", gpuId, computeMode)
			continue
		}
		indexInt, err := parseInt(index)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse index %s", index)
		}
		memSize, err := parseMemSize(memTotal)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse memSize %s", memTotal)
		}

		pciOutput, err := isolated_device.GetPCIStrByAddr(gpuPciAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", gpuPciAddr)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])

		driverInfoPath := fmt.Sprintf("/proc/driver/nvidia/gpus/0000:%s/information", dev.Addr)
		driverContent, err := procutils.NewRemoteCommandAsFarAsPossible("cat", driverInfoPath).Output()
		if err != nil {
			return nil, errors.Wrapf(err, "failed get driver content from: %s", driverInfoPath)
		}
		driverInfo, err := parseNvidiaGPUDriverInformation(string(driverContent))
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse driver content from: %s", driverInfoPath)
		}

		gpuDev := &nvidiaGPU{
			BaseDevice:  NewBaseDevice(dev, isolated_device.ContainerDeviceTypeNvidiaGpu, gpuId),
			memSize:     memSize,
			gpuIndex:    indexInt,
			deviceMinor: driverInfo.DeviceMinor,
		}
		gpuDev.SetModelName(gpuName)
		devs = append(devs, gpuDev)
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}

type NvidiaGPUDriverInformation struct {
	Model       string
	IRQ         int
	UUID        string
	VideoBIOS   string
	BusType     string
	DMASize     string
	DMAMask     string
	BusLocation string
	DeviceMinor int
	Firmware    string
	Excluded    bool
}

// parseNvidiaGPUDriverInformation 解析下面文件的内容
// cat /proc/driver/nvidia/gpus/0000\:61\:00.0/information
// Model:           NVIDIA GeForce RTX 4060
// IRQ:             483
// GPU UUID:        GPU-2e1ab7a2-fda6-8b93-eba2-fa59e6135199
// Video BIOS:      95.07.36.00.04
// Bus Type:        PCIe
// DMA Size:        47 bits
// DMA Mask:        0x7fffffffffff
// Bus Location:    0000:61:00.0
// Device Minor:    0
// GPU Firmware:    570.133.07
// GPU Excluded:    No
func parseNvidiaGPUDriverInformation(content string) (*NvidiaGPUDriverInformation, error) {
	info := &NvidiaGPUDriverInformation{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 key: value 格式
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Model":
			info.Model = value
		case "IRQ":
			if irq, err := parseInt(value); err != nil {
				return nil, errors.Wrapf(err, "failed parse IRQ %s", value)
			} else {
				info.IRQ = irq
			}
		case "GPU UUID":
			info.UUID = value
		case "Video BIOS":
			info.VideoBIOS = value
		case "Bus Type":
			info.BusType = value
		case "DMA Size":
			info.DMASize = value
		case "DMA Mask":
			info.DMAMask = value
		case "Bus Location":
			info.BusLocation = value
		case "Device Minor":
			if minor, err := parseInt(value); err != nil {
				return nil, errors.Wrapf(err, "failed parse Device Minor %s", value)
			} else {
				info.DeviceMinor = minor
			}
		case "GPU Firmware":
			info.Firmware = value
		case "GPU Excluded":
			info.Excluded = (value != "No")
		}
	}

	return info, nil
}

func parseInt(s string) (int, error) {
	stringValue := strings.TrimSpace(s)
	return strconv.Atoi(stringValue)
}
