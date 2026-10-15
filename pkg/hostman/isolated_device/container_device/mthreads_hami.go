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

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const mthreadsSgpuPolicyWeak = 1

func init() {
	isolated_device.RegisterContainerDeviceManager(newMthreadsGPUHamiManager())
}

type mthreadsGPUHamiManager struct{}

func newMthreadsGPUHamiManager() *mthreadsGPUHamiManager {
	return &mthreadsGPUHamiManager{}
}

func (m *mthreadsGPUHamiManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeMthreadsGpuHami
}

func (m *mthreadsGPUHamiManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	devs, err := getMthreadsGPUs(m, computeapi.DEVICE_SHARING_MODE_HAMI)
	if err != nil {
		return nil, err
	}
	for _, dev := range devs {
		gpu, ok := dev.(*mthreadsGPU)
		if !ok {
			continue
		}
		configureMthreadsSgpu(gpu.GetIndex())
	}
	return devs, nil
}

func (m *mthreadsGPUHamiManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *mthreadsGPUHamiManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return newMthreadsContainerDevices(dev)
}

func (m *mthreadsGPUHamiManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	indices := make([]string, 0)
	memoryLimit := 0
	smLimit := 0
	seenLimit := false
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		if iDev == nil {
			continue
		}
		if _, ok := iDev.GetContainerDeviceManager().(*mthreadsGPUHamiManager); !ok {
			continue
		}
		if dev.IsolatedDevice.Path != "" {
			indices = append(indices, dev.IsolatedDevice.Path)
		} else if dev.IsolatedDevice.Index >= 0 {
			indices = append(indices, strconv.Itoa(dev.IsolatedDevice.Index))
		}
		if !seenLimit {
			memoryLimit = dev.IsolatedDevice.MemoryLimit
			smLimit = dev.IsolatedDevice.SmUtilLimit
			seenLimit = true
		}
	}
	envs, mounts := buildMthreadsExtraConfigures(indices)
	if len(envs) == 0 {
		return nil, nil
	}
	envs = append(envs, buildMthreadsHamiQosEnvs(memoryLimit, smLimit)...)
	return envs, mounts
}

func configureMthreadsSgpu(index int) {
	writeMthreadsSgpuProc(index, "policy", mthreadsSgpuPolicyWeak)
	writeMthreadsSgpuProc(index, "max_inst", computeapi.MthreadsSgpuMaxInstances)
}

func writeMthreadsSgpuProc(index int, node string, value int) {
	procPath := fmt.Sprintf("/proc/sgpu_km/%d/%s", index, node)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", fmt.Sprintf("echo %d > %s", value, procPath)).Output()
	if err != nil {
		log.Warningf("mthreads sgpu set %s=%d: %v %s", procPath, value, err, out)
	}
}
