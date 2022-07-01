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
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/process"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var systemCpu *CPU

type CPU struct {
	DieList []*CPUDie
}

func NewCPU() (*CPU, error) {
	var cpu = new(CPU)
	cpu.DieList = make([]*CPUDie, 0)

	cpuinfo, err := fileutils2.FileGetContents("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}

	var core *CPUCore
	for _, line := range strings.Split(cpuinfo, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key == "processor" {
				iVal, err := strconv.Atoi(val)
				if err != nil {
					return nil, err
				}
				if core != nil {
					cpu.AddCore(core)
				}
				core = NewCPUCore(iVal)
			} else if core != nil {
				core.setInfo(key, val)
			}
		}
	}
	if core != nil {
		cpu.AddCore(core)
	}
	return cpu, nil
}

func (c *CPU) AddCore(core *CPUCore) {
	for len(c.DieList) < core.PhysicalId+1 {
		c.DieList = append(c.DieList, NewCPUDie(len(c.DieList)))
	}
	c.DieList[core.PhysicalId].AddCore(core)
}

func (c *CPU) GetCpuset(idx int) string {
	if idx >= 0 && idx < len(c.DieList) {
		return c.DieList[idx].GetCoreStr()
	} else {
		return ""
	}
}

func (c *CPU) GetPhysicalNum() int {
	return len(c.DieList)
}

func (c *CPU) GetPhysicalId(cstr string) int {
	for _, d := range c.DieList {
		if d.GetCoreStr() == cstr {
			return d.Index
		}
	}
	return -1
}

type CPUCore struct {
	Index      int
	VendorId   string
	Mhz        float64
	CacheSize  string
	PhysicalId int
	CoreId     int
}

func NewCPUCore(index int) *CPUCore {
	return &CPUCore{Index: index}
}

func (c *CPUCore) setInfo(k, v string) {
	switch k {
	case "vendor_id":
		c.VendorId = v
	case "cpu MHz":
		val, _ := strconv.ParseFloat(v, 64)
		c.Mhz = val
	case "cache size":
		c.CacheSize = strings.Split(v, " ")[0]
	case "physical id":
		val, _ := strconv.Atoi(v)
		c.PhysicalId = val
	case "core id":
		val, _ := strconv.Atoi(v)
		c.CoreId = val
	}
}

type CPUDie struct {
	Index    int
	CoreList []*CPUCore
}

func (d *CPUDie) AddCore(core *CPUCore) {
	d.CoreList = append(d.CoreList, core)
}

func (d *CPUDie) GetCoreStr() string {
	coreIdx := []string{}
	for _, c := range d.CoreList {
		coreIdx = append(coreIdx, strconv.Itoa(c.Index))
	}
	sort.Slice(coreIdx, func(i int, j int) bool {
		si, _ := strconv.Atoi(coreIdx[i])
		sj, _ := strconv.Atoi(coreIdx[j])
		return si < sj
	})
	return strings.Join(coreIdx, ",")
}

func NewCPUDie(index int) *CPUDie {
	return &CPUDie{Index: index}
}

func GetSystemCpu() (*CPU, error) {
	if systemCpu == nil {
		var err error
		systemCpu, err = NewCPU()
		if err != nil {
			return nil, err
		}
	}
	return systemCpu, nil
}

func ParseCpusetStr(cpuset string) string {
	var (
		idxList = make([]string, 0)
		sets    = strings.Split(cpuset, ",")
	)

	for _, idxstr := range sets {
		if strings.Contains(idxstr, "-") {
			ses := strings.Split(idxstr, "-")
			start, _ := strconv.Atoi(ses[0])
			end, _ := strconv.Atoi(ses[1])
			for start <= end {
				idxList = append(idxList, strconv.Itoa(start))
				start += 1
			}
		} else {
			idxList = append(idxList, idxstr)
		}
	}
	sort.Slice(idxList, func(i int, j int) bool {
		si, _ := strconv.Atoi(idxList[i])
		sj, _ := strconv.Atoi(idxList[j])
		return si < sj
	})
	return strings.Join(idxList, ",")
}

type ProcessCPUinfo struct {
	Pid    int
	Share  *float64
	Cpuset *int
	Util   float64
	Weight float64
}

func (p *ProcessCPUinfo) String() string {
	share := -1.0
	if p.Share != nil {
		share = *p.Share
	}
	cpuset := -1
	if p.Cpuset != nil {
		cpuset = *p.Cpuset
	}
	return fmt.Sprintf("(%d, %f, %f, %f, %d)", p.Pid, p.Weight, p.Util, share, cpuset)
}

func Average(arr []float64) float64 {
	var total = 0.0
	for _, a := range arr {
		total += a
	}
	return total / float64(len(arr))
}

func GetProcessWeight(share *float64, util float64) float64 {
	if share != nil {
		return (*share) * (util*0.8 + 30)
	}
	return 0.0
}

func NewProcessCPUinfo(pid int) (*ProcessCPUinfo, error) {
	cpuinfo := new(ProcessCPUinfo)
	cpuinfo.Pid = pid
	spid := strconv.Itoa(pid)

	cpuTask := NewCGroupCPUTask(spid, "", 0)
	if cpuTask.taskIsExist() {
		share := cpuTask.GetParam("cpu.shares")
		ishare, err := strconv.ParseFloat(share, 64)
		if err != nil {
			log.Errorln(err)
		} else {
			ishare /= 1024.0
			cpuinfo.Share = &ishare
		}
	}

	cpusetTask := NewCGroupCPUSetTask(fmt.Sprintf("%d", pid), "", 0, "")
	if cpusetTask.taskIsExist() {
		cpuset := cpusetTask.GetParam("cpuset.cpus")
		if len(cpuset) > 0 {
			c, err := GetSystemCpu()
			if err != nil {
				log.Errorln(err)
			} else {
				icpuset := c.GetPhysicalId(ParseCpusetStr(cpuset))
				cpuinfo.Cpuset = &icpuset
			}
		}
	}

	if cpuinfo.Share != nil {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		util, err := proc.CPUPercent()
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		util /= float64(*cpuinfo.Share)
		uHistory := FetchHistoryUtil()

		var utils = []float64{}
		if _, ok := uHistory[spid]; ok {
			utils = uHistory[spid]
		}
		utils = append(utils, util)
		for len(utils) > MAX_HISTORY_UTIL_COUNT {
			utils = utils[1:]
		}
		uHistory[spid] = utils

		cpuinfo.Util = Average(utils)
	}

	cpuinfo.Weight = GetProcessWeight(cpuinfo.Share, cpuinfo.Util)
	return cpuinfo, nil
}

type CPULoad struct {
	Processes []*ProcessCPUinfo
}

func (c *CPULoad) AddProcess(proc *ProcessCPUinfo) {
	if c.Processes == nil {
		c.Processes = make([]*ProcessCPUinfo, 0)
	}
	c.Processes = append(c.Processes, proc)
}

func (c *CPULoad) GetWeight() float64 {
	var wt = 0.0
	for _, p := range c.Processes {
		wt += p.Weight
	}
	return wt
}

func (c *CPULoad) Sort() {
	sort.Slice(c.Processes, func(i, j int) bool {
		return c.Processes[i].Weight < c.Processes[j].Weight
	})
}
