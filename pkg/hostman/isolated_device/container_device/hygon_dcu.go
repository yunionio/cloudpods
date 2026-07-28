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
	isolated_device.RegisterContainerDeviceManager(newHygonDCUManager())
}

type hygonDCUManager struct{}

func newHygonDCUManager() *hygonDCUManager {
	return &hygonDCUManager{}
}

func (m *hygonDCUManager) GetRegisterType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeHygonDcu
}

func (m *hygonDCUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getHygonDCUs(m, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE)
}

func (m *hygonDCUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (m *hygonDCUManager) GetDevType() string {
	return computeapi.GPU_TYPE
}

func (m *hygonDCUManager) GetSharingMode() string {
	return computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
}

type hygonDCU struct {
	manager isolated_device.IContainerDeviceManager

	*BaseDevice
	gpuIndex     int
	renderPath   string
	memorySize   int
	computeUnits int
}

func (dev *hygonDCU) GetMemorySize() int {
	return dev.memorySize
}

func (dev *hygonDCU) GetIndex() int {
	return dev.gpuIndex
}

func (dev *hygonDCU) GetRenderPath() string {
	return dev.renderPath
}

func (dev *hygonDCU) GetComputeUnits() int {
	return dev.computeUnits
}

func (dev *hygonDCU) GetContainerDeviceManager() isolated_device.IContainerDeviceManager {
	return dev.manager
}

func (m *hygonDCUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
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
		if _, ok := devMan.(*hygonDCUManager); !ok {
			continue
		}
		indices = append(indices, dev.IsolatedDevice.Path)
	}
	if len(indices) == 0 {
		return nil, nil
	}
	return m.buildHygonExtraConfigures(indices)
}

func buildHygonRuntimeMounts() []*runtimeapi.Mount {
	hyhalPath := options.HostOptions.HygonHyhalPath
	dtkPath := options.HostOptions.HygonDtkPath
	mounts := []*runtimeapi.Mount{}
	if hygonPathExists(hyhalPath) {
		mounts = append(mounts, &runtimeapi.Mount{
			ContainerPath: hyhalPath,
			HostPath:      hyhalPath,
			Readonly:      true,
		})
	}
	if hygonPathExists(dtkPath) {
		mounts = append(mounts, &runtimeapi.Mount{
			ContainerPath: dtkPath,
			HostPath:      dtkPath,
			Readonly:      true,
		})
	}
	return mounts
}

func buildHygonRuntimeEnvs(indices []string) []*runtimeapi.KeyValue {
	hyhalPath := options.HostOptions.HygonHyhalPath
	dtkPath := options.HostOptions.HygonDtkPath
	envs := []*runtimeapi.KeyValue{
		{
			Key:   "HYGON_VISIBLE_DEVICES",
			Value: strings.Join(indices, ","),
		},
		{
			Key:   "HIP_VISIBLE_DEVICES",
			Value: strings.Join(indices, ","),
		},
		{
			Key:   "ROCM_PATH",
			Value: dtkPath,
		},
		{
			Key:   "DTK_HOME",
			Value: dtkPath,
		},
		{
			Key:   "ROCM_SMI_LIB_PATH",
			Value: path.Join(hyhalPath, "lib"),
		},
	}
	return envs
}

func (m *hygonDCUManager) buildHygonExtraConfigures(indices []string) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return buildHygonRuntimeEnvs(indices), buildHygonRuntimeMounts()
}

func hygonCommonDevices() []*runtimeapi.Device {
	perms := "rwm"
	devs := []*runtimeapi.Device{}
	for _, devPath := range []string{"/dev/kfd", "/dev/mkfd"} {
		if hygonPathExists(devPath) {
			devs = append(devs, &runtimeapi.Device{
				ContainerPath: devPath,
				HostPath:      devPath,
				Permissions:   perms,
			})
		}
	}
	return devs
}

func (m *hygonDCUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	if dev.IsolatedDevice == nil {
		return nil, nil, errors.Errorf("isolated device is nil")
	}
	iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
	if iDev == nil {
		return nil, nil, errors.Errorf("device %s not found", dev.IsolatedDevice.Id)
	}
	dcuDev, ok := iDev.(*hygonDCU)
	if !ok {
		return nil, nil, errors.Errorf("device %s is not hygon dcu", dev.IsolatedDevice.Id)
	}
	renderPath := dcuDev.GetRenderPath()
	if renderPath == "" {
		return nil, nil, errors.Errorf("hygon dcu %s has empty render path", dev.IsolatedDevice.Id)
	}
	perms := "rwm"
	ctrDevs := []*runtimeapi.Device{
		{
			ContainerPath: renderPath,
			HostPath:      renderPath,
			Permissions:   perms,
		},
	}
	return ctrDevs, hygonCommonDevices(), nil
}

func hygonDebugOutputSnippet(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func getHygonDCUs(manager isolated_device.IContainerDeviceManager, sharingMode string) ([]isolated_device.IDevice, error) {
	log.Infof("==== getHygonDCUs start: sharingMode=%s", sharingMode)
	hySmiPath := options.HostOptions.HygonHySmiPath
	log.Infof("==== getHygonDCUs config: hySmiPath=%s hyhalPath=%s dtkPath=%s remoteExecutor=%v",
		hySmiPath, options.HostOptions.HygonHyhalPath, options.HostOptions.HygonDtkPath, hygonUseRemoteFS())
	if !hygonPathExists(hySmiPath) {
		log.Infof("==== getHygonDCUs abort: hy-smi not found at %s (remote=%v)", hySmiPath, hygonUseRemoteFS())
		return nil, nil
	}

	memOut, err := procutils.NewRemoteCommandAsFarAsPossible(hySmiPath, "--showmeminfo", "vram").Output()
	if err != nil {
		log.Warningf("==== getHygonDCUs: hy-smi --showmeminfo vram failed: %v, fallback to default output", err)
		memOut, err = procutils.NewRemoteCommandAsFarAsPossible(hySmiPath).Output()
		if err != nil {
			log.Warningf("==== getHygonDCUs abort: hy-smi failed: %v", err)
			return nil, errors.Wrap(err, "hy-smi")
		}
	}
	log.Infof("==== getHygonDCUs hy-smi output (first 10 lines):\n%s", hygonDebugOutputSnippet(string(memOut), 10))

	indices := parseHySmiDeviceIndices(string(memOut))
	log.Infof("==== getHygonDCUs parsed indices: %v", indices)
	if len(indices) == 0 {
		log.Infof("==== getHygonDCUs abort: no HCU indices parsed from hy-smi output")
		return nil, nil
	}
	memMap := parseHySmiMemInfoVram(string(memOut))
	log.Infof("==== getHygonDCUs parsed memMap: %v", memMap)
	renderToPCI, err := getHygonRenderPathToPCIMap()
	if err != nil {
		log.Warningf("==== getHygonDCUs abort: getHygonRenderPathToPCIMap failed: %v", err)
		return nil, err
	}
	log.Infof("==== getHygonDCUs render to pci map: %v", renderToPCI)
	computeUnitsMap := parseHySmiComputeUnits(string(memOut))

	devs := make([]isolated_device.IDevice, 0, len(indices))
	for i, idx := range indices {
		renderPath := hygonRenderPathForHCUIndex(idx)
		pciAddr := renderToPCI[renderPath]
		if pciAddr == "" {
			log.Warningf("==== getHygonDCUs: no pci addr for render path %s (HCU idx=%d)", renderPath, idx)
		}

		var pciDev *isolated_device.PCIDevice
		if pciAddr != "" {
			pciOutput, err := isolated_device.GetPCIStrByAddr(pciAddr)
			if err != nil {
				return nil, errors.Wrapf(err, "GetPCIStrByAddr %s", pciAddr)
			}
			pciDev = isolated_device.NewPCIDevice2(pciOutput[0])
		} else {
			pciDev = &isolated_device.PCIDevice{
				VendorId: computeapi.HYGON_VENDOR_ID,
				Addr:     fmt.Sprintf("hygon-%d", idx),
			}
		}

		modelName := hygonModelNameFromPCIDevice(pciDev)

		memSize := memMap[idx]
		if memSize == 0 {
			memSize = memMap[i]
		}
		computeUnits := computeUnitsMap[idx]
		if computeUnits == 0 {
			computeUnits = 60
		}

		dcuDev := &hygonDCU{
			manager:      manager,
			BaseDevice:   NewBaseDevice(pciDev, computeapi.GPU_TYPE, strconv.Itoa(idx), sharingMode, 1),
			gpuIndex:     idx,
			renderPath:   renderPath,
			memorySize:   memSize,
			computeUnits: computeUnits,
		}
		dcuDev.SetModelName(modelName)
		log.Infof("==== getHygonDCUs device[%d]: idx=%d model=%s pci=%s render=%s vendorDeviceId=%s memMiB=%d cu=%d",
			i, idx, modelName, pciAddr, renderPath, pciDev.GetVendorDeviceId(), memSize, computeUnits)
		devs = append(devs, dcuDev)
	}
	if len(devs) == 0 {
		log.Infof("==== getHygonDCUs finished: no devices built")
		return nil, nil
	}
	log.Infof("==== getHygonDCUs finished: built %d devices", len(devs))
	return devs, nil
}

func hygonRenderPathForHCUIndex(idx int) string {
	return fmt.Sprintf("/dev/dri/renderD%d", 128+idx)
}

func hygonModelNameFromPCIDevice(pciDev *isolated_device.PCIDevice) string {
	if pciDev == nil {
		return "Hygon DCU"
	}
	if pciDev.ModelName != "" {
		return pciDev.ModelName
	}
	if pciDev.DeviceName != "" {
		return pciDev.DeviceName
	}
	return "Hygon DCU"
}

func buildHygonRenderPathToPCIMap(pciToRender map[string]string) map[string]string {
	ret := make(map[string]string, len(pciToRender))
	for pciAddr, renderPath := range pciToRender {
		ret[renderPath] = pciAddr
	}
	return ret
}

func getHygonRenderPathToPCIMap() (map[string]string, error) {
	pciToRender, err := getHygonRenderPathMap()
	if err != nil {
		return nil, err
	}
	return buildHygonRenderPathToPCIMap(pciToRender), nil
}

func getHygonRenderPathMap() (map[string]string, error) {
	const byPathDir = "/dev/dri/by-path"
	ret := map[string]string{}
	if !hygonPathExists(byPathDir) {
		return ret, nil
	}
	entries, err := hygonReadDir(byPathDir)
	if err != nil {
		return nil, errors.Wrapf(err, "read %s", byPathDir)
	}
	for _, entry := range entries {
		entryName := entry.Name()
		if !strings.HasSuffix(entryName, "-render") {
			continue
		}
		pciAddr, err := getGPUPCIAddr(entryName)
		if err != nil {
			continue
		}
		fp := path.Join(byPathDir, entryName)
		linkPath, err := hygonReadlink(fp)
		if err != nil {
			continue
		}
		renderPath := hygonRenderPathFromLink(linkPath)
		ret[pciAddr] = renderPath
	}
	return ret, nil
}

var (
	hySmiIndexRe       = regexp.MustCompile(`(?m)^\s*(\d+)\s+`)
	hySmiHCUBracketRe  = regexp.MustCompile(`HCU\[(\d+)\]`)
	hySmiHCUMemTotalRe = regexp.MustCompile(`HCU\[(\d+)\]\s*:\s*vram Total Memory \(MiB\):\s*(\d+)`)
	hySmiComputeRe     = regexp.MustCompile(`(?i)(?:Compute units|CU)\s*:\s*(\d+)`)
)

func parseHySmiDeviceIndices(output string) []int {
	indices := []int{}
	seen := map[int]bool{}
	addIndex := func(idx int) {
		if !seen[idx] {
			seen[idx] = true
			indices = append(indices, idx)
		}
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := hySmiHCUBracketRe.FindStringSubmatch(line); len(m) == 2 {
			idx, _ := strconv.Atoi(m[1])
			addIndex(idx)
			continue
		}
		if strings.HasPrefix(line, "GPU[") {
			if m := regexp.MustCompile(`GPU\[(\d+)\]`).FindStringSubmatch(line); len(m) == 2 {
				idx, _ := strconv.Atoi(m[1])
				addIndex(idx)
			}
			continue
		}
		if strings.HasPrefix(line, "HCU") || strings.HasPrefix(line, "GPU") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		idx, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		addIndex(idx)
	}
	if len(indices) == 0 {
		for i, line := range strings.Split(output, "\n") {
			if m := hySmiIndexRe.FindStringSubmatch(line); len(m) == 2 {
				idx, err := strconv.Atoi(m[1])
				if err == nil && i > 0 {
					addIndex(idx)
				}
			}
		}
	}
	return indices
}

func parseHySmiMemInfoVram(output string) map[int]int {
	ret := map[int]int{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// HCU[0]          : vram Total Memory (MiB): 65520
		if m := hySmiHCUMemTotalRe.FindStringSubmatch(line); len(m) == 3 {
			idx, _ := strconv.Atoi(m[1])
			mem, _ := strconv.Atoi(m[2])
			ret[idx] = mem
			continue
		}
		if m := regexp.MustCompile(`GPU\[(\d+)\]\s*:\s*(?:Total Memory|VRAM Total Memory).*?(\d+)`).FindStringSubmatch(line); len(m) == 3 {
			idx, _ := strconv.Atoi(m[1])
			val, _ := strconv.ParseInt(m[2], 10, 64)
			if val > 1024*1024 {
				ret[idx] = int(val / 1024 / 1024)
			} else {
				ret[idx] = int(val)
			}
			continue
		}
		if m := regexp.MustCompile(`(?i)^(\d+)\s+.*?(\d+)\s*MiB`).FindStringSubmatch(line); len(m) == 3 {
			idx, _ := strconv.Atoi(m[1])
			mem, _ := strconv.Atoi(m[2])
			ret[idx] = mem
		}
	}
	return ret
}

func parseHySmiComputeUnits(output string) map[int]int {
	ret := map[int]int{}
	currentIdx := -1
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := regexp.MustCompile(`(?i)(?:Device|GPU|HCU)\s*(\d+)`).FindStringSubmatch(line); len(m) == 2 {
			currentIdx, _ = strconv.Atoi(m[1])
		}
		if m := hySmiComputeRe.FindStringSubmatch(line); len(m) == 2 && currentIdx >= 0 {
			cu, _ := strconv.Atoi(m[1])
			ret[currentIdx] = cu
		}
	}
	return ret
}
