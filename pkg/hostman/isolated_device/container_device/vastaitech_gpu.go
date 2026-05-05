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
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/pkg/errors"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	fileutils "yunion.io/x/onecloud/pkg/util/fileutils2"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newVastaitechGPUManager())
}

type vastaitechGPUManager struct{}

func newVastaitechGPUManager() isolated_device.IContainerDeviceManager {
	return &vastaitechGPUManager{}
}

func (v vastaitechGPUManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeVastaitechGpu
}

const (
	VASTAITECH_VA_CTL   = "va_ctl"
	VASTAITECH_VA_VIDEO = "va_video"
	VASTAITECH_VACC     = "vacc"
)

var vastaitechRelatedDevices = map[string]string{
	VASTAITECH_VA_CTL:   "/dev/va%d_ctl",
	VASTAITECH_VA_VIDEO: "/dev/va_video%d",
	VASTAITECH_VACC:     "/dev/vacc%d",
}

func (v vastaitechGPUManager) getRelatedDevices(index int) map[string]string {
	devs := make(map[string]string)
	for key, devFmt := range vastaitechRelatedDevices {
		devs[key] = fmt.Sprintf(devFmt, index)
	}
	return devs
}

func (v vastaitechGPUManager) getDriRenderPrefix() string {
	return "/dev/dri/renderD"
}

// getVastaitechDriStartIndexFromByPath 扫描 /dev/dri/by-path/，找到名称含 va_card 的 -render 链接对应的最小 renderD 编号
func (v vastaitechGPUManager) getVastaitechDriStartIndexFromByPath() (int, error) {
	const byPathDir = "/dev/dri/by-path"
	entries, err := os.ReadDir(byPathDir)
	if err != nil {
		return 0, errors.Wrapf(err, "read %s", byPathDir)
	}
	const renderDSuffix = "renderD"
	var minIdx *int
	for _, entry := range entries {
		entryName := entry.Name()
		if !strings.Contains(entryName, "va_card") || !strings.HasSuffix(entryName, "-render") {
			continue
		}
		fp := path.Join(byPathDir, entryName)
		linkPath, err := os.Readlink(fp)
		if err != nil {
			return 0, errors.Wrapf(err, "readlink %s", fp)
		}
		// linkPath is e.g. "../renderD129"
		base := path.Base(linkPath)
		if !strings.HasPrefix(base, renderDSuffix) {
			continue
		}
		idxStr := strings.TrimPrefix(base, renderDSuffix)
		driIdx, err := strconv.Atoi(idxStr)
		if err != nil {
			return 0, errors.Wrapf(err, "parse render index from %s", linkPath)
		}
		if minIdx == nil || driIdx < *minIdx {
			minIdx = &driIdx
		}
	}
	if minIdx == nil {
		return 0, errors.Errorf("no va_card render device found in %s", byPathDir)
	}
	return *minIdx, nil
}

func (v vastaitechGPUManager) getDriStartIndex() (int, error) {
	return v.getVastaitechDriStartIndexFromByPath()
}

func (v vastaitechGPUManager) getRelatedDeviceStartIndex(driPath string) (int, error) {
	prefix := v.getDriRenderPrefix()
	if !strings.HasPrefix(driPath, prefix) {
		return -1, errors.Errorf("device path %q doesn't start with /dev/dri/renderD", driPath)
	}
	idxStr := strings.ReplaceAll(driPath, prefix, "")
	driIdx, err := strconv.Atoi(idxStr)
	if err != nil {
		return -1, errors.Wrapf(err, "convert %s to int", idxStr)
	}
	startIndex, err := v.getDriStartIndex()
	if err != nil {
		return -1, err
	}
	idx := driIdx - startIndex
	if idx < 0 {
		return -1, errors.Errorf("%s index is less than %d", driPath, startIndex)
	}
	return idx, nil
}

func (v vastaitechGPUManager) NewDevices(dev *isolated_device.ContainerDevice) ([]isolated_device.IDevice, error) {
	idx, err := v.getRelatedDeviceStartIndex(dev.Path)
	if err != nil {
		return nil, errors.Wrap(err, "get related device start index")
	}
	// check related devices
	for _, devPath := range v.getRelatedDevices(idx) {
		if !fileutils.Exists(devPath) {
			return nil, errors.Wrapf(errors.ErrNotFound, "related device %s not found of %s", devPath, dev.Path)
		}
	}
	if err := CheckVirtualNumber(dev); err != nil {
		return nil, err
	}
	gpuDevs := make([]isolated_device.IDevice, 0)
	for i := 0; i < dev.VirtualNumber; i++ {
		gpuDev, err := newVastaitechGPU(dev.Path, i)
		if err != nil {
			return nil, errors.Wrapf(err, "new CPH AMD GPU with index %d", i)
		}
		gpuDevs = append(gpuDevs, gpuDev)
	}
	return gpuDevs, nil
}

func (v vastaitechGPUManager) getCommonDevices() []*runtimeapi.Device {
	vatools := "/dev/vatools"
	vaSync := "/dev/va_sync"
	devs := []*runtimeapi.Device{}
	for _, devPath := range []string{vatools, vaSync} {
		devs = append(devs, &runtimeapi.Device{
			ContainerPath: devPath,
			HostPath:      devPath,
			Permissions:   "rwm",
		})
	}
	return devs
}

func (v vastaitechGPUManager) NewContainerDevices(input *hostapi.ContainerCreateInput, dev *hostapi.ContainerDevice) ([]*runtimeapi.Device, []*runtimeapi.Device, error) {
	driHostPath := dev.IsolatedDevice.Path
	idx, err := v.getRelatedDeviceStartIndex(driHostPath)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get related device start index by %s", driHostPath)
	}
	perms := "rwm"
	devs := []*runtimeapi.Device{
		{
			HostPath:      driHostPath,
			ContainerPath: driHostPath,
			Permissions:   perms,
		},
	}
	for _, devPath := range v.getRelatedDevices(idx) {
		devs = append(devs, &runtimeapi.Device{
			HostPath:      devPath,
			ContainerPath: devPath,
			Permissions:   perms,
		})
	}
	return devs, v.getCommonDevices(), nil
}

func (v vastaitechGPUManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return nil, nil
}

func (v vastaitechGPUManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	return nil, nil
}

type vastaitechGPU struct {
	*BaseDevice
}

func newVastaitechGPU(devPath string, index int) (*vastaitechGPU, error) {
	dev, err := NewPCIGPURenderBaseDevice(devPath, index, isolated_device.ContainerDeviceTypeVastaitechGpu)
	if err != nil {
		return nil, errors.Wrap(err, "new PCIGPURenderBaseDevice")
	}
	return &vastaitechGPU{BaseDevice: dev}, nil
}
