package handler

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func transToSchedForecastResult(result *core.SchedResultItemList) interface{} {
	unit := result.Unit
	reqCount := unit.SchedData().Count
	var readyCount int64
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
		return fmt.Sprintf("%s:%s", item.Candidater.Get("Name"), item.Candidater.Get("ID"))
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
		hostType := item.Candidater.Get("HostType")
		if result.Unit.SchedData().Hypervisor == hostType {
			items = append(items, item)
		}
	}

	var results []api.ForecastResult
	for _, item := range items {
		addInfos(result.Unit.LogManager.FailedLogs(), item)
	}

	for _, item := range items {
		if item.Count <= 0 {
			continue
		}
		readyCount += item.Count
		ret := api.ForecastResult{
			Candidate: logIndex(item),
			Count:     item.Count,
			Capacity:  item.Capacity,
		}
		results = append(results, ret)
	}
	canCreate := true
	if readyCount < reqCount {
		canCreate = false
	}
	return &api.SchedForecastResult{
		CanCreate: canCreate,
		Filters:   filters,
		Results:   results,
	}
}
