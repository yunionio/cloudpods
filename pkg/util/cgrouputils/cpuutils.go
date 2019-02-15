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
	cpuinfo, err := fileutils2.FileGetContents("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}

	var core *CPUCore
	for _, line := range strings.Split(cpuinfo, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			if parts[0] == "processor" {
				val, err := strconv.Atoi(parts[1])
				if err != nil {
					return nil, err
				}
				core = NewCPUCore(val)
				cpu.AddCore(core)
			} else if core != nil {
				core.setInfo(parts[0], parts[1])
			}
		}
	}
	return cpu, nil
}

func (c *CPU) AddCore(core *CPUCore) {
	for len(c.DieList) < core.PhysicalId {
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
	sort.Slice(coreIdx, func(i int, j int) bool { return coreIdx[i] < coreIdx[j] })
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
			for start < end {
				idxList = append(idxList, strconv.Itoa(start))
				start += 1
			}
		} else {
			idxList = append(idxList, idxstr)
		}
	}
	sort.Slice(idxList, func(i int, j int) bool { return idxList[i] < idxList[j] })
	return strings.Join(idxList, ",")
}

type ProcessCPUinfo struct {
	Pid    int
	Share  *int
	Cpuset *int
	Util   float64
	Weight float64
}

func Average(arr []float64) float64 {
	var total = 0.0
	for _, a := range arr {
		total += a
	}
	return total / float64(len(arr))
}

func GetProcessWeight(share int, util float64) float64 {
	return float64(share) * (util*0.8 + 30)
}

func NewProcessCPUinfo(pid int) (*ProcessCPUinfo, error) {
	cpuinfo := new(ProcessCPUinfo)
	cpuinfo.Pid = pid
	spid := strconv.Itoa(pid)

	share := NewCGroupCPUTask(spid, 0).GetParam("cpu.shares")
	ishare, err := strconv.Atoi(share)
	if err != nil {
		log.Errorln(err)
	} else {
		if ishare != 0 {
			cpuinfo.Share = &ishare
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
			util /= float64(ishare)
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
	}

	cpuset := NewCGroupCPUSetTask(fmt.Sprintf("%s", pid), 0, "").GetParam("cpuset.cpus")
	if len(cpuset) > 0 {
		c, err := GetSystemCpu()
		if err != nil {
			log.Errorln(err)
		} else {
			icpuset := c.GetPhysicalId(ParseCpusetStr(cpuset))
			cpuinfo.Cpuset = &icpuset
		}
	}

	cpuinfo.Weight = GetProcessWeight(*cpuinfo.Share, cpuinfo.Util)
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
