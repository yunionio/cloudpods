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

package hostinfo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostdhcp"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SCPUInfo struct {
	CpuCount        int
	cpuFreq         int64 // MHZ
	cpuFeatures     []string
	CpuArchitecture string

	cpuInfoProc *types.SCPUInfo
	cpuInfoDmi  *types.SDMICPUInfo
}

func DetectCpuInfo() (*SCPUInfo, error) {
	cpuinfo := new(SCPUInfo)
	cpuCount, _ := cpu.Counts(true)
	cpuinfo.CpuCount = cpuCount
	spec, err := cpuinfo.fetchCpuSpecs()
	if err != nil {
		return nil, err
	}
	var freq float64
	strCpuFreq, ok := spec["cpu_freq"]
	if ok {
		freq, err = strconv.ParseFloat(strCpuFreq, 64)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
	}

	cpuinfo.cpuFreq = int64(freq)
	log.Infof("cpuinfo freq %d", cpuinfo.cpuFreq)

	cpuinfo.cpuFeatures = strings.Split(spec["flags"], " ")

	// cpu.Percent(interval, false)
	ret, err := fileutils2.FileGetContents("/proc/cpuinfo")
	if err != nil {
		return nil, errors.Wrap(err, "get cpuinfo")
	}
	cpuinfo.cpuInfoProc, err = sysutils.ParseCPUInfo(strings.Split(ret, "\n"))
	if err != nil {
		return nil, errors.Wrap(err, "parse cpu info")
	}
	bret, err := procutils.NewCommand("dmidecode", "-t", "4").Output()
	if err != nil {
		log.Errorf("dmidecode -t 4 error: %s(%s)", err, string(bret))
		cpuinfo.cpuInfoDmi = &types.SDMICPUInfo{Nodes: 1}
	} else {
		cpuinfo.cpuInfoDmi = sysutils.ParseDMICPUInfo(strings.Split(string(bret), "\n"))
	}
	cpuArch, err := procutils.NewCommand("uname", "-m").Output()
	if err != nil {
		return nil, errors.Wrap(err, "get cpu architecture")
	}
	cpuinfo.CpuArchitecture = strings.TrimSpace(string(cpuArch))
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
	MemInfo *types.SDMIMemInfo
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
	ret, err := procutils.NewCommand("dmidecode", "-t", "17").Output()
	if err != nil {
		// ignore
		log.Errorf("dmidecode fail %s: %s", err, ret)
		smem.MemInfo = &types.SDMIMemInfo{}
	} else {
		smem.MemInfo = sysutils.ParseDMIMemInfo(strings.Split(string(ret), "\n"))
	}
	if smem.MemInfo.Total == 0 {
		// in case dmidecode is not work, use gopsutil
		smem.MemInfo.Total = smem.Total
	}
	return smem, nil
}

func (m *SMemory) GetHugepages() (sysutils.THugepages, error) {
	return sysutils.GetHugepages()
}

type SNIC struct {
	Inter   string
	Bridge  string
	Ip      string
	Network string
	WireId  string
	Mask    int

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
	if len(options.HostOptions.DhcpRelay) > 0 && !netutils.IsExitAddress(v4Ip) {
		return true
	} else {
		return false
	}
}

func (n *SNIC) SetupDhcpRelay() error {
	if n.EnableDHCPRelay() {
		if err := n.dhcpServer.RelaySetup(n.Ip); err != nil {
			return err
		}
	}
	return nil
}

func (n *SNIC) SetWireId(wire, wireId string, bandwidth int64) {
	n.Network = wire
	n.WireId = wireId
	n.Bandwidth = int(bandwidth)
}

func (n *SNIC) ExitCleanup() {
	n.BridgeDev.CleanupConfig()
	log.Infof("Stop DHCP Server")
	// TODO stop dhcp server
}

func NewNIC(desc string) (*SNIC, error) {
	nic := new(SNIC)
	data := strings.Split(desc, "/")
	if len(data) < 3 {
		return nil, fmt.Errorf("Parse nic conf %s failed, too short", desc)
	}
	nic.Inter = data[0]
	nic.Bridge = data[1]
	if regutils.MatchIP4Addr(data[2]) {
		nic.Ip = data[2]
	} else {
		nic.Network = data[2]
	}
	nic.Bandwidth = 1000

	log.Infof("IP %s/%s/%s", nic.Ip, nic.Bridge, nic.Inter)
	if len(nic.Ip) > 0 {
		// waiting for interface assign ip
		// in case nic bonding is too slow
		var max, wait = 30, 0
		for wait < max {
			inf := netutils2.NewNetInterfaceWithExpectIp(nic.Inter, nic.Ip)
			if inf.Addr == nic.Ip {
				mask, _ := inf.Mask.Size()
				if mask > 0 {
					nic.Mask = mask
				}
				break
			}
			br := netutils2.NewNetInterface(nic.Bridge)
			if br.Addr == nic.Ip {
				mask, _ := br.Mask.Size()
				if nic.Mask == 0 && mask > 0 {
					nic.Mask = mask
				}
				break
			}
			time.Sleep(time.Second * 2)
			wait += 1
		}
		if wait >= max {
			// if ip not found in inter or bridge
			return nil, fmt.Errorf("Ip %s is not configure on %s/%s ?", nic.Ip, nic.Bridge, nic.Inter)
		}
	}

	var err error

	nic.BridgeDev, err = hostbridge.NewDriver(options.HostOptions.BridgeDriver,
		nic.Bridge, nic.Inter, nic.Ip)
	if err != nil {
		return nil, errors.Wrapf(err, "hostbridge.NewDriver driver: %s, bridge: %s, interface: %s, ip: %s", options.HostOptions.BridgeDriver, nic.Bridge, nic.Inter, nic.Ip)
	}

	confirm, err := nic.BridgeDev.ConfirmToConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "nic.BridgeDev.ConfirmToConfig %#v", nic.BridgeDev)
	}
	if !confirm {
		log.Infof("Not confirm to configuration")
		if err = nic.BridgeDev.Setup(nic.BridgeDev); err != nil {
			return nil, errors.Wrapf(err, "nic.BridgeDev.Setup %v", nic.BridgeDev)
		}
		time.Sleep(time.Second * 1)
	} else {
		log.Infof("Confirm to configuration!!")
	}
	if err := nic.BridgeDev.PersistentConfig(); err != nil {
		return nil, errors.Wrapf(err, "nic.BridgeDev.PersistentConfig %v", nic.BridgeDev)
	}
	if isDHCP, err := nic.BridgeDev.DisableDHCPClient(); err != nil {
		return nil, errors.Wrap(err, "disable dhcp client")
	} else if isDHCP {
		Instance().SysWarning["dhcp"] = "dhcp client is enabled before host agent start, please disable it"
	}

	var dhcpRelay []string
	if nic.EnableDHCPRelay() {
		dhcpRelay = options.HostOptions.DhcpRelay
	}
	nic.dhcpServer, err = hostdhcp.NewGuestDHCPServer(nic.Bridge, options.HostOptions.DhcpServerPort, dhcpRelay)
	if err != nil {
		return nil, err
	}
	// dhcp server start after guest manager init
	return nic, nil
}

type SSysInfo struct {
	*types.SSystemInfo

	Nest           string `json:"nest,omitempty"`
	OsDistribution string `json:"os_distribution"`
	OsVersion      string `json:"os_version"`
	KernelVersion  string `json:"kernel_version"`
	QemuVersion    string `json:"qemu_version"`
	OvsVersion     string `json:"ovs_version"`
	KvmModule      string `json:"kvm_module"`
	CpuModelName   string `json:"cpu_model_name"`
	CpuMicrocode   string `json:"cpu_microcode"`

	StorageType string `json:"storage_type"`

	HugepagesOption string `json:"hugepages_option"`
	HugepageSizeKb  int    `json:"hugepage_size_kb"`

	Topology *hostapi.HostTopology `json:"topology"`
	CPUInfo  *hostapi.HostCPUInfo  `json:"cpu_info"`
}

func StartDetachStorages(hs []jsonutils.JSONObject) {
	for len(hs) > 0 {
		hostId, _ := hs[0].GetString("host_id")
		storageId, _ := hs[0].GetString("storage_id")
		_, err := modules.Hoststorages.Detach(
			hostutils.GetComputeSession(context.Background()),
			hostId, storageId, nil)
		if err != nil {
			log.Errorf("Host %s detach storage %s failed: %s",
				hostId, storageId, err)
			time.Sleep(30 * time.Second)
		} else {
			hs = hs[1:]
		}
	}
}

func IsRootPartition(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}

	path = strings.TrimSuffix(path, "/")
	pathSegs := strings.Split(path, "/")
	for len(pathSegs) > 1 {
		err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", path).Run()
		if err != nil {
			pathSegs = pathSegs[:len(pathSegs)-1]
			path = strings.Join(pathSegs, "/")
			continue
		} else {
			return false
		}
	}
	return true
}
