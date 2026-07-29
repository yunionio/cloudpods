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
	"regexp"
	"strconv"
	"strings"
	"sync"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newHygonDCUHamiManager())
}

type hygonDCUHamiManager struct {
	*hygonDCUManager

	vdevMu    sync.Mutex
	allocated map[string]int // containerDeviceId -> vdevIndex
	vdevInUse map[int]string // vdevIndex -> containerDeviceId
}

func newHygonDCUHamiManager() *hygonDCUHamiManager {
	return &hygonDCUHamiManager{
		hygonDCUManager: newHygonDCUManager(),
		allocated:       make(map[string]int),
		vdevInUse:       make(map[int]string),
	}
}

func (m *hygonDCUHamiManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeHygonDcuHami
}

func (m *hygonDCUHamiManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getHygonDCUs(m, computeapi.DEVICE_SHARING_MODE_HAMI)
}

func (m *hygonDCUHamiManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	if len(devs) == 0 {
		return nil, nil
	}
	dev := devs[0]
	if dev.IsolatedDevice == nil {
		return nil, nil
	}
	iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
	if iDev == nil {
		return nil, nil
	}
	dcuDev, ok := iDev.(*hygonDCU)
	if !ok {
		return nil, nil
	}

	memLimitMiB := dev.IsolatedDevice.MemoryLimit
	if memLimitMiB <= 0 {
		memLimitMiB = dcuDev.GetMemorySize()
	}
	smLimit := dev.IsolatedDevice.SmUtilLimit
	if smLimit <= 0 {
		smLimit = 100
	}
	computeUnits := dcuDev.GetComputeUnits()
	if computeUnits <= 0 {
		computeUnits = 60
	}
	cuCount := computeUnits * smLimit / 100
	if cuCount <= 0 {
		cuCount = 1
	}

	vdevIdx, err := m.ensureVdev(dev.IsolatedDevice.Id, dcuDev.GetIndex(), memLimitMiB, cuCount)
	if err != nil {
		log.Errorf("ensure hygon vdev for device %s: %v", dev.IsolatedDevice.Id, err)
		return nil, nil
	}

	envs := buildHygonRuntimeEnvs([]string{strconv.Itoa(vdevIdx)})
	mounts := buildHygonRuntimeMounts()

	vdevConfHost := path.Join(options.HostOptions.HygonVdevConfDir, fmt.Sprintf("vdev%d.conf", vdevIdx))
	vdevConfContainer := path.Join("/etc/vdev/docker", fmt.Sprintf("vdev%d.conf", vdevIdx))
	if hygonPathExists(vdevConfHost) {
		mounts = append(mounts, &runtimeapi.Mount{
			ContainerPath: vdevConfContainer,
			HostPath:      vdevConfHost,
			Readonly:      true,
		})
	}
	return envs, mounts
}

func (m *hygonDCUHamiManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return m.hygonDCUManager.NewContainerDevices(input, dev)
}

func (m *hygonDCUHamiManager) ensureVdev(containerDevId string, physIdx, memMiB, computeUnits int) (int, error) {
	m.vdevMu.Lock()
	defer m.vdevMu.Unlock()

	if vdevIdx, ok := m.allocated[containerDevId]; ok {
		return vdevIdx, nil
	}

	vdevIdx, err := m.findOrCreateVdev(physIdx, memMiB, computeUnits)
	if err != nil {
		return -1, err
	}
	m.allocated[containerDevId] = vdevIdx
	m.vdevInUse[vdevIdx] = containerDevId
	return vdevIdx, nil
}

func (m *hygonDCUHamiManager) findOrCreateVdev(physIdx, memMiB, computeUnits int) (int, error) {
	existing, err := m.listAvailableVdevIndices(physIdx)
	if err != nil {
		log.Warningf("list hygon vdev indices: %v", err)
	}
	for _, idx := range existing {
		if _, used := m.vdevInUse[idx]; !used {
			return idx, nil
		}
	}

	hySmiPath := options.HostOptions.HygonHySmiPath
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		hySmiPath, "virtual", "-create-vdevices", "1",
		"-d", strconv.Itoa(physIdx),
		"-vdevice-compute-units", strconv.Itoa(computeUnits),
		"-vdevice-memory-size", strconv.Itoa(memMiB),
	).Output()
	if err != nil {
		return -1, errors.Wrapf(err, "create vdev on dcu %d: %s", physIdx, out)
	}

	created, err := m.listAvailableVdevIndices(physIdx)
	if err != nil || len(created) == 0 {
		return physIdx, nil
	}
	for _, idx := range created {
		if _, used := m.vdevInUse[idx]; !used {
			return idx, nil
		}
	}
	return created[len(created)-1], nil
}

func (m *hygonDCUHamiManager) listAvailableVdevIndices(physIdx int) ([]int, error) {
	hySmiPath := options.HostOptions.HygonHySmiPath
	out, err := procutils.NewRemoteCommandAsFarAsPossible(hySmiPath, "virtual", "-show-vdevice-info").Output()
	if err != nil {
		return nil, errors.Wrap(err, "hy-smi virtual -show-vdevice-info")
	}
	return parseHySmiVdeviceIndices(string(out), physIdx), nil
}

func parseHySmiVdeviceIndices(output string, physIdx int) []int {
	indices := []int{}
	currentVdev := -1
	currentPhys := -1
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Virtual Device") {
			if m := regexp.MustCompile(`Virtual Device\s*(\d+)`).FindStringSubmatch(line); len(m) == 2 {
				currentVdev, _ = strconv.Atoi(m[1])
			}
			continue
		}
		if strings.HasPrefix(line, "Actual Device:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentPhys, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
			continue
		}
		if currentVdev >= 0 && currentPhys == physIdx {
			indices = append(indices, currentVdev)
			currentVdev = -1
			currentPhys = -1
		}
	}
	if len(indices) == 0 {
		vdevDir := options.HostOptions.HygonVdevConfDir
		if hygonPathExists(vdevDir) {
			entries, err := hygonReadDir(vdevDir)
			if err == nil {
				for _, e := range entries {
					name := e.Name()
					if strings.HasPrefix(name, "vdev") && strings.HasSuffix(name, ".conf") {
						numStr := strings.TrimSuffix(strings.TrimPrefix(name, "vdev"), ".conf")
						if idx, err := strconv.Atoi(numStr); err == nil {
							indices = append(indices, idx)
						}
					}
				}
			}
		}
	}
	return indices
}

func (m *hygonDCUHamiManager) releaseVdev(containerDevId string) {
	m.vdevMu.Lock()
	defer m.vdevMu.Unlock()
	vdevIdx, ok := m.allocated[containerDevId]
	if !ok {
		return
	}
	delete(m.allocated, containerDevId)
	delete(m.vdevInUse, vdevIdx)
}
