// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain the copy of the License at
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
	"os"
	"path"
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
	isolated_device.RegisterContainerDeviceManager(newIluvatarGPUManager())
}

type iluvatarGPUManager struct{}

func newIluvatarGPUManager() *iluvatarGPUManager {
	return &iluvatarGPUManager{}
}

func (m *iluvatarGPUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeIluvatarGpu
}

func (m *iluvatarGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getIluvatarGPUs(m)
}

func (m *iluvatarGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *iluvatarGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	if dev.IsolatedDevice == nil {
		return nil, nil, errors.Errorf("isolated device is nil")
	}
	iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
	if iDev == nil {
		return nil, nil, errors.Errorf("device %s not found", dev.IsolatedDevice.Id)
	}
	gpuDev, ok := iDev.(*iluvatarGPU)
	if !ok {
		return nil, nil, errors.Errorf("device %s is not iluvatar gpu", dev.IsolatedDevice.Id)
	}
	minor := gpuDev.GetDeviceMinor()
	if minor < 0 {
		minor = gpuDev.GetIndex()
	}
	ctrDevs := []*runtimeapi.Device{}
	node := iluvatarDevNode(minor)
	if hygonPathExists(node) {
		ctrDevs = append(ctrDevs, iluvatarDeviceSpec(node))
	} else {
		log.Warningf("iluvatar container device %s not found, skip", node)
	}
	return ctrDevs, iluvatarCommonDevices(), nil
}

func (m *iluvatarGPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	indices := collectIluvatarVisibleIndices(devs)
	return buildIluvatarExtraConfigures(indices, iluvatarCorexHome(), hygonPathExists)
}

type iluvatarGPU struct {
	manager isolated_device.IContainerDeviceManager
	*BaseDevice

	memSize     int
	gpuIndex    int
	deviceMinor int
	uuid        string
}

func (dev *iluvatarGPU) GetMemorySize() int {
	return dev.memSize
}

func (dev *iluvatarGPU) GetIndex() int {
	return dev.gpuIndex
}

func (dev *iluvatarGPU) GetDeviceMinor() int {
	return dev.deviceMinor
}

func (dev *iluvatarGPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

func iluvatarCommonDevices() []*runtimeapi.Device {
	devs := []*runtimeapi.Device{}
	for _, p := range collectIluvatarCommonDevicePaths(hygonPathExists) {
		devs = append(devs, iluvatarDeviceSpec(p))
	}
	return devs
}

func iluvatarCorexHome() string {
	home := options.HostOptions.IluvatarCorexHome
	if home == "" {
		return defaultIluvatarCorexHome
	}
	return home
}

func iluvatarIxsmiPath() string {
	p := options.HostOptions.IluvatarIxsmiPath
	if p != "" {
		return p
	}
	return path.Join(iluvatarCorexHome(), "bin", "ixsmi")
}

func iluvatarLDLibraryPath() string {
	lib64 := path.Join(iluvatarCorexHome(), "lib64")
	existing := os.Getenv("LD_LIBRARY_PATH")
	if existing == "" {
		return lib64
	}
	return existing + ":" + lib64
}

func runIxsmi(args ...string) (string, error) {
	ixsmiPath := iluvatarIxsmiPath()
	cmd := procutils.NewRemoteCommandAsFarAsPossible(ixsmiPath, args...)
	cmd.SetEnv([]string{"LD_LIBRARY_PATH=" + iluvatarLDLibraryPath()})
	out, err := cmd.Output()
	if err != nil {
		return string(out), errors.Wrapf(err, "ixsmi %s", strings.Join(args, " "))
	}
	return string(out), nil
}

func lookupIluvatarPCIDevice(busId, modelName string) *isolated_device.PCIDevice {
	cands := iluvatarPCIAddrCandidates(busId)
	for _, addr := range cands {
		pciOutput, err := isolated_device.GetPCIStrByAddr(addr)
		if err != nil || len(pciOutput) == 0 {
			log.Warningf("iluvatar GetPCIStrByAddr %s: %v", addr, err)
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
		VendorId:   computeapi.ILUVATAR_VENDOR_ID,
		VendorName: "ILUVATAR",
		Addr:       fallbackAddr,
		ModelName:  modelName,
		DeviceName: modelName,
	}
}

func collectIluvatarVisibleIndices(devs []*hostapi.ContainerDevice) []string {
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
		if _, ok := devMan.(*iluvatarGPUManager); !ok {
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

func getIluvatarGPUs(manager isolated_device.IContainerDeviceManager) ([]isolated_device.IDevice, error) {
	ixsmiPath := iluvatarIxsmiPath()
	if !hygonPathExists(ixsmiPath) {
		log.Infof("iluvatar gpu probe skipped: ixsmi not found at %s", ixsmiPath)
		return nil, nil
	}

	tableOut, err := runIxsmi()
	if err != nil {
		return nil, err
	}
	parsed := parseIxsmiTable(tableOut)
	if len(parsed) == 0 {
		log.Infof("iluvatar gpu probe: no devices parsed from ixsmi table")
		return nil, nil
	}

	listOut, err := runIxsmi("-L")
	if err != nil {
		log.Warningf("ixsmi -L failed: %v", err)
	} else {
		parsed = mergeIluvatarProbe(parsed, parseIxsmiList(listOut))
	}

	devs := make([]isolated_device.IDevice, 0, len(parsed))
	for _, gpu := range parsed {
		if gpu.ComputeMode != "" && gpu.ComputeMode != iluvatarComputeModeOK {
			log.Warningf("iluvatar gpu %d compute mode %s, skip", gpu.Index, gpu.ComputeMode)
			continue
		}
		pciDev := lookupIluvatarPCIDevice(gpu.BusId, gpu.Name)
		indexStr := strconv.Itoa(gpu.Index)
		minor := resolveIluvatarDeviceMinor(gpu.Index, hygonPathExists)
		if minor < 0 {
			log.Warningf("iluvatar gpu %d: %s not found, fallback to index for container device", gpu.Index, iluvatarDevNode(gpu.Index))
		}
		dev := &iluvatarGPU{
			manager:     manager,
			BaseDevice:  NewBaseDevice(pciDev, computeapi.GPU_TYPE, indexStr, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, 1),
			memSize:     gpu.MemorySizeMB,
			gpuIndex:    gpu.Index,
			deviceMinor: minor,
			uuid:        gpu.UUID,
		}
		if gpu.Name != "" {
			dev.SetModelName(gpu.Name)
		}
		log.Infof("iluvatar gpu idx=%d minor=%d model=%s pci=%s uuid=%s memMiB=%d path=%s node=%s",
			gpu.Index, minor, gpu.Name, pciDev.Addr, gpu.UUID, gpu.MemorySizeMB, indexStr, iluvatarDevNode(gpu.Index))
		devs = append(devs, dev)
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}
