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
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	napi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/host_health"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/hardware"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/kubelet"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/k8s/tokens"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type SHostInfo struct {
	isRegistered     bool
	IsRegistered     chan struct{}
	registerCallback func()
	stopped          bool
	isLoged          bool

	saved  bool
	pinger *SHostPingTask

	Cpu     *SCPUInfo
	Mem     *SMemory
	sysinfo *SSysInfo

	kubeletConfig kubelet.KubeletConfig

	isInit           bool
	onHostDown       string
	reservedCpusInfo *api.HostReserveCpusInput

	IsolatedDeviceMan isolated_device.IsolatedDeviceManager

	MasterNic *netutils2.SNetInterface
	Nics      []*SNIC

	HostId         string
	Zone           string
	ZoneId         string
	Cloudregion    string
	CloudregionId  string
	ZoneManagerUri string
	Project_domain string
	Domain_id      string

	FullName   string
	SysError   map[string]string
	SysWarning map[string]string

	IoScheduler string
}

func (h *SHostInfo) GetIsolatedDeviceManager() isolated_device.IsolatedDeviceManager {
	return h.IsolatedDeviceMan
}

func (h *SHostInfo) GetBridgeDev(bridge string) hostbridge.IBridgeDriver {
	for _, n := range h.Nics {
		if bridge == n.Bridge {
			return n.BridgeDev
		}
	}
	if bridge == options.HostOptions.OvnIntegrationBridge || bridge == api.HostVpcBridge {
		drv, err := hostbridge.NewOVSBridgeDriverByName(bridge)
		if err != nil {
			log.Errorf("create ovn bridge driver: %v", err)
			return nil
		}
		return drv
	} else if bridge == api.HostTapBridge {
		drv, err := hostbridge.NewOVSBridgeDriverByName(options.HostOptions.TapBridgeName)
		if err != nil {
			log.Errorf("create ovn bridge driver: %v", err)
			return nil
		}
		return drv
	}
	return nil
}

func (h *SHostInfo) StartDHCPServer() {
	for _, nic := range h.Nics {
		nic.dhcpServer.Start(false)
	}
}

func (h *SHostInfo) GetHostId() string {
	return h.HostId
}

func (h *SHostInfo) GetZoneId() string {
	return h.ZoneId
}

/*func (h *SHostInfo) GetMediumType() string {
	if h.sysinfo != nil {
		return h.sysinfo.StorageType
	}
	return ""
}*/

func (h *SHostInfo) IsKvmSupport() bool {
	return sysutils.IsKvmSupport()
}

func (h *SHostInfo) IsNestedVirtualization() bool {
	return utils.IsInStringArray("hypervisor", h.Cpu.cpuFeatures)
}

func (h *SHostInfo) IsHugepagesEnabled() bool {
	return options.HostOptions.HugepagesOption == "native"
}

func (h *SHostInfo) HugepageSizeKb() int {
	return h.sysinfo.HugepageSizeKb
}

/* In this order init host service:
 * 1. prepare env, fix environment variable path
 * 2. detect hostinfo, fill host capability and custom host field
 * 3. prepare hostbridge, start openvswitch service
 * 4. parse host config, config ip address
 * 5. check is ovn support, setup ovn chassis
 */
func (h *SHostInfo) Init() error {
	if err := h.prepareEnv(); err != nil {
		return errors.Wrap(err, "Prepare environment")
	}

	log.Infof("Start detectHostInfo")
	if err := h.detectHostInfo(); err != nil {
		return errors.Wrap(err, "detectHostInfo")
	}

	if err := hostbridge.Prepare(options.HostOptions.BridgeDriver); err != nil {
		return errors.Wrapf(err, "Prepare host bridge %q", options.HostOptions.BridgeDriver)
	}

	log.Infof("Start parseConfig")
	if err := h.parseConfig(); err != nil {
		return errors.Wrap(err, "parseConfig")
	}
	if HasOvnSupport() {
		if err := h.setupOvnChassis(); err != nil {
			return errors.Wrap(err, "Setup OVN Chassis")
		}
	}

	return nil
}

func (h *SHostInfo) setupOvnChassis() error {
	opts := &options.HostOptions
	if opts.BridgeDriver != hostbridge.DRV_OPEN_VSWITCH {
		return nil
	}
	log.Infof("Start setting up ovn chassis")
	oh := NewOvnHelper(h)
	if err := oh.Init(); err != nil {
		return err
	}
	return nil
}

func (h *SHostInfo) generateLocalNetworkConfig() (string, error) {
	netIp, dev, err := netutils2.DefaultSrcIpDev()
	if err != nil {
		return "", errors.Wrap(err, "find default source address & device")
	}
	log.Infof("Find dev: %s ip: %s", dev, netIp)

	var bridgeName string
	// test if dev is bridge
	if err := procutils.NewCommand("ovs-vsctl", "br-exists", dev).Run(); err == nil {
		portStr, err := procutils.NewCommand("ovs-vsctl", "list-ports", dev).Output()
		if err != nil {
			return "", errors.Wrap(err, "list port")
		}
		ports := strings.Split(string(portStr), "\n")

		devs := []string{}
		for i := 0; i < len(ports); i++ {
			portName := strings.TrimSpace(ports[i])
			if len(portName) > 0 {
				lk, err := netlink.LinkByName(portName)
				if err != nil {
					log.Errorf("netlink.LinkByName %s failed %s", portName, err)
					continue
				} else {
					log.Infof("port %s link type %s", portName, lk.Type())
					if !utils.IsInStringArray(lk.Type(), []string{"veth", "tun"}) {
						devs = append(devs, portName)
					}
				}
			}
		}
		if len(devs) != 1 {
			return "", fmt.Errorf("list ports of br got %v", dev)
		}
		bridgeName = dev
		dev = devs[0]
	} else {
		log.Errorf("br-exists %s get error %s", dev, err)

		// test if dev is port of bridge
		output, err := procutils.NewCommand("ovs-vsctl", "port-to-br", dev).Output()
		if err != nil && !strings.Contains(string(output), "no port named") {
			return "", errors.Wrapf(err, "port to br failed %s", output)
		} else if err == nil {
			bridgeName = strings.TrimSpace(string(output))
		}
	}

	if len(bridgeName) == 0 {
		bridgeName = "br"
		index := 0
		for {
			if _, err := net.InterfaceByName(bridgeName + strconv.Itoa(index)); err != nil {
				bridgeName = bridgeName + strconv.Itoa(index)
				break
			}
			index += 1
		}
	}

	log.Infof("bridge name %s", bridgeName)
	return fmt.Sprintf("%s/%s/%s", dev, bridgeName, netIp), nil
}

func (h *SHostInfo) parseConfig() error {
	mem := h.GetMemory()
	if mem < 64 { // MB
		return fmt.Errorf("Not enough memory!")
	}
	if len(options.HostOptions.Networks) == 0 {
		netConf, err := h.generateLocalNetworkConfig()
		if err != nil {
			return errors.Wrap(err, "generateLocalNetworkConfig")
		}
		log.Infof("Generate network config %s", netConf)
		options.HostOptions.Networks = []string{netConf}
		if len(options.HostOptions.Config) > 0 {
			if err = fileutils2.FilePutContents(
				options.HostOptions.Config,
				jsonutils.Marshal(options.HostOptions).YAMLString(),
				false,
			); err != nil {
				log.Errorf("write config file failed %s", err)
			}
		}
	}
	for _, n := range options.HostOptions.Networks {
		nic, err := NewNIC(n)
		if err != nil {
			return errors.Wrapf(err, "NewNIC %s", n)
		}
		h.Nics = append(h.Nics, nic)
	}
	for i := 0; i < len(h.Nics); i++ {
		if err := h.Nics[i].SetupDhcpRelay(); err != nil {
			return errors.Wrapf(err, "SetupDhcpRelay %s/%s/%s", h.Nics[i].Inter, h.Nics[i].Bridge, h.Nics[i].Ip)
		}
	}
	if len(options.HostOptions.ListenInterface) > 0 {
		h.MasterNic = netutils2.NewNetInterface(options.HostOptions.ListenInterface)
		if len(h.MasterNic.Addr) == 0 {
			return fmt.Errorf("Listen interface %s master not have IP", options.HostOptions.ListenInterface)
		}
	} else {
		h.MasterNic = nil
	}

	h.IsolatedDeviceMan = isolated_device.NewManager(h)

	return nil
}

func (h *SHostInfo) prepareEnv() error {
	if err := h.fixPathEnv(); err != nil {
		return errors.Wrap(err, "Fix path environment")
	}
	if options.HostOptions.ReportInterval > 300 {
		return fmt.Errorf("Option report_interval must no longer than 5 min")
	}

	for _, dirPath := range []string{
		options.HostOptions.ServersPath,
		options.HostOptions.MemorySnapshotsPath,
		options.HostOptions.LocalBackupTempPath,
	} {
		output, err := procutils.NewCommand("mkdir", "-p", dirPath).Output()
		if err != nil {
			return errors.Wrapf(err, "failed to create path %s: %s", dirPath, output)
		}
	}

	_, err := procutils.NewCommand("ethtool", "-h").Output()
	if err != nil {
		return errors.Wrap(err, "Execute 'ethtool -h'")
	}

	supportedSchedulers, _ := fileutils2.GetAllBlkdevsIoSchedulers()
	// IoScheduler default to none scheduler
	ioParams := make(map[string]string, 0)
	switch options.HostOptions.BlockIoScheduler {
	case "deadline":
		if utils.IsInStringArray("mq-deadline", supportedSchedulers) {
			h.IoScheduler = "mq-deadline"
		} else if utils.IsInStringArray("deadline", supportedSchedulers) {
			h.IoScheduler = "deadline"
		} else {
			h.IoScheduler = "none"
		}
	case "cfq":
		if utils.IsInStringArray("bfq", supportedSchedulers) {
			h.IoScheduler = "bfq"
		} else if utils.IsInStringArray("cfq", supportedSchedulers) {
			h.IoScheduler = "cfq"
		} else {
			h.IoScheduler = "none"
		}
	default:
		if utils.IsInStringArray(options.HostOptions.BlockIoScheduler, supportedSchedulers) {
			h.IoScheduler = options.HostOptions.BlockIoScheduler
		} else {
			h.IoScheduler = "none"
		}
	}

	log.Infof("I/O Scheduler switch to %s", h.IoScheduler)

	ioParams["queue/scheduler"] = h.IoScheduler
	switch h.IoScheduler {
	case "cfq":
		ioParams["queue/iosched/group_isolation"] = "1"
		ioParams["queue/iosched/slice_idle"] = "0"
		ioParams["queue/iosched/group_idle"] = "0"
		ioParams["queue/iosched/quantum"] = "32"
	}
	fileutils2.ChangeAllBlkdevsParams(ioParams)
	_, err = procutils.NewRemoteCommandAsFarAsPossible("modprobe", "tun").Output()
	if err != nil {
		return errors.Wrap(err, "Failed to activate tun/tap device")
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", "vhost_net").Output()
	if err != nil {
		log.Warningf("modprobe vhost_net error: %s", output)
	}
	if !options.HostOptions.DisableSetCgroup {
		if !cgrouputils.Init(h.IoScheduler) {
			return fmt.Errorf("Cannot initialize control group subsystem")
		}
	}

	// err = h.resetIptables()
	// if err != nil {
	//  return err
	// }

	if options.HostOptions.EnableKsm && options.HostOptions.HugepagesOption == "disable" {
		h.EnableKsm(900)
	} else {
		h.DisableKsm()
	}

	switch options.HostOptions.HugepagesOption {
	case "disable":
		h.DisableHugepages()
	case "native":
		err := h.EnableNativeHugepages(0)
		if err != nil {
			return errors.Wrap(err, "EnableNativeHugepages")
		}
		hp, err := h.Mem.GetHugepages()
		if err != nil {
			return errors.Wrap(err, "Mem.GetHugepages")
		}
		szlist := hp.PageSizes()
		if len(szlist) == 0 {
			return errors.Error("invalid hugepages total size")
		}
		if len(szlist) > 1 {
			return errors.Error("cannot support more than 1 type of hugepage size")
		}
		h.sysinfo.HugepageSizeKb = szlist[0]
	case "transparent":
		h.EnableTransparentHugepages()
	default:
		return fmt.Errorf("Invalid hugepages option")
	}

	h.sysinfo.HugepagesOption = options.HostOptions.HugepagesOption

	h.PreventArpFlux()
	h.tuneSystem()
	return nil
}

func (h *SHostInfo) detectHostInfo() error {
	output, err := procutils.NewCommand("dmidecode", "-t", "1").Output()
	if err != nil {
		log.Errorf("dmidecode -t 1 error %s(%s)", err, string(output))
		h.sysinfo.SSystemInfo = &types.SSystemInfo{}
	} else {
		h.sysinfo.SSystemInfo, err = sysutils.ParseDMISysinfo(strings.Split(string(output), "\n"))
		if err != nil {
			return err
		}
	}

	h.detectKvmModuleSupport()
	h.detectNestSupport()

	if err := h.detectSyssoftwareInfo(); err != nil {
		return err
	}

	h.detectStorageSystem()

	topoInfo, err := hardware.GetTopology()
	if err != nil {
		return errors.Wrap(err, "Get hardware topology")
	}
	cpuInfo, err := hardware.GetCPU()
	if err != nil {
		return errors.Wrap(err, "Get CPU info")
	}
	h.sysinfo.Topology = topoInfo
	h.sysinfo.CPUInfo = cpuInfo

	system_service.Init()
	if options.HostOptions.CheckSystemServices {
		if err := h.checkSystemServices(); err != nil {
			return err
		}
	}
	return nil
}

func (h *SHostInfo) checkSystemServices() error {
	funcEn := func(srv string, srvinst system_service.ISystemService) {
		if !srvinst.IsInstalled() {
			log.Warningf("Service %s not installed", srv)
		} else if !srvinst.IsActive() {
			srvinst.Start(false)
		}
	}

	if options.HostOptions.ManageNtpConfiguration {
		for _, srv := range []string{"ntpd"} {
			srvinst := system_service.GetService(srv)
			funcEn(srv, srvinst)
		}
	}

	svcs := os.Getenv("HOST_SYSTEM_SERVICES_OFF")
	for _, srv := range []string{"host_sdnagent", "host-deployer", "telegraf"} {
		srvinst := system_service.GetService(srv)
		if strings.Contains(svcs, srv) {
			if srvinst.IsActive() || srvinst.IsEnabled() {
				srvinst.Stop(true)
			}
		} else {
			funcEn(srv, srvinst)
		}
	}

	return nil
}

func (h *SHostInfo) detectStorageSystem() {
	stype, _ := sysutils.DetectStorageType()
	switch stype {
	case "hdd":
		stype = api.DISK_TYPE_ROTATE
	case "ssd":
		stype = api.DISK_TYPE_SSD
	case "hybird":
		stype = api.DISK_TYPE_HYBRID
	default:
		stype = ""
	}
	h.sysinfo.StorageType = stype
}

func (h *SHostInfo) fixPathEnv() error {
	var paths = []string{
		"/usr/bin", // usr bin at first for host container deploy
		"/usr/sbin",
		"/usr/local/sbin",
		"/usr/local/bin",
		"/sbin",
		"/bin",
	}
	return os.Setenv("PATH", strings.Join(paths, ":"))
}

func (h *SHostInfo) DisableHugepages() {
	kv := map[string]string{
		"/proc/sys/vm/nr_hugepages":                   "0",
		"/sys/kernel/mm/transparent_hugepage/enabled": "never",
		"/sys/kernel/mm/transparent_hugepage/defrag":  "never",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
}

func (h *SHostInfo) EnableTransparentHugepages() {
	kv := map[string]string{
		"/proc/sys/vm/nr_hugepages":                   "0",
		"/sys/kernel/mm/transparent_hugepage/enabled": "always",
		"/sys/kernel/mm/transparent_hugepage/defrag":  "always",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
}

func (h *SHostInfo) GetMemory() int {
	return h.Mem.Total
}

/* func (h *SHostInfo) getCurrentHugepageNr() (int64, error) {
	nrStr, err := fileutils2.FileGetContents("/proc/sys/vm/nr_hugepages")
	if err != nil {
		return 0, errors.Wrap(err, "file get content nr hugepages")
	}
	nr, err := strconv.Atoi(strings.TrimSpace(nrStr))
	if err != nil {
		return 0, errors.Wrap(err, "nr str atoi")
	}
	return int64(nr), nil
} */

func (h *SHostInfo) EnableNativeHugepages(reservedMb int) error {
	kv := map[string]string{
		"/sys/kernel/mm/transparent_hugepage/enabled": "never",
		"/sys/kernel/mm/transparent_hugepage/defrag":  "never",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
	// check reserved memory
	hp, err := h.Mem.GetHugepages()
	if err != nil {
		return err
	}
	pgList := hp.PageSizes()
	if len(pgList) == 0 {
		// not initialized yet, manually setup, usage page_size 2MB
	} else if len(pgList) == 1 {
		// already setup, depends on the PageSize
		if pgList[0] == 2048 {
		} else {
			// readonly, cannot adjust any more
			return nil
		}
	} else {
		return errors.Error("cannot support more than 1 type of hugepage sizes")
	}
	mem := h.GetMemory()
	if reservedMb > 0 {
		mem -= reservedMb
	} else {
		mem -= h.getReservedMemMb()
	}
	desiredSz := 2 // ONLY 2MB Hugepage Supported
	desiredNr := mem / desiredSz
	if desiredNr*desiredSz < mem {
		desiredNr += 1
	}
	log.Infof("Hugepage %dGB(%dMB) available Mem %dMB, to reserve %d hugepages with size %d", hp.BytesMb()/1024, hp.BytesMb(), mem, desiredNr, desiredSz)
	// not setup hugepage yet, or reserved too many hugepages
	err = timeutils2.CommandWithTimeout(1, "sh", "-c",
		fmt.Sprintf("echo %d > /sys/kernel/mm/hugepages/hugepages-%dkB/nr_hugepages", desiredNr, desiredSz*1024)).Run()
	if err != nil {
		return err
	}
	return nil
}

func (h *SHostInfo) EnableKsm(sleepSec int) {
	sysutils.SetSysConfig("/sys/kernel/mm/ksm/run", "1")
	sysutils.SetSysConfig("/sys/kernel/mm/ksm/sleep_millisecs",
		fmt.Sprintf("%d", sleepSec*1000))
}

func (h *SHostInfo) DisableKsm() {
	sysutils.SetSysConfig("/sys/kernel/mm/ksm/run", "0")
}

func (h *SHostInfo) PreventArpFlux() {
	sysutils.SetSysConfig("/proc/sys/net/ipv4/conf/all/arp_filter", "1")
}

// Any system wide optimizations
// set swappiness=0 to avoid swap
// set vfs_cache_pressure=300 to avoid stale pagecache
func (h *SHostInfo) tuneSystem() {
	minMemMb := h.getKubeReservedMemMb()
	if minMemMb < 100 {
		minMemMb = 100
	}
	minMemKB := fmt.Sprintf("%d", 2*minMemMb*1024)
	kv := map[string]string{
		"/proc/sys/vm/swappiness":                        "0",
		"/proc/sys/vm/vfs_cache_pressure":                "350",
		"/proc/sys/vm/min_free_kbytes":                   minMemKB,
		"/proc/sys/net/ipv4/tcp_mtu_probing":             "2",
		"/proc/sys/net/ipv4/neigh/default/gc_thresh1":    "1024",
		"/proc/sys/net/ipv4/neigh/default/gc_thresh2":    "4096",
		"/proc/sys/net/ipv4/neigh/default/gc_thresh3":    "8192",
		"/sys/module/kvm/parameters/ignore_msrs":         "1",
		"/sys/module/kvm/parameters/report_ignored_msrs": "0",

		"/proc/sys/net/netfilter/nf_conntrack_tcp_be_liberal": "1",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
}

func (h *SHostInfo) resetIptables() error {
	for _, tbl := range []string{"filter", "nat", "mangle"} {
		output, err := procutils.NewCommand("iptables", "-t", tbl, "-F").Output()
		if err != nil {
			return errors.Wrapf(err, "fail to clean NAT iptables: %s", output)
		}
	}
	return nil
}

func (h *SHostInfo) initCgroup() error {
	reservedCpus := cpuset.NewCPUSet()
	if h.reservedCpusInfo != nil {
		var err error
		reservedCpus, err = cpuset.Parse(h.reservedCpusInfo.Cpus)
		if err != nil {
			return errors.Wrap(err, "failed parse reserved cpus")
		}
	}

	hostCpusetBuilder := cpuset.NewBuilder()
	for i := 0; i < h.Cpu.CpuCount; i++ {
		if reservedCpus.Contains(i) {
			continue
		}
		hostCpusetBuilder.Add(i)
	}
	hostCpuset := hostCpusetBuilder.Result()
	hostCpusetStr := hostCpuset.String()
	// init host cpuset root group
	if !cgrouputils.NewCGroupCPUSetTask("", hostconsts.HOST_CGROUP, 0, hostCpusetStr).Configure() {
		return fmt.Errorf("failed init host root cpuset")
	}
	// init host cpu root group
	cgrouputils.CgroupSet("", hostconsts.HOST_CGROUP, hostCpuset.Size()*1024)
	// init host blkio root group
	cgrouputils.CgroupIoHardlimitSet("", hostconsts.HOST_CGROUP, 0, nil, "")

	if h.reservedCpusInfo != nil {
		reservedCpusTask := cgrouputils.NewCGroupCPUSetTask("", hostconsts.HOST_RESERVED_CPUSET, 0, h.reservedCpusInfo.Cpus)
		if !reservedCpusTask.Configure() {
			return fmt.Errorf("failed init host reserved cpuset %s", h.reservedCpusInfo.Cpus)
		}
		if h.reservedCpusInfo.Mems != "" &&
			!reservedCpusTask.CustomConfig(cgrouputils.CPUSET_MEMS, h.reservedCpusInfo.Mems) {
			return fmt.Errorf("failed init host reserved cpuset mems %s", h.reservedCpusInfo.Mems)
		}
		if h.reservedCpusInfo.DisableSchedLoadBalance != nil &&
			*h.reservedCpusInfo.DisableSchedLoadBalance &&
			!reservedCpusTask.CustomConfig(cgrouputils.CPUSET_SCHED_LOAD_BALANCE, "0") {
			return fmt.Errorf("failed init host reserved cpuset sched load balance")
		}
	}
	return nil
}

func (h *SHostInfo) detectKvmModuleSupport() string {
	h.sysinfo.KvmModule = sysutils.GetKVMModuleSupport()
	return h.sysinfo.KvmModule
}

func (h *SHostInfo) detectNestSupport() {
	if sysutils.IsNestEnabled() {
		h.sysinfo.Nest = "enabled"
	} else {
		h.sysinfo.Nest = "disabled"
	}
}

func (h *SHostInfo) detectOsDist() {
	files, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", "ls /etc/*elease").Output()
	if err != nil {
		log.Errorln(err)
		return
	}
	re := regexp.MustCompile(`(.+) release ([\d.]+)[^(]*(?:\((.+)\))?`)
	for _, file := range strings.Split(string(files), "\n") {
		content, err := procutils.NewRemoteCommandAsFarAsPossible("cat", file).Output()
		if err != nil {
			log.Errorln(err)
			continue
		}
		m := re.FindStringSubmatch(string(content))
		if len(m) == 4 {
			h.sysinfo.OsDistribution = m[1]
			h.sysinfo.OsVersion = m[2]
			break
		}
	}
	log.Infof("DetectOsDist %s %s", h.sysinfo.OsDistribution, h.sysinfo.OsVersion)
	if len(h.sysinfo.OsDistribution) == 0 {
		log.Errorln("Failed to detect distribution info")
		content, err := procutils.NewRemoteCommandAsFarAsPossible("cat", "/etc/os-release").Output()
		if err != nil {
			log.Errorln(err)
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "ID=") {
				h.sysinfo.OsDistribution = line[3:]
				continue
			}
			if strings.HasPrefix(line, "VERSION=") {
				h.sysinfo.OsVersion = strings.Trim(line[8:], "\"")
				continue
			}
		}
	}
	if utils.IsInStringArray(strings.ToLower(h.sysinfo.OsDistribution), []string{"uos", "debian", "ubuntu"}) {
		if err := procutils.NewRemoteCommandAsFarAsPossible("systemctl", "cat", "--", "openvswitch").Run(); err != nil {
			log.Warningf("system_service.SetOpenvswitchName to openvswitch-switch")
			system_service.SetOpenvswitchName("openvswitch-switch")
		}
	}
}

func (h *SHostInfo) detectKernelVersion() {
	out, err := procutils.NewCommand("uname", "-r").Output()
	if err != nil {
		log.Errorf("detectKernelVersion error: %v", err)
	}
	h.sysinfo.KernelVersion = strings.TrimSpace(string(out))
}

func (h *SHostInfo) detectSyssoftwareInfo() error {
	h.detectOsDist()
	h.detectKernelVersion()
	if err := h.detectQemuVersion(); err != nil {
		h.SysError["qemu"] = err.Error()
	}
	h.detectOvsVersion()
	if err := h.detectOvsKOVersion(); err != nil {
		h.SysError["openvswitch"] = err.Error()
	}
	return nil
}

func (h *SHostInfo) detectQemuVersion() error {
	if len(qemutils.GetQemu("")) == 0 {
		return fmt.Errorf("Qemu not installed")
	}

	out, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemu(""), "-version").Output()
	if err != nil {
		return errors.Errorf("exec qemu version failed %s", out)
	}

	cmd := qemutils.GetQemu(options.HostOptions.DefaultQemuVersion)
	version, err := procutils.NewRemoteCommandAsFarAsPossible(cmd, "--version").Output()
	if err != nil {
		log.Errorln(err)
		return err
	} else {
		versions := strings.Split(string(version), "\n")
		parts := strings.Split(versions[0], " ")
		v := parts[len(parts)-1]
		if len(v) > 0 {
			log.Infof("Detect qemu version is %s", v)
			h.sysinfo.QemuVersion = v
		} else {
			return fmt.Errorf("Failed to detect qemu version")
		}
	}
	return nil
}

func (h *SHostInfo) detectOvsVersion() {
	version, err := procutils.NewCommand("ovs-vsctl", "--version").Output()
	if err != nil {
		log.Errorln(err)
	} else {
		versions := strings.Split(string(version), "\n")
		parts := strings.Split(versions[0], " ")
		v := parts[len(parts)-1]
		if len(v) > 0 {
			log.Infof("Detect OVS version is %s", v)
			h.sysinfo.OvsVersion = v
		} else {
			log.Errorln("Failed to detect ovs version")
		}
	}
}

func (h *SHostInfo) detectOvsKOVersion() error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible("modinfo", "openvswitch").Output()
	if err != nil {
		return errors.Errorf("modinfo openvswitch failed %s", output)
	}
	lines := strings.Split(string(output), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "version:") || strings.HasPrefix(line, "vermagic") {
			log.Infof("kernel module openvswitch %s", line)
			return nil
		}
	}
	return errors.Errorf("kernel module openvswitch paramters version not found, is kernel version correct ??")
}

func (h *SHostInfo) GetMasterNicIpAndMask() (string, int) {
	log.Errorf("MasterNic %#v", h.MasterNic)
	if h.MasterNic != nil {
		mask, _ := h.MasterNic.Mask.Size()
		return h.MasterNic.Addr, mask
	}
	for _, n := range h.Nics {
		if len(n.Ip) > 0 {
			return n.Ip, n.Mask
		}
	}
	return "", 0
}

func (h *SHostInfo) GetMasterIp() string {
	if h.MasterNic != nil {
		return h.MasterNic.Addr
	}
	for _, n := range h.Nics {
		if len(n.Ip) > 0 {
			return n.Ip
		}
	}
	return ""
}

func (h *SHostInfo) GetMasterMac() string {
	return h.getMasterMacWithRefresh(false)
}

func (h *SHostInfo) getMasterMacWithRefresh(refresh bool) string {
	if h.MasterNic != nil {
		if refresh {
			h.MasterNic.FetchConfig()
		}
		return h.MasterNic.Mac
	}
	for _, n := range h.Nics {
		if len(n.Ip) > 0 {
			if refresh {
				n.BridgeDev.FetchConfig()
			}
			return n.BridgeDev.GetMac()
		}
	}
	return ""
}

func (h *SHostInfo) GetMatchNic(bridge, iface, mac string) *SNIC {
	for _, nic := range h.Nics {
		if nic.BridgeDev.GetMac() == mac ||
			(nic.Bridge == bridge && nic.Inter == iface) {
			return nic
		}
	}
	return nil
}

func (h *SHostInfo) StartRegister(delay int, callback func()) {
	if callback != nil {
		h.registerCallback = callback
	}

	timeutils2.AddTimeout(time.Duration(delay)*time.Second, h.register)
}

func (h *SHostInfo) register() {
	if !h.isRegistered {
		h.fetchAccessNetworkInfo()
	}
}

func (h *SHostInfo) onFail(reason interface{}) {
	if len(h.HostId) > 0 && !h.isLoged {
		logclient.AddSimpleActionLog(h, logclient.ACT_ONLINE, reason, hostutils.GetComputeSession(context.Background()).GetToken(), false)
		data := jsonutils.NewDict()
		data.Add(jsonutils.NewString(h.GetName()), "name")
		data.Add(jsonutils.NewString(fmt.Sprintf("register failed: %v", reason)), "message")
		notifyclient.SystemExceptionNotify(context.TODO(), napi.ActionSystemException, napi.TOPIC_RESOURCE_HOST, data)
		h.isLoged = true
	}
	log.Errorf("register failed: %s", reason)
	h.StartRegister(30, nil)
	panic("register failed, try 30 seconds later...")
}

// try to create network on region.
func (h *SHostInfo) tryCreateNetworkOnWire() {
	masterIp, mask := h.GetMasterNicIpAndMask()
	log.Debugf("Get master ip %s and mask %d", masterIp, mask)
	if len(masterIp) == 0 || mask == 0 {
		h.onFail(fmt.Sprintf("master ip %s mask %d", masterIp, mask))
		return
	}
	params := jsonutils.NewDict()
	params.Set("ip", jsonutils.NewString(masterIp))
	params.Set("mask", jsonutils.NewInt(int64(mask)))
	params.Set("is_classic", jsonutils.JSONTrue)
	params.Set("server_type", jsonutils.NewString(api.NETWORK_TYPE_BAREMETAL))
	params.Set("is_on_premise", jsonutils.JSONTrue)
	ret, err := modules.Networks.PerformClassAction(
		hostutils.GetComputeSession(context.Background()),
		"try-create-network", params)
	if err != nil {
		h.onFail(fmt.Sprintf("try create network: %v", err))
		return
	}
	if !jsonutils.QueryBoolean(ret, "find_matched", false) {
		h.onFail("try create network: find_matched == false")
		return
	}
	wireId, err := ret.GetString("wire_id")
	if err != nil {
		h.onFail(fmt.Sprintf("try create network: get wire_id: %v", err))
		return
	}
	h.onGetWireId(wireId)
}

func (h *SHostInfo) fetchAccessNetworkInfo() {
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		h.onFail("master ip not found")
		return
	}
	log.Debugf("Master ip %s to fetch wire", masterIp)
	params := jsonutils.NewDict()
	params.Set("ip", jsonutils.NewString(masterIp))
	params.Set("is_classic", jsonutils.JSONTrue)
	params.Set("provider", jsonutils.NewString(api.CLOUD_PROVIDER_ONECLOUD))
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("limit", jsonutils.NewInt(0))
	// use default vpc
	params.Set("vpc", jsonutils.NewString(api.DEFAULT_VPC_ID))

	res, err := modules.Networks.List(h.GetSession(), params)
	if err != nil {
		h.onFail(err)
		return
	}
	if len(res.Data) == 0 {
		h.tryCreateNetworkOnWire()
	} else if len(res.Data) == 1 {
		wireId, _ := res.Data[0].GetString("wire_id")
		h.onGetWireId(wireId)
	} else {
		h.onFail("Fail to get network info: no networks")
		return
	}
}

func (h *SHostInfo) onGetWireId(wireId string) {
	wire, err := hostutils.GetWireInfo(context.Background(), wireId)
	if err != nil {
		h.onFail(err)
		return
	}
	h.ZoneId, err = wire.GetString("zone_id")
	if err != nil {
		h.onFail(err)
		return
	}
	h.getZoneInfo(h.ZoneId)
}

func (h *SHostInfo) GetSession() *mcclient.ClientSession {
	return hostutils.GetComputeSession(context.Background())
}

func (h *SHostInfo) getZoneInfo(zoneId string) {
	log.Debugf("Start GetZoneInfo %s", zoneId)
	var params = jsonutils.NewDict()
	params.Set("provider", jsonutils.NewString(api.CLOUD_PROVIDER_ONECLOUD))
	res, err := modules.Zones.Get(h.GetSession(), zoneId, params)
	if err != nil {
		h.onFail(err)
		return
	}
	zone := api.ZoneDetails{}
	jsonutils.Update(&zone, res)
	h.Zone = zone.Name
	h.ZoneId = zone.Id
	h.Cloudregion = zone.Cloudregion
	h.CloudregionId = zone.CloudregionId
	h.ZoneManagerUri = zone.ManagerUri
	if len(h.Zone) == 0 {
		h.onFail(fmt.Errorf("failed to found zone with id %s", zoneId))
		return
	}
	consts.SetZone(zone.Name)

	h.getHostInfo(h.ZoneId)
}

func (h *SHostInfo) getHostInfo(zoneId string) {
	masterMac := h.getMasterMacWithRefresh(true)
	if len(masterMac) == 0 {
		h.onFail("master mac not found")
		return
	}
	params := jsonutils.NewDict()
	params.Set("any_mac", jsonutils.NewString(masterMac))
	params.Set("provider", jsonutils.NewString(api.CLOUD_PROVIDER_ONECLOUD))
	params.Set("details", jsonutils.JSONTrue)
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.Hosts.List(h.GetSession(), params)
	if err != nil {
		h.onFail(err)
		return
	}
	hosts := []api.HostDetails{}
	jsonutils.Update(&hosts, res.Data)
	if len(hosts) == 0 {
		h.updateHostRecord("")
		return
	}
	if len(hosts) > 1 {
		for i := range hosts {
			h.HostId = hosts[i].Id
			logclient.AddSimpleActionLog(h, logclient.ACT_ONLINE, fmt.Errorf("duplicate host with %s", params), hostutils.GetComputeSession(context.Background()).GetToken(), false)
		}
		return
	}
	h.Domain_id = hosts[0].DomainId
	h.HostId = hosts[0].Id
	h.Project_domain = strings.ReplaceAll(hosts[0].ProjectDomain, " ", "+")
	// 上次未能正常offline, 补充一次健康日志
	if hosts[0].HostStatus == api.HOST_ONLINE {
		reason := fmt.Sprintf("The host status is online when it staring. Maybe the control center was down earlier")
		logclient.AddSimpleActionLog(h, logclient.ACT_HEALTH_CHECK, map[string]string{"reason": reason}, hostutils.GetComputeSession(context.Background()).GetToken(), false)
		data := jsonutils.NewDict()
		data.Add(jsonutils.NewString(h.GetName()), "name")
		data.Add(jsonutils.NewString(reason), "message")
		notifyclient.SystemExceptionNotify(context.TODO(), napi.ActionSystemException, napi.TOPIC_RESOURCE_HOST, data)
	}
	h.updateHostRecord(hosts[0].Id)
}

func (h *SHostInfo) UpdateSyncInfo(hostId string, body jsonutils.JSONObject) (interface{}, error) {
	if h.GetHostId() != hostId {
		return nil, nil
	}
	descObj, err := body.Get("desc")
	if err != nil {
		return nil, err
	}
	domainId, _ := descObj.GetString("domain_id")
	projectDomain, _ := descObj.GetString("project_domain")
	if len(domainId) != 0 {
		h.Domain_id = domainId
	}
	if len(projectDomain) != 0 {
		h.Project_domain = strings.ReplaceAll(projectDomain, " ", "+")
	}
	return nil, nil
}

func (h *SHostInfo) ProbeSyncIsolatedDevices(hostId string, body jsonutils.JSONObject) (interface{}, error) {
	if h.GetHostId() != hostId {
		return nil, nil
	}
	return h.probeSyncIsolatedDevices()
}

func (h *SHostInfo) setHostname(name string) {
	h.FullName = name
	err := sysutils.SetHostname(name)
	if err != nil {
		log.Errorf("Fail to set system hostname: %s", err)
	}
}

func (h *SHostInfo) fetchHostname() string {
	if len(options.HostOptions.Hostname) > 0 {
		return options.HostOptions.Hostname
	} else {
		hn, err := os.Hostname()
		if err != nil {
			log.Fatalf("fail to get hostname %s", err)
			return ""
		}
		dotIdx := strings.IndexByte(hn, '.')
		if dotIdx >= 0 {
			hn = hn[:dotIdx]
		}
		hn = strings.ToLower(hn)
		if len(hn) == 0 {
			hn = "host"
		}
		masterIp := h.GetMasterIp()
		return hn + "-" + strings.Replace(masterIp, ".", "-", -1)
	}
}

func (h *SHostInfo) getSysInfo() *SSysInfo {
	return h.sysinfo
}

func (h *SHostInfo) updateHostRecord(hostId string) {
	if len(hostId) == 0 {
		h.isInit = true
	}
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		h.onFail("master ip is none")
		return
	}

	input := api.HostCreateInput{}
	if len(hostId) == 0 {
		input.GenerateName = h.fetchHostname()
	}
	input.AccessIp = masterIp
	input.AccessMac = h.GetMasterMac()
	var schema = "http"
	if options.HostOptions.EnableSsl {
		schema = "https"
	}
	input.ManagerUri = fmt.Sprintf("%s://%s:%d", schema, masterIp, options.HostOptions.Port)
	input.CpuCount = &h.Cpu.cpuInfoProc.Count
	nodeCount := int8(h.Cpu.cpuInfoDmi.Nodes)
	if sysutils.IsHypervisor() {
		nodeCount = 1
	}
	input.NodeCount = &nodeCount
	input.CpuDesc = h.Cpu.cpuInfoProc.Model
	input.CpuMicrocode = h.Cpu.cpuInfoProc.Microcode
	input.CpuArchitecture = h.Cpu.CpuArchitecture

	if h.Cpu.cpuInfoProc.Freq > 0 {
		input.CpuMhz = &h.Cpu.cpuInfoProc.Freq
	}
	input.CpuCache = fmt.Sprintf("%d", h.Cpu.cpuInfoProc.Cache)
	input.MemSize = fmt.Sprintf("%d", h.GetMemory())
	if len(hostId) == 0 {
		// first time create
		input.MemReserved = fmt.Sprintf("%d", h.getReservedMemMb())
	}
	input.StorageDriver = api.DISK_DRIVER_LINUX
	input.StorageType = h.sysinfo.StorageType
	storageSize := storageman.GetManager().GetTotalCapacity()
	input.StorageSize = &storageSize

	// TODO optimize content data struct
	input.SysInfo = jsonutils.Marshal(h.getSysInfo())
	input.SN = h.sysinfo.SN
	input.HostType = options.HostOptions.HostType
	if len(options.HostOptions.Rack) > 0 {
		input.Rack = options.HostOptions.Rack
	}
	if len(options.HostOptions.Slots) > 0 {
		input.Slots = options.HostOptions.Slots
	}
	meta, _ := jsonutils.Marshal(h.getSysInfo()).GetMap()
	input.Metadata = map[string]string{}
	for k, v := range meta {
		val, _ := v.GetString()
		input.Metadata[k] = val
	}
	input.Version = version.GetShortString()
	input.OvnVersion = MustGetOvnVersion()

	var (
		res jsonutils.JSONObject
		err error
	)
	if !h.isInit {
		res, err = modules.Hosts.Update(h.GetSession(), hostId, jsonutils.Marshal(input))
	} else {
		res, err = modules.Hosts.CreateInContext(h.GetSession(), jsonutils.Marshal(input), &modules.Zones, h.ZoneId)
	}
	if err != nil {
		h.onFail(errors.Wrapf(err, "host create or update with %s", jsonutils.Marshal(input)))
		return
	}
	h.onUpdateHostInfoSucc(res)
}

func (h *SHostInfo) updateHostMetadata(hostname string) error {
	onK8s, _ := tokens.IsInsideKubernetesCluster()
	meta := api.HostRegisterMetadata{
		OnKubernetes: onK8s,
		Hostname:     hostname,
	}
	if len(h.SysError) > 0 {
		meta.SysError = jsonutils.Marshal(h.SysError).String()
	}
	if len(h.SysWarning) > 0 {
		meta.SysWarn = jsonutils.Marshal(h.SysWarning).String()
	}
	meta.RootPartitionTotalCapacityMB = int64(storageman.GetRootPartTotalCapacity())
	meta.RootPartitionUsedCapacityMB = int64(storageman.GetRootPartUsedCapacity())
	data := meta.JSON(meta)
	_, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, data)
	return err
}

// func (h *SHostInfo) SyncRootPartitionUsedCapacity() error {
//	data := jsonutils.NewDict()
//	data.Set("root_partition_used_capacity_mb", jsonutils.NewInt(int64(storageman.GetRootPartUsedCapacity())))
//	_, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, data)
//	return err
// }

func (h *SHostInfo) onUpdateHostInfoSucc(hostbody jsonutils.JSONObject) {
	h.HostId, _ = hostbody.GetString("id")
	hostname, _ := hostbody.GetString("name")
	if err := h.updateHostMetadata(hostname); err != nil {
		h.onFail(err)
		return
	}
	if jsonutils.QueryBoolean(hostbody, "auto_migrate_on_host_down", false) {
		h.onHostDown = hostconsts.SHUTDOWN_SERVERS
	}
	log.Infof("on host down %s", h.onHostDown)
	host_health.SetOnHostDown(h.onHostDown)

	// fetch host reserved cpus info
	reservedCpusStr, _ := hostbody.GetString("metadata", api.HOSTMETA_RESERVED_CPUS_INFO)
	if reservedCpusStr != "" {
		reservedCpusJson, err := jsonutils.ParseString(reservedCpusStr)
		if err != nil {
			h.onFail(fmt.Sprintf("parse reserved cpus info failed %s", err))
			return
		}
		reservedCpusInfo := api.HostReserveCpusInput{}
		err = reservedCpusJson.Unmarshal(&reservedCpusInfo)
		if err != nil {
			h.onFail(fmt.Sprintf("unmarshal host reserved cpus info failed %s", err))
			return
		}
		h.reservedCpusInfo = &reservedCpusInfo
	}
	h.initCgroup()

	memReservedMb, _ := hostbody.Int("mem_reserved")
	if options.HostOptions.HugepagesOption == "native" && memReservedMb > int64(h.getReservedMemMb()) {
		err := h.EnableNativeHugepages(int(memReservedMb))
		if err != nil {
			h.onFail(err)
			return
		}
	}

	reserved := h.getReportedReservedMemMb()
	if reserved != int(memReservedMb) {
		h.updateHostReservedMem(reserved)
		return
	}
	h.getNetworkInfo()
}

func (h *SHostInfo) updateHostReservedMem(reserved int) {
	content := jsonutils.NewDict()
	content.Set("mem_reserved", jsonutils.NewInt(int64(reserved)))
	res, err := modules.Hosts.Update(h.GetSession(), h.HostId, content)
	if err != nil {
		h.onFail(err)
		return
	} else {
		h.onUpdateHostInfoSucc(res)
	}
}

func (h *SHostInfo) getKubeReservedMemMb() int {
	// reserved for Kubelet
	if h.kubeletConfig != nil {
		memThreshold := h.kubeletConfig.GetEvictionConfig().GetHard().GetMemoryAvailable()
		memBytes, _ := memThreshold.Value.Quantity.AsInt64()
		memMb := int(math.Ceil(float64(memBytes) / 1024 / 1024))
		log.Infof("Kubelet memory threshold subtracted: %dMB", memMb)
		return memMb
	}
	return 0
}

func (h *SHostInfo) getOSReservedMemMb() int {
	// reserved memory for OS
	reserved := h.Mem.MemInfo.Total / 10
	if reserved > options.HostOptions.MaxReservedMemory {
		return options.HostOptions.MaxReservedMemory
	}
	if reserved == 0 {
		panic("memory reserve value is 0, need help")
	}
	return reserved
}

func (h *SHostInfo) getReservedMemMb() int {
	return h.getOSReservedMemMb() + h.getKubeReservedMemMb()
}

func (h *SHostInfo) getReportedReservedMemMb() int {
	if options.HostOptions.HugepagesOption == "native" {
		// return total minus mem in huagepage pool
		hp, _ := h.Mem.GetHugepages()
		return h.GetMemory() - int(hp.BytesMb())
	}
	return h.getReservedMemMb()
}

func (h *SHostInfo) PutHostOnline() error {
	if len(h.SysError) > 0 && !options.HostOptions.StartHostIgnoreSysError {
		log.Fatalf("Can't put host online, unless resolve these problem %v", h.SysError)
	} else if len(h.SysError) > 0 && options.HostOptions.StartHostIgnoreSysError {
		log.Errorf("Host sys error: %v", h.SysError)
	}

	data := jsonutils.NewDict()
	if len(h.SysWarning) > 0 {
		data.Set("warning", jsonutils.Marshal(h.SysWarning))
	}

	_, err := modules.Hosts.PerformAction(
		h.GetSession(), h.HostId, "online", data)
	if err != nil {
		logclient.AddSimpleActionLog(h, logclient.ACT_ONLINE, data, hostutils.GetComputeSession(context.Background()).GetToken(), false)
	}
	return err
}

func (h *SHostInfo) getNetworkInfo() {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.Hostwires.ListDescendent(
		h.GetSession(),
		h.HostId, params)
	if err != nil {
		h.onFail(errors.Wrapf(err, "Hostwires.ListDescendent: %s", params))
		return
	}
	hostwires := []api.HostwireDetails{}
	jsonutils.Update(&hostwires, res.Data)
	for _, hw := range hostwires {
		nic := h.GetMatchNic(hw.Bridge, hw.Interface, hw.MacAddr)
		if nic != nil {
			if hw.Bandwidth < 1 {
				hw.Bandwidth = 1000
			}
			nic.SetWireId(hw.Wire, hw.WireId, int64(hw.Bandwidth))
		} else {
			log.Warningf("NIC not present %s", jsonutils.Marshal(hw).String())
		}
	}
	h.uploadNetworkInfo()
}

func (h *SHostInfo) uploadNetworkInfo() {
	phyNics, err := sysutils.Nics()
	if err != nil {
		h.onFail(errors.Wrap(err, "parse physical nics info"))
		return
	}
	for _, pnic := range phyNics {
		err := h.doSendPhysicalNicInfo(pnic)
		if err != nil {
			h.onFail(errors.Wrapf(err, "doSendPhysicalNicInfo %s", pnic.Dev))
			return
		}
	}
	for _, nic := range h.Nics {
		if len(nic.WireId) == 0 {
			if len(nic.Network) == 0 {
				kwargs := jsonutils.NewDict()
				kwargs.Set("ip", jsonutils.NewString(nic.Ip))
				kwargs.Set("is_classic", jsonutils.JSONTrue)
				kwargs.Set("scope", jsonutils.NewString("system"))
				kwargs.Set("limit", jsonutils.NewInt(0))

				wireInfo, err := hostutils.GetWireOfIp(context.Background(), kwargs)
				if err != nil {
					h.onFail(errors.Wrapf(err, "GetWireOfIp args: %s", kwargs.String()))
					return
				} else {
					nic.Network, _ = wireInfo.GetString("name")
					err := h.doUploadNicInfo(nic)
					if err != nil {
						h.onFail(errors.Wrapf(err, "doUploadNicInfo %s", nic.Inter))
						return
					}
				}
			} else {
				err := h.doUploadNicInfo(nic)
				if err != nil {
					h.onFail(errors.Wrapf(err, "doUploadNicInfo %s", nic.Inter))
					return
				}
			}
		} else {
			err := h.doSyncNicInfo(nic)
			if err != nil {
				h.onFail(errors.Wrapf(err, "doSyncNicInfo %s", nic.Inter))
				return
			}
		}
	}
	h.getStoragecacheInfo()
}

func (h *SHostInfo) doSendPhysicalNicInfo(nic *types.SNicDevInfo) error {
	log.Infof("upload physical nic: %s(%s)", nic.Dev, nic.Mac)
	return h.doUploadNicInfoInternal(nic.Dev, nic.Mac.String(), "", "", "", nic.Up != nil && *nic.Up)
}

func (h *SHostInfo) doUploadNicInfo(nic *SNIC) error {
	err := h.doUploadNicInfoInternal(nic.Inter, nic.BridgeDev.GetMac(), nic.Network, nic.Bridge, nic.Ip, true)
	if err != nil {
		return errors.Wrap(err, "doUploadNicInfoInternal")
	}
	return h.onUploadNicInfoSucc(nic)
}

func (h *SHostInfo) doUploadNicInfoInternal(ifname, mac, net, bridge, ipaddr string, isUp bool) error {
	log.Infof("Upload NIC br:%s if:%s", bridge, ifname)
	content := jsonutils.NewDict()
	content.Set("mac", jsonutils.NewString(mac))
	content.Set("wire", jsonutils.NewString(net))
	content.Set("bridge", jsonutils.NewString(bridge))
	content.Set("interface", jsonutils.NewString(ifname))
	if isUp {
		content.Set("link_up", jsonutils.JSONTrue)
	} else {
		content.Set("link_up", jsonutils.JSONFalse)
	}
	if len(ipaddr) > 0 {
		content.Set("ip_addr", jsonutils.NewString(ipaddr))
		if ipaddr == h.GetMasterIp() {
			content.Set("nic_type", jsonutils.NewString(api.NIC_TYPE_ADMIN))
		}
		// always try to allocate from reserved pool
		content.Set("reserve", jsonutils.JSONTrue)
	}
	_, err := modules.Hosts.PerformAction(h.GetSession(),
		h.HostId, "add-netif", content)
	if err != nil {
		return errors.Wrap(err, "modules.Hosts.PerformAction add-netif")
	}
	return nil
}

func (h *SHostInfo) doSyncNicInfo(nic *SNIC) error {
	content := jsonutils.NewDict()
	content.Set("bridge", jsonutils.NewString(nic.Bridge))
	content.Set("interface", jsonutils.NewString(nic.Inter))
	query := jsonutils.NewDict()
	query.Set("mac_addr", jsonutils.NewString(nic.BridgeDev.GetMac()))
	_, err := modules.Hostwires.Update(h.GetSession(),
		h.HostId, nic.WireId, query, content)
	if err != nil {
		return errors.Wrap(err, "modules.Hostwires.Update")
	}
	return nil
}

func (h *SHostInfo) onUploadNicInfoSucc(nic *SNIC) error {
	res, err := modules.Hostwires.Get(h.GetSession(), h.HostId, nic.Network, nil)
	if err != nil {
		return errors.Wrap(err, "modules.Hostwires.Get")
	} else {
		bridge, _ := res.GetString("bridge")
		iface, _ := res.GetString("interface")
		macAddr, _ := res.GetString("mac_addr")
		nic = h.GetMatchNic(bridge, iface, macAddr)
		if nic != nil {
			wire, _ := res.GetString("wire")
			wireId, _ := res.GetString("wire_id")
			bandwidth, err := res.Int("bandwidth")
			if err != nil {
				bandwidth = 1000
			}
			nic.SetWireId(wire, wireId, bandwidth)
		} else {
			return errors.Error("GetMatchNic failed!!!")
		}
	}
	return nil
}

func (h *SHostInfo) getStoragecacheInfo() {
	path := storageman.GetManager().LocalStorageImagecacheManager.GetPath()
	params := jsonutils.NewDict()
	params.Set("external_id", jsonutils.NewString(h.HostId))
	params.Set("path", jsonutils.NewString(path))
	res, err := modules.Storagecaches.List(
		h.GetSession(), params)
	if err != nil {
		h.onFail(err)
		return
	} else {
		if len(res.Data) == 0 {
			body := jsonutils.NewDict()
			body.Set("name",
				jsonutils.NewString(fmt.Sprintf(
					"local-%s-%s", h.FullName, time.Now().String())))
			body.Set("path", jsonutils.NewString(path))
			body.Set("external_id", jsonutils.NewString(h.HostId))
			sc, err := modules.Storagecaches.Create(h.GetSession(), body)
			if err != nil {
				h.onFail(err)
				return
			} else {
				scid, _ := sc.GetString("id")
				storageman.GetManager().
					LocalStorageImagecacheManager.SetStoragecacheId(scid)
				h.getStorageInfo()
			}
		} else {
			scid, _ := res.Data[0].GetString("id")
			storageman.GetManager().
				LocalStorageImagecacheManager.SetStoragecacheId(scid)
			h.getStorageInfo()
		}
	}
}

func (h *SHostInfo) getStorageInfo() {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.Hoststorages.ListDescendent(
		h.GetSession(),
		h.HostId, params)
	if err != nil {
		h.onFail(err)
		return
	} else {
		h.onGetStorageInfoSucc(res.Data)
	}
}

func (h *SHostInfo) onGetStorageInfoSucc(hoststorages []jsonutils.JSONObject) {
	var detachStorages = []jsonutils.JSONObject{}
	storageManager := storageman.GetManager()

	for _, hs := range hoststorages {
		storagetype, _ := hs.GetString("storage_type")
		mountPoint, _ := hs.GetString("mount_point")
		storagecacheId, _ := hs.GetString("storagecache_id")
		imagecachePath, _ := hs.GetString("imagecache_path")
		storageId, _ := hs.GetString("storage_id")
		storageName, _ := hs.GetString("storage")
		storageConf, _ := hs.Get("storage_conf")

		log.Infof("Storage %s(%s) mountpoint %s", storageName, storagetype, mountPoint)

		if !utils.IsInStringArray(storagetype, api.STORAGE_LOCAL_TYPES) {
			storage := storageManager.NewSharedStorageInstance(mountPoint, storagetype)
			if storage != nil {
				storage.SetStoragecacheId(storagecacheId)
				if err := storage.SetStorageInfo(storageId, storageName, storageConf); err != nil {
					h.onFail(err)
					return
				}
				storageManager.Storages = append(storageManager.Storages, storage)
				if err := storage.Accessible(); err != nil {
					h.onFail(err)
					return
				}
				storageManager.InitSharedStorageImageCache(
					storagetype, storagecacheId, imagecachePath, storage)
			}
		} else {
			// Storage type local
			storage, _ := storageManager.GetStorageByPath(mountPoint)
			if storage != nil {
				storage.SetStoragecacheId(storagecacheId)
				if IsRootPartition(mountPoint) {
					// update host storage is root partition
					params := jsonutils.NewDict()
					params.Set("is_root_partiton", jsonutils.JSONTrue)
					_, err := modules.Hoststorages.Update(h.GetSession(), h.HostId, storageId, nil, params)
					if err != nil {
						h.onFail(errors.Wrapf(err, "Update host storage %s with params %s", storageId, params))
						return
					}
				}
				if err := storage.SetStorageInfo(storageId, storageName, storageConf); err != nil {
					h.onFail(errors.Wrapf(err, "Set storage info %s/%s/%s", storageId, storageName, storageConf))
					return
				}
				if storagetype == api.STORAGE_LVM {
					// lvm set storage image cache info
					storageManager.InitLVMStorageImageCache(storagecacheId, mountPoint)
				}
			} else {
				// XXX hack: storage type baremetal is a converted host，reserve storage
				if storagetype != api.STORAGE_BAREMETAL {
					detachStorages = append(detachStorages, hs)
				}
			}
		}
	}

	if len(detachStorages) > 0 {
		go StartDetachStorages(detachStorages)
	}

	h.uploadStorageInfo()
}

func (h *SHostInfo) uploadStorageInfo() {
	for _, s := range storageman.GetManager().Storages {
		if err := s.SetStorageInfo(s.GetId(), s.GetStorageName(), s.GetStorageConf()); err != nil {
			h.onFail(errors.Wrapf(err, "Upload storage %s info with config %s", s.GetStorageName(), s.GetStorageConf()))
			return
		}
		res, err := s.SyncStorageInfo()
		if err != nil {
			h.onFail(errors.Wrapf(err, "Sync storage %s info", s.GetStorageName()))
			return
		} else {
			h.onSyncStorageInfoSucc(s, res)
		}
	}
	// go storageman.StartSyncStorageSizeTask(
	//	time.Duration(options.HostOptions.SyncStorageInfoDurationSecond) * time.Second,
	// )
	h.probeSyncIsolatedDevicesStep()
}

func (h *SHostInfo) onSyncStorageInfoSucc(storage storageman.IStorage, storageInfo jsonutils.JSONObject) {
	if len(storage.GetId()) == 0 {
		id, _ := storageInfo.GetString("id")
		name, _ := storageInfo.GetString("name")
		storageConf, _ := storageInfo.Get("storage_conf")
		if err := storage.SetStorageInfo(id, name, storageConf); err != nil {
			h.onFail(err)
			return
		}
		h.attachStorage(storage)
	}
}

func (h *SHostInfo) attachStorage(storage storageman.IStorage) {
	content := jsonutils.NewDict()
	content.Set("mount_point", jsonutils.NewString(storage.GetPath()))
	content.Set("is_root_partition", jsonutils.NewBool(IsRootPartition(storage.GetPath())))
	_, err := modules.Hoststorages.Attach(h.GetSession(),
		h.HostId, storage.GetId(), content)
	if err != nil {
		h.onFail(err)
		return
	}
}

func (h *SHostInfo) getRemoteIsolatedDevices() ([]jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("host", jsonutils.NewString(h.GetHostId()))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.IsolatedDevices.List(h.GetSession(), params)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

func (h *SHostInfo) probeSyncIsolatedDevicesStep() {
	_, err := h.probeSyncIsolatedDevices()
	if err != nil {
		h.onFail(errors.Wrap(err, "probeSyncIsolatedDevices"))
		return
	}

	h.deployAdminAuthorizedKeys()
}

func (h *SHostInfo) probeSyncIsolatedDevices() (*jsonutils.JSONArray, error) {
	for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
		if out, err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Output(); err != nil {
			log.Errorf("failed probe driver %s: %s %s", driver, out, err)
		}
	}
	if err := h.IsolatedDeviceMan.ProbePCIDevices(options.HostOptions.DisableGPU, options.HostOptions.DisableUSB); err != nil {
		return nil, errors.Wrap(err, "ProbePCIDevices")
	}

	objs, err := h.getRemoteIsolatedDevices()
	if err != nil {
		return nil, errors.Wrap(err, "getRemoteIsolatedDevices")
	}
	for _, obj := range objs {
		info := isolated_device.CloudDeviceInfo{}
		if err := obj.Unmarshal(&info); err != nil {
			return nil, errors.Wrapf(err, "unmarshal isolated device %s to cloud device info", obj)
		}
		dev := h.IsolatedDeviceMan.GetDeviceByIdent(info.VendorDeviceId, info.Addr)
		if dev != nil {
			dev.SetDeviceInfo(info)
		} else {
			// detach device
			h.IsolatedDeviceMan.AppendDetachedDevice(&info)
		}
	}
	h.IsolatedDeviceMan.StartDetachTask()
	if err := h.IsolatedDeviceMan.BatchCustomProbe(); err != nil {
		return nil, errors.Wrap(err, "Device probe")
	}

	// sync each isolated device found
	updateDevs := jsonutils.NewArray()
	for _, dev := range h.IsolatedDeviceMan.GetDevices() {
		if obj, err := isolated_device.SyncDeviceInfo(h.GetSession(), h.HostId, dev); err != nil {
			return nil, errors.Wrapf(err, "Sync device %s", dev)
		} else {
			updateDevs.Add(obj)
		}
	}
	return updateDevs, nil
}

func (h *SHostInfo) deployAdminAuthorizedKeys() {
	onErr := func(format string, args ...interface{}) {
		h.onFail(fmt.Sprintf(format, args...))
		return
	}

	sshDir := path.Join("/root", ".ssh")
	output, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", sshDir).Output()
	if err != nil {
		onErr("mkdir .ssh failed %s %s", output, err)
	}

	query := jsonutils.NewDict()
	query.Set("admin", jsonutils.JSONTrue)
	ret, err := modules.Sshkeypairs.List(h.GetSession(), query)
	if err != nil {
		onErr("Get admin sshkey: %v", err)
	}
	if len(ret.Data) == 0 {
		onErr("Not found admin sshkey")
	}
	keys := ret.Data[0]
	adminPublicKey, _ := keys.GetString("public_key")
	pubKeys := &deployapi.SSHKeys{AdminPublicKey: adminPublicKey}

	var oldKeys string
	authFile := path.Join(sshDir, "authorized_keys")
	if procutils.NewRemoteCommandAsFarAsPossible("test", "-f", authFile).Run() == nil {
		output, err := procutils.NewRemoteCommandAsFarAsPossible("cat", authFile).Output()
		if err != nil {
			onErr("cat auth file %s %s", output, err)
		}
		oldKeys = string(output)
	}
	newKeys := fsdriver.MergeAuthorizedKeys(oldKeys, pubKeys)
	if output, err := procutils.NewRemoteCommandAsFarAsPossible(
		"sh", "-c", fmt.Sprintf("echo '%s' > %s", newKeys, authFile)).Output(); err != nil {
		onErr("write public keys: %s %s", output, err)
	}
	if output, err := procutils.NewRemoteCommandAsFarAsPossible(
		"chmod", "0644", authFile).Output(); err != nil {
		onErr("chmod failed %s %s", output, err)
	}
	h.onSucc()
}

func (h *SHostInfo) onSucc() {
	if !h.stopped && !h.isRegistered {
		log.Infof("Host registration process success....")
		if err := h.save(); err != nil {
			panic(err.Error())
		}
		h.StartPinger()
		if h.registerCallback != nil {
			h.registerCallback()
		}
		h.isRegistered = true

		// Notify caller, host register is success
		close(h.IsRegistered)
	}
}

func (h *SHostInfo) StartPinger() {
	h.pinger = NewHostPingTask(options.HostOptions.PingRegionInterval)
	if h.pinger != nil {
		go h.pinger.Start()
	}
}

func (h *SHostInfo) save() error {
	if h.saved {
		return nil
	} else {
		h.saved = true
	}

	if err := h.registerHostlocalServer(); err != nil {
		return err
	}
	// TODO XXX >>> ???
	// file put content
	if err := h.setupBridges(); err != nil {
		return err
	}
	return nil
}

func (h *SHostInfo) setupBridges() error {
	for _, n := range h.Nics {
		if err := n.BridgeDev.WarmupConfig(); err != nil {
			log.Errorln(err)
			return err
		}
	}
	return nil
}

func (h *SHostInfo) registerHostlocalServer() error {
	for _, n := range h.Nics {
		mac := h.GetMasterMac()
		if len(mac) == 0 {
			panic("len mac == 0")
		}
		ip := h.GetMasterIp()
		if len(ip) == 0 {
			panic("len ip == 0")
		}

		err := n.BridgeDev.RegisterHostlocalServer(mac, ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *SHostInfo) GetId() string {
	return h.HostId
}

func (h *SHostInfo) GetName() string {
	return h.getHostname()
}

func (h *SHostInfo) Keyword() string {
	return "host"
}

func (h *SHostInfo) stop() {
	log.Infof("Host Info stop ...")
	h.unregister()
	if h.pinger != nil {
		h.pinger.Stop()
	}
	for _, nic := range h.Nics {
		nic.ExitCleanup()
	}
}

func (h *SHostInfo) unregister() {
	isLog := false
	for {
		updateHealthStatus := true
		input := api.HostOfflineInput{
			UpdateHealthStatus: &updateHealthStatus,
			Reason:             "host stop",
		}
		_, err := modules.Hosts.PerformAction(h.GetSession(), h.HostId, "offline", jsonutils.Marshal(input))
		if err != nil {
			if errors.Cause(err) == httperrors.ErrResourceNotFound {
				log.Errorf("host not found on region, may be removed, exit cleanly")
				break
			}
			if !isLog {
				logclient.AddSimpleActionLog(h, logclient.ACT_OFFLINE, err, hostutils.GetComputeSession(context.Background()).GetToken(), false)
				isLog = true
			}
			time.Sleep(time.Second * 1)
			continue
		}
		break
	}
	h.stopped = true
}

func (h *SHostInfo) OnCatalogChanged(catalog mcclient.KeystoneServiceCatalogV3) {
	// TODO: dynamic probe endpoint type
	svcs := os.Getenv("HOST_SYSTEM_SERVICES_OFF")
	defaultEndpointType := options.HostOptions.SessionEndpointType
	if len(defaultEndpointType) == 0 {
		defaultEndpointType = identityapi.EndpointInterfacePublic
	}
	s := auth.AdminSession(context.Background(), options.HostOptions.Region, h.Zone, defaultEndpointType)
	// replace session catalog
	s.SetServiceCatalog(catalog)

	if options.HostOptions.ManageNtpConfiguration {
		ntpd := system_service.GetService("ntpd")
		urls, _ := s.GetServiceURLs("ntp", defaultEndpointType)
		if len(urls) > 0 {
			log.Infof("Get Ntp urls: %v", urls)
		} else {
			urls = []string{"ntp://cn.pool.ntp.org",
				"ntp://0.cn.pool.ntp.org",
				"ntp://1.cn.pool.ntp.org",
				"ntp://2.cn.pool.ntp.org",
				"ntp://3.cn.pool.ntp.org"}
		}
		if !reflect.DeepEqual(ntpd.GetConf(), urls) || (!strings.Contains(svcs, "ntpd") && !ntpd.IsActive()) {
			ntpd.SetConf(urls)
			ntpd.BgReload(map[string]interface{}{"servers": urls})
		}
	}
	telegraf := system_service.GetService("telegraf")
	conf := map[string]interface{}{}
	conf["hostname"] = h.getHostname()
	conf["tags"] = map[string]string{
		"id":                                  h.HostId,
		"host_id":                             h.HostId,
		"zone_id":                             h.ZoneId,
		"zone":                                h.Zone,
		"cloudregion_id":                      h.CloudregionId,
		"cloudregion":                         h.Cloudregion,
		"domain_id":                           h.Domain_id,
		"project_domain":                      h.Project_domain,
		"region":                              options.HostOptions.Region,
		"host_ip":                             h.GetMasterIp(),
		hostconsts.TELEGRAF_TAG_KEY_BRAND:     hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND,
		hostconsts.TELEGRAF_TAG_KEY_RES_TYPE:  hostconsts.TELEGRAF_TAG_ONECLOUD_RES_TYPE,
		hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE: hostconsts.TELEGRAF_TAG_ONECLOUD_HOST_TYPE_HOST,
	}
	conf["nics"] = h.getNicsTelegrafConf()
	urls, _ := s.GetServiceURLs("kafka", defaultEndpointType)
	if len(urls) > 0 {
		conf["kafka"] = map[string]interface{}{"brokers": urls, "topic": "telegraf"}
	}
	urls, _ = s.GetServiceURLs("influxdb", defaultEndpointType)
	if len(urls) > 0 {
		conf["influxdb"] = map[string]interface{}{"url": urls, "database": "telegraf"}
	}
	if !reflect.DeepEqual(telegraf.GetConf(), conf) || (!strings.Contains(svcs, "telegraf") && !telegraf.IsActive()) {
		log.Debugf("telegraf config: %s", conf)
		telegraf.SetConf(conf)
		if !strings.Contains(svcs, "telegraf") {
			telegraf.BgReload(conf)
		} else {
			telegraf.BgReloadConf(conf)
		}
	}

	/*urls, _ = catalog.GetServiceURLs("elasticsearch",
		options.HostOptions.Region, "zone", defaultEndpointType)
	if len(urls) > 0 {
		conf["elasticsearch"] = map[string]interface{}{"url": urls[0]}
		fluentbit := system_service.GetService("fluentbit")
		if !reflect.DeepEqual(fluentbit.GetConf(), conf) || !fluentbit.IsActive() {
			fluentbit.SetConf(conf)
			fluentbit.BgReload(conf)
		}
	}*/
}

func (h *SHostInfo) getNicsTelegrafConf() []map[string]interface{} {
	var ret = make([]map[string]interface{}, 0)
	existing := make(map[string]struct{})
	for i, n := range h.Nics {
		ret = append(ret, map[string]interface{}{
			"name":  n.Inter,
			"alias": fmt.Sprintf("eth%d", i),
			"speed": n.Bandwidth,
		})
		ret = append(ret, map[string]interface{}{
			"name":  n.Inter,
			"alias": fmt.Sprintf("br%d", i),
			"speed": n.Bandwidth,
		})
		existing[n.Inter] = struct{}{}
	}
	phyNics, _ := sysutils.Nics()
	for _, pnic := range phyNics {
		if _, ok := existing[pnic.Dev]; !ok {
			ret = append(ret, map[string]interface{}{
				"name":  pnic.Dev,
				"speed": pnic.Speed,
			})
		}
	}
	return ret
}

func (h *SHostInfo) getHostname() string {
	if len(h.FullName) > 0 {
		return h.FullName
	}
	return h.fetchHostname()
}

func (h *SHostInfo) GetCpuArchitecture() string {
	return h.Cpu.CpuArchitecture
}

func (h *SHostInfo) IsAarch64() bool {
	return h.GetCpuArchitecture() == apis.OS_ARCH_AARCH64
}

func (h *SHostInfo) IsX8664() bool {
	return h.GetCpuArchitecture() == apis.OS_ARCH_X86_64
}

func (h *SHostInfo) GetKubeletConfig() kubelet.KubeletConfig {
	return h.kubeletConfig
}

func NewHostInfo() (*SHostInfo, error) {
	var res = new(SHostInfo)
	res.sysinfo = &SSysInfo{}
	cpu, err := DetectCpuInfo()
	if err != nil {
		return nil, err
	} else {
		res.Cpu = cpu
	}

	if res.IsAarch64() {
		qemutils.UseAarch64()
	} else if !res.IsX8664() {
		return nil, fmt.Errorf("unsupport cpu architecture %s", cpu.CpuArchitecture)
	}

	log.Infof("CPU Model %s Microcode %s", cpu.cpuInfoProc.Model, cpu.cpuInfoProc.Microcode)

	mem, err := DetectMemoryInfo()
	if err != nil {
		return nil, err
	} else {
		res.Mem = mem
	}

	res.Nics = make([]*SNIC, 0)
	res.IsRegistered = make(chan struct{})
	res.SysError = make(map[string]string)
	res.SysWarning = make(map[string]string)

	if !options.HostOptions.DisableProbeKubelet {
		kubeletDir := options.HostOptions.KubeletRunDirectory
		kubeletConfig, err := kubelet.NewKubeletConfigByDirectory(kubeletDir)
		if err != nil {
			return nil, errors.Wrapf(err, "New kubelet config by dir: %s", kubeletDir)
		}
		res.kubeletConfig = kubeletConfig
		log.Infof("Get kubelet container image Fs: %s, eviction config: %s", res.kubeletConfig.GetImageFs(), res.kubeletConfig.GetEvictionConfig())
	}

	return res, nil
}

var hostInfo *SHostInfo

func Instance() *SHostInfo {
	if hostInfo == nil {
		var err error
		hostInfo, err = NewHostInfo()
		if err != nil {
			log.Fatalf("NewHostInfo: %s", err)
		}
	}
	return hostInfo
}

func Stop() {
	hostInfo.stop()
}
