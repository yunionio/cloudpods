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
	"os"
	"path"
	"strconv"
	"sync"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newHygonDCUHamiManager())
}

type hygonVdevAllocation struct {
	cacheDir string
	vdevIdx  int
	pipeID   int
	devIdx   int
	coremsk1 string
	coremsk2 string
}

type hygonDCUHamiManager struct {
	*hygonDCUManager

	mu        sync.Mutex
	vidx      [hygonMaxVdevIdx]bool
	pipeid    map[int][hygonMaxPipePerDev]bool
	coremask  map[int][2]string
	allocated map[string]*hygonVdevAllocation // containerDeviceId -> allocation
}

func newHygonDCUHamiManager() *hygonDCUHamiManager {
	return &hygonDCUHamiManager{
		hygonDCUManager: newHygonDCUManager(),
		pipeid:          make(map[int][hygonMaxPipePerDev]bool),
		coremask:        make(map[int][2]string),
		allocated:       make(map[string]*hygonVdevAllocation),
	}
}

func (m *hygonDCUHamiManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeHygonDcuHami
}

func (m *hygonDCUHamiManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getHygonDCUs(m, computeapi.DEVICE_SHARING_MODE_HAMI)
}

func isHygonVDcuRequest(devs []*hostapi.ContainerDevice, fullMemMiB int) bool {
	if len(devs) == 0 {
		return false
	}
	if len(devs) >= 2 {
		return false
	}
	dev := devs[0]
	if dev.IsolatedDevice == nil {
		return false
	}
	memLimit := dev.IsolatedDevice.MemoryLimit
	if memLimit <= 0 {
		memLimit = fullMemMiB
	}
	return memLimit < fullMemMiB
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

	fullMemMiB := dcuDev.GetMemorySize()
	isVDcu := isHygonVDcuRequest(devs, fullMemMiB)
	mounts := buildHygonRuntimeMounts(isVDcu)

	if !isVDcu {
		return buildHygonRuntimeEnvs([]string{strconv.Itoa(dcuDev.GetIndex())}), mounts
	}

	memLimitMiB := dev.IsolatedDevice.MemoryLimit
	if memLimitMiB <= 0 {
		memLimitMiB = fullMemMiB
	}
	smLimit := dev.IsolatedDevice.SmUtilLimit
	if smLimit <= 0 {
		smLimit = 100
	}
	computeUnits := dcuDev.GetComputeUnits()
	if computeUnits <= 0 {
		computeUnits = 60
	}
	reqCores := int32(computeUnits * smLimit / 100)
	if reqCores <= 0 {
		reqCores = 1
	}

	guestId, containerName := getHygonContainerContext()
	if guestId == "" {
		guestId = "unknown"
	}
	if containerName == "" {
		containerName = dev.IsolatedDevice.Id
	}

	alloc, err := m.ensureVdevAllocation(
		dev.IsolatedDevice.Id,
		guestId,
		containerName,
		dcuDev,
		memLimitMiB,
		reqCores,
	)
	if err != nil {
		log.Errorf("ensure hygon vdev for device %s: %v", dev.IsolatedDevice.Id, err)
		return nil, mounts
	}

	mounts = append(mounts, &runtimeapi.Mount{
		ContainerPath: "/etc/vdev/docker/",
		HostPath:      alloc.cacheDir,
		Readonly:      false,
	})

	return buildHygonRuntimeEnvs([]string{strconv.Itoa(alloc.vdevIdx)}), mounts
}

func (m *hygonDCUHamiManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return m.newHygonDrmContainerDevices(dev, true)
}

func (m *hygonDCUHamiManager) ensureCoreMask(devIdx, computeUnits int) {
	if _, ok := m.coremask[devIdx]; ok {
		return
	}
	init := initHygonCoreUsage(computeUnits)
	m.coremask[devIdx] = [2]string{init, init}
}

func (m *hygonDCUHamiManager) allocateVdevIdx() (int, error) {
	for idx := range m.vidx {
		if !m.vidx[idx] {
			m.vidx[idx] = true
			return idx, nil
		}
	}
	return 0, errors.Error("hygon vdev index out of bound (>200)")
}

func (m *hygonDCUHamiManager) allocatePipeID(devIdx int) (int, error) {
	pipes := m.pipeid[devIdx]
	for idx := range pipes {
		if !pipes[idx] {
			pipes[idx] = true
			m.pipeid[devIdx] = pipes
			return idx, nil
		}
	}
	return 0, errors.Errorf("hygon pipe index out of bound for device %d", devIdx)
}

func (m *hygonDCUHamiManager) ensureVdevAllocation(
	containerDevId, guestId, containerName string,
	dcuDev *hygonDCU,
	memMiB int,
	reqCores int32,
) (*hygonVdevAllocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if alloc, ok := m.allocated[containerDevId]; ok {
		return alloc, nil
	}

	devIdx := dcuDev.GetIndex()
	computeUnits := dcuDev.GetComputeUnits()
	if computeUnits <= 0 {
		computeUnits = 60
	}
	m.ensureCoreMask(devIdx, computeUnits)

	coremsk1, reqTmp, err := allocHygonCoreUsage(m.coremask[devIdx][0], int(reqCores))
	if err != nil {
		return nil, errors.Wrap(err, "alloc core mask 1")
	}
	coremsk2 := initHygonCoreUsage(computeUnits)
	if reqTmp > 0 {
		coremsk2, _, err = allocHygonCoreUsage(m.coremask[devIdx][1], reqTmp)
		if err != nil {
			return nil, errors.Wrap(err, "alloc core mask 2")
		}
	}

	vdevIdx, err := m.allocateVdevIdx()
	if err != nil {
		return nil, err
	}
	pipeID, err := m.allocatePipeID(devIdx)
	if err != nil {
		m.vidx[vdevIdx] = false
		return nil, err
	}

	pciBusId := hygonPciBusIdFromAddr(dcuDev.GetAddr())
	dirName := hygonVgpuCacheDirName(guestId, containerName, devIdx, pipeID, vdevIdx, coremsk1, coremsk2)
	cacheDir := path.Join(options.HostOptions.HygonVgpuCacheDir, dirName)

	if err := createHygonVdevConfFile(pciBusId, coremsk1, coremsk2, reqCores, int32(memMiB), devIdx, vdevIdx, pipeID, cacheDir, "vdev0.conf"); err != nil {
		m.vidx[vdevIdx] = false
		pipes := m.pipeid[devIdx]
		pipes[pipeID] = false
		m.pipeid[devIdx] = pipes
		return nil, err
	}
	vdevConfDir := options.HostOptions.HygonVdevConfDir
	if err := createHygonVdevConfFile(pciBusId, coremsk1, coremsk2, reqCores, int32(memMiB), devIdx, vdevIdx, pipeID, vdevConfDir, fmt.Sprintf("vdev%d.conf", vdevIdx)); err != nil {
		_ = hygonRemoveAll(cacheDir)
		m.vidx[vdevIdx] = false
		pipes := m.pipeid[devIdx]
		pipes[pipeID] = false
		m.pipeid[devIdx] = pipes
		return nil, err
	}

	coreUsage1, err := addHygonCoreUsage(m.coremask[devIdx][0], coremsk1)
	if err != nil {
		return nil, errors.Wrap(err, "add core usage 1")
	}
	mask := m.coremask[devIdx]
	mask[0] = coreUsage1
	m.coremask[devIdx] = mask
	coreUsage2, err := addHygonCoreUsage(m.coremask[devIdx][1], coremsk2)
	if err != nil {
		return nil, errors.Wrap(err, "add core usage 2")
	}
	mask = m.coremask[devIdx]
	mask[1] = coreUsage2
	m.coremask[devIdx] = mask

	alloc := &hygonVdevAllocation{
		cacheDir: cacheDir,
		vdevIdx:  vdevIdx,
		pipeID:   pipeID,
		devIdx:   devIdx,
		coremsk1: coremsk1,
		coremsk2: coremsk2,
	}
	m.allocated[containerDevId] = alloc
	return alloc, nil
}

func (m *hygonDCUHamiManager) ReleaseContainerDevices(devs []*hostapi.ContainerDevice) {
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		m.releaseVdev(dev.IsolatedDevice.Id)
	}
}

func (m *hygonDCUHamiManager) releaseVdev(containerDevId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alloc, ok := m.allocated[containerDevId]
	if !ok {
		return
	}
	delete(m.allocated, containerDevId)

	m.vidx[alloc.vdevIdx] = false
	if pipes, ok := m.pipeid[alloc.devIdx]; ok {
		pipes[alloc.pipeID] = false
		m.pipeid[alloc.devIdx] = pipes
	}

	if _, ok := m.coremask[alloc.devIdx]; ok {
		m.rebuildCoreMaskLocked(alloc.devIdx)
	}

	if alloc.cacheDir != "" {
		if err := hygonRemoveAll(alloc.cacheDir); err != nil {
			log.Warningf("remove hygon vdev cache dir %s: %v", alloc.cacheDir, err)
		}
	}
	vdevConfPath := path.Join(options.HostOptions.HygonVdevConfDir, fmt.Sprintf("vdev%d.conf", alloc.vdevIdx))
	if err := hygonRemove(vdevConfPath); err != nil && !os.IsNotExist(err) {
		log.Warningf("remove hygon vdev conf %s: %v", vdevConfPath, err)
	}
}

func (m *hygonDCUHamiManager) rebuildCoreMaskLocked(devIdx int) {
	dcuDev := m.findHygonDCUByIndex(devIdx)
	computeUnits := 60
	if dcuDev != nil && dcuDev.GetComputeUnits() > 0 {
		computeUnits = dcuDev.GetComputeUnits()
	}
	init := initHygonCoreUsage(computeUnits)
	m.coremask[devIdx] = [2]string{init, init}
	for _, alloc := range m.allocated {
		if alloc.devIdx != devIdx {
			continue
		}
		mask := m.coremask[devIdx]
		if usage, err := addHygonCoreUsage(mask[0], alloc.coremsk1); err == nil {
			mask[0] = usage
		}
		if usage, err := addHygonCoreUsage(mask[1], alloc.coremsk2); err == nil {
			mask[1] = usage
		}
		m.coremask[devIdx] = mask
	}
}

func (m *hygonDCUHamiManager) findHygonDCUByIndex(devIdx int) *hygonDCU {
	for _, dev := range hostinfo.Instance().IsolatedDeviceMan.GetDevices() {
		if dcu, ok := dev.(*hygonDCU); ok && dcu.GetIndex() == devIdx {
			return dcu
		}
	}
	return nil
}
