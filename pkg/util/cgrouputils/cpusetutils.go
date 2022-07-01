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

package cgrouputils

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strconv"
	"sync"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	utilHistoryFile        = "/tmp/util.history"
	MAX_HISTORY_UTIL_COUNT = 5
)

var (
	rebalanceProcessesLock    = sync.Mutex{}
	rebalanceProcessesRunning = false

	utilHistory map[string][]float64
)

func RebalanceProcesses(pids []string) {
	rebalanceProcessesLock.Lock()
	if rebalanceProcessesRunning {
		rebalanceProcessesLock.Unlock()
		return
	} else {
		rebalanceProcessesRunning = true
		rebalanceProcessesLock.Unlock()
	}

	err := rebalanceProcesses(pids)
	if err != nil {
		log.Errorf("rebalance processes error: %s", err)
	}
	rebalanceProcessesRunning = false
}

func rebalanceProcesses(pids []string) error {
	FetchHistoryUtil()

	cpu, err := GetSystemCpu()
	if err != nil {
		log.Errorln(err)
		return err
	}
	cpuCount := cpu.GetPhysicalNum()
	if cpuCount <= 1 {
		return nil
	}

	info, err := GetProcessesCpuinfo(pids)
	if err != nil {
		return err
	}
	ret := ArrangeProcesses(info, cpuCount)
	if len(ret) != 0 {
		CommitProcessesCpuset(ret)
	}

	SaveHistoryUtil()
	return nil
}

func CommitProcessesCpuset(cpus []CPULoad) {
	for i, cpu := range cpus {
		for _, proc := range cpu.Processes {
			if proc.Cpuset == nil || *proc.Cpuset != i {
				CommitProcessCpuset(proc, i)
			}
		}
	}
}

func CommitProcessCpuset(proc *ProcessCPUinfo, idx int) {
	cpu, _ := GetSystemCpu()
	sets := cpu.GetCpuset(idx)
	if len(sets) > 0 {
		cpuset := NewCGroupCPUSetTask(strconv.Itoa(proc.Pid), "", 0, sets)
		cpuset.SetTask()
	}
}

func GetMaxLoadDiff(cpus []CPULoad) float64 {
	var maxLd, minLd = -1.0, -1.0
	for _, cpu := range cpus {
		wt := cpu.GetWeight()
		if maxLd < 0 || maxLd < wt {
			maxLd = wt
		}
		if minLd < 0 || minLd > wt {
			minLd = wt
		}
	}
	if minLd > 0 {
		return (maxLd - minLd) / minLd
	} else {
		return 0.0
	}
}

func GetMinLoadCpu(cpus []CPULoad, cpuset *int) *CPULoad {
	var (
		mintWt = -1.0
		minCpu *CPULoad
	)

	for i := 0; i < len(cpus); i++ {
		wt := cpus[i].GetWeight()
		if mintWt < 0.0 || mintWt > wt || (mintWt == wt && cpuset != nil && *cpuset == i) {
			mintWt = wt
			minCpu = &cpus[i]
		}
	}
	return minCpu
}

func ArrangeProcesses(infos []*ProcessCPUinfo, cpuCount int) []CPULoad {
	var (
		newProc = false
		cpus    = make([]CPULoad, cpuCount)
		procs   = CPULoad{}
	)

	for _, info := range infos {
		procs.AddProcess(info)
		if info.Cpuset != nil && *info.Cpuset < cpuCount {
			cpus[*info.Cpuset].AddProcess(info)
		} else {
			newProc = true
		}
	}
	prevDiff := GetMaxLoadDiff(cpus)
	if !newProc && prevDiff < 0.2 {
		return nil
	}
	for _, c := range cpus {
		c.Processes = nil
	}
	procs.Sort()

	for _, p := range procs.Processes {
		mincpu := GetMinLoadCpu(cpus, p.Cpuset)
		mincpu.AddProcess(p)
	}
	return cpus
}

func GetProcessesCpuinfo(pids []string) ([]*ProcessCPUinfo, error) {
	if len(pids) == 0 {
		var err error
		pids, err = GetAllPids()
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
	}
	cpu, err := GetSystemCpu()
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	coreCnt := len(cpu.DieList[0].CoreList)

	var ret = []*ProcessCPUinfo{}
	for _, pid := range pids {
		ipid, err := strconv.Atoi(pid)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		info, err := NewProcessCPUinfo(ipid)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		if info.Share != nil && *info.Share < float64(coreCnt) {
			ret = append(ret, info)
		}
	}
	return ret, nil
}

func GetAllPids() ([]string, error) {
	var pids = []string{}
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`^\d+$`)
	for _, f := range files {
		if re.MatchString(f.Name()) {
			pids = append(pids, f.Name())
		}
	}
	return pids, nil
}

func FetchHistoryUtil() map[string][]float64 {
	if utilHistory == nil {
		utilHistory = make(map[string][]float64)
		if fileutils2.Exists(utilHistoryFile) {
			contents, err := fileutils2.FileGetContents(utilHistoryFile)
			if err != nil {
				log.Errorf("FetchHistoryUtil error: %s", err)
				return utilHistory
			}
			var objmap map[string]*json.RawMessage
			if err := json.Unmarshal([]byte(contents), &objmap); err != nil {
				log.Errorf("FetchHistoryUtil error: %s", err)
				return utilHistory
			}
			for k, v := range objmap {
				var s = []float64{}
				if err := json.Unmarshal(*v, &s); err != nil {
					log.Errorf("FetchHistoryUtil error: %s", err)
					break
				}
				utilHistory[k] = s
			}
		}
	}
	return utilHistory
}

func SaveHistoryUtil() {
	content, err := json.Marshal(utilHistory)
	if err != nil {
		log.Errorf("SaveHistoryUtil error: %s", err)
	} else {
		fileutils2.FilePutContents(utilHistoryFile, string(content), false)
	}
}
