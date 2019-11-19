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
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type DiskSchedtagPredicate struct {
	*BaseSchedtagPredicate
}

func (p *DiskSchedtagPredicate) Name() string {
	return "disk_schedtag"
}

func (p *DiskSchedtagPredicate) Clone() core.FitPredicate {
	return &DiskSchedtagPredicate{
		BaseSchedtagPredicate: NewBaseSchedtagPredicate(),
	}
}

func (p *DiskSchedtagPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	return p.BaseSchedtagPredicate.PreExecute(p, u, cs)
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

func (p *DiskSchedtagPredicate) IsResourceMatchInput(input ISchedtagCustomer, res ISchedtagCandidateResource) bool {
	return true
}

func (p *DiskSchedtagPredicate) IsResourceFitInput(u *core.Unit, c core.Candidater, res ISchedtagCandidateResource, input ISchedtagCustomer) core.PredicateFailureReason {
	storage := res.(*api.CandidateStorage)
	if storage.Status == computeapi.STORAGE_OFFLINE || !storage.Enabled.Bool() {
		return &FailReason{
			fmt.Sprintf("Storage status is %s, enable is %v", storage.Status, storage.Enabled),
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

	storageTypes := p.GetHypervisorDriver().GetStorageTypes()
	if len(storageTypes) != 0 && !utils.IsInStringArray(storage.StorageType, storageTypes) {
		return &FailReason{
			fmt.Sprintf("Storage %s storage type %s not in %v", storage.Name, storage.StorageType, storageTypes),
			StorageType,
		}
	}
	return nil
}

func (p *DiskSchedtagPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return p.BaseSchedtagPredicate.Execute(p, u, c)
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
	return selectRes.(*api.CandidateStorage).GetFreeCapacity()
}

func (p *DiskSchedtagPredicate) AddSelectResult(index int, selectRes []ISchedtagCandidateResource, output *core.AllocatedResource) {
	storageIds := []string{}
	for _, res := range selectRes {
		storageIds = append(storageIds, res.GetId())
	}
	ret := &schedapi.CandidateDisk{
		Index:      index,
		StorageIds: storageIds,
	}
	log.Debugf("Suggestion storages %v for disk%d", storageIds, index)
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
	if len(backendStorages) == 0 {
		backendStorages = storages
	}
	return backendStorages
}
