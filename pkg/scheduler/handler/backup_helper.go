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

package handler

import (
	"fmt"
	"sort"

	"yunion.io/x/log"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func transToBackupSchedResult(
	result *core.SchedResultItemList, preferMainHost, preferBackupHost string, count int64, sid string,
) *schedapi.ScheduleOutput {
	// clean each result sched result item's count
	for _, item := range result.Data {
		item.Count = 0
	}

	apiResults := newBackupSchedResult(result, preferMainHost, preferBackupHost, count, sid)
	return apiResults
}

func newBackupSchedResult(
	result *core.SchedResultItemList,
	preferMainHost, preferBackupHost string,
	count int64,
	sid string,
) *schedapi.ScheduleOutput {
	ret := new(schedapi.ScheduleOutput)
	apiResults := make([]*schedapi.CandidateResource, 0)
	storageUsed := core.NewStorageUsed()
	var wireHostMap map[string]core.SchedResultItems
	for i := 0; i < int(count); i++ {
		log.V(10).Debugf("Select backup host from result: %s", result)
		target, err := getSchedBackupResult(result, preferMainHost, preferBackupHost, sid, wireHostMap, storageUsed)
		if err != nil {
			er := &schedapi.CandidateResource{Error: err.Error()}
			apiResults = append(apiResults, er)
			continue
		}
		apiResults = append(apiResults, target)
	}
	ret.Candidates = apiResults
	return ret
}

func getSchedBackupResult(
	result *core.SchedResultItemList,
	preferMainHost, preferBackupHost string,
	sid string, wireHostMap map[string]core.SchedResultItems,
	storageUsed *core.StorageUsed,
) (*schedapi.CandidateResource, error) {
	if wireHostMap == nil {
		wireHostMap = buildWireHostMap(result)
	} else {
		reviseWireHostMap(wireHostMap)
	}

	mainHost, backupHost := selectHosts(wireHostMap, preferMainHost, preferBackupHost)
	if mainHost == nil {
		return nil, fmt.Errorf("Can't find main host %q", preferMainHost)
	}
	if backupHost == nil {
		return nil, fmt.Errorf("Can't find backup host %q by main %q", preferBackupHost, mainHost.ID)
	}

	markHostUsed(mainHost)
	markHostUsed(backupHost)

	ret := mainHost.ToCandidateResource(storageUsed)
	ret.BackupCandidate = backupHost.ToCandidateResource(storageUsed)
	ret.SessionId = sid
	ret.BackupCandidate.SessionId = sid
	return ret, nil
}

func buildWireHostMap(result *core.SchedResultItemList) map[string]core.SchedResultItems {
	sort.Sort(sort.Reverse(result.Data))
	wireHostMap := make(map[string]core.SchedResultItems)
	for i := 0; i < len(result.Data); i++ {
		networks := result.Data[i].Candidater.Getter().Networks()
		for j := 0; j < len(networks); j++ {
			if hosts, ok := wireHostMap[networks[j].WireId]; ok {
				if hostInResultItemsIndex(result.Data[i].ID, hosts) < 0 {
					wireHostMap[networks[j].WireId] = append(hosts, result.Data[i])
				}
			} else {
				wireHostMap[networks[j].WireId] = core.SchedResultItems{result.Data[i]}
			}
		}
	}
	return wireHostMap
}

func reviseWireHostMap(wireHostMap map[string]core.SchedResultItems) {
	for _, hosts := range wireHostMap {
		sort.Sort(sort.Reverse(hosts))
	}
}

func markHostUsed(host *core.SchedResultItem) {
	host.Count++
	host.Capacity--
}

func hostInResultItemsIndex(hostId string, hosts core.SchedResultItems) int {
	for i := 0; i < len(hosts); i++ {
		if hosts[i].ID == hostId {
			return i
		}
	}
	return -1
}

func selectHosts(
	wireHostMap map[string]core.SchedResultItems, preferMainHost, preferBackupHost string,
) (*core.SchedResultItem, *core.SchedResultItem) {
	var scroe int64
	var mainIdx, backupIdx int
	var selectedWireId string
	for wireId, hosts := range wireHostMap {
		mainIdx, backupIdx = -1, -1
		if len(hosts) < 2 {
			continue
		}
		if len(preferMainHost) > 0 {
			if mainIdx = hostInResultItemsIndex(preferMainHost, hosts); mainIdx < 0 {
				continue
			}
		}
		if len(preferBackupHost) > 0 {
			if backupIdx = hostInResultItemsIndex(preferBackupHost, hosts); backupIdx < 0 {
				continue
			}
		}

		// select main host index
		if mainIdx < 0 {
			for i := 0; i < len(hosts); i++ {
				if hosts[i].ID != preferBackupHost {
					mainIdx = i
				}
			}
		}
		if hosts[mainIdx].Capacity <= 0 {
			if len(preferMainHost) > 0 {
				// in case prefer main host capacity isn't enough
				break
			} else {
				continue
			}
		}

		// select backup host index
		if backupIdx < 0 {
			for i := 0; i < len(hosts); i++ {
				if i != mainIdx {
					backupIdx = i
				}
			}
		}
		if hosts[backupIdx].Capacity <= 0 {
			if len(preferBackupHost) > 0 {
				// in case perfer backup host capacity isn't enough
				break
			} else {
				continue
			}
		}

		// the highest total score wins
		curScore := hosts[mainIdx].Capacity + hosts[backupIdx].Capacity
		if curScore > scroe {
			selectedWireId = wireId
			scroe = curScore
		}
	}
	if len(selectedWireId) == 0 {
		return nil, nil
	}
	return wireHostMap[selectedWireId][mainIdx], wireHostMap[selectedWireId][backupIdx]
}
