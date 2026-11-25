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
	"strconv"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newAscendNPUManager())
}

type ascendNPUManager struct{}

func extractPartitionNumber(device string) (int, error) {
	re := regexp.MustCompile(`(\d+)$`)
	matches := re.FindStringSubmatch(device)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no partition number found in %s", device)
	}
	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (m *ascendNPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	npus := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		devMan := iDev.GetContainerDeviceManager()
		if _, ok := devMan.(*ascendNPUManager); !ok {
			continue
		}
		idx, err := extractPartitionNumber(dev.IsolatedDevice.Path)
		if err != nil {
			npus = append(npus, strconv.Itoa(idx))
		}

	}
	if len(npus) == 0 {
		return nil, nil
	}

	return []*runtimeapi.KeyValue{
		{
			Key:   "ASCEND_VISIBLE_DEVICES",
			Value: strings.Join(npus, ","),
		},
	}, nil
}

func newAscendNPUManager() *ascendNPUManager {
	return &ascendNPUManager{}
}

func (m *ascendNPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getAscendNpus(m, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE)
}

func (m *ascendNPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *ascendNPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return nil, nil, nil
}

func (m *ascendNPUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeAscendNpu
}

func (m *ascendNPUManager) GetDevType() string {
	return computeapi.NPU_TYPE
}

func (m *ascendNPUManager) GetSharingMode() string {
	return computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
}

type ascnedNPU struct {
	manager isolated_device.IContainerDeviceManager

	*BaseDevice
	memorySize    int
	chipsOnDevice uint
}

func (dev *ascnedNPU) GetMemorySize() int {
	return dev.memorySize
}

func (dev *ascnedNPU) GetChipsOnDevice() uint {
	if dev.chipsOnDevice == 0 {
		return 1
	}
	return dev.chipsOnDevice
}

func (dev *ascnedNPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

func parseNpuSmiChipCounts(output string) map[int]uint {
	counts := map[int]uint{}
	npuID := -1
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "NPU ID":
			id, err := strconv.Atoi(val)
			if err != nil {
				npuID = -1
				continue
			}
			npuID = id
		case "Chip Count":
			if npuID < 0 {
				continue
			}
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				continue
			}
			counts[npuID] = uint(n)
		}
	}
	return counts
}

// parseNpuSmiBoardInfo extracts the PCI address reported by
// `npu-smi info -t board`.
func parseNpuSmiBoardInfo(output string) string {
	var pciAddr string
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "PCIe Bus Info":
			pciAddr = value
		}
	}
	return pciAddr
}

func parseNpuSmiNames(output string) map[int]string {
	names := make(map[int]string)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue // header or non-data line
		}
		// Chip Name is the final column; only chip 0 represents the NPU.
		if fields[1] == "0" && !strings.EqualFold(fields[len(fields)-1], "Mcu") {
			names[id] = strings.Join(fields[3:], " ")
		}
	}
	return names
}

func parseNpuSmiHBMCapacity(output string) int {
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != "HBM Capacity(MB)" {
			continue
		}
		capacity, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err == nil && capacity > 0 {
			return capacity
		}
	}
	return 0
}

func getAscendNpus(m isolated_device.IContainerDeviceManager, sharingMode string) ([]isolated_device.IDevice, error) {
	devs := make([]isolated_device.IDevice, 0)
	chipCounts := map[int]uint{}
	chipOut, err := procutils.NewRemoteCommandAsFarAsPossible("npu-smi", "info", "-l").Output()
	if err != nil {
		log.Warningf("npu-smi info -l: %s", err)
		return nil, errors.Wrap(err, "npu-smi info -l")
	}
	chipCounts = parseNpuSmiChipCounts(string(chipOut))
	nameOut, err := procutils.NewRemoteCommandAsFarAsPossible("npu-smi", "info", "-m").Output()
	if err != nil {
		return nil, errors.Wrap(err, "npu-smi info -m")
	}
	npuNames := parseNpuSmiNames(string(nameOut))
	for npuId, chipCount := range chipCounts {
		// parse pci addr
		out, err := procutils.NewRemoteCommandAsFarAsPossible("npu-smi", "info", "-t", "board", "-i", strconv.Itoa(npuId)).Output()
		if err != nil {
			return nil, errors.Wrap(err, "npu-smi")
		}
		pciAddr := parseNpuSmiBoardInfo(string(out))
		if pciAddr == "" {
			continue
		}
		memOut, err := procutils.NewRemoteCommandAsFarAsPossible("npu-smi", "info", "-t", "memory", "-i", strconv.Itoa(npuId)).Output()
		if err != nil {
			return nil, errors.Wrapf(err, "npu-smi info -t memory -i %d", npuId)
		}
		hbmMem := parseNpuSmiHBMCapacity(string(memOut))
		pciOutput, err := isolated_device.GetPCIStrByAddr(pciAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", pciAddr)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])

		devPath := fmt.Sprintf("/dev/davinci%d", npuId)
		npuDev := &ascnedNPU{
			manager:       m,
			BaseDevice:    NewBaseDevice(dev, computeapi.NPU_TYPE, devPath, sharingMode, 1),
			chipsOnDevice: chipCount,
		}
		npuDev.SetModelName(npuNames[npuId])
		npuDev.memorySize = hbmMem

		devs = append(devs, npuDev)
	}

	if len(devs) == 0 {
		return nil, nil
	}

	return devs, nil
}
