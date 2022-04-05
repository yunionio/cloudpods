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
	if len(data.IsolatedDevices) == 0 {
		return false, nil
	}
	return true, nil
}

func (f *IsolatedDevicePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(f, u, c)
	reqIsoDevs := u.SchedData().IsolatedDevices
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
		freeCount := len(getter.UnusedIsolatedDevicesByType(devType))
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

	h.SetCapacity(minCapacity)
	return h.GetResult()
}
