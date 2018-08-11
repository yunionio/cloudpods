package guest

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// IsolatedDevicePredicate check mode, and number of scheduled
// device configurations and current resources.
type IsolatedDevicePredicate struct {
	predicates.BasePredicate
}

func (f *IsolatedDevicePredicate) Name() string {
	return "host_isolated_device"
}

func (f *IsolatedDevicePredicate) Clone() core.FitPredicate {
	return &IsolatedDevicePredicate{}
}

func (f *IsolatedDevicePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	data := u.SchedData()
	if len(data.IsolatedDevices) == 0 {
		return false, nil
	}
	return true, nil
}

func (f *IsolatedDevicePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)
	reqIsoDevs := u.SchedData().IsolatedDevices
	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	minCapacity := int64(0xFFFFFFFF)

	// check by specify device id
	for _, dev := range reqIsoDevs {
		if len(dev.ID) == 0 {
			continue
		}
		if fDev := hc.GetIsolatedDevice(dev.ID); fDev != nil {
			if len(fDev.GuestID) != 0 {
				h.Exclude(fmt.Sprintf("IsolatedDevice %q already used by guest %q", dev.ID, fDev.GuestID))
				return h.GetResult()
			}
		} else {
			h.Exclude(fmt.Sprintf("Not found IsolatedDevice %q", dev.ID))
			return h.GetResult()
		}
		minCapacity = 1
	}

	reqCount := len(reqIsoDevs)
	freeCount := len(hc.UnusedIsolatedDevices())
	totalCount := len(hc.IsolatedDevices)

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
		if len(dev.Type) != 0 {
			devTypeRequest[dev.Type] += 1
		}
	}
	for devType, reqCount := range devTypeRequest {
		freeCount := len(hc.UnusedIsolatedDevicesByType(devType))
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
		freeCount := len(hc.UnusedIsolatedDevicesByVendorModel(vendorModel))
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
