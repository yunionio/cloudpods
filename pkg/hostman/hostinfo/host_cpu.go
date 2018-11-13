package hostinfo

import (
	"github.com/shirou/gopsutil/cpu"
	"yunion.io/x/log"
)

type SCPUInfo struct {
	CpuCount    int
	cpuFreq     float32 // MHZ
	cpuFeatures []string
}

func DetectCpuInfo() *SCPUInfo {
	cpuCount, _ := cpu.Counts(true)
	log.Errorln(cpu.Percent(0, false)) // ///
	return &SCPUInfo{
		CpuCount: cpuCount,
	}
}
