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

package hostinfo

import (
	"context"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/pod"
	"yunion.io/x/onecloud/pkg/util/pod/cadvisor"
	"yunion.io/x/onecloud/pkg/util/pod/stats"
)

func (h *SHostInfo) initCRI() error {
	cri, err := pod.NewCRI(h.GetContainerRuntimeEndpoint(), 3*time.Second)
	if err != nil {
		return errors.Wrapf(err, "New CRI by endpoint %q", h.GetContainerRuntimeEndpoint())
	}
	ver, err := cri.Version(context.Background())
	if err != nil {
		return errors.Wrap(err, "get runtime version")
	}
	log.Infof("Init container runtime: %s", ver)
	h.cri = cri
	return nil
}

func (h *SHostInfo) initContainerCPUMap(topo *hostapi.HostTopology) error {
	statefile := path.Join(options.HostOptions.ServersPath, "container_cpu_map")
	cm, err := pod.NewHostContainerCPUMap(topo, statefile)
	if err != nil {
		return errors.Wrap(err, "NewHostContainerCPUMap")
	}
	h.containerCPUMap = cm
	return nil
}

func (h *SHostInfo) startContainerStatsProvider(cri pod.CRI) error {
	ca, err := cadvisor.New(nil, "/opt/cloud/workspace", []string{"cloudpods"})
	if err != nil {
		return errors.Wrap(err, "new cadvisor")
	}
	if err := ca.Start(); err != nil {
		return errors.Wrap(err, "start cadvisor")
	}
	h.containerStatsProvider = stats.NewCRIContainerStatsProvider(ca, cri.GetRuntimeClient(), cri.GetImageClient())
	return nil
}

func (h *SHostInfo) GetCRI() pod.CRI {
	return h.cri
}

func (h *SHostInfo) GetContainerCPUMap() *pod.HostContainerCPUMap {
	return h.containerCPUMap
}

func (h *SHostInfo) GetContainerStatsProvider() stats.ContainerStatsProvider {
	return h.containerStatsProvider
}

type INvidiaGpuIndexMemoryInterface interface {
	GetMemorySize() int
	GetNvidiaDevIndex() string
}

func (h *SHostInfo) GetNvidiaGpuIndexMemoryMap() map[string]int {
	res := map[string]int{}
	for i := range h.containerNvidiaGpus {
		iDev, ok := h.containerNvidiaGpus[i].(INvidiaGpuIndexMemoryInterface)
		if !ok {
			continue
		}
		index := iDev.GetNvidiaDevIndex()
		memSize := iDev.GetMemorySize()
		res[index] = memSize
	}
	return res
}

func (h *SHostInfo) GetNvidiaGpuIndexByDeviceIds(ids []string) map[string]int {
	h.HasContainerNvidiaGpu()
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			want[id] = struct{}{}
		}
	}
	res := map[string]int{}
	for i := range h.containerNvidiaGpus {
		iDev := h.containerNvidiaGpus[i]
		cloudId := iDev.GetCloudId()
		if _, ok := want[cloudId]; !ok {
			continue
		}
		if _, exists := res[cloudId]; exists {
			continue
		}
		nv, ok := iDev.(INvidiaGpuIndexMemoryInterface)
		if !ok {
			continue
		}
		idx, err := strconv.Atoi(nv.GetNvidiaDevIndex())
		if err != nil {
			log.Errorf("failed parse nvidia gpu index %s: %s", nv.GetNvidiaDevIndex(), err)
			continue
		}
		res[cloudId] = idx
	}
	return res
}

func (h *SHostInfo) HasContainerVastaitechGpu() bool {
	if !h.IsContainerHost() {
		return false
	}
	if h.hasVastaitechGpus != nil {
		return *h.hasVastaitechGpus
	}
	hasVastaitechGpus := false
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.VASTAITECH_VENDOR_ID {
			hasVastaitechGpus = true
		}
	}
	h.hasVastaitechGpus = &hasVastaitechGpus
	return *h.hasVastaitechGpus
}

func (h *SHostInfo) HasContainerAscendNpu() bool {
	if !h.IsContainerHost() {
		return false
	}
	if h.hasAscendNpus != nil {
		return *h.hasAscendNpus
	}
	hasAscendNpus := false
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.ASCEND_VENDOR_ID {
			hasAscendNpus = true
		}
	}
	h.hasAscendNpus = &hasAscendNpus
	return *h.hasAscendNpus
}

func (h *SHostInfo) HasContainerKunlunxinXpu() bool {
	if !h.IsContainerHost() {
		return false
	}
	if h.hasKunlunxinXpu != nil {
		return *h.hasKunlunxinXpu
	}
	hasKunlunxinXpu := false
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.KUNLUNXIN_VENDOR_ID {
			hasKunlunxinXpu = true
		}
	}
	h.hasKunlunxinXpu = &hasKunlunxinXpu
	return *h.hasKunlunxinXpu
}

func (h *SHostInfo) HasContainerAmdGpu() bool {
	if !h.IsContainerHost() {
		return false
	}
	if h.hasAmdGpus != nil {
		return *h.hasAmdGpus
	}
	hasAmdGpus := false
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.AMD_VENDOR_ID {
			hasAmdGpus = true
		}
	}
	h.hasAmdGpus = &hasAmdGpus
	return *h.hasAmdGpus
}

func (h *SHostInfo) HasContainerIluvatarGpu() bool {
	if !h.IsContainerHost() {
		return false
	}
	if h.hasIluvatarGpus != nil {
		return *h.hasIluvatarGpus
	}
	hasIluvatarGpus := false
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.ILUVATAR_VENDOR_ID {
			hasIluvatarGpus = true
		}
	}
	h.hasIluvatarGpus = &hasIluvatarGpus
	return *h.hasIluvatarGpus
}

func (h *SHostInfo) GetContainerIluvatarGpuMemSizeByIndex(index int) int {
	if h.iluvatarGpuMemSize != nil {
		if size, ok := h.iluvatarGpuMemSize[index]; ok {
			return size
		}
	} else {
		h.iluvatarGpuMemSize = map[int]int{}
	}

	retSize := -1
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.ILUVATAR_VENDOR_ID && devs[i].GetIndex() == index {
			retSize = devs[i].GetMemorySize()
		}
	}
	h.iluvatarGpuMemSize[index] = retSize
	return retSize
}

func (h *SHostInfo) HasContainerNvidiaGpu() bool {
	if !h.IsContainerHost() {
		return false
	}
	if h.hasNvidiaGpus != nil {
		return *h.hasNvidiaGpus
	}
	hasNvidiaGpus := false
	nvDevs := make([]isolated_device.IDevice, 0)
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.Split(devs[i].GetVendorDeviceId(), ":")[0]
		if vendorId == apis.NVIDIA_VENDOR_ID {
			hasNvidiaGpus = true
			nvDevs = append(nvDevs, devs[i])
		}

	}
	h.hasNvidiaGpus = &hasNvidiaGpus
	h.containerNvidiaGpus = nvDevs
	return *h.hasNvidiaGpus
}

func (h *SHostInfo) HasContainerMthreadsGpu() bool {
	if h.hasMthreadsGpus != nil {
		return *h.hasMthreadsGpus
	}
	hasMthreadsGpus := false
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		vendorId := strings.ToLower(strings.Split(devs[i].GetVendorDeviceId(), ":")[0])
		if vendorId == apis.MTHREADS_VENDOR_ID {
			hasMthreadsGpus = true
			break
		}
	}
	h.hasMthreadsGpus = &hasMthreadsGpus
	return *h.hasMthreadsGpus
}

func (h *SHostInfo) GetMthreadsGpuIndexes(ids, addrs []string) []int {
	if len(ids) == 0 && len(addrs) == 0 {
		return nil
	}
	wantId := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			wantId[id] = struct{}{}
		}
	}
	wantAddr := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr != "" {
			wantAddr = append(wantAddr, addr)
		}
	}
	seen := map[int]struct{}{}
	out := make([]int, 0)
	devs := h.IsolatedDeviceMan.GetDevices()
	for i := range devs {
		dev := devs[i]
		vendorId := strings.ToLower(strings.Split(dev.GetVendorDeviceId(), ":")[0])
		if vendorId != apis.MTHREADS_VENDOR_ID {
			continue
		}
		_, idMatch := wantId[dev.GetCloudId()]
		addrMatch := false
		if !idMatch {
			for _, addr := range wantAddr {
				if samePCIAddr(dev.GetAddr(), addr) {
					addrMatch = true
					break
				}
			}
		}
		if !idMatch && !addrMatch {
			continue
		}
		idx := dev.GetIndex()
		if idx < 0 {
			n, err := strconv.Atoi(dev.GetDevicePath())
			if err != nil {
				continue
			}
			idx = n
		}
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		out = append(out, idx)
	}
	sort.Ints(out)
	return out
}

func samePCIAddr(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return strings.HasSuffix(a, ":"+b) || strings.HasSuffix(b, ":"+a)
}
