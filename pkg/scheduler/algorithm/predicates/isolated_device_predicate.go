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
	if devType != compute.CONTAINER_DEV_NVIDIA_MPS {
		return len(devs)
	} else {
		devMap := map[string]struct{}{}
		for _, dev := range devs {
			devMap[dev.DevicePath] = struct{}{}
		}
		return len(devMap)
	}
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
