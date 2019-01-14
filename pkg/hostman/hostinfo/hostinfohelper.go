package hostinfo

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostdhcp"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SCPUInfo struct {
	CpuCount    int
	cpuFreq     int64 // MHZ
	cpuFeatures []string

	cpuInfoProc *types.CPUInfo
	cpuInfoDmi  *types.DMICPUInfo
}

func DetectCpuInfo() (*SCPUInfo, error) {
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
	cpuinfo.cpuFreq = freq

	// cpu.Percent(interval, false)
	ret, err := fileutils2.FileGetContents("/proc/cpuinfo")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	cpuinfo.cpuInfoProc, err = sysutils.ParseCPUInfo(strings.Split(ret, "\n"))
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	bret, err := exec.Command("dmidecode", "-t", "4").Output()
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	cpuinfo.cpuInfoDmi = sysutils.ParseDMICPUInfo(strings.Split(string(bret), "\n"))
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return cpuinfo, nil
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

type SMemory struct {
	Total   int
	Free    int
	Used    int
	MemInfo *types.DMIMemInfo
}

func DetectMemoryInfo() (*SMemory, error) {
	var smem = new(SMemory)
	info, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	smem.Total = int(info.Total / 1024 / 1024)
	smem.Free = int(info.Available / 1024 / 1024)
	smem.Used = smem.Total - smem.Free
	ret, err := exec.Command("dmidecode", "-t", "17").Output()
	if err != nil {
		return nil, err
	}
	smem.MemInfo = sysutils.ParseDMIMemInfo(strings.Split(string(ret), "\n"))
	return smem, nil
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

type SNIC struct {
	Inter   string
	Bridge  string
	Ip      string
	Network string
	WireId  string

	Bandwidth  int
	BridgeDev  hostbridge.IBridgeDriver
	dhcpServer *hostdhcp.SGuestDHCPServer
}

func (n *SNIC) EnableDHCPRelay() bool {
	v4Ip, err := netutils.NewIPV4Addr(n.Ip)
	if err != nil {
		log.Errorln(err)
		return false
	}
	if options.HostOptions.DhcpRelay != nil && netutils.IsExitAddress(v4Ip) {
		return true
	} else {
		return false
	}
}

func (n *SNIC) SetupDhcpRelay() {
	if n.EnableDHCPRelay() {
		n.dhcpServer.RelaySetup(n.Ip)
	}
}

func (n *SNIC) SetWireId(wire, wireId string, bandwidth int64) {
	n.Network = wire
	n.WireId = wireId
	n.Bandwidth = int(bandwidth)
}

func NewNIC(desc string) (*SNIC, error) {
	nic := new(SNIC)
	data := strings.Split(desc, "/")
	nic.Inter = data[0]
	nic.Bridge = data[1]
	if regutils.MatchIP4Addr(data[2]) {
		nic.Ip = data[2]
	} else {
		nic.Network = data[2]
	}
	nic.Bandwidth = 1000

	// 这是干啥呢 ？？？
	if len(nic.Ip) > 0 {
		var max, wait = 30, 0
		for wait < max {
			inf := netutils2.NewNetInterface(nic.Inter)
			if inf.Addr == nic.Ip {
				break
			}
			br := netutils2.NewNetInterface(nic.Bridge)
			if br.Addr == nic.Ip {
				break
			}
			time.Sleep(time.Second * 2)
			wait += 1
		}
	}

	var err error

	nic.BridgeDev, err = hostbridge.NewDriver(options.HostOptions.BridgeDriver,
		nic.Bridge, nic.Inter, nic.Ip)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	confirm, err := nic.BridgeDev.ConfirmToConfig(nic.BridgeDev.Exists(), nic.BridgeDev.Interfaces())
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	if !confirm {
		log.Infof("Not confirm to configuration")
		if err = nic.BridgeDev.Setup(); err != nil {
			log.Errorln(err)
			return nil, err
		}
		time.Sleep(time.Second * 1)
	} else {
		log.Infof("Confirm to configuration!!")
	}

	var dhcpRelay []string
	if nic.EnableDHCPRelay() {
		dhcpRelay = options.HostOptions.DhcpRelay
	}
	nic.dhcpServer = hostdhcp.NewGuestDHCPServer(nic.Bridge, dhcpRelay)
	nic.dhcpServer.Start()
	return nic, nil
}

type SSysInfo struct {
	*types.DMISystemInfo

	Nest           string `json:"nest,omitempty"`
	OsDistribution string `json:"os_distribution"`
	OsVersion      string `json:"os_version"`
	KernelVersion  string `json:"kernel_version"`
	QemuVersion    string `json:"qemu_version"`
	OvsVersion     string `json:"ovs_version"`

	StorageType string `json:"storage_type"`
}

func StartDetachStorages(hs []jsonutils.JSONObject) {
	for len(hs) > 0 {
		hostId, _ := hs[0].GetString("host_id")
		storageId, _ := hs[0].GetString("storage_id")
		_, err := modules.Hoststorages.Detach(
			hostutils.GetComputeSession(context.Background()),
			hostId, storageId)
		if err != nil {
			log.Errorf("Host %s detach storage %s failed: %s",
				hostId, storageId, err)
			time.Sleep(30 * time.Second)
		} else {
			hs = hs[1:]
		}
	}
}
