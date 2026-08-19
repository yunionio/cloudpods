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
	"path"
	"strings"

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

func (f *IsolatedDevicePredicate) getIsolatedDeviceCountBySharingMode(sharingMode string, devs []*core.IsolatedDeviceDesc) int {
	return isolatedDeviceHostFree(sharingMode, devs)
}

// isolatedDeviceHostFree is the value shown as hostFree in shortage messages.
// HAMI: sum of (memory_size - memory_size_allocated) on the matched pool,
// counted before min-memory filtering so remaining-but-too-small cards are
// not reported as 0.
func isolatedDeviceHostFree(sharingMode string, devs []*core.IsolatedDeviceDesc) int {
	if sharingMode == compute.DEVICE_SHARING_MODE_HAMI {
		ret := 0
		for i := range devs {
			ret += devs[i].AvailableMemorySize()
		}
		return ret
	}
	ret := 0
	for i := range devs {
		ret += devs[i].AvailableNum()
	}
	return ret
}

// countDevicesWithMinMemory counts available capacity for devices of the given
// dev_type whose MemorySize satisfies the minimum requirement. Devices with
// MemorySize == 0 are treated as "unknown" and pass through for non-HAMI modes
// so rows that have not been backfilled yet do not exclude every host.
func (f *IsolatedDevicePredicate) countDevicesWithMinMemory(getter core.CandidatePropertyGetter, devType, sharingMode string, minMemoryMb int) int {
	devs := getter.AvailableIsolatedDevicesByTypeSharingMode(devType, sharingMode)
	return countDevicesWithMinMemoryFromList(devs, sharingMode, minMemoryMb)
}

// countDevicesWithMinMemoryFromList is the pure-function core of the memory
// fit count, factored out for unit testing. Callers pass an already-filtered
// list (typically by dev_type).
func countDevicesWithMinMemoryFromList(devs []*core.IsolatedDeviceDesc, sharingMode string, minMemoryMb int) int {
	return len(filterDescsByMinMemory(devs, sharingMode, minMemoryMb))
}

func filterDevicesByTypeSharingMode(devs []*core.IsolatedDeviceDesc, devType, sharingMode string) []*core.IsolatedDeviceDesc {
	ret := make([]*core.IsolatedDeviceDesc, 0)
	for _, dev := range devs {
		if devType != "" && devType != dev.DevType {
			continue
		}
		if sharingMode != "" && dev.SharingMode != sharingMode {
			continue
		}
		ret = append(ret, dev)
	}
	return ret
}

// filterDescsByMinMemory drops devices that cannot satisfy minMemoryMb.
// Non-HAMI: MemorySize > 0 and < minMemoryMb are excluded; MemorySize == 0
// (unknown / not yet reported) is allowed through. HAMI uses remaining
// AvailableMemorySize. minMemoryMb <= 0 is a no-op.
func filterDescsByMinMemory(devs []*core.IsolatedDeviceDesc, sharingMode string, minMemoryMb int) []*core.IsolatedDeviceDesc {
	if minMemoryMb <= 0 {
		return devs
	}
	out := make([]*core.IsolatedDeviceDesc, 0, len(devs))
	for _, d := range devs {
		if sharingMode == compute.DEVICE_SHARING_MODE_HAMI {
			if d.AvailableMemorySize() < minMemoryMb {
				continue
			}
		} else if d.MemorySize > 0 && d.MemorySize < minMemoryMb {
			continue
		}
		out = append(out, d)
	}
	return out
}

func isolatedDeviceRequestAmount(dev *compute.IsolatedDeviceConfig) int {
	if dev.SharingMode == compute.DEVICE_SHARING_MODE_HAMI {
		return dev.MemoryRequest
	}
	return 1
}

func isolatedDeviceMinMemory(dev *compute.IsolatedDeviceConfig) int {
	minMemMb := dev.MemoryMb
	if dev.SharingMode == compute.DEVICE_SHARING_MODE_HAMI && dev.MemoryRequest > minMemMb {
		minMemMb = dev.MemoryRequest
	}
	return minMemMb
}

func maxMinMemoryForRequests(reqs []*compute.IsolatedDeviceConfig, match func(*compute.IsolatedDeviceConfig) bool) int {
	maxMem := 0
	for _, dev := range reqs {
		if !match(dev) {
			continue
		}
		if m := isolatedDeviceMinMemory(dev); m > maxMem {
			maxMem = m
		}
	}
	return maxMem
}

type isolatedDeviceShortageSpec struct {
	devType        string
	sharingMode    string
	vendor         string
	model          string
	devicePath     string
	minMemMb       int
	amountIsMemory bool
}

func uniqueJoinedFields(reqs []*compute.IsolatedDeviceConfig, match func(*compute.IsolatedDeviceConfig) bool, field func(*compute.IsolatedDeviceConfig) string) string {
	seen := make(map[string]struct{})
	vals := make([]string, 0)
	for _, dev := range reqs {
		if !match(dev) {
			continue
		}
		v := field(dev)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		vals = append(vals, v)
	}
	return strings.Join(vals, ",")
}

func shortageSpecFromRequests(reqs []*compute.IsolatedDeviceConfig, match func(*compute.IsolatedDeviceConfig) bool, devType, sharingMode string, minMemMb int) isolatedDeviceShortageSpec {
	spec := isolatedDeviceShortageSpec{
		devType:        devType,
		sharingMode:    sharingMode,
		minMemMb:       minMemMb,
		amountIsMemory: sharingMode == compute.DEVICE_SHARING_MODE_HAMI,
	}
	spec.vendor = uniqueJoinedFields(reqs, match, func(d *compute.IsolatedDeviceConfig) string { return d.Vendor })
	spec.model = uniqueJoinedFields(reqs, match, func(d *compute.IsolatedDeviceConfig) string { return d.Model })
	return spec
}

func isolatedDeviceAmountLabel(n int, asMemory bool) string {
	if asMemory {
		return fmt.Sprintf("%d MiB", n)
	}
	return fmt.Sprintf("%d", n)
}

func isolatedDeviceShortageMessage(spec isolatedDeviceShortageSpec, reqCount, hostFree, pending int) string {
	var b strings.Builder
	b.WriteString("IsolatedDevice")
	if spec.devType != "" {
		fmt.Fprintf(&b, " type %q", spec.devType)
	}
	if spec.vendor != "" {
		fmt.Fprintf(&b, " vendor %q", spec.vendor)
	}
	if spec.model != "" {
		fmt.Fprintf(&b, " model %q", spec.model)
	}
	if spec.sharingMode != "" {
		fmt.Fprintf(&b, " sharing_mode %q", spec.sharingMode)
	}
	if spec.devicePath != "" {
		fmt.Fprintf(&b, " device_path %q", spec.devicePath)
	}
	if spec.minMemMb > 0 {
		fmt.Fprintf(&b, " memory>=%dMiB", spec.minMemMb)
	}
	fmt.Fprintf(&b, " not enough, request: %s, hostFree: %s",
		isolatedDeviceAmountLabel(reqCount, spec.amountIsMemory),
		isolatedDeviceAmountLabel(hostFree, spec.amountIsMemory))
	if pending > 0 {
		fmt.Fprintf(&b, ", pending: %s", isolatedDeviceAmountLabel(pending, spec.amountIsMemory))
	}
	return b.String()
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
	pendingUsage := getter.GetPendingUsage().IsolatedDevice

	// check by specify device id
	for _, dev := range reqIsoDevs {
		if len(dev.Id) == 0 {
			continue
		}
		if fDev := getter.GetIsolatedDevice(dev.Id); fDev != nil {
			if fDev.IsUsedUp() {
				h.Exclude(fmt.Sprintf("IsolatedDevice %q already used up", dev.Id))
				return h.GetResult()
			}
		} else {
			h.Exclude(fmt.Sprintf("Not found IsolatedDevice %q", dev.Id))
			return h.GetResult()
		}
		minCapacity = 1
	}
	type reqKey struct {
		devType     string
		sharingMode string
	}
	// check host device by type
	devTypeRequest := make(map[reqKey]int, 0)
	for _, dev := range reqIsoDevs {
		if len(dev.DevType) != 0 {
			key := reqKey{devType: dev.DevType, sharingMode: dev.SharingMode}
			reqAmount := isolatedDeviceRequestAmount(dev)
			if reqAmount <= 0 {
				h.Exclude(fmt.Sprintf("IsolatedDevice type %q sharing_mode %q request amount must be positive", dev.DevType, dev.SharingMode))
				return h.GetResult()
			}
			devTypeRequest[key] += reqAmount
		}
	}
	for key, reqCount := range devTypeRequest {
		devType, sharingMode := key.devType, key.sharingMode
		matched := getter.AvailableIsolatedDevicesByTypeSharingMode(devType, sharingMode)
		match := func(d *compute.IsolatedDeviceConfig) bool {
			return d.DevType == devType && d.SharingMode == sharingMode
		}
		minMem := maxMinMemoryForRequests(reqIsoDevs, match)
		hostFree := isolatedDeviceHostFree(sharingMode, matched)
		devs := filterDescsByMinMemory(matched, sharingMode, minMem)
		pendingCnt := pendingUsage.Get(path.Join(devType, sharingMode))
		freeCount := f.getIsolatedDeviceCountBySharingMode(sharingMode, devs)
		if freeCount < (reqCount + pendingCnt) {
			spec := shortageSpecFromRequests(reqIsoDevs, match, devType, sharingMode, minMem)
			h.Exclude(isolatedDeviceShortageMessage(spec, reqCount, hostFree, pendingCnt))
			return h.GetResult()
		}
		cap := freeCount / reqCount
		if int64(cap) < minCapacity {
			minCapacity = int64(cap)
		}
	}

	// check host device by model
	type modelReqKey struct {
		vendorModel string
		devType     string
		sharingMode string
	}
	devVendorModelRequest := make(map[modelReqKey]int, 0)
	for _, dev := range reqIsoDevs {
		if len(dev.Model) != 0 {
			key := modelReqKey{
				vendorModel: fmt.Sprintf("%s:%s", dev.Vendor, dev.Model),
				devType:     dev.DevType,
				sharingMode: dev.SharingMode,
			}
			reqAmount := isolatedDeviceRequestAmount(dev)
			if reqAmount <= 0 {
				h.Exclude(fmt.Sprintf("IsolatedDevice vendor:model %q request amount must be positive", key.vendorModel))
				return h.GetResult()
			}
			devVendorModelRequest[key] += reqAmount
		}
	}
	for key, reqCount := range devVendorModelRequest {
		matched := filterDevicesByTypeSharingMode(getter.AvailableIsolatedDevicesByVendorModel(key.vendorModel), key.devType, key.sharingMode)
		match := func(d *compute.IsolatedDeviceConfig) bool {
			return fmt.Sprintf("%s:%s", d.Vendor, d.Model) == key.vendorModel &&
				d.DevType == key.devType && d.SharingMode == key.sharingMode
		}
		minMem := maxMinMemoryForRequests(reqIsoDevs, match)
		hostFree := isolatedDeviceHostFree(key.sharingMode, matched)
		devs := filterDescsByMinMemory(matched, key.sharingMode, minMem)
		spec := shortageSpecFromRequests(reqIsoDevs, match, key.devType, key.sharingMode, minMem)
		pendingCnt := pendingUsage.Get(path.Join(key.devType, key.sharingMode))
		freeCount := f.getIsolatedDeviceCountBySharingMode(key.sharingMode, devs)
		if len(devs) == 0 || freeCount < (reqCount+pendingCnt) {
			h.Exclude(isolatedDeviceShortageMessage(spec, reqCount, hostFree, pendingCnt))
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
		devType     string
		sharingMode string
		minMemMb    int
	}
	vramReq := make(map[vramReqKey]int)
	for _, dev := range reqIsoDevs {
		minMemMb := isolatedDeviceMinMemory(dev)
		if minMemMb <= 0 {
			continue
		}
		vramReq[vramReqKey{dev.DevType, dev.SharingMode, minMemMb}]++
	}
	for k, reqCnt := range vramReq {
		fit := f.countDevicesWithMinMemory(getter, k.devType, k.sharingMode, k.minMemMb)
		if fit < reqCnt {
			match := func(d *compute.IsolatedDeviceConfig) bool {
				return d.DevType == k.devType && d.SharingMode == k.sharingMode && isolatedDeviceMinMemory(d) == k.minMemMb
			}
			spec := shortageSpecFromRequests(reqIsoDevs, match, k.devType, k.sharingMode, k.minMemMb)
			// VRAM fit is a device count even for HAMI (cards with enough remaining memory).
			spec.amountIsMemory = false
			h.Exclude(isolatedDeviceShortageMessage(spec, reqCnt, fit, 0))
			return h.GetResult()
		}
		cap := fit / reqCnt
		if int64(cap) < minCapacity {
			minCapacity = int64(cap)
		}
	}

	// check host device by device_path
	type devicePathReqKey struct {
		devicePath  string
		devType     string
		sharingMode string
	}
	devicePathReq := make(map[devicePathReqKey]int, 0)
	for _, dev := range reqIsoDevs {
		if len(dev.DevicePath) != 0 {
			key := devicePathReqKey{
				devicePath:  dev.DevicePath,
				devType:     dev.DevType,
				sharingMode: dev.SharingMode,
			}
			reqAmount := isolatedDeviceRequestAmount(dev)
			if reqAmount <= 0 {
				h.Exclude(fmt.Sprintf("IsolatedDevice device_path %q request amount must be positive", dev.DevicePath))
				return h.GetResult()
			}
			devicePathReq[key] += reqAmount
		}
	}
	for key, reqCnt := range devicePathReq {
		matched := filterDevicesByTypeSharingMode(getter.AvailableIsolatedDevicesByDevicePath(key.devicePath), key.devType, key.sharingMode)
		match := func(d *compute.IsolatedDeviceConfig) bool {
			return d.DevicePath == key.devicePath && d.DevType == key.devType && d.SharingMode == key.sharingMode
		}
		minMem := maxMinMemoryForRequests(reqIsoDevs, match)
		hostFree := isolatedDeviceHostFree(key.sharingMode, matched)
		devs := filterDescsByMinMemory(matched, key.sharingMode, minMem)
		spec := shortageSpecFromRequests(reqIsoDevs, match, key.devType, key.sharingMode, minMem)
		spec.devicePath = key.devicePath
		pendingCnt := pendingUsage.Get(path.Join(key.devType, key.sharingMode))
		freeCount := f.getIsolatedDeviceCountBySharingMode(key.sharingMode, devs)
		if len(devs) == 0 || freeCount < (reqCnt+pendingCnt) {
			h.Exclude(isolatedDeviceShortageMessage(spec, reqCnt, hostFree, pendingCnt))
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
