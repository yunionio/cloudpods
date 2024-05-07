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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type DiskSchedtagPredicate struct {
	*BaseSchedtagPredicate
	storageUsed map[string]int64
}

func (p *DiskSchedtagPredicate) Name() string {
	return "disk_schedtag"
}

func (p *DiskSchedtagPredicate) Clone() core.FitPredicate {
	return &DiskSchedtagPredicate{
		BaseSchedtagPredicate: NewBaseSchedtagPredicate(),
		storageUsed:           make(map[string]int64),
	}
}

func (p *DiskSchedtagPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	return p.BaseSchedtagPredicate.PreExecute(ctx, p, u, cs)
}

type diskW struct {
	*computeapi.DiskConfig
}

func (d diskW) Keyword() string {
	return "disk"
}

func (d diskW) ResourceKeyword() string {
	return "storage"
}

func (d diskW) IsSpecifyResource() bool {
	return d.Storage != ""
}

func (d diskW) GetSchedtags() []*computeapi.SchedtagConfig {
	return d.DiskConfig.Schedtags
}

func (d diskW) GetDynamicConditionInput() *jsonutils.JSONDict {
	return d.JSON(d)
}

func (p *DiskSchedtagPredicate) GetInputs(u *core.Unit) []ISchedtagCustomer {
	ret := make([]ISchedtagCustomer, 0)
	for _, disk := range u.SchedData().Disks {
		ret = append(ret, &diskW{disk})
	}
	return ret
}

func (p *DiskSchedtagPredicate) GetResources(c core.Candidater) []ISchedtagCandidateResource {
	ret := make([]ISchedtagCandidateResource, 0)
	for _, storage := range c.Getter().Storages() {
		ret = append(ret, storage)
	}
	return ret
}

func (p *DiskSchedtagPredicate) IsResourceMatchInput(ctx context.Context, input ISchedtagCustomer, res ISchedtagCandidateResource) bool {
	return true
}

func (p *DiskSchedtagPredicate) IsResourceFitInput(ctx context.Context, u *core.Unit, c core.Candidater, res ISchedtagCandidateResource, input ISchedtagCustomer) core.PredicateFailureReason {
	storage := res.(*api.CandidateStorage)
	if storage.Status != computeapi.STORAGE_ONLINE || storage.Enabled.IsFalse() {
		return &FailReason{
			fmt.Sprintf("Storage %s status is %s, enable is %v", storage.GetName(), storage.Status, storage.Enabled),
			StorageEnable,
		}
	}

	d := input.(*diskW)
	if d.Storage != "" {
		if storage.Id != d.Storage && storage.Name != d.Storage {
			return &FailReason{
				fmt.Sprintf("Storage name %s != (%s:%s)", d.Storage, storage.Name, storage.Id),
				StorageMatch,
			}
		}
	}
	if c.Getter().ResourceType() == computeapi.HostResourceTypePrepaidRecycle {
		return nil
	}
	if len(d.Backend) != 0 && storage.StorageType != computeapi.STORAGE_BAREMETAL {
		if storage.StorageType != d.Backend {
			return &FailReason{
				fmt.Sprintf("Storage %s backend %s != %s", storage.Name, storage.StorageType, d.Backend),
				StorageType,
			}
		}
	}
	if len(d.Medium) != 0 {
		if !computeapi.IsDiskTypeMatch(storage.MediumType, d.Medium) {
			return &FailReason{
				fmt.Sprintf("Storage %s medium %s != %s", storage.Name, storage.MediumType, d.Medium),
				StorageMedium,
			}
		}
	}
	storageTypes := []string{}
	driver := p.GetHypervisorDriver()
	if driver != nil {
		storageTypes = driver.GetStorageTypes()
	}
	if len(storageTypes) != 0 && !utils.IsInStringArray(storage.StorageType, storageTypes) {
		return &FailReason{
			fmt.Sprintf("Storage %s storage type %s not in %v", storage.Name, storage.StorageType, storageTypes),
			StorageType,
		}
	}

	// domain ownership filter
	if storage.DomainId == u.SchedInfo.Domain {
	} else if storage.IsPublic && storage.PublicScope == string(rbacscope.ScopeSystem) {
	} else if storage.IsPublic && storage.PublicScope == string(rbacscope.ScopeDomain) && utils.IsInStringArray(u.SchedInfo.Domain, storage.GetSharedDomains()) {
	} else {
		return &FailReason{
			Reason: fmt.Sprintf("Storage %s is not accessible due to domain ownership", storage.Name),
			Type:   StorageOwnership,
		}
	}

	if driver != nil && driver.DoScheduleStorageFilter() {
		// free capacity check
		isMigrate := len(u.SchedData().HostId) > 0
		if !isMigrate || !utils.IsInStringArray(storage.StorageType, computeapi.SHARED_STORAGE) {
			if storage.FreeCapacity < int64(d.SizeMb) {
				return &FailReason{
					Reason: fmt.Sprintf("Storage %s free capacity %d < %d(request)", storage.Name, storage.FreeCapacity, d.SizeMb),
					Type:   StorageCapacity,
				}
			}

		}
		// only check ActualFreeCapacity when storage_type is local
		if storage.StorageType == computeapi.STORAGE_LOCAL && storage.ActualFreeCapacity < int64(d.SizeMb) {
			return &FailReason{
				Reason: fmt.Sprintf("Storage %s actual free capacity %d < %d(request)", storage.Name, storage.ActualFreeCapacity, d.SizeMb),
				Type:   StorageCapacity,
			}
		}
	}

	return nil
}

func (p *DiskSchedtagPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return p.BaseSchedtagPredicate.Execute(ctx, p, u, c)
}

func (p *DiskSchedtagPredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	p.BaseSchedtagPredicate.OnPriorityEnd(p, u, c)
}

func (p *DiskSchedtagPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
	p.BaseSchedtagPredicate.OnSelectEnd(p, u, c, count)
}

func (p *DiskSchedtagPredicate) DoSelect(
	c core.Candidater,
	input ISchedtagCustomer,
	res []ISchedtagCandidateResource,
) []ISchedtagCandidateResource {
	return p.GetUsedStorages(res, input.(*diskW).Backend)

}

func (p *DiskSchedtagPredicate) GetCandidateResourceSortScore(selectRes ISchedtagCandidateResource) int64 {
	s := selectRes.(*api.CandidateStorage)
	return s.GetFreeCapacity()
}

func (p *DiskSchedtagPredicate) AddSelectResult(index int, input ISchedtagCustomer, selectRes []ISchedtagCandidateResource, output *core.AllocatedResource) {
	storages := []*schedapi.CandidateStorage{}
	for _, res := range selectRes {
		cs := res.(*api.CandidateStorage)
		storages = append(storages, &schedapi.CandidateStorage{
			Id:           cs.GetId(),
			Name:         cs.GetName(),
			FreeCapacity: cs.GetFreeCapacity(),
		})
	}
	ret := &schedapi.CandidateDiskV2{
		Index:    index,
		Storages: storages,
	}
	output.Disks = append(output.Disks, ret)
}

func (p *DiskSchedtagPredicate) GetUsedStorages(res []ISchedtagCandidateResource, backend string) []ISchedtagCandidateResource {
	storages := make([]ISchedtagCandidateResource, 0)
	for _, s := range res {
		storages = append(storages, s.(*api.CandidateStorage))
	}
	backendStorages := make([]ISchedtagCandidateResource, 0)
	if backend != "" {
		for _, s := range storages {
			if s.(*api.CandidateStorage).StorageType == backend {
				backendStorages = append(backendStorages, s)
			}
		}
	} else {
		backendStorages = storages
	}
	/*if len(backendStorages) == 0 {
		backendStorages = storages
	}*/
	return backendStorages
}
