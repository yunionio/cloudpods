package handler

import (
	"fmt"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

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
			if cnt <= 0 {
				info, exist := getOrNewFilter(preName)
				info.Count++
				var msg string
				if failedLog := logs.Get(logIndex(item)); failedLog != nil {
					msg = failedLog.String()
				}
				info.Messages = append(info.Messages, msg)
				if !exist {
					filters = append(filters, info)
				}
			}
		}
	}

	items := make([]*core.SchedResultItem, 0)
	for _, item := range result.Data {
		hostType := item.Candidater.Getter().HostType()
		if schedData.Hypervisor == hostType {
			items = append(items, item)
		}
	}

	for _, item := range items {
		addInfos(result.Unit.LogManager.FailedLogs(), item)
	}

	var output *schedapi.ScheduleOutput
	sid := schedData.SessionId
	if schedData.Backup {
		output = transToBackupSchedResult(result, schedData.PreferHost, schedData.PreferBackupHost, int64(schedData.Count), sid)
	} else {
		output = transToRegionSchedResult(result.Data, int64(schedData.Count), sid)
	}

	var readyCount int64
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
	}
	return &api.SchedForecastResult{
		CanCreate: canCreate,
		Filters:   filters,
		Results:   output.Candidates,
	}
}
