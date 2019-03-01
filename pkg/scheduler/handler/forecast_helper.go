package handler

import (
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func isSelectedHost(item *core.SchedResultItem) {
	if item.Count > 0 {
		return true
	}
	return false
}

func transToSchedForecastResult(result *core.SchedResultItemList) interface{} {
	unit := result.Unit
	reqCount := unit.SchedData().Count
	readyCount := 0
	infos := make([]api.ForecastInfo, 0)

	infoMap := make(map[string]*ForecastInfo)
	getOrNewInfo := func(preName string) *ForecastInfo {
		if info, ok := infoMap[preName]; !ok {
			i := &ForecastInfo{
				Filter: preName,
				Count:  1,
			}
			infoMap[preName] = true
			return i
		} else {
			return info
		}
	}
	addInfos := func(capaDetail map[string]int64) {
		for preName, cnt := range capaDetail {
			if cnt <= 0 {
				getOrNewInfo(preName)
			}
		}
	}
	for _, item := range result.Data {
		readyCount += item.Count

	}
	canCreate := true
	if readyCount < reqCount {
		canCreate = false
	}
	return &api.SchedForecastResult{
		CanCreate: false,
		Infos:     nil,
	}
}
