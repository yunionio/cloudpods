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
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	candidatecache "yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/common"
	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
	"yunion.io/x/onecloud/pkg/util/k8s"
)

const defaultIgnorePool = true

var schedManager *SchedulerManager

type SchedulerManager struct {
	ExpireManager    *ExpireManager
	CompletedManager *CompletedManager
	HistoryManager   *HistoryManager
	TaskManager      *TaskManager

	DataManager        *data_manager.DataManager
	CandidateManager   *data_manager.CandidateManager
	KubeClusterManager *k8s.SKubeClusterManager
}

func newSchedulerManager(stopCh <-chan struct{}) *SchedulerManager {
	sm := &SchedulerManager{}
	sm.DataManager = data_manager.NewDataManager(stopCh)
	sm.CandidateManager = data_manager.NewCandidateManager(sm.DataManager, stopCh)
	sm.ExpireManager = NewExpireManager(stopCh)
	sm.CompletedManager = NewCompletedManager(stopCh)
	sm.HistoryManager = NewHistoryManager(stopCh)
	sm.TaskManager = NewTaskManager(stopCh)
	sm.KubeClusterManager = k8s.NewKubeClusterManager(o.Options.Region, 30*time.Second)

	common.RegisterCacheManager(sm.CandidateManager)

	return sm
}

func GetScheduleManager() *SchedulerManager {
	return schedManager
}

func GetK8sClient() (*kubernetes.Clientset, error) {
	return GetScheduleManager().KubeClusterManager.GetK8sClient()
}

func InitAndStart(stopCh <-chan struct{}) {
	if schedManager != nil {
		log.Warningf("Global scheduler already init.")
		return
	}
	schedManager = newSchedulerManager(stopCh)
	go schedManager.start()
	log.Infof("InitAndStart ok")
}

func (sm *SchedulerManager) start() {
	startFuncs := []func(){
		sm.ExpireManager.Run,
		sm.CompletedManager.Run,
		sm.HistoryManager.Run,
		sm.TaskManager.Run,
		sm.DataManager.Run,
		sm.CandidateManager.Run,
		//sm.KubeClusterManager.Start,
	}
	for _, f := range startFuncs {
		go f()
	}
}

func (sm *SchedulerManager) schedule(info *api.SchedInfo) (*core.ScheduleResult, error) {
	// force sync clean expire cache before do schedule
	sm.ExpireManager.Trigger()

	log.V(10).Infof("SchedulerManager do schedule, input: %#v", info)
	task, err := sm.TaskManager.AddTask(sm, info)
	if err != nil {
		return nil, err
	}

	sm.HistoryManager.NewHistoryItem(task)
	results, err := task.Wait()
	if err != nil {
		return nil, err
	}
	log.V(10).Infof("SchedulerManager finish schedule, selected candidates: %#v", results)
	return results, nil
}

// NewSessionID returns the current timestamp of a string type with precision of
// milliseconds. And it should be consistent with the format of Region.
// just like: 1509699887616
func NewSessionID() string {
	return fmt.Sprintf("%v", time.Now().UnixNano()/1000000)
}

// Schedule process the request data that is scheduled for dispatch and complements
// the session information.
func Schedule(info *api.SchedInfo) (*core.ScheduleResult, error) {
	if len(info.SessionId) == 0 {
		info.SessionId = NewSessionID()
	}
	return schedManager.schedule(info)
}

func IsReady() bool {
	return schedManager != nil
}

func GetCandidateManager() *data_manager.CandidateManager {
	return schedManager.CandidateManager
}

func Expire(expireArgs *api.ExpireArgs, trigger bool) (*api.ExpireResult, error) {
	schedManager.ExpireManager.Add(expireArgs)
	if trigger {
		schedManager.ExpireManager.Trigger()
	}
	return &api.ExpireResult{}, nil
}

func CompletedNotify(completedNotifyArgs *api.CompletedNotifyArgs) (*api.CompletedNotifyResult, error) {
	schedManager.CompletedManager.Add(completedNotifyArgs)
	return &api.CompletedNotifyResult{}, nil
}

func getHostCandidatesList(args *api.CandidateListArgs) (*api.CandidateListResult, error) {
	r := new(api.CandidateListResult)
	r.Limit = args.Limit
	r.Offset = args.Offset
	cs, err := GetCandidateManager().GetCandidates(data_manager.CandidateGetArgs{
		ResType:  "host",
		ZoneID:   args.Zone,
		RegionID: args.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("Get host candidates err: %v", err)
	}
	return GetCandidateHostList(cs, args, r)
}

func getBaremetalCandidatesList(args *api.CandidateListArgs) (*api.CandidateListResult, error) {
	r := new(api.CandidateListResult)
	r.Limit = args.Limit
	r.Offset = args.Offset
	cs, err := GetCandidateManager().GetCandidates(data_manager.CandidateGetArgs{
		ResType:  "baremetal",
		ZoneID:   args.Zone,
		RegionID: args.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("Get baremetal candidates err: %v", err)
	}
	return GetCandidateBaremetalList(cs, args, r)
}

func mergeAllCandidateList(host, baremetal *api.CandidateListResult, args *api.CandidateListArgs) (*api.CandidateListResult, error) {
	res := make([]api.CandidateListResultItem, 0)
	if host.Total == args.Limit {
		return host, nil
	} else {
		// must < args.Limit
		res = append(res, host.Data...)
	}

	for _, bm := range baremetal.Data {
		if int64(len(res)) >= args.Limit {
			break
		}
		res = append(res, bm)
	}

	r := new(api.CandidateListResult)
	r.Limit = args.Limit
	r.Offset = args.Offset
	r.Total = int64(len(res))
	r.Data = res

	return r, nil
}

func GetCandidateList(args *api.CandidateListArgs) (*api.CandidateListResult, error) {
	var (
		hostRes, bmRes *api.CandidateListResult
		err            error
	)
	switch args.Type {
	case "all":
		hostRes, err = getHostCandidatesList(args)
		if err != nil {
			return nil, err
		}

		bmRes, err = getBaremetalCandidatesList(args)
		if err != nil {
			return nil, err
		}
		return mergeAllCandidateList(hostRes, bmRes, args)

	case "host":
		return getHostCandidatesList(args)

	case "baremetal":
		return getBaremetalCandidatesList(args)

	default:
		return nil, fmt.Errorf("Unsupport candidate type %q", args.Type)
	}
}

func GetCandidateHostList(
	candidates []core.Candidater,
	args *api.CandidateListArgs,
	r *api.CandidateListResult,
) (*api.CandidateListResult, error) {
	r.Total = int64(len(candidates))

	for _, cc := range candidates {
		if int64(len(r.Data)) >= args.Limit {
			break
		}
		c := cc.(*candidate.HostDesc)
		mem := api.NewResultResourceInt64(
			c.GetFreeMemSize(false),
			c.GetReservedMemSize(),
			c.GetTotalMemSize(false))

		cpu := api.NewResultResourceInt64(
			c.GetFreeCPUCount(false),
			c.GetReservedCPUCount(),
			c.GetTotalCPUCount(false))

		storage := api.NewResultResourceInt64(
			c.GetFreeLocalStorageSize(false),
			c.GetReservedStorageSize(),
			c.GetTotalLocalStorageSize(false))

		item := api.CandidateListResultItem{
			ID:      c.IndexKey(),
			Name:    c.Name,
			Mem:     *mem,
			Cpu:     *cpu,
			Storage: *storage,

			Status:       c.Status,
			HostStatus:   c.HostStatus,
			HostType:     c.GetHostType(),
			EnableStatus: c.GetEnableStatus(),
		}

		pendingUsage, _ := schedmodels.HostPendingUsageManager.GetPendingUsage(c.GetId())
		if pendingUsage != nil {
			item.PendingUsage = pendingUsage.ToMap()
		}
		r.Data = append(r.Data, item)
	}
	return r, nil
}

func GetCandidateBaremetalList(
	candidates []core.Candidater,
	args *api.CandidateListArgs,
	r *api.CandidateListResult,
) (*api.CandidateListResult, error) {
	r.Total = int64(len(candidates))

	for _, cc := range candidates {
		if int64(len(r.Data)) >= args.Limit {
			break
		}
		c := cc.(*candidate.BaremetalDesc)

		mem := api.NewResultResourceInt64(
			c.FreeMemSize(),
			0,
			int64(c.MemSize))

		cpu := api.NewResultResourceInt64(
			c.FreeCPUCount(),
			0,
			int64(c.CpuCount))

		storage := api.NewResultResourceInt64(
			c.FreeStorageSize(),
			0,
			c.StorageSize)

		item := api.CandidateListResultItem{
			ID:      c.IndexKey(),
			Name:    c.Name,
			Mem:     *mem,
			Cpu:     *cpu,
			Storage: *storage,

			Status:       c.Status,
			HostStatus:   c.HostStatus,
			HostType:     c.GetHostType(),
			EnableStatus: c.GetEnableStatus(),
		}
		r.Data = append(r.Data, item)
	}
	return r, nil
}

func GetCandidateDetail(args *api.CandidateDetailArgs) (*api.CandidateDetailResult, error) {
	r := new(api.CandidateDetailResult)
	candidate, err := GetCandidateManager().GetCandidate(args.ID, args.Type)
	if err != nil {
		return nil, err
	}

	r.Candidate = candidate
	return r, nil
}

func Cleanup(cleanupArgs *api.CleanupArgs) (*api.CleanupResult, error) {
	r := new(api.CleanupResult)
	cm := GetCandidateManager()

	if cleanupArgs.ResType != "" {
		cm.ReloadAll(cleanupArgs.ResType)
	} else {
		cm.ReloadAll("host")
		cm.ReloadAll("baremetal")
	}

	return r, nil
}

func GetHistoryList(args *api.HistoryArgs) (*api.HistoryResult, error) {
	offset, limit, all, isSuggestion := args.Offset, args.Limit, args.All, args.IsSuggestion
	if limit == int64(0) {
		limit = int64(50)
	}

	historyItems, total := schedManager.HistoryManager.GetHistoryList(offset, limit, all, isSuggestion)
	items := []*api.HistoryItem{}

	for idx := range historyItems {
		hi := historyItems[idx]
		items = append(items, hi.ToAPI())
	}

	return &api.HistoryResult{
		Items:  items,
		Offset: offset,
		Limit:  int64(len(items)),
		Total:  total,
	}, nil
}

func GetHistoryDetail(historyDetailArgs *api.HistoryDetailArgs) (*api.HistoryDetailResult, error) {
	historyItem := schedManager.HistoryManager.GetHistory(historyDetailArgs.ID)
	if historyItem == nil {
		return nil, fmt.Errorf("History '%v' not found", historyDetailArgs.ID)
	}

	task := historyItem.Task
	schedInfo := task.SchedInfo
	historyTasks := []api.HistoryTask{}

	data := schedInfo
	taskExecutor := task.GetTaskExecutor(data.Tag)
	historyTask := api.HistoryTask{
		Type: data.Hypervisor,
		Data: data,
	}

	if taskExecutor != nil {
		historyTask.Status = taskExecutor.Status
		historyTask.Time = taskExecutor.Time.Local().Format("2006-01-02 15:04:05")
		historyTask.Consuming = fmt.Sprintf("%s", taskExecutor.Consuming)

		resultItems, err := taskExecutor.GetResult()
		historyTask.Result = resultItems
		if err != nil {
			historyTask.Error = fmt.Sprintf("%v", err)
		}

		if historyDetailArgs.Log {
			historyTask.Logs = taskExecutor.GetLogs()
			historyTask.CapacityMap = taskExecutor.GetCapacityMap()
		}
	}

	historyTasks = append(historyTasks, historyTask)

	var inputStr, outputStr, errStr string
	result, err := task.GetResult()

	if err != nil {
		errStr = fmt.Sprintf("%v", err)
	} else {
		if bytes, err0 := json.MarshalIndent(result, "", "  "); err0 == nil {
			outputStr = string(bytes)
		}
	}

	if historyDetailArgs.Raw {
		inputStr = schedInfo.Raw
	}

	historyDetail := &api.HistoryDetail{
		Time:      historyItem.Time.Local().Format("2006-01-02 15:04:05"),
		Consuming: fmt.Sprintf("%s", task.Consuming),
		SessionID: task.GetSessionID(),
		Tasks:     historyTasks,
		Input:     inputStr,
		Output:    outputStr,
		Error:     errStr,
	}

	return &api.HistoryDetailResult{
		Detail: historyDetail,
	}, nil
}

func GetCandidateHostsDesc() ([]core.Candidater, error) {
	return GetCandidateManager().GetCandidates(data_manager.CandidateGetArgs{ResType: "host"})
}

func GetK8sCandidateHosts(nodesName ...string) ([]*candidatecache.HostDesc, error) {
	hosts, err := GetCandidateHostsDesc()
	if err != nil {
		return nil, err
	}
	findHost := func(nodeName string) *candidatecache.HostDesc {
		for _, host := range hosts {
			if host.Getter().Name() == nodeName {
				return host.(*candidatecache.HostDesc)
			}
		}
		return nil
	}
	ret := make([]*candidatecache.HostDesc, 0)
	for _, nodeName := range nodesName {
		if host := findHost(nodeName); host != nil {
			ret = append(ret, host)
		}
	}
	return ret, nil
}
