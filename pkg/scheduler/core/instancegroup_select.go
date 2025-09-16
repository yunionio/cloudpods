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
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"

	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

func transToInstanceGroupSchedResult(result *SchedResultItemList, schedInfo *api.SchedInfo) *schedapi.ScheduleOutput {
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
	*SchedResultItem
	instanceGroupCapacity map[string]int64
	masterCount           int64
	backupCount           int64
}

func (item *sSchedResultItem) minInstanceGroupCapacity(groupSet map[string]*models.SGroup) int64 {
	var mincapa int64 = -1
	for id, capa := range item.instanceGroupCapacity {
		if _, ok := groupSet[id]; !ok {
			continue
		}
		if mincapa == -1 || capa < mincapa {
			mincapa = capa
		}
	}
	return mincapa
}

func buildHosts(result *SchedResultItemList, groups map[string]*models.SGroup) []*sSchedResultItem {
	hosts := make([]*sSchedResultItem, result.Data.Len())
	for i := 0; i < len(result.Data); i++ {
		getter := result.Data[i].Candidater.Getter()
		igCapacity := make(map[string]int64)
		for id, group := range groups {
			c, err := getter.GetFreeGroupCount(id)
			if err != nil {
				if errors.Cause(err) == ErrInstanceGroupNotFound {
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
	return hosts
}

// sortHost sorts the host for guest that is the backup one of the high-availability guest
// if isBackup is true and the master one if isBackup is false.
func sortHosts(hosts []*sSchedResultItem, guestInfo *sGuestInfo, isBackup *bool) {
	sortIndexi, sortIndexj := make([]int64, 5), make([]int64, 5)
	sort.Slice(hosts, func(i, j int) bool {
		switch {
		case isBackup == nil:
			sortIndexi[0], sortIndexj[0] = hosts[i].Count, hosts[j].Count
		case *isBackup:
			sortIndexi[0], sortIndexj[0] = hosts[i].backupCount, hosts[j].backupCount
		default:
			sortIndexi[0], sortIndexj[0] = hosts[i].masterCount, hosts[j].masterCount
		}
		sortIndexi[1], sortIndexj[1] = hosts[i].Count, hosts[j].Count
		sortIndexi[2], sortIndexj[2] = -(hosts[i].minInstanceGroupCapacity(guestInfo.instanceGroupsDetail)), -(hosts[j].minInstanceGroupCapacity(guestInfo.instanceGroupsDetail))
		sortIndexi[3], sortIndexj[3] = scoreNormalization(hosts[i].Score, hosts[j].Score)
		sortIndexi[4], sortIndexj[4] = -(hosts[i].Capacity), -(hosts[j].Capacity)
		for i := 0; i < 5; i++ {
			if sortIndexi[i] == sortIndexj[i] {
				continue
			}
			return sortIndexi[i] < sortIndexj[i]
		}
		return true
	})
}

// scoreNormalization compare the value of s1 and s2.
// If s1 is less than s2, return 1, 0 which means s2 is better than s1.
func scoreNormalization(s1, s2 Score) (int64, int64) {
	preferScore1, preferScore2 := s1.PreferScore()-s1.AvoidScore(), s2.PreferScore()-s2.AvoidScore()
	normalScore1, normalScore2 := s1.NormalScore(), s2.NormalScore()
	if preferScore1 < preferScore2 {
		return 0, 1
	}
	if preferScore1 > preferScore2 {
		return 1, 0
	}
	if normalScore1 < normalScore2 {
		return 0, 1
	}
	if normalScore1 > normalScore2 {
		return 1, 0
	}
	return 0, 0
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
	storageUsed :=
		NewStorageUsed()
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
	storageUsed := NewStorageUsed()
	isBackup := true
	isMaster := false
	for i := 0; i < len(guestInfos); i++ {
		for wireid, hosts := range wireHostMap {
			if nowireIds.Has(wireid) {
				continue
			}
			masterItem := selectHost(hosts, guestInfos[i], &isMaster, true)
			if masterItem == nil {
				masterItem = selectHost(hosts, guestInfos[i], &isMaster, false)
				if masterItem == nil {
					nowireIds.Insert(wireid)
					continue
				}
			}
			// mark master used for now
			markHostUsed(masterItem, guestInfos[i], &isMaster)
			backupItem := selectHost(hosts, backGuestInfos[i], &isBackup, false)
			if backupItem == nil {
				nowireIds.Insert(wireid)
				unMarkHostUsed(masterItem, guestInfos[i], &isMaster)
				continue
			}
			markHostUsed(backupItem, backGuestInfos[i], &isBackup)
			canRe := masterItem.ToCandidateResource(storageUsed)
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
		host.masterCount++
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
		host.masterCount--
	}
}

// selectHost select host from hosts for guest described by guestInfo.
// If forced is true, all instanceGroups will be forced.
// Otherwise, the instanceGroups with ForceDispersion 'false' will be unforced.
func selectHost(hosts []*sSchedResultItem, guestInfo sGuestInfo, isBackup *bool, forced bool) *sSchedResultItem {
	sortHosts(hosts, &guestInfo, isBackup)
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
