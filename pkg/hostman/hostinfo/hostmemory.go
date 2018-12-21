package hostinfo

import (
	"bufio"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SMemory struct {
	Total   int
	Free    int
	Used    int
	MemInfo *types.DMIMemInfo
}

func DetectMemoryInfo() (*SMemory, error) {
	var mem = new(SMemory)
	info, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	mem.Total = int(info.Total / 1024 / 1024)
	mem.Free = int(info.Available / 1024 / 1024)
	mem.Used = mem.Total - mem.Free
	ret, err := exec.Command("dmidecode", "-t", "17").Output()
	if err != nil {
		return nil, err
	}
	mem.MemInfo = sysutils.ParseDMIMemInfo(strings.Split(string(ret), "\n"))
	return mem, nil
}

func (m *SMemory) GetHugepagesizeMb() int {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		log.Errorln(err)
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Hugepagesize:") {
			re := regexp.MustCompile(`\s+`)
			segs := re.Split(line, -1)
			v, err := strconv.Atoi(segs[1])
			if err != nil {
				log.Errorln(err)
				return 0
			}
			return int(v) / 1024
		}
	}
	if err := scanner.Err(); err != nil {
		log.Errorln(err)
	}
	return 0
}
