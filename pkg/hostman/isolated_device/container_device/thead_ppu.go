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
	isolated_device.RegisterContainerDeviceManager(newTHeadPPUManager())
}

type tHeadPPUManager struct{}

func newTHeadPPUManager() *tHeadPPUManager {
	return &tHeadPPUManager{}
}

func (m *tHeadPPUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeTHeadPpu
}

func (m *tHeadPPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getTHeadPPUs(m)
}

func (m *tHeadPPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *tHeadPPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	if dev.IsolatedDevice == nil {
		return nil, nil, errors.Errorf("isolated device is nil")
	}
	iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
	if iDev == nil {
		return nil, nil, errors.Errorf("device %s not found", dev.IsolatedDevice.Id)
	}
	gpuDev, ok := iDev.(*tHeadPPU)
	if !ok {
		return nil, nil, errors.Errorf("device %s is not t-head ppu", dev.IsolatedDevice.Id)
	}
	ctrDevs := []*runtimeapi.Device{}
	node := tHeadPpuDevNode(gpuDev.GetIndex())
	if hygonPathExists(node) {
		ctrDevs = append(ctrDevs, tHeadPpuDeviceSpec(node))
	} else {
		log.Warningf("t-head ppu container device %s not found, skip", node)
	}
	return ctrDevs, tHeadPpuCommonDevices(), nil
}

func (m *tHeadPPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	indices := collectTHeadPpuVisibleIndices(devs)
	return buildTHeadPpuExtraConfigures(indices, tHeadPpuSdkHome(), hygonPathExists)
}

type tHeadPPU struct {
	manager isolated_device.IContainerDeviceManager
	*BaseDevice

	memSize  int
	gpuIndex int
	uuid     string
}

func (dev *tHeadPPU) GetMemorySize() int {
	return dev.memSize
}

func (dev *tHeadPPU) GetIndex() int {
	return dev.gpuIndex
}

func (dev *tHeadPPU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

func tHeadPpuCommonDevices() []*runtimeapi.Device {
	devs := []*runtimeapi.Device{}
	for _, p := range collectTHeadPpuCommonDevicePaths(hygonPathExists) {
		devs = append(devs, tHeadPpuDeviceSpec(p))
	}
	return devs
}

func tHeadPpuSdkHome() string {
	home := options.HostOptions.THeadPpuSdkHome
	if home == "" {
		return defaultTHeadPpuSdkHome
	}
	return home
}

func tHeadPpuSmiPath() string {
	p := options.HostOptions.THeadPpuSmiPath
	if p != "" {
		return p
	}
	return defaultTHeadPpuSmiPath
}

func tHeadPpuLDLibraryPath() string {
	libDir := tHeadPpuLibDir(tHeadPpuSdkHome(), hygonPathExists)
	existing := os.Getenv("LD_LIBRARY_PATH")
	if existing == "" {
		return libDir
	}
	return existing + ":" + libDir
}

func runPpuSmi(args ...string) (string, error) {
	smiPath := tHeadPpuSmiPath()
	cmd := procutils.NewRemoteCommandAsFarAsPossible(smiPath, args...)
	cmd.SetEnv([]string{"LD_LIBRARY_PATH=" + tHeadPpuLDLibraryPath()})
	out, err := cmd.Output()
	if err != nil {
		return string(out), errors.Wrapf(err, "ppu-smi %s", strings.Join(args, " "))
	}
	return string(out), nil
}

func lookupTHeadPCIDevice(busId, modelName string) *isolated_device.PCIDevice {
	cands := tHeadPpuPCIAddrCandidates(busId)
	for _, addr := range cands {
		pciOutput, err := isolated_device.GetPCIStrByAddr(addr)
		if err != nil || len(pciOutput) == 0 {
			log.Warningf("t-head ppu GetPCIStrByAddr %s: %v", addr, err)
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
		VendorId:   computeapi.THEAD_VENDOR_ID,
		VendorName: "THEAD",
		Addr:       fallbackAddr,
		ModelName:  modelName,
		DeviceName: modelName,
	}
}

func collectTHeadPpuVisibleIndices(devs []*hostapi.ContainerDevice) []string {
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
		if _, ok := devMan.(*tHeadPPUManager); !ok {
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

func getTHeadPPUs(manager isolated_device.IContainerDeviceManager) ([]isolated_device.IDevice, error) {
	smiPath := tHeadPpuSmiPath()
	if !hygonPathExists(smiPath) {
		log.Infof("t-head ppu probe skipped: ppu-smi not found at %s", smiPath)
		return nil, nil
	}

	parsed := []*parsedTHeadPPU{}
	queryOut, err := runPpuSmi("--query-ppu=index,name,uuid,pci.bus_id,memory.total", "--format=csv,noheader,nounits")
	if err != nil {
		log.Warningf("ppu-smi --query-ppu failed: %v", err)
	} else {
		parsed = parsePpuSmiQueryCSV(queryOut)
	}

	listOut, err := runPpuSmi("-L")
	if err != nil {
		log.Warningf("ppu-smi -L failed: %v", err)
	} else {
		parsed = mergeTHeadPpuProbe(parsed, parsePpuSmiList(listOut))
		if len(parsed) == 0 {
			for idx, uuid := range parsePpuSmiList(listOut) {
				parsed = append(parsed, &parsedTHeadPPU{Index: idx, UUID: uuid})
			}
		}
	}

	if len(parsed) == 0 {
		log.Infof("t-head ppu probe: no devices parsed from ppu-smi")
		return nil, nil
	}

	devs := make([]isolated_device.IDevice, 0, len(parsed))
	for _, gpu := range parsed {
		pciDev := lookupTHeadPCIDevice(gpu.BusId, gpu.Name)
		indexStr := strconv.Itoa(gpu.Index)
		node := tHeadPpuDevNode(gpu.Index)
		if !hygonPathExists(node) {
			log.Warningf("t-head ppu %d: %s not found, still register by index", gpu.Index, node)
		}
		dev := &tHeadPPU{
			manager:    manager,
			BaseDevice: NewBaseDevice(pciDev, computeapi.GPU_TYPE, indexStr, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, 1),
			memSize:    gpu.MemorySizeMB,
			gpuIndex:   gpu.Index,
			uuid:       gpu.UUID,
		}
		if gpu.Name != "" {
			dev.SetModelName(gpu.Name)
		}
		log.Infof("t-head ppu idx=%d model=%s pci=%s uuid=%s memMiB=%d path=%s node=%s",
			gpu.Index, gpu.Name, pciDev.Addr, gpu.UUID, gpu.MemorySizeMB, indexStr, node)
		devs = append(devs, dev)
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}
