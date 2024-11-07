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

package core

import (
	"encoding/json"
	"sort"

	"yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

type ScheduleResult struct {
	// Result is sync schedule result
	Result *schedapi.ScheduleOutput
	// ForecastResult is forecast schedule result
	ForecastResult *api.SchedForecastResult
	// TestResult is test schedule result
	TestResult interface{}
}

type SchedResultItem struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Count    int64                  `json:"count"`
	Data     map[string]interface{} `json:"data"`
	Capacity int64                  `json:"capacity"`
	Score    Score                  `json:"score"`

	CapacityDetails map[string]int64 `json:"capacity_details"`
	ScoreDetails    string           `json:"score_details"`

	Candidater Candidater `json:"-"`

	*AllocatedResource

	SchedData *api.SchedInfo
}

type SchedResultItemList struct {
	Unit *Unit
	Data SchedResultItems
}

func (its SchedResultItemList) String() string {
	bytes, _ := json.Marshal(its.Data)
	return string(bytes)
}

type SchedResultItems []*SchedResultItem

func (its SchedResultItems) Len() int {
	return len(its)
}

func (its SchedResultItems) Swap(i, j int) {
	its[i], its[j] = its[j], its[i]
}

func (its SchedResultItems) Less(i, j int) bool {
	it1, it2 := its[i], its[j]
	return it1.Capacity < it2.Capacity
}

func (item *SchedResultItem) ToCandidateResource(storageUsed *StorageUsed) *schedapi.CandidateResource {
	return &schedapi.CandidateResource{
		HostId:     item.ID,
		CpuNumaPin: item.selectCpuNumaPin(),
		Name:       item.Name,
		Disks:      item.getDisks(storageUsed),
		Nets:       item.Nets,
	}
}

func (item *SchedResultItem) selectCpuNumaPin() []schedapi.SCpuNumaPin {
	vcpuCount := item.SchedData.Ncpu
	if item.SchedData.ExtraCpuCount > 0 {
		vcpuCount += item.SchedData.ExtraCpuCount
	}

	var res []schedapi.SCpuNumaPin
	if item.SchedData.LiveMigrate && len(item.SchedData.CpuNumaPin) > 0 {
		res = item.Candidater.AllocCpuNumaPinWithNodeCount(vcpuCount, item.SchedData.Memory*1024, len(item.SchedData.CpuNumaPin))
	} else {
		res = item.Candidater.AllocCpuNumaPin(vcpuCount, item.SchedData.Memory*1024, item.SchedData.PreferNumaNodes)
	}

	if item.SchedData.ExtraCpuCount > 0 {
		extraCpuCnt := item.SchedData.ExtraCpuCount
		for extraCpuCnt > 0 {
			cpuMaxIdx := 0
			cpuMax := -1
			for i := range res {
				if len(res[i].CpuPin)-res[i].ExtraCpuCount > cpuMax {
					cpuMax = len(res[i].CpuPin) - res[i].ExtraCpuCount
					cpuMaxIdx = i
				}
			}
			res[cpuMaxIdx].ExtraCpuCount += 1
			extraCpuCnt -= 1
		}
	}
	return res
}

func (item *SchedResultItem) getDisks(used *StorageUsed) []*schedapi.CandidateDisk {
	inputs := item.SchedData.Disks
	ret := make([]*schedapi.CandidateDisk, 0)
	for idx, disk := range item.Disks {
		ret = append(ret, &schedapi.CandidateDisk{
			Index:      idx,
			StorageIds: item.getSortStorageIds(used, inputs[idx], disk.Storages),
		})
	}
	return ret
}

func (item *SchedResultItem) getSortStorageIds(
	used *StorageUsed,
	disk *compute.DiskConfig,
	storages []*schedapi.CandidateStorage) []string {
	reqSize := disk.SizeMb
	ss := make([]sortStorage, 0)
	for _, s := range storages {
		ss = append(ss, sortStorage{
			Id:      s.Id,
			FeeSize: s.FreeCapacity - used.Get(s.Id),
		})
	}
	toSort := sortStorages(ss)
	sort.Sort(toSort)
	sortedStorages := toSort.getIds()
	ret := make([]string, 0)
	for idx, id := range sortedStorages {
		if idx == 0 {
			used.Add(id, int64(reqSize))
		}
		ret = append(ret, id)
	}
	return ret
}
