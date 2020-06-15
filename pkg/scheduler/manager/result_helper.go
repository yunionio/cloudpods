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

package manager

import (
	"fmt"
	"sort"

	"yunion.io/x/log"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
)

func transToSchedResult(result *core.SchedResultItemList, schedInfo *api.SchedInfo) *schedapi.ScheduleOutput {
	if schedInfo.Backup {
		return transToBackupSchedResult(result,
			schedInfo.PreferHost, schedInfo.PreferBackupHost, int64(schedInfo.Count), schedInfo.SessionId)
	} else {
		return transToRegionSchedResult(result.Data, int64(schedInfo.Count), schedInfo.SessionId)
	}
}

func setSchedPendingUsage(driver computemodels.IGuestDriver, req *api.SchedInfo, resp *schedapi.ScheduleOutput) error {
	if req.IsSuggestion || IsDriverSkipScheduleDirtyMark(driver) || req.SkipDirtyMarkHost() {
		return nil
	}
	for _, item := range resp.Candidates {
		schedmodels.HostPendingUsageManager.AddPendingUsage(req, item)
	}
	return nil
}

func IsDriverSkipScheduleDirtyMark(driver computemodels.IGuestDriver) bool {
	return !(driver.DoScheduleCPUFilter() && driver.DoScheduleMemoryFilter() && driver.DoScheduleStorageFilter())
}

func transToRegionSchedResult(result core.SchedResultItems, count int64, sid string) *schedapi.ScheduleOutput {
	apiResults := make([]*schedapi.CandidateResource, 0)
	succCount := 0
	storageUsed := core.NewStorageUsed()
	for _, nr := range result {
		for {
			if nr.Count <= 0 {
				break
			}
			tr := nr.ToCandidateResource(storageUsed)
			tr.SessionId = sid
			apiResults = append(apiResults, tr)
			nr.Count--
			succCount++
		}
	}

	for {
		if int64(succCount) >= count {
			break
		}
		er := &schedapi.CandidateResource{Error: "Out of resource"}
		apiResults = append(apiResults, er)
		succCount++
	}

	return &schedapi.ScheduleOutput{
		Candidates: apiResults,
	}
}

func transToBackupSchedResult(
	result *core.SchedResultItemList, preferMasterHost, preferBackupHost string, count int64, sid string,
) *schedapi.ScheduleOutput {
	// clean each result sched result item's count
	for _, item := range result.Data {
		item.Count = 0
	}

	apiResults := newBackupSchedResult(result, preferMasterHost, preferBackupHost, count, sid)
	return apiResults
}

func newBackupSchedResult(
	result *core.SchedResultItemList,
	preferMasterHost, preferBackupHost string,
	count int64,
	sid string,
) *schedapi.ScheduleOutput {
	ret := new(schedapi.ScheduleOutput)
	apiResults := make([]*schedapi.CandidateResource, 0)
	storageUsed := core.NewStorageUsed()
	var wireHostMap map[string]core.SchedResultItems
	for i := 0; i < int(count); i++ {
		log.V(10).Debugf("Select backup host from result: %s", result)
		target, err := getSchedBackupResult(result, preferMasterHost, preferBackupHost, sid, wireHostMap, storageUsed)
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
	preferMasterHost, preferBackupHost string,
	sid string, wireHostMap map[string]core.SchedResultItems,
	storageUsed *core.StorageUsed,
) (*schedapi.CandidateResource, error) {
	if wireHostMap == nil {
		wireHostMap = buildWireHostMap(result)
	} else {
		reviseWireHostMap(wireHostMap)
	}

	masterHost, backupHost := selectHosts(wireHostMap, preferMasterHost, preferBackupHost)
	if masterHost == nil {
		return nil, fmt.Errorf("Can't find master host %q", preferMasterHost)
	}
	if backupHost == nil {
		return nil, fmt.Errorf("Can't find backup host %q by master %q", preferBackupHost, masterHost.ID)
	}

	markHostUsed(masterHost)
	markHostUsed(backupHost)

	ret := masterHost.ToCandidateResource(storageUsed)
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
	wireHostMap map[string]core.SchedResultItems, preferMasterHost, preferBackupHost string,
) (*core.SchedResultItem, *core.SchedResultItem) {
	var scroe int64
	var masterIdx, backupIdx int
	var selectedWireId string
	for wireId, hosts := range wireHostMap {
		masterIdx, backupIdx = -1, -1
		if len(hosts) < 2 {
			continue
		}
		if len(preferMasterHost) > 0 {
			if masterIdx = hostInResultItemsIndex(preferMasterHost, hosts); masterIdx < 0 {
				continue
			}
		}
		if len(preferBackupHost) > 0 {
			if backupIdx = hostInResultItemsIndex(preferBackupHost, hosts); backupIdx < 0 {
				continue
			}
		}

		// select master host index
		if masterIdx < 0 {
			for i := 0; i < len(hosts); i++ {
				if hosts[i].ID != preferBackupHost {
					masterIdx = i
				}
			}
		}
		if hosts[masterIdx].Capacity <= 0 {
			if len(preferMasterHost) > 0 {
				// in case prefer master host capacity isn't enough
				break
			} else {
				continue
			}
		}

		// select backup host index
		if backupIdx < 0 {
			for i := 0; i < len(hosts); i++ {
				if i != masterIdx {
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
		curScore := hosts[masterIdx].Capacity + hosts[backupIdx].Capacity
		if curScore > scroe {
			selectedWireId = wireId
			scroe = curScore
		}
	}
	if len(selectedWireId) == 0 {
		return nil, nil
	}
	return wireHostMap[selectedWireId][masterIdx], wireHostMap[selectedWireId][backupIdx]
}

func transToSchedTestResult(result *core.SchedResultItemList, limit int64) interface{} {
	return &api.SchedTestResult{
		Data:   result.Data,
		Total:  int64(result.Data.Len()),
		Limit:  limit,
		Offset: 0,
	}
}

func transToSchedForecastResult(result *core.SchedResultItemList) interface{} {
	unit := result.Unit
	schedData := unit.SchedData()
	reqCount := int64(schedData.Count)
	filters := make([]*api.ForecastFilter, 0)

	filtersMap := make(map[string]*api.ForecastFilter)
	getOrNewFilter := func(preName string) (*api.ForecastFilter, bool) {
		if info, ok := filtersMap[preName]; !ok {
			i := &api.ForecastFilter{
				Filter:   preName,
				Count:    0,
				Messages: make([]string, 0),
			}
			filtersMap[preName] = i
			return i, false
		} else {
			return info, true
		}
	}

	logIndex := func(item *core.SchedResultItem) string {
		getter := item.Candidater.Getter()
		name := getter.Name()
		id := getter.Id()
		return fmt.Sprintf("%s:%s", name, id)
	}
	addInfos := func(logs core.SchedLogList, item *core.SchedResultItem) {
		for preName, cnt := range item.CapacityDetails {
			if cnt > 0 {
				continue
			}
			failedLog := logs.Get(logIndex(item))
			if failedLog == nil {
				log.Errorf("predicate %q count is 0, but not found failed log", preName)
				continue
			}
			for _, msg := range failedLog.Messages {
				info, exist := getOrNewFilter(msg.Type)
				info.Count++
				info.Messages = append(info.Messages, msg.Info)
				if !exist {
					filters = append(filters, info)
				}
			}
		}
	}

	items := make(core.SchedResultItems, 0)
	for _, item := range result.Data {
		hostType := item.Candidater.Getter().HostType()
		if schedData.Hypervisor == hostType {
			items = append(items, item)
		}
	}

	for _, item := range items {
		addInfos(result.Unit.LogManager.FailedLogs(), item)
	}

	var (
		output     = transToSchedResult(result, schedData)
		readyCount int64
	)

	for _, candi := range output.Candidates {
		if len(candi.Error) != 0 {
			info, exist := getOrNewFilter("select_candidate")
			msg := candi.Error
			info.Messages = append(info.Messages, msg)
			if !exist {
				filters = append(filters, info)
			}
		} else {
			readyCount++
		}
	}

	canCreate := true
	if readyCount < reqCount {
		canCreate = false
		filters = append(filters, &api.ForecastFilter{
			Messages: []string{
				fmt.Sprintf("No enough resources: %d/%d(free/request)", readyCount, reqCount),
			},
		})
	}
	return &api.SchedForecastResult{
		CanCreate: canCreate,
		Filters:   filters,
		Results:   output.Candidates,
	}
}
