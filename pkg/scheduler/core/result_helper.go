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
	"fmt"

	"yunion.io/x/log"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

func ResultHelp(result *SchedResultItemList, schedInfo *api.SchedInfo) *ScheduleResult {
	out := new(ScheduleResult)
	out.Result = transToSchedResult(result, schedInfo)
	return out
}

func ResultHelpForForcast(result *SchedResultItemList, _ *api.SchedInfo) *ScheduleResult {
	out := new(ScheduleResult)
	out.ForecastResult = transToSchedForecastResult(result)
	return out
}

func ResultHelpForTest(result *SchedResultItemList, schedInfo *api.SchedInfo) *ScheduleResult {
	out := new(ScheduleResult)
	out.TestResult = transToSchedTestResult(result, schedInfo.SuggestionLimit)
	return out
}

type IResultHelper interface {
	ResultHelp(result *SchedResultItemList, schedInfo *api.SchedInfo) *ScheduleResult
}

// ResultHelperFunc type is an adapter to allwo the use of ordinary functions as a ResultHelper.
// If f is a function with the appropriate signature, ResultHelperFunc(f) is a ResultHelper that calls f.
type SResultHelperFunc func(result *SchedResultItemList, schedInfo *api.SchedInfo) *ScheduleResult

func (r SResultHelperFunc) ResultHelp(result *SchedResultItemList, schedInfo *api.SchedInfo) *ScheduleResult {
	return r(result, schedInfo)
}

func transToSchedResult(result *SchedResultItemList, schedInfo *api.SchedInfo) *schedapi.ScheduleOutput {
	if schedInfo.Backup || len(schedInfo.InstanceGroupsDetail) > 0 {
		return transToInstanceGroupSchedResult(result, schedInfo)
	} else {
		return transToRegionSchedResult(result.Data, int64(schedInfo.Count), schedInfo.SessionId)
	}
}

func transToRegionSchedResult(result SchedResultItems, count int64, sid string) *schedapi.ScheduleOutput {
	apiResults := make([]*schedapi.CandidateResource, 0)
	succCount := 0
	storageUsed := NewStorageUsed()
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

func hostInResultItemsIndex(hostId string, hosts SchedResultItems) int {
	for i := 0; i < len(hosts); i++ {
		if hosts[i].ID == hostId {
			return i
		}
	}
	return -1
}

func transToSchedTestResult(result *SchedResultItemList, limit int64) interface{} {
	return &api.SchedTestResult{
		Data:   result.Data,
		Total:  int64(result.Data.Len()),
		Limit:  limit,
		Offset: 0,
	}
}

func transToSchedForecastResult(result *SchedResultItemList) interface{} {
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

	logIndex := func(item *SchedResultItem) string {
		getter := item.Candidater.Getter()
		name := getter.Name()
		id := getter.Id()
		return fmt.Sprintf("%s:%s", name, id)
	}
	addInfos := func(logs SchedLogList, item *SchedResultItem) {
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

	items := make(SchedResultItems, 0)
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
