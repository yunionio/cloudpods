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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
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
	driver := u.GetHypervisorDriver()
	if driver != nil && !driver.DoScheduleStorageFilter() {
		return false, nil
	}
	if u.SchedData().ResetCpuNumaPin {
		return false, nil
	}

	return true, nil
}

type diskSizeRequest struct {
	backend string
	max     int64
	total   int64
	disks   []*compute.DiskConfig
}

func newDiskSizeRequest(backend string) *diskSizeRequest {
	return &diskSizeRequest{
		backend: backend,
		max:     -1,
		total:   0,
		disks:   make([]*compute.DiskConfig, 0),
	}
}

func (req *diskSizeRequest) GetMax() int64 {
	return req.max
}

func (req *diskSizeRequest) GetTotal() int64 {
	return req.total
}

func (req *diskSizeRequest) Add(disk *compute.DiskConfig) *diskSizeRequest {
	req.total += int64(disk.SizeMb)
	if req.max < int64(disk.SizeMb) {
		req.max = int64(disk.SizeMb)
	}

	req.disks = append(req.disks, disk)
	return req
}

func (req *diskSizeRequest) NewByMediumType(mt string) *diskSizeRequest {
	newReq := newDiskSizeRequest(req.backend)
	for _, d := range req.disks {
		if d.Medium == mt {
			newReq.Add(d)
		}
	}
	return newReq
}

type diskBackendSizeRequest struct {
	reqs map[string]*diskSizeRequest
	// beMdm is a map that use backend as key and medium types as value
	beMdm map[string]sets.String
}

func newDiskBackendSizeRequest() *diskBackendSizeRequest {
	return &diskBackendSizeRequest{
		reqs:  make(map[string]*diskSizeRequest),
		beMdm: make(map[string]sets.String),
	}
}

func (ds *diskBackendSizeRequest) get(backend string) (*diskSizeRequest, bool) {
	req, ok := ds.reqs[backend]
	return req, ok
}

func (ds *diskBackendSizeRequest) set(backend string, req *diskSizeRequest) *diskBackendSizeRequest {
	ds.reqs[backend] = req
	return ds
}

func (ds *diskBackendSizeRequest) Add(disk *compute.DiskConfig) *diskBackendSizeRequest {
	backend := disk.Backend
	req, ok := ds.get(backend)
	if !ok {
		req = newDiskSizeRequest(backend)
	}
	req.Add(disk)
	ds.set(backend, req)

	mds, ok := ds.beMdm[backend]
	if !ok {
		mds = sets.NewString()
	}
	mds.Insert(disk.Medium)
	ds.beMdm[backend] = mds

	return ds
}

func (ds *diskBackendSizeRequest) GetBackendMediumMap() map[string]sets.String {
	return ds.beMdm
}

func (ds *diskBackendSizeRequest) Get(backend string, mediumType string) (*diskSizeRequest, error) {
	req, ok := ds.get(backend)
	if !ok {
		return nil, errors.Errorf("Not found diskBackendSizeRequest by backend %q", backend)
	}
	return req.NewByMediumType(mediumType), nil
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

	getStorageCapacity := func(backend string, mediumType string, reqMaxSize int64, reqTotalSize int64, useRsvd bool) (*storageCapacity, *storageCapacity, error) {
		totalFree, actualFree, err := getter.GetFreeStorageSizeOfType(backend, mediumType, useRsvd, reqMaxSize)
		if err != nil {
			return nil, nil, err
		}
		reqTotalSize = utils.Max(reqTotalSize, 1)
		capacity := totalFree / reqTotalSize
		actualCapacity := actualFree / reqTotalSize
		return newStorageCapacity(capacity, totalFree, false), newStorageCapacity(actualCapacity, actualFree, true), nil
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

	getStorageFreeStr := func(backend string, mediumType string, useRsvd bool, isActual bool) string {
		ss := []string{}
		for _, s := range getter.Storages() {
			if candidate.IsStorageBackendMediumMatch(s, backend, mediumType) {
				if isActual {
					total := s.Capacity
					free := total - s.ActualCapacityUsed
					ss = append(ss, fmt.Sprintf("storage %q, actual_total:%d - actual_used:%d = free:%d", s.GetName(), total, s.ActualCapacityUsed, free))
				} else {
					total := int64(float32(s.Capacity) * s.Cmtbound)
					used := s.GetUsedCapacity(tristate.True)
					waste := s.GetUsedCapacity(tristate.False)
					free := total - int64(used) - int64(waste)
					ss = append(ss, fmt.Sprintf("storage %q, total:%d - used:%d - waste:%d = free:%d", s.GetName(), total, used, waste, free))
				}
			}
		}
		return strings.Join(ss, " + ")
	}

	sizeRequest := newDiskBackendSizeRequest()
	storeRequest := make(map[string]int64, 0)
	for _, disk := range d.Disks {
		if isMigrate() && !isLocalhostBackend(disk.Backend) {
			storeRequest[disk.Storage] = 1
		} else if len(disk.DiskId) > 0 && len(disk.Storage) > 0 {
			// server attach to an existing disk
			storeRequest[disk.Storage] = 1
		} else if !isMigrate() || (isMigrate() && isLocalhostBackend(disk.Backend)) {
			// if migrate, only local storage need check capacity constraint
			sizeRequest.Add(disk)
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

	appendFailMsg := func(backend string, mediumType string, req *diskSizeRequest, useRsvd bool, capacity *storageCapacity) {
		reqStr := fmt.Sprintf("no enough backend %q, mediumType %q storage, req=%v(%v)", backend, mediumType, req.GetTotal(), getReqSizeStr(backend))
		freePrex := "free"
		isActual := capacity.isActual
		if isActual {
			freePrex = "actual_free"
		}
		freeStr := fmt.Sprintf("%s=%v(%v)", freePrex, capacity.free, getStorageFreeStr(backend, mediumType, useRsvd, isActual))
		msg := reqStr + ", " + freeStr
		h.AppendPredicateFailMsg(msg)
	}

	for be, mds := range sizeRequest.GetBackendMediumMap() {
		for _, medium := range mds.List() {
			req, err := sizeRequest.Get(be, medium)
			if err != nil {
				h.Exclude(fmt.Sprintf("get request size by backend %q, medium %q: %v", be, medium, err))
				return h.GetResult()
			}
			capacity, actualCapacity, err := getStorageCapacity(be, medium, req.GetMax(), req.GetTotal(), useRsvd)
			if err != nil {
				h.Exclude(err.Error())
				return h.GetResult()
			}
			tmpCap := utils.Min(capacity.capacity, actualCapacity.capacity)
			if capacity.capacity <= 0 {
				appendFailMsg(be, medium, req, useRsvd, capacity)
			} else if actualCapacity.capacity <= 0 {
				appendFailMsg(be, medium, req, useRsvd, actualCapacity)
			}
			minCapacity = utils.Min(minCapacity, tmpCap)
		}
	}

	h.SetCapacity(minCapacity)

	return h.GetResult()
}
