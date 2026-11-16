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

package gpudb

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/pkg/errors"
)

var GPUInfos []*SGPUInfo
var gpuInfosMap map[string]map[string][]*SGPUInfo

var gpuInfoList [][]SGPUInfo = [][]SGPUInfo{
	nvidiaGPUInfos,
	amdGPUInfos,
	huaweiGPUInfos,
	kunlunxinGPUInfos,
	iluvatarGPUInfos,
	vastaitechGPUInfos,
	intelGPUInfos,
	hygonGPUInfos,
	birenGPUInfos,
	mthreadsGPUInfos,
	theadGPUInfos,
	cambriconGPUInfos,
	enflameGPUInfos,
	metaxGPUInfos,
}

func init() {
	gpuInfosMap = make(map[string]map[string][]*SGPUInfo)
	for _, gpuInfo := range gpuInfoList {
		for i := range gpuInfo {
			rec := new(SGPUInfo)
			*rec = gpuInfo[i]
			indexGPUInfo(rec)
			GPUInfos = append(GPUInfos, rec)
		}
	}
	sortGPUInfosMap()
}

func sortGPUInfosMap() {
	for _, devices := range gpuInfosMap {
		for _, list := range devices {
			sort.Slice(list, func(i, j int) bool {
				if list[i].MemorySizeGB != list[j].MemorySizeGB {
					return list[i].MemorySizeGB > list[j].MemorySizeGB
				}
				return list[i].ChipsOnDevice < list[j].ChipsOnDevice
			})
		}
	}
}

func indexGPUInfo(gpu *SGPUInfo) {
	vendorId := strings.ToLower(strings.TrimSpace(gpu.VendorId))
	if vendorId == "" {
		return
	}
	devices := gpuInfosMap[vendorId]
	if devices == nil {
		devices = make(map[string][]*SGPUInfo)
		gpuInfosMap[vendorId] = devices
	}
	deviceId := strings.ToLower(gpu.DeviceId)
	for _, old := range devices[deviceId] {
		if old.ChipsOnDevice == gpu.ChipsOnDevice && old.MemorySizeGB == gpu.MemorySizeGB {
			panic(fmt.Sprintf("gpudb: duplicate %s:%s chips=%d memory=%dGB (%s, %s)",
				vendorId, deviceId, gpu.ChipsOnDevice, gpu.MemorySizeGB, old.Name, gpu.Name))
		}
	}
	devices[deviceId] = append(devices[deviceId], gpu)
}

func GetGPUInfo(vendorId string, deviceId string, memSizeGB uint, chipsOnDevice uint) *SGPUInfo {
	vendorId = strings.ToLower(strings.TrimSpace(vendorId))
	deviceId = strings.ToLower(deviceId)
	devices, ok := gpuInfosMap[vendorId]
	if !ok {
		return nil
	}
	list := devices[deviceId]
	if len(list) == 0 {
		return nil
	} else if len(list) == 1 {
		copied := *list[0]
		return &copied
	}
	if memSizeGB >= list[0].MemorySizeGB || memSizeGB < list[len(list)-1].MemorySizeGB {
		memSizeGB = list[0].MemorySizeGB
	}
	var matched []*SGPUInfo
	for i := range list {
		if (i == 0 && memSizeGB >= list[i].MemorySizeGB) || (i > 0 && memSizeGB < list[i-1].MemorySizeGB && memSizeGB >= list[i].MemorySizeGB) {
			matched = append(matched, list[i])
		}
	}
	if len(matched) == 0 {
		return nil
	} else if len(matched) == 1 {
		copied := *matched[0]
		return &copied
	}
	if chipsOnDevice <= matched[0].ChipsOnDevice || chipsOnDevice > matched[len(matched)-1].ChipsOnDevice {
		chipsOnDevice = matched[0].ChipsOnDevice
	}
	for i := range matched {
		if (i == 0 && chipsOnDevice <= matched[i].ChipsOnDevice) || (i > 0 && chipsOnDevice > matched[i-1].ChipsOnDevice && chipsOnDevice <= matched[i].ChipsOnDevice) {
			copied := *matched[i]
			return &copied
		}
	}
	return nil
}

func GetGPUInfo2(vendorDeviceId string, memSizeGB uint, chipsOnDevice uint) (*SGPUInfo, error) {
	vendorId, deviceId, ok := strings.Cut(vendorDeviceId, ":")
	if !ok {
		return nil, errors.Wrapf(errors.ErrInvalidFormat, "invalid vendorDeviceId: %s", vendorDeviceId)
	}
	return GetGPUInfo(vendorId, deviceId, memSizeGB, chipsOnDevice), nil
}
