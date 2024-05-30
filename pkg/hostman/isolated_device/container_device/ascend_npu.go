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
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newAscendNPUManager())
}

type ascendNPUManager struct{}

func (m *ascendNPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	npus := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		if isolated_device.ContainerDeviceType(dev.IsolatedDevice.DeviceType) != isolated_device.ContainerDeviceTypeAscendNpu {
			continue
		}
		npus = append(npus, dev.IsolatedDevice.Path)
	}
	if len(npus) == 0 {
		return nil, nil
	}

	var (
		ASCEND_TOOLKIT_HOME = "/usr/local/Ascend/ascend-toolkit/latest"
		LD_LIBRARY_PATH     = fmt.Sprintf("/usr/local/Ascend/driver/lib64:/usr/local/Ascend/driver/lib64/common:/usr/local/Ascend/driver/lib64/driver:"+
			"%s/lib64:%s/lib64/plugin/opskernel:%s/lib64/plugin/nnengine", ASCEND_TOOLKIT_HOME, ASCEND_TOOLKIT_HOME, ASCEND_TOOLKIT_HOME)
		ASCEND_AICPU_PATH = ASCEND_TOOLKIT_HOME
		ASCEND_OPP_PATH   = fmt.Sprintf("%s/opp", ASCEND_TOOLKIT_HOME)
		TOOLCHAIN_HOME    = fmt.Sprintf("%s/toolkit", ASCEND_TOOLKIT_HOME)
		ASCEND_HOME_PATH  = ASCEND_AICPU_PATH
	)

	return []*runtimeapi.KeyValue{
			{
				Key:   "ASCEND_TOOLKIT_HOME",
				Value: ASCEND_TOOLKIT_HOME,
			}, {
				Key:   "LD_LIBRARY_PATH",
				Value: LD_LIBRARY_PATH,
			}, {
				Key:   "ASCEND_AICPU_PATH",
				Value: ASCEND_AICPU_PATH,
			}, {
				Key:   "ASCEND_OPP_PATH",
				Value: ASCEND_OPP_PATH,
			}, {
				Key:   "TOOLCHAIN_HOME",
				Value: TOOLCHAIN_HOME,
			}, {
				Key:   "ASCEND_HOME_PATH",
				Value: ASCEND_HOME_PATH,
			},
		}, []*runtimeapi.Mount{
			{
				ContainerPath: "/usr/local/Ascend",
				HostPath:      "/usr/local/Ascend",
				Readonly:      true,
			},
			{
				ContainerPath: "/usr/local/dcmi",
				HostPath:      "/usr/local/dcmi",
				Readonly:      true,
			},
			{
				ContainerPath: "/usr/local/bin/npu-smi",
				HostPath:      "/usr/local/bin/npu-smi",
				Readonly:      true,
			},
		}
}

func newAscendNPUManager() *ascendNPUManager {
	return &ascendNPUManager{}
}

func (m *ascendNPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getAscendNpus()
}

func (m *ascendNPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *ascendNPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return []*runtimeapi.Device{
			&runtimeapi.Device{
				ContainerPath: dev.IsolatedDevice.Path,
				HostPath:      dev.IsolatedDevice.Path,
				Permissions:   "rwm",
			},
		}, []*runtimeapi.Device{
			&runtimeapi.Device{
				ContainerPath: "/dev/davinci_manager",
				HostPath:      "/dev/davinci_manager",
				Permissions:   "rwm",
			},
			&runtimeapi.Device{
				ContainerPath: "/dev/devmm_svm",
				HostPath:      "/dev/devmm_svm",
				Permissions:   "rwm",
			},
			&runtimeapi.Device{
				ContainerPath: "/dev/hisi_hdc",
				HostPath:      "/dev/hisi_hdc",
				Permissions:   "rwm",
			},
		}, nil
}

func (m *ascendNPUManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeAscendNpu
}

type ascnedNPU struct {
	*BaseDevice
}

func getAscendNpus() ([]isolated_device.IDevice, error) {
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
		if len(fileds2) < 4 {
			return nil, errors.Errorf("failed parse npu-smi unknonw chip line get busid")
		}
		log.Debugf("fileds2 %v", fileds2)
		busID := fileds2[3]
		pciOutput, err := isolated_device.GetPCIStrByAddr(busID)
		if err != nil {
			return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", busID)
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])
		npuDev := &ascnedNPU{
			BaseDevice: NewBaseDevice(dev, isolated_device.ContainerDeviceTypeAscendNpu, devPath),
		}
		npuDev.SetModelName(npuName)

		devs = append(devs, npuDev)
	}

	if len(devs) == 0 {
		return nil, nil
	}

	return devs, nil
}
