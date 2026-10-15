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
	"strconv"
	"strings"

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
	isolated_device.RegisterContainerDeviceManager(newMthreadsGPUManager())
}

type mthreadsGPUManager struct{}

func newMthreadsGPUManager() *mthreadsGPUManager {
	return &mthreadsGPUManager{}
}

func (m *mthreadsGPUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeMthreadsGpu
}

func (m *mthreadsGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getMthreadsGPUs(m, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE)
}

func (m *mthreadsGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *mthreadsGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	return newMthreadsContainerDevices(dev)
}

func (m *mthreadsGPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return buildMthreadsExtraConfigures(collectMthreadsVisibleIndices(devs))
}

type mthreadsGPU struct {
	manager isolated_device.IContainerDeviceManager
	*BaseDevice

	memSize  int
	gpuIndex int
}

func (dev *mthreadsGPU) GetMemorySize() int {
	return dev.memSize
}

func (dev *mthreadsGPU) GetIndex() int {
	return dev.gpuIndex
}

func (dev *mthreadsGPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

const defaultMthreadsGmiPath = "/usr/bin/mthreads-gmi"

func mthreadsGmiPath() string {
	p := options.HostOptions.MthreadsGmiPath
	if p != "" {
		return p
	}
	return defaultMthreadsGmiPath
}

func runMthreadsGmi(args ...string) (string, error) {
	bin := mthreadsGmiPath()
	cmd := procutils.NewRemoteCommandAsFarAsPossible(bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return string(out), errors.Wrapf(err, "mthreads-gmi %s", strings.Join(args, " "))
	}
	return string(out), nil
}

func lookupMthreadsPCIDevice(busId, modelName string) *isolated_device.PCIDevice {
	cands := mthreadsPCIAddrCandidates(busId)
	for _, addr := range cands {
		pciOutput, err := isolated_device.GetPCIStrByAddr(addr)
		if err != nil || len(pciOutput) == 0 {
			log.Warningf("mthreads GetPCIStrByAddr %s: %v", addr, err)
			continue
		}
		dev := isolated_device.NewPCIDevice2(pciOutput[0])
		if modelName != "" {
			dev.ModelName = modelName
		}
		return dev
	}
	fallbackAddr := busId
	if len(cands) > 0 {
		fallbackAddr = cands[len(cands)-1]
	}
	return &isolated_device.PCIDevice{
		VendorId:   computeapi.MTHREADS_VENDOR_ID,
		VendorName: "MTHREADS",
		Addr:       fallbackAddr,
		ModelName:  modelName,
		DeviceName: modelName,
	}
}

func collectMthreadsVisibleIndices(devs []*hostapi.ContainerDevice) []string {
	indices := []string{}
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		if iDev == nil {
			continue
		}
		devMan := iDev.GetContainerDeviceManager()
		if _, ok := devMan.(*mthreadsGPUManager); !ok {
			continue
		}
		if dev.IsolatedDevice.Path != "" {
			indices = append(indices, dev.IsolatedDevice.Path)
			continue
		}
		if dev.IsolatedDevice.Index >= 0 {
			indices = append(indices, strconv.Itoa(dev.IsolatedDevice.Index))
		}
	}
	return indices
}

func newMthreadsContainerDevices(dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	if dev.IsolatedDevice == nil {
		return nil, nil, errors.Errorf("isolated device is nil")
	}
	iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
	if iDev == nil {
		return nil, nil, errors.Errorf("device %s not found", dev.IsolatedDevice.Id)
	}
	gpuDev, ok := iDev.(*mthreadsGPU)
	if !ok {
		return nil, nil, errors.Errorf("device %s is not mthreads gpu", dev.IsolatedDevice.Id)
	}
	ctrDevs := []*runtimeapi.Device{}
	node := mthreadsDevNode(gpuDev.GetIndex())
	if hygonPathExists(node) {
		ctrDevs = append(ctrDevs, mthreadsDeviceSpec(node))
	} else {
		log.Warningf("mthreads container device %s not found, skip", node)
	}
	return ctrDevs, nil, nil
}

func getMthreadsGPUs(manager isolated_device.IContainerDeviceManager, sharingMode string) ([]isolated_device.IDevice, error) {
	bin := mthreadsGmiPath()
	if !hygonPathExists(bin) {
		log.Infof("mthreads gpu probe skipped: mthreads-gmi not found at %s", bin)
		return nil, nil
	}

	tableOut, err := runMthreadsGmi()
	if err != nil {
		return nil, err
	}
	parsed := parseMthreadsGmi(tableOut)
	devs := make([]isolated_device.IDevice, 0, len(parsed))
	for _, gpu := range parsed {
		if !isMthreadsPhysical(gpu) {
			log.Infof("mthreads gpu %d device type %q, skip", gpu.Index, gpu.DeviceType)
			continue
		}
		pciDev := lookupMthreadsPCIDevice(gpu.BusId, gpu.Name)
		indexStr := strconv.Itoa(gpu.Index)
		dev := &mthreadsGPU{
			manager:    manager,
			BaseDevice: NewBaseDevice(pciDev, computeapi.GPU_TYPE, indexStr, sharingMode, 1),
			memSize:    gpu.MemorySizeMB,
			gpuIndex:   gpu.Index,
		}
		if gpu.Name != "" {
			dev.SetModelName(gpu.Name)
		}
		log.Infof("mthreads gpu idx=%d model=%s pci=%s memMiB=%d path=%s node=%s",
			gpu.Index, gpu.Name, pciDev.Addr, gpu.MemorySizeMB, indexStr, mthreadsDevNode(gpu.Index))
		devs = append(devs, dev)
	}
	if len(devs) == 0 {
		log.Infof("mthreads gpu probe finished: no physical devices found")
		return nil, nil
	}
	return devs, nil
}
