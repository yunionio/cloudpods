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

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

// The MPS /dev/shm is needed to allow MPS daemon health-checking
var shmPath = "/dev/shm"

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaMPSManager())
}

type nvidiaMPSManager struct{}

func newNvidiaMPSManager() *nvidiaMPSManager {
	return &nvidiaMPSManager{}
}

func (m *nvidiaMPSManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNvidiaMps
}

func (m *nvidiaMPSManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getNvidiaMPSGpus()
}

func (m *nvidiaMPSManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *nvidiaMPSManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return nil, nil, nil
}

func (m *nvidiaMPSManager) getMPSPipeDirectory() string {
	return options.HostOptions.CudaMPSPipeDirectory
}

func (m *nvidiaMPSManager) getSHMPath() string {
	return shmPath
}

func (m *nvidiaMPSManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	gpuIds := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		if isolated_device.ContainerDeviceType(dev.IsolatedDevice.DeviceType) != isolated_device.ContainerDeviceTypeNvidiaMps {
			continue
		}
		gpuIds = append(gpuIds, dev.IsolatedDevice.Path)
	}
	if len(gpuIds) == 0 {
		return nil, nil
	}

	return []*runtimeapi.KeyValue{
			{
				Key:   "CUDA_MPS_PIPE_DIRECTORY",
				Value: m.getMPSPipeDirectory(),
			},
			{
				Key:   "NVIDIA_VISIBLE_DEVICES",
				Value: strings.Join(gpuIds, ","),
			},
			{
				Key:   "NVIDIA_DRIVER_CAPABILITIES",
				Value: "all",
			},
		}, []*runtimeapi.Mount{
			{
				ContainerPath: m.getSHMPath(),
				HostPath:      m.getSHMPath(),
			},
			{
				ContainerPath: m.getMPSPipeDirectory(),
				HostPath:      m.getMPSPipeDirectory(),
			},
		}
}

type nvidiaMPS struct {
	*BaseDevice

	MemSizeMB        int
	MemTotalMB       int
	ThreadPercentage int
}

func (c *nvidiaMPS) GetNvidiaMpsMemoryLimit() int {
	return c.MemSizeMB
}

func (c *nvidiaMPS) GetNvidiaMpsMemoryTotal() int {
	return c.MemTotalMB
}

func (c *nvidiaMPS) GetNvidiaMpsThreadPercentage() int {
	return c.ThreadPercentage
}

func parseMemSize(memTotalStr string) (int, error) {
	if !strings.HasSuffix(memTotalStr, " MiB") {
		return -1, errors.Errorf("unknown mem string suffix")
	}
	memStr := strings.TrimSpace(strings.TrimSuffix(memTotalStr, " MiB"))
	return strconv.Atoi(memStr)
}

func getNvidiaMPSGpus() ([]isolated_device.IDevice, error) {
	devs := make([]isolated_device.IDevice, 0)
	// nvidia-smi  --query-gpu=gpu_uuid,gpu_name,gpu_bus_id,memory.total,compute_mode --format=csv
	// GPU-76aef7ff-372d-2432-b4b4-beca4d8d3400, Tesla P40, 00000000:00:08.0, 23040 MiB, Exclusive_Process
	out, err := procutils.NewRemoteCommandAsFarAsPossible("nvidia-smi", "--query-gpu=gpu_uuid,gpu_name,gpu_bus_id,memory.total,compute_mode", "--format=csv").Output()
	if err != nil {
		return nil, errors.Wrap(err, "nvidia-smi")
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "uuid") {
			continue
		}
		segs := strings.Split(line, ",")
		if len(segs) != 5 {
			log.Errorf("unknown nvidia-smi out line %s", line)
			continue
		}
		gpuId, gpuName, gpuPciAddr, memTotal, computeMode := strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1]), strings.TrimSpace(segs[2]), strings.TrimSpace(segs[3]), strings.TrimSpace(segs[4])
		if computeMode != "Exclusive_Process" {
			log.Warningf("gpu device %s compute mode %s, skip.", gpuId, computeMode)
			continue
		}
		memSize, err := parseMemSize(memTotal)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse memSize %s", memTotal)
		}

		pciOutput, err := isolated_device.GetPCIStrByAddr(gpuPciAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", gpuPciAddr)
		}
		for i := 0; i < options.HostOptions.CudaMPSReplicas; i++ {
			dev := isolated_device.NewPCIDevice2(pciOutput[0])
			gpuDev := &nvidiaMPS{
				BaseDevice:       NewBaseDevice(dev, isolated_device.ContainerDeviceTypeNvidiaMps, gpuId),
				MemSizeMB:        memSize / options.HostOptions.CudaMPSReplicas,
				MemTotalMB:       memSize,
				ThreadPercentage: 100 / options.HostOptions.CudaMPSReplicas,
			}
			gpuDev.SetModelName(gpuName)
			devAddr := gpuDev.GetAddr()
			gpuDev.SetAddr(fmt.Sprintf("%s-%d", devAddr, i), devAddr)
			devs = append(devs, gpuDev)
		}
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}
