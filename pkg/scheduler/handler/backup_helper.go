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
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
)

func transToBackupSchedResult(result *core.SchedResultItemList, preferMasterHost, preferBackupHost string, count int64, setDirty bool) *schedapi.ScheduleOutput {
	// clean each result sched result item's count
	for _, item := range result.Data {
		item.Count = 0
	}

	apiResults := newBackupSchedResult(result, preferMasterHost, preferBackupHost, count, setDirty)
	return apiResults
}

func newBackupSchedResult(
	result *core.SchedResultItemList,
	preferMasterHost, preferBackupHost string,
	count int64,
	setDirty bool,
) *schedapi.ScheduleOutput {
	ret := new(schedapi.ScheduleOutput)
	apiResults := make([]*schedapi.CandidateResource, 0)
	for i := 0; i < int(count); i++ {
		log.V(10).Debugf("Select backup host from result: %s", result)
		target, err := getSchedBackupResult(result, preferMasterHost, preferBackupHost, setDirty)
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
	setDirty bool,
) (*schedapi.CandidateResource, error) {
	masterHost := selectMasterHost(result.Data, preferMasterHost, preferBackupHost)
	if masterHost == nil {
		return nil, fmt.Errorf("Can't find master host %q", preferMasterHost)
	}
	backupHost := selectBackupHost(masterHost.ID, preferBackupHost, result.Data)
	if backupHost == nil {
		return nil, fmt.Errorf("Can't find backup host %q by master %q", preferBackupHost, masterHost.ID)
	}

	markHostUsed(masterHost, setDirty)
	markHostUsed(backupHost, setDirty)
	sort.Sort(sort.Reverse(result))

	ret := masterHost.ToCandidateResource()
	ret.BackupCandidate = backupHost.ToCandidateResource()
	return ret, nil
}

func markHostUsed(host *core.SchedResultItem, setDirty bool) {
	host.Count++
	host.Capacity--
	if setDirty {
		setHostDirty(host)
	}
}

// selectMasterID find master host id run VM
// return nil if not found
func selectMasterHost(result []*core.SchedResultItem, preferMasterHost, preferBackupHost string) *core.SchedResultItem {
	if len(result) == 0 {
		return nil
	}
	host := result[0]
	if host.ID == preferMasterHost {
		if host.Capacity >= 1 {
			return host
		} else {
			return nil
		}
	}
	if host.Capacity >= 1 && host.ID != preferBackupHost {
		if len(preferMasterHost) == 0 {
			return host
		}
		if len(result) == 1 {
			return nil
		}
		restHosts := result[1:]
		return selectMasterHost(restHosts, preferMasterHost, preferBackupHost)
	}
	if len(result) == 1 {
		return nil
	}
	restHosts := result[1:]
	return selectMasterHost(restHosts, preferMasterHost, preferBackupHost)
}

func selectBackupHost(masterID, preferBackupHost string, result []*core.SchedResultItem) *core.SchedResultItem {
	if len(result) == 0 {
		return nil
	}
	firstHost := result[0]
	if canHostAsBackup(masterID, preferBackupHost, firstHost) {
		return firstHost
	}
	if len(result) == 1 {
		return nil
	}
	restHosts := result[1:]
	return selectBackupHost(masterID, preferBackupHost, restHosts)
}

func canHostAsBackup(masterID, preferBackupHost string, host *core.SchedResultItem) bool {
	if host.ID == masterID {
		return false
	}
	if host.Capacity == 0 {
		return false
	}
	if preferBackupHost != "" {
		if host.ID != preferBackupHost {
			return false
		}
	}
	return true
}

type dirtyItemAdapter struct {
	*core.SchedResultItem
}

func (a *dirtyItemAdapter) Index() (string, error) {
	return a.ID, nil
}

func (a *dirtyItemAdapter) GetCount() uint64 {
	return uint64(a.Count)
}

func setHostDirty(host *core.SchedResultItem) {
	schedman.GetCandidateManager().SetCandidateDirty(&dirtyItemAdapter{SchedResultItem: host})
}
