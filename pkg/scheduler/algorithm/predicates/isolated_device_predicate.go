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

package predicates

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// IsolatedDevicePredicate check mode, and number of scheduled
// device configurations and current resources.
type IsolatedDevicePredicate struct {
	BasePredicate
}

func (f *IsolatedDevicePredicate) Name() string {
	return "host_isolated_device"
}

func (f *IsolatedDevicePredicate) Clone() core.FitPredicate {
	return &IsolatedDevicePredicate{}
}

func (f *IsolatedDevicePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	data := u.SchedData()

	if data.ResetCpuNumaPin {
		return false, nil
	}

	if len(data.IsolatedDevices) > 0 {
		return true, nil
	}
	networks := data.Networks
	for i := 0; i < len(networks); i++ {
		if networks[i].SriovDevice != nil {
			return true, nil
		}
	}
	disks := data.Disks
	for i := 0; i < len(disks); i++ {
		if disks[i].NVMEDevice != nil {
			return true, nil
		}
	}
	return false, nil
}

func (f *IsolatedDevicePredicate) getIsolatedDeviceCountByType(getter core.CandidatePropertyGetter, devType string) int {
	devs := getter.UnusedIsolatedDevicesByType(devType)
	if devType != compute.CONTAINER_DEV_NVIDIA_MPS && devType != compute.CONTAINER_DEV_NVIDIA_GPU_SHARE {
		return len(devs)
	} else {
		devMap := map[string]struct{}{}
		for _, dev := range devs {
			devMap[dev.DevicePath] = struct{}{}
		}
		return len(devMap)
	}
}

// countDevicesWithMinMemory counts free devices of the given dev_type whose
// MemorySize satisfies the minimum requirement. Devices with MemorySize == 0
// are treated as "unknown" and pass through (so newly-introduced rows that
// haven't been backfilled yet don't accidentally exclude every host).
// For NVIDIA_MPS / NVIDIA_GPU_SHARE the count is deduplicated by DevicePath,
// matching getIsolatedDeviceCountByType.
func (f *IsolatedDevicePredicate) countDevicesWithMinMemory(getter core.CandidatePropertyGetter, devType string, minMemoryMb int) int {
	devs := getter.UnusedIsolatedDevicesByType(devType)
	isShared := devType == compute.CONTAINER_DEV_NVIDIA_MPS || devType == compute.CONTAINER_DEV_NVIDIA_GPU_SHARE
	return countDevicesWithMinMemoryFromList(devs, isShared, minMemoryMb)
}

// countDevicesWithMinMemoryFromList is the pure-function core of the memory
// fit count, factored out for unit testing. Callers pass an already-filtered
// list (typically by dev_type).
func countDevicesWithMinMemoryFromList(devs []*core.IsolatedDeviceDesc, isShared bool, minMemoryMb int) int {
	if !isShared {
		n := 0
		for _, d := range devs {
			if d.MemorySize > 0 && d.MemorySize < minMemoryMb {
				continue
			}
			n++
		}
		return n
	}
	seen := map[string]struct{}{}
	for _, d := range devs {
		if d.MemorySize > 0 && d.MemorySize < minMemoryMb {
			continue
		}
		seen[d.DevicePath] = struct{}{}
	}
	return len(seen)
}

func (f *IsolatedDevicePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(f, u, c)
	reqIsoDevs := u.SchedData().IsolatedDevices
	if reqIsoDevs == nil {
		reqIsoDevs = []*compute.IsolatedDeviceConfig{}
	}
	networks := u.SchedData().Networks
	for i := 0; i < len(networks); i++ {
		if networks[i].SriovDevice != nil {
			reqIsoDevs = append(reqIsoDevs, networks[i].SriovDevice)
		}
	}
	disks := u.SchedData().Disks
	for i := 0; i < len(disks); i++ {
		if disks[i].NVMEDevice != nil {
			reqIsoDevs = append(reqIsoDevs, disks[i].NVMEDevice)
		}
	}

	getter := c.Getter()
	minCapacity := int64(0xFFFFFFFF)

	// check by specify device id
	for _, dev := range reqIsoDevs {
		if len(dev.Id) == 0 {
			continue
		}
		if fDev := getter.GetIsolatedDevice(dev.Id); fDev != nil {
			if len(fDev.GuestID) != 0 {
				h.Exclude(fmt.Sprintf("IsolatedDevice %q already used by guest %q", dev.Id, fDev.GuestID))
				return h.GetResult()
			}
		} else {
			h.Exclude(fmt.Sprintf("Not found IsolatedDevice %q", dev.Id))
			return h.GetResult()
		}
		minCapacity = 1
	}

	reqCount := len(reqIsoDevs)
	freeCount := len(getter.UnusedIsolatedDevices()) - getter.GetPendingUsage().IsolatedDevice
	totalCount := len(getter.GetIsolatedDevices())

	// check host isolated device count
	if freeCount < reqCount {
		h.AppendInsufficientResourceError(int64(reqCount), int64(totalCount), int64(freeCount))
		h.Exclude(fmt.Sprintf(
			"IsolatedDevice count not enough, request: %d, hostTotal: %d, hostFree: %d",
			reqCount, totalCount, freeCount))
		return h.GetResult()
	}

	// check host device by type
	devTypeRequest := make(map[string]int, 0)
	for _, dev := range reqIsoDevs {
		if len(dev.DevType) != 0 {
			devTypeRequest[dev.DevType] += 1
		}
	}
	for devType, reqCount := range devTypeRequest {
		freeCount := f.getIsolatedDeviceCountByType(getter, devType)
		if freeCount < reqCount {
			h.Exclude(fmt.Sprintf("IsolatedDevice type %q not enough, request: %d, hostFree: %d", devType, reqCount, freeCount))
			return h.GetResult()
		}
		cap := freeCount / reqCount
		if int64(cap) < minCapacity {
			minCapacity = int64(cap)
		}
	}

	// check host device by model
	devVendorModelRequest := make(map[string]int, 0)
	for _, dev := range reqIsoDevs {
		if len(dev.Model) != 0 {
			devVendorModelRequest[fmt.Sprintf("%s:%s", dev.Vendor, dev.Model)] += 1
		}
	}
	for vendorModel, reqCount := range devVendorModelRequest {
		freeCount := len(getter.UnusedIsolatedDevicesByVendorModel(vendorModel))
		if freeCount < reqCount {
			h.Exclude(fmt.Sprintf("IsolatedDevice vendor:model %q not enough, request: %d, hostFree: %d", vendorModel, reqCount, freeCount))
			return h.GetResult()
		}
		cap := freeCount / reqCount
		if int64(cap) < minCapacity {
			minCapacity = int64(cap)
		}
	}

	// check host device by (type, min_memory_mb) — VRAM-aware fit for GPUs.
	// LLM scheduling stamps MemoryMb on each request entry so a SKU's
	// vram_claim_mb is honoured. Devices with memory_size == 0 are passed
	// through as unknown (see countDevicesWithMinMemory).
	type vramReqKey struct {
		devType  string
		minMemMb int
	}
	vramReq := make(map[vramReqKey]int)
	for _, dev := range reqIsoDevs {
		if dev.MemoryMb <= 0 {
			continue
		}
		vramReq[vramReqKey{dev.DevType, dev.MemoryMb}]++
	}
	for k, reqCnt := range vramReq {
		fit := f.countDevicesWithMinMemory(getter, k.devType, k.minMemMb)
		if fit < reqCnt {
			h.Exclude(fmt.Sprintf(
				"IsolatedDevice type %q with memory >= %d MiB not enough, request: %d, hostFree: %d",
				k.devType, k.minMemMb, reqCnt, fit))
			return h.GetResult()
		}
		cap := fit / reqCnt
		if int64(cap) < minCapacity {
			minCapacity = int64(cap)
		}
	}

	// check host device by device_path
	devicePathReq := make(map[string]int, 0)
	for _, dev := range reqIsoDevs {
		if len(dev.DevicePath) != 0 {
			devicePathReq[dev.DevicePath] += 1
		}
	}
	for devPath, reqCnt := range devicePathReq {
		freeCount := len(getter.UnusedIsolatedDevicesByDevicePath(devPath))
		if freeCount < reqCount {
			h.Exclude(fmt.Sprintf("IsolatedDevice device_path %q not enough, request: %d, hostFree: %d", devPath, reqCount, freeCount))
			return h.GetResult()
		}
		cap := freeCount / reqCnt
		if int64(cap) < minCapacity {
			minCapacity = int64(cap)
		}
	}

	h.SetCapacity(minCapacity)
	return h.GetResult()
}
