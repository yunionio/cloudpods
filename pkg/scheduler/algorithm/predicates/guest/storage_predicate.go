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

package guest

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
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

func (p *StoragePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	if !u.GetHypervisorDriver().DoScheduleStorageFilter() {
		return false, nil
	}
	return true, nil
}

func (p *StoragePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	d := u.SchedData()
	getter := c.Getter()
	storages := getter.Storages()

	isMigrate := func() bool {
		return len(d.HostId) > 0
	}

	isLocalhostBackend := func(backend string) bool {
		return utils.IsLocalStorage(backend)
	}

	isStorageAccessible := func(storage string) bool {
		for _, s := range storages {
			if storage == s.Id || storage == s.Name {
				return true
			}
		}

		return false
	}

	type storageCapacity struct {
		capacity int64
		free     int64
		isActual bool
	}

	newStorageCapacity := func(capacity int64, free int64, isActual bool) *storageCapacity {
		return &storageCapacity{
			capacity: capacity,
			free:     free,
			isActual: isActual,
		}
	}

	getStorageCapacity := func(backend string, reqMaxSize int64, reqTotalSize int64, useRsvd bool) (*storageCapacity, *storageCapacity) {
		totalFree, actualFree := getter.GetFreeStorageSizeOfType(backend, useRsvd)
		reqTotalSize = utils.Max(reqTotalSize, 1)
		capacity := totalFree / reqTotalSize
		actualCapacity := actualFree / reqTotalSize
		return newStorageCapacity(capacity, totalFree, false), newStorageCapacity(actualCapacity, actualFree, true)
	}

	getReqSizeStr := func(backend string) string {
		ss := make([]string, 0, len(d.Disks))
		for _, disk := range d.Disks {
			if disk.Backend == backend {
				ss = append(ss, fmt.Sprintf("%v", disk.SizeMb))
			}
		}

		return strings.Join(ss, "+")
	}

	getStorageFreeStr := func(backend string, useRsvd bool, isActual bool) string {
		ss := []string{}
		for _, s := range getter.Storages() {
			if s.StorageType == backend {
				if isActual {
					total := s.Capacity
					free := total - s.ActualCapacityUsed
					ss = append(ss, fmt.Sprintf("actual_total:%d - actual_used:%d = free:%d", total, s.ActualCapacityUsed, free))
				} else {
					total := int64(float32(s.Capacity) * s.Cmtbound)
					used := s.GetUsedCapacity(tristate.True)
					waste := s.GetUsedCapacity(tristate.False)
					free := total - int64(used) - int64(waste)
					ss = append(ss, fmt.Sprintf("total:%d - used:%d - waste:%d = free:%d", total, used, waste, free))
				}
			}
		}
		return strings.Join(ss, " + ")
	}

	sizeRequest := make(map[string]map[string]int64, 0)
	storeRequest := make(map[string]int64, 0)
	for _, disk := range d.Disks {
		if isMigrate() && !isLocalhostBackend(disk.Backend) {
			storeRequest[disk.Storage] = 1
		} else if len(disk.DiskId) > 0 && len(disk.Storage) > 0 {
			// server attach to an existing disk
			storeRequest[disk.Storage] = 1
		} else if !isMigrate() || (isMigrate() && isLocalhostBackend(disk.Backend)) {
			// if migrate, only local storage need check capacity constraint
			if _, ok := sizeRequest[disk.Backend]; !ok {
				sizeRequest[disk.Backend] = map[string]int64{"max": -1, "total": 0}
			}
			max := sizeRequest[disk.Backend]["max"]
			if max < int64(disk.SizeMb) {
				sizeRequest[disk.Backend]["max"] = int64(disk.SizeMb)
			}
			sizeRequest[disk.Backend]["total"] += int64(disk.SizeMb)
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

	appendFailMsg := func(backend string, req map[string]int64, useRsvd bool, capacity *storageCapacity) {
		reqStr := fmt.Sprintf("no enough %q storage, req=%v(%v)", backend, req["total"], getReqSizeStr(backend))
		freePrex := "free"
		isActual := capacity.isActual
		if isActual {
			freePrex = "actual_free"
		}
		freeStr := fmt.Sprintf("%s=%v(%v)", freePrex, capacity.free, getStorageFreeStr(backend, useRsvd, isActual))
		msg := reqStr + ", " + freeStr
		h.AppendPredicateFailMsg(msg)
	}

	for be, req := range sizeRequest {
		capacity, actualCapacity := getStorageCapacity(be, req["max"], req["total"], useRsvd)
		tmpCap := utils.Min(capacity.capacity, actualCapacity.capacity)
		if capacity.capacity <= 0 {
			appendFailMsg(be, req, useRsvd, capacity)
		} else if actualCapacity.capacity <= 0 {
			appendFailMsg(be, req, useRsvd, actualCapacity)
		}
		minCapacity = utils.Min(minCapacity, tmpCap)
	}

	h.SetCapacity(minCapacity)

	return h.GetResult()
}
