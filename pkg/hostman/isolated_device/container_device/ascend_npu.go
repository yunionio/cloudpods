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
	memorySize int
}

func (dev *ascnedNPU) GetMemorySize() int {
	return dev.memorySize
}

func (dev *ascnedNPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

func getAscendNpus(m isolated_device.IContainerDeviceManager, sharingMode string) ([]isolated_device.IDevice, error) {
	devs := make([]isolated_device.IDevice, 0)
	// Show all device's topology information
	out, err := procutils.NewRemoteCommandAsFarAsPossible("npu-smi", "info").Output()
	if err != nil {
		return nil, errors.Wrap(err, "npu-smi")
	}
	lines := strings.Split(string(out), "\n")
	for i := 6; i < len(lines); i += 3 {
		if !strings.HasPrefix(lines[i], "|") {
			continue
		}
		if len(lines) <= (i + 1) {
			return nil, errors.Errorf("failed parse npu-smi unknown chip line")
		}

		fields := strings.Fields(lines[i])
		if len(fields) < 3 {
			return nil, errors.Errorf("failed parse npu-smi unknown npu line")
		}

		log.Debugf("fields %v", fields)
		strNpuID := fields[1]
		npuId, err := strconv.Atoi(strNpuID)
		if err != nil {
			log.Warningf("failed parse npuid %s: %s. break", strNpuID, err)
			break
		}
		npuName := fields[2]
		devPath := fmt.Sprintf("/dev/davinci%d", npuId)

		fileds2 := strings.Fields(lines[i+1])
		if len(fileds2) < 12 {
			return nil, errors.Errorf("failed parse npu-smi unknonw chip line get busid, memory size")
		}
		log.Debugf("fileds2 %v", fileds2)
		busID := fileds2[3]
		hbmMemStr := fileds2[11]
		hbmMem, err := strconv.Atoi(strings.TrimSpace(hbmMemStr))
		if err != nil {
			return nil, errors.Errorf("failed parse npu-smi unknonw chip line get hbm memory size %s: %s. break", hbmMemStr, err)
		}

		pciOutput, err := isolated_device.GetPCIStrByAddr(busID)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", busID)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])
		npuDev := &ascnedNPU{
			manager:    m,
			BaseDevice: NewBaseDevice(dev, computeapi.NPU_TYPE, devPath, sharingMode, 1),
		}
		npuDev.SetModelName(npuName)
		npuDev.memorySize = hbmMem

		devs = append(devs, npuDev)
	}

	if len(devs) == 0 {
		return nil, nil
	}

	return devs, nil
}
