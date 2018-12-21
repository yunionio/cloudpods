package hostinfo

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/cpu"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SCPUInfo struct {
	CpuCount    int
	cpuFreq     float32 // MHZ
	cpuFeatures []string

	cpuInfoProc *types.CPUInfo
	cpuInfoDmi  *types.DMICPUInfo
}

func DetectCpuInfo() (*SCPUInfo, err) {
	cpuinfo := new(SCPUInfo)
	cpuCount, _ := cpu.Counts(true)
	cpuinfo.CpuCount = cpuCount
	spec, err := cpuinfo.fetchCpuSpecs()
	if err != nil {
		return nil, err
	}
	strCpuFreq := spec["cpu_freq"]
	freq, err := strconv.ParseInt(strCpuFreq, 10, 0)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	cpu.Percent(interval, percpu)
	ret, err := cloudcommon.FileGetContents("/proc/cpuinfo")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	cpuinfo.cpuInfoProc, err = sysutils.ParseCPUInfo(strings.Split(ret, "\n"))
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	ret, err = exec.Command("dmidecode", "-t", "4").Output()
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	cpuinfo.cpuInfoDmi, err = sysutils.ParseDMICPUInfo(strings.Split(string(ret), "\n"))
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return cpuinfo
}

func (c *SCPUInfo) fetchCpuSpecs() (map[string]string, error) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var spec = make(map[string]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		colon := strings.Index(line, ":")
		if colon > 0 {
			key := strings.TrimSpace(line[:colon])
			val := strings.TrimSpace(line[colon+1:])
			if key == "cpu MHz" {
				spec["cpu_freq"] = val
			} else if key == "flags" {
				spec["flags"] = val
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Errorln(err)
		return nil, err
	}
	return spec, nil
}

// percentInterval(ms)
func (c *SCPUInfo) GetJsonDesc(percentInterval int) {
	// perc, err := cpu.Percent(time.Millisecond*percentInterval, false)
	// os. ?????可能不需要要写
}
