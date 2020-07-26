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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func transToInstanceGroupSchedResult(result *core.SchedResultItemList, schedInfo *api.SchedInfo) *schedapi.ScheduleOutput {
	for _, item := range result.Data {
		item.Count = 0
	}
	guestInfos, backGuestInfos, groups := generateGuestInfo(schedInfo)
	hosts := buildHosts(result, groups)
	if len(backGuestInfos) > 0 {
		return getBackupSchedResult(hosts, guestInfos, backGuestInfos, schedInfo.SessionId)
	}
	return getSchedResult(hosts, guestInfos, schedInfo.SessionId)
}

type sGuestInfo struct {
	schedInfo            *api.SchedInfo
	instanceGroupsDetail map[string]*models.SGroup
	preferHost           string
}

type sSchedResultItem struct {
	*core.SchedResultItem
	instanceGroupCapacity map[string]int64
	mainCount           int64
	backupCount           int64
}

func buildHosts(result *core.SchedResultItemList, groups map[string]*models.SGroup) []*sSchedResultItem {
	hosts := make([]*sSchedResultItem, result.Data.Len())
	for i := 0; i < len(result.Data); i++ {
		getter := result.Data[i].Candidater.Getter()
		igCapacity := make(map[string]int64)
		for id, group := range groups {
			c, err := getter.GetFreeGroupCount(id)
			if err != nil {
				if errors.Cause(err) == candidate.ErrInstanceGroupNotFound {
					igCapacity[id] = int64(group.Granularity)
				} else {
					igCapacity[id] = 0
					log.Errorf("GetFreeGroupCount: %s", err.Error())
				}
			} else {
				igCapacity[id] = int64(c)
			}
		}
		hosts[i] = &sSchedResultItem{
			SchedResultItem:       result.Data[i],
			instanceGroupCapacity: igCapacity,
		}
	}
	sortHosts(hosts, nil)
	return hosts
}

// sortHost sorts the host for guest that is the backup one of the high-availability guest
// if isBackup is true and the main one if isBackup is false.
func sortHosts(hosts []*sSchedResultItem, isBackup *bool) {
	sort.Slice(hosts, func(i, j int) bool {
		var counti, countj int64
		switch {
		case isBackup == nil:
			counti, countj = hosts[i].Count, hosts[j].Count
		case *isBackup:
			counti, countj = hosts[i].backupCount, hosts[j].backupCount
		default:
			counti, countj = hosts[i].mainCount, hosts[j].mainCount
		}
		if counti == countj {
			return hosts[i].Capacity > hosts[j].Capacity
		}
		return counti < countj
	})
}

// buildWireHosts classify hosts according to their wire
func buildWireHosts(hosts []*sSchedResultItem) map[string][]*sSchedResultItem {
	wireHostMap := make(map[string][]*sSchedResultItem)
	for _, host := range hosts {
		networks := host.Candidater.Getter().Networks()
		for j := 0; j < len(networks); j++ {
			if hosts, ok := wireHostMap[networks[j].WireId]; ok {
				if hostsIndex(host.ID, hosts) < 0 {
					wireHostMap[networks[j].WireId] = append(hosts, host)
				}
			} else {
				wireHostMap[networks[j].WireId] = []*sSchedResultItem{host}
			}
		}
	}
	return wireHostMap
}

// generateGuestInfo return guestInfos, backupGuestInfos and all instanceGroups
func generateGuestInfo(schedInfo *api.SchedInfo) ([]sGuestInfo, []sGuestInfo, map[string]*models.SGroup) {
	infos := make([]sGuestInfo, 0, schedInfo.Count)
	infobs := make([]sGuestInfo, 0, schedInfo.Count)
	groups := make(map[string]*models.SGroup)
	name := schedInfo.Name
	if len(name) == 0 {
		name = "default"
	}
	for id, group := range schedInfo.InstanceGroupsDetail {
		groups[id] = group
	}
	for i := 0; i < schedInfo.Count; i++ {
		info := sGuestInfo{
			schedInfo:            schedInfo,
			instanceGroupsDetail: make(map[string]*models.SGroup),
			preferHost:           schedInfo.PreferHost,
		}
		for id, group := range schedInfo.InstanceGroupsDetail {
			info.instanceGroupsDetail[id] = group
		}
		infos = append(infos, info)
		if !schedInfo.Backup {
			continue
		}
		infob := sGuestInfo{
			schedInfo:            schedInfo,
			preferHost:           schedInfo.PreferBackupHost,
			instanceGroupsDetail: make(map[string]*models.SGroup),
		}
		infobs = append(infobs, infob)
		// Virtual an instanceGroup
		group := models.SGroup{
			Granularity:     1,
			ForceDispersion: tristate.True,
		}
		groupid := fmt.Sprintf("virtual-%s-%d", name, i)
		group.Id = groupid
		infos[i].instanceGroupsDetail[groupid] = &group
		infobs[i].instanceGroupsDetail[groupid] = &group
		groups[groupid] = &group
	}
	return infos, infobs, groups
}

func hostsIndex(hostId string, hosts []*sSchedResultItem) int {
	for i := 0; i < len(hosts); i++ {
		if hosts[i].ID == hostId {
			return i
		}
	}
	return -1
}

// getBackupSchedResult return the ScheduleOutput for guest without backup
func getSchedResult(hosts []*sSchedResultItem, guestInfos []sGuestInfo, sid string) *schedapi.ScheduleOutput {
	apiResults := make([]*schedapi.CandidateResource, 0)
	storageUsed := core.NewStorageUsed()
	var i int = 0
	for ; i < len(guestInfos); i++ {
		host := selectHost(hosts, guestInfos[i], nil, true)
		if host == nil {
			host = selectHost(hosts, guestInfos[i], nil, false)
			if host == nil {
				er := &schedapi.CandidateResource{Error: fmt.Sprintf("no suitable Host for No.%d Guest", i+1)}
				apiResults = append(apiResults, er)
				break
			}
		}
		markHostUsed(host, guestInfos[i], nil)
		tr := host.ToCandidateResource(storageUsed)
		tr.SessionId = sid
		apiResults = append(apiResults, tr)
	}
	for ; i < len(guestInfos); i++ {
		er := &schedapi.CandidateResource{Error: fmt.Sprintf("no suitable Host for No.%d Guest", i+1)}
		apiResults = append(apiResults, er)
	}
	ret := new(schedapi.ScheduleOutput)
	ret.Candidates = apiResults
	return ret
}

// getBackupSchedResult return the ScheduleOutput for guest with backup
func getBackupSchedResult(hosts []*sSchedResultItem, guestInfos, backGuestInfos []sGuestInfo, sid string) *schedapi.ScheduleOutput {
	wireHostMap := buildWireHosts(hosts)
	apiResults := make([]*schedapi.CandidateResource, 0, len(guestInfos))
	nowireIds := sets.NewString()
	storageUsed := core.NewStorageUsed()
	isBackup := true
	isMain := false
	for i := 0; i < len(guestInfos); i++ {
		for wireid, hosts := range wireHostMap {
			if nowireIds.Has(wireid) {
				continue
			}
			mainItem := selectHost(hosts, guestInfos[i], &isMain, true)
			if mainItem == nil {
				mainItem = selectHost(hosts, guestInfos[i], &isMain, false)
				if mainItem == nil {
					nowireIds.Insert(wireid)
					continue
				}
			}
			// mark main used for now
			markHostUsed(mainItem, guestInfos[i], &isMain)
			backupItem := selectHost(hosts, backGuestInfos[i], &isBackup, false)
			if backupItem == nil {
				nowireIds.Insert(wireid)
				unMarkHostUsed(mainItem, guestInfos[i], &isMain)
				continue
			}
			markHostUsed(backupItem, backGuestInfos[i], &isBackup)
			canRe := mainItem.ToCandidateResource(storageUsed)
			canRe.BackupCandidate = backupItem.ToCandidateResource(storageUsed)
			canRe.SessionId = sid
			canRe.BackupCandidate.SessionId = sid
			apiResults = append(apiResults, canRe)
			break
		}
		if len(apiResults) == i+1 {
			continue
		}
		er := &schedapi.CandidateResource{Error: fmt.Sprintf("no suitable Host for No.%d Highly available Guest", i+1)}
		apiResults = append(apiResults, er)
	}
	ret := new(schedapi.ScheduleOutput)
	ret.Candidates = apiResults
	return ret
}

func markHostUsed(host *sSchedResultItem, guestInfo sGuestInfo, isBackup *bool) {
	for gid := range guestInfo.instanceGroupsDetail {
		host.instanceGroupCapacity[gid] = host.instanceGroupCapacity[gid] - 1
	}
	host.Capacity--
	host.Count++
	if isBackup == nil {
		return
	}
	if *isBackup {
		host.backupCount++
	} else {
		host.mainCount++
	}
}

// unMarkHostUsed is the reverse operation of markHostUsed
func unMarkHostUsed(host *sSchedResultItem, guestInfo sGuestInfo, isBackup *bool) {
	for gid := range guestInfo.instanceGroupsDetail {
		host.instanceGroupCapacity[gid] = host.instanceGroupCapacity[gid] + 1
	}
	host.Capacity++
	host.Count--
	if isBackup == nil {
		return
	}
	if *isBackup {
		host.backupCount--
	} else {
		host.mainCount--
	}
}

// selectHost select host from hosts for guest described by guestInfo.
// If forced is true, all instanceGroups will be forced.
// Otherwise, the instanceGroups with ForceDispersion 'false' will be unforced.
func selectHost(hosts []*sSchedResultItem, guestInfo sGuestInfo, isBackup *bool, forced bool) *sSchedResultItem {
	sortHosts(hosts, isBackup)
	var idx = -1
	if len(guestInfo.preferHost) > 0 {
		if idx = hostsIndex(guestInfo.preferHost, hosts); idx < 0 {
			return nil
		}
	}
	var choosed bool
Loop:
	for i, host := range hosts {
		if idx >= 0 && idx != i {
			continue
		}
		if host.Capacity <= 0 {
			continue
		}
		// check forced instanceGroup
		for id, group := range guestInfo.instanceGroupsDetail {
			capacity := host.instanceGroupCapacity[id]
			checkCapacity := forced || group.ForceDispersion.IsTrue()
			if checkCapacity && capacity <= 0 {
				continue Loop
			}
		}
		idx = i
		choosed = true
		break
	}
	if choosed {
		return hosts[idx]
	}
	return nil
}
