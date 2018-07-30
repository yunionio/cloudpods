package guest

import (
	"fmt"
	"strings"

	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/predicates"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
	"github.com/yunionio/pkg/utils"
)

// StoragePredicate used to filter whether the storage capacity of the
// current candidate matches the type of the disk. If not matched, the
// storage capacity will be set to 0.
type StoragePredicate struct {
	predicates.BasePredicate
}

func (p *StoragePredicate) Name() string {
	return "host_storage"
}

func (p *StoragePredicate) Clone() core.FitPredicate {
	return &StoragePredicate{}
}

func (p *StoragePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.IsPublicCloudProvider() {
		return false, nil
	}
	return true, nil
}

func (p *StoragePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	d := u.SchedData()

	isMigrate := func() bool {
		return len(d.HostID) > 0
	}

	isLocalhostBackend := func(backend string) bool {
		return utils.IsLocalStorage(backend)
	}

	isStorageAccessible := func(storage string) bool {
		for _, s := range hc.Storages {
			if storage == s.ID || storage == s.Name {
				return true
			}
		}

		return false
	}

	getStorageCapacity := func(backend string, reqMaxSize int64, reqTotalSize int64, useRsvd bool) (int64, int64) {
		totalFree := hc.GetFreeStorageSizeOfType(backend, useRsvd)
		capacity := totalFree / utils.Max(reqTotalSize, 1)

		return capacity, totalFree
	}

	getReqSizeStr := func(backend string) string {
		ss := make([]string, 0, len(d.Disks))
		for _, disk := range d.Disks {
			if disk.Backend == backend {
				ss = append(ss, fmt.Sprintf("%v", disk.Size))
			}
		}

		return strings.Join(ss, "+")
	}

	getStorageFreeStr := func(backend string, useRsvd bool) string {
		ss := []string{}
		for _, s := range hc.Storages {
			if s.StorageType == backend {
				total := int64(float64(s.Capacity) * s.Cmtbound)
				free := total - s.UsedCapacity - s.WasteCapacity
				ss = append(ss, fmt.Sprintf("(%v-%v-%v=%v)", total, s.UsedCapacity, s.WasteCapacity, free))
			}
		}
		return strings.Join(ss, " + ")
	}

	sizeRequest := make(map[string]map[string]int64, 0)
	storeRequest := make(map[string]int64, 0)
	for _, disk := range d.Disks {
		if isMigrate() && !isLocalhostBackend(disk.Backend) {
			storeRequest[*disk.Storage] = 1
		} else {
			if _, ok := sizeRequest[disk.Backend]; !ok {
				sizeRequest[disk.Backend] = map[string]int64{"max": -1, "total": 0}
			}
			max := sizeRequest[disk.Backend]["max"]
			if max < disk.Size {
				sizeRequest[disk.Backend]["max"] = disk.Size
			}
			sizeRequest[disk.Backend]["total"] += disk.Size
		}
	}

	for store := range storeRequest {
		if !isStorageAccessible(store) {
			h.Exclude(fmt.Sprintf("storage %v not accessible", store))
			return h.GetResult()
		}
	}

	useRsvd := h.UseReserved()
	minCapacity := int64(0xFFFFFFFF)
	for be, req := range sizeRequest {
		capacity, totalFree := getStorageCapacity(be, req["max"], req["total"], useRsvd)
		if capacity == 0 {
			s := fmt.Sprintf("no enough %q storage, req=%v(%v), free=%v(%v)",
				be, req["total"], getReqSizeStr(be), totalFree, getStorageFreeStr(be, useRsvd))
			h.AppendPredicateFailMsg(s)
		}
		minCapacity = utils.Min(minCapacity, capacity)
	}

	h.SetCapacity(minCapacity)
	if minCapacity <= 0 {
		h.AppendPredicateFailMsg(predicates.ErrNoEnoughStorage)
	}

	return h.GetResult()
}
