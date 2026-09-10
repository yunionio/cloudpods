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
	"os"
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
	isolated_device.RegisterContainerDeviceManager(newKunlunxinXPUManager())
}

type kunlunxinXPUManager struct{}

func newKunlunxinXPUManager() *kunlunxinXPUManager {
	return &kunlunxinXPUManager{}
}

func (m *kunlunxinXPUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeKunlunxinXpu
}

func (m *kunlunxinXPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getKunlunxinXPUs(m)
}

func (m *kunlunxinXPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *kunlunxinXPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	if dev.IsolatedDevice == nil {
		return nil, nil, errors.Errorf("isolated device is nil")
	}
	iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
	if iDev == nil {
		return nil, nil, errors.Errorf("device %s not found", dev.IsolatedDevice.Id)
	}
	gpuDev, ok := iDev.(*kunlunxinXPU)
	if !ok {
		return nil, nil, errors.Errorf("device %s is not kunlunxin xpu", dev.IsolatedDevice.Id)
	}
	ctrDevs := []*runtimeapi.Device{}
	node := kunlunxinXpuDevNode(gpuDev.GetIndex())
	if hygonPathExists(node) {
		ctrDevs = append(ctrDevs, kunlunxinXpuDeviceSpec(node))
	} else {
		log.Warningf("kunlunxin xpu container device %s not found, skip", node)
	}
	return ctrDevs, kunlunxinXpuCommonDevices(), nil
}

func (m *kunlunxinXPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	indices := collectKunlunxinXpuVisibleIndices(devs)
	return buildKunlunxinXpuExtraConfigures(indices, kunlunxinXreHome(), hygonPathExists)
}

type kunlunxinXPU struct {
	manager isolated_device.IContainerDeviceManager
	*BaseDevice

	memSize  int
	gpuIndex int
	uuid     string
}

func (dev *kunlunxinXPU) GetMemorySize() int {
	return dev.memSize
}

func (dev *kunlunxinXPU) GetIndex() int {
	return dev.gpuIndex
}

func (dev *kunlunxinXPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

func kunlunxinXpuCommonDevices() []*runtimeapi.Device {
	devs := []*runtimeapi.Device{}
	for _, p := range collectKunlunxinXpuCommonDevicePaths(hygonPathExists) {
		devs = append(devs, kunlunxinXpuDeviceSpec(p))
	}
	return devs
}

func kunlunxinXreHome() string {
	home := options.HostOptions.KunlunxinXreHome
	if home == "" {
		return defaultKunlunxinXreHome
	}
	return home
}

func kunlunxinXpuSmiPath() string {
	p := options.HostOptions.KunlunxinXpuSmiPath
	if p != "" {
		return p
	}
	return defaultKunlunxinXpuSmiPath
}

func kunlunxinXpuLDLibraryPath() string {
	libDir := kunlunxinXpuLibDir(kunlunxinXreHome(), hygonPathExists)
	existing := os.Getenv("LD_LIBRARY_PATH")
	if existing == "" {
		return libDir
	}
	return existing + ":" + libDir
}

func runXpuSmi(args ...string) (string, error) {
	smiPath := kunlunxinXpuSmiPath()
	cmd := procutils.NewRemoteCommandAsFarAsPossible(smiPath, args...)
	cmd.SetEnv([]string{"LD_LIBRARY_PATH=" + kunlunxinXpuLDLibraryPath()})
	out, err := cmd.Output()
	if err != nil {
		return string(out), errors.Wrapf(err, "xpu-smi %s", strings.Join(args, " "))
	}
	return string(out), nil
}

func lookupKunlunxinPCIDevice(busId, modelName string) *isolated_device.PCIDevice {
	cands := kunlunxinXpuPCIAddrCandidates(busId)
	for _, addr := range cands {
		pciOutput, err := isolated_device.GetPCIStrByAddr(addr)
		if err != nil || len(pciOutput) == 0 {
			log.Warningf("kunlunxin xpu GetPCIStrByAddr %s: %v", addr, err)
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
		VendorId:   computeapi.KUNLUNXIN_VENDOR_ID,
		VendorName: "KUNLUNXIN",
		Addr:       fallbackAddr,
		ModelName:  modelName,
		DeviceName: modelName,
	}
}

func collectKunlunxinXpuVisibleIndices(devs []*hostapi.ContainerDevice) []string {
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
		if _, ok := devMan.(*kunlunxinXPUManager); !ok {
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

func getKunlunxinXPUs(manager isolated_device.IContainerDeviceManager) ([]isolated_device.IDevice, error) {
	smiPath := kunlunxinXpuSmiPath()
	if !hygonPathExists(smiPath) {
		log.Infof("kunlunxin xpu probe skipped: xpu-smi not found at %s", smiPath)
		return nil, nil
	}

	out, err := runXpuSmi()
	if err != nil {
		log.Warningf("xpu-smi failed: %v", err)
		return nil, nil
	}
	parsed := parseXpuSmiTable(out)
	if len(parsed) == 0 {
		log.Infof("kunlunxin xpu probe: no devices parsed from xpu-smi")
		return nil, nil
	}

	devs := make([]isolated_device.IDevice, 0, len(parsed))
	for _, gpu := range parsed {
		pciDev := lookupKunlunxinPCIDevice(gpu.BusId, gpu.Name)
		indexStr := strconv.Itoa(gpu.Index)
		node := kunlunxinXpuDevNode(gpu.Index)
		if !hygonPathExists(node) {
			log.Warningf("kunlunxin xpu %d: %s not found, still register by index", gpu.Index, node)
		}
		dev := &kunlunxinXPU{
			manager:    manager,
			BaseDevice: NewBaseDevice(pciDev, computeapi.GPU_TYPE, indexStr, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, 1),
			memSize:    gpu.MemorySizeMB,
			gpuIndex:   gpu.Index,
			uuid:       gpu.UUID,
		}
		if gpu.Name != "" {
			dev.SetModelName(gpu.Name)
		}
		log.Infof("kunlunxin xpu idx=%d model=%s pci=%s uuid=%s memMiB=%d path=%s node=%s",
			gpu.Index, gpu.Name, pciDev.Addr, gpu.UUID, gpu.MemorySizeMB, indexStr, node)
		devs = append(devs, dev)
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}
