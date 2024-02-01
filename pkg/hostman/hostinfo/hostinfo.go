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
	"bytes"
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
	"syscall"
	"time"

	"github.com/vishvananda/netlink"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	napi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/hardware"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/kubelet"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	_ "yunion.io/x/onecloud/pkg/hostman/isolated_device/container_device"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	_ "yunion.io/x/onecloud/pkg/hostman/storageman/container_storage"
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
	"yunion.io/x/onecloud/pkg/util/ovnutils"
	"yunion.io/x/onecloud/pkg/util/pod"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SHostInfo struct {
	isRegistered bool
	// IsRegistered     chan struct{}
	// registerCallback func()
	stopped bool
	isLoged bool

	saved  bool
	pinger *SHostPingTask

	Cpu                 *SCPUInfo
	Mem                 *SMemory
	sysinfo             *SSysInfo
	qemuMachineInfoList []monitor.MachineInfo
	kvmMaxCpus          uint

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

	FullName string
	SysError map[string][]api.HostError

	IoScheduler string

	cri pod.CRI
}

func (h *SHostInfo) GetContainerDeviceConfigurationFilePath() string {
	return options.HostOptions.ContainerDeviceConfigFile
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
	if ovnutils.HasOvnSupport() && !options.HostOptions.DisableLocalVpc {
		if err := h.setupOvnChassis(); err != nil {
			return errors.Wrap(err, "Setup OVN Chassis")
		}
	}

	if h.IsContainerHost() {
		if err := h.initCRI(); err != nil {
			return errors.Wrap(err, "init container runtime interface")
		}
	}

	return nil
}

func (h *SHostInfo) initCRI() error {
	cri, err := pod.NewCRI(h.GetContainerRuntimeEndpoint(), 3*time.Second)
	if err != nil {
		return errors.Wrapf(err, "New CRI by endpoint %q", h.GetContainerRuntimeEndpoint())
	}
	ver, err := cri.Version(context.Background())
	if err != nil {
		return errors.Wrap(err, "get runtime version")
	}
	log.Infof("Init container runtime: %s", ver)
	h.cri = cri
	return nil
}

func (h *SHostInfo) GetCRI() pod.CRI {
	return h.cri
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
			return fmt.Errorf("Listen interface %s master have no IP", options.HostOptions.ListenInterface)
		}
	} else {
		// set MasterNic to the first NIC with IP
		h.MasterNic = nil
		for _, n := range h.Nics {
			if len(n.Ip) > 0 {
				h.MasterNic = netutils2.NewNetInterface(n.Bridge)
			}
		}
		if h.MasterNic == nil {
			return fmt.Errorf("No interface suitable to be master NIC")
		}
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

	// setup tuned-adm
	_, err = procutils.NewRemoteCommandAsFarAsPossible("tuned-adm", "profile", "virtual-host").Output()
	if err != nil {
		log.Errorf("tuned-adm profile virtual-host fail: %s", err)
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
		h.EnableNativeHugepages()
		hp, err := h.Mem.GetHugepages()
		if err != nil {
			return errors.Wrap(err, "MEM.GetHugepages")
		}
		for i := 0; i < len(hp); i++ {
			if hp[i].SizeKb == options.HostOptions.HugepageSizeMb*1024 {
				nr := hp[i].Total
				h.sysinfo.HugepageNr = &nr
				h.sysinfo.HugepageSizeKb = hp[i].SizeKb
				break
			}
		}
		if h.sysinfo.HugepageNr == nil || *h.sysinfo.HugepageNr == 0 {
			return errors.Errorf("hugepage %d nr 0", options.HostOptions.HugepageSizeMb)
		}
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

func runDmidecode(hType string) (*types.SSystemInfo, error) {
	output, err := procutils.NewCommand("dmidecode", "-t", hType).Output()
	if err != nil {
		return &types.SSystemInfo{}, errors.Wrapf(err, "cmd: dmidecode -t %s, output: %s", hType, output)
	}
	info, err := sysutils.ParseDMISysinfo(strings.Split(string(output), "\n"))
	if err != nil {
		return &types.SSystemInfo{}, errors.Wrapf(err, "ParseDMISysinfo with line: %s", output)
	}
	return info, nil
}

func (h *SHostInfo) detectHostInfo() error {
	sysInfo, err := runDmidecode("1")
	if err != nil {
		log.Warningf("get system info error: %v", err)
	}
	h.sysinfo.SSystemInfo = sysInfo

	motherboardInfo, err := runDmidecode("2")
	if err != nil {
		log.Warningf("get motherboard info error: %v", err)
	}
	h.sysinfo.MotherboardInfo = motherboardInfo

	h.detectKvmModuleSupport()
	h.detectKVMMaxCpus()
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

func (h *SHostInfo) GetMemoryTotal() int {
	if h.Mem.MemInfo == nil {
		return h.Mem.MemInfo.Total
	}
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

func (h *SHostInfo) EnableNativeHugepages() {
	kv := map[string]string{
		"/sys/kernel/mm/transparent_hugepage/enabled": "never",
		"/sys/kernel/mm/transparent_hugepage/defrag":  "never",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
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

/*func (h *SHostInfo) resetIptables() error {
	for _, tbl := range []string{"filter", "nat", "mangle"} {
		output, err := procutils.NewCommand("iptables", "-t", tbl, "-F").Output()
		if err != nil {
			return errors.Wrapf(err, "fail to clean NAT iptables: %s", output)
		}
	}
	return nil
}*/

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
	/*if err := h.detectQemuVersion(); err != nil {
		log.Errorf("detect qemu version: %s", err.Error())
		h.AppendHostError(fmt.Sprintf("detect qemu version: %s", err.Error()))
	}*/
	h.detectOvsVersion()
	if err := h.detectOvsKOVersion(); err != nil {
		log.Errorf("detect ovs kernel version: %s", err.Error())
		h.AppendHostError(fmt.Sprintf("detect ovs kernel version: %s", err.Error()))
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
		var v = parts[len(parts)-1]
		if strings.HasPrefix(parts[len(parts)-1], "(") {
			v = parts[len(parts)-2]
		}

		if len(v) > 0 {
			log.Infof("Detect qemu version is %s", v)
			h.sysinfo.QemuVersion = v
		} else {
			return fmt.Errorf("Failed to detect qemu version")
		}
	}
	return h.detectQemuCapabilities(h.sysinfo.QemuVersion)
}

const (
	KVM_GET_API_VERSION = uintptr(44544)
	KVM_CREATE_VM       = uintptr(44545)
	KVM_CHECK_EXTENSION = uintptr(44547)

	KVM_CAP_NR_VCPUS  = 9
	KVM_CAP_MAX_VCPUS = 66
	// TODO: arm mem ipa size for max memsize
	// KVM_CAP_ARM_VM_IPA_SIZE
)

func (h *SHostInfo) detectKVMMaxCpus() error {
	ioctl := func(fd, op, arg uintptr) (uintptr, uintptr, syscall.Errno) {
		return syscall.Syscall(syscall.SYS_IOCTL, fd, op, arg)
	}
	kvm, err := syscall.Open("/dev/kvm", syscall.O_RDONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed open /dev/kvm")
	}
	defer syscall.Close(kvm)
	r, _, errno := ioctl(uintptr(kvm), KVM_GET_API_VERSION, uintptr(0))
	if errno != 0 {
		return errors.Errorf("get api version: %d", errno)
	}
	log.Infof("KVM API VERSION %d", r)
	r, _, errno = ioctl(uintptr(kvm), KVM_CHECK_EXTENSION, uintptr(KVM_CAP_MAX_VCPUS))
	if errno != 0 {
		return errors.Errorf("kvm check extension KVM_CAP_MAX_VCPUS errno: %d", errno)
	}
	log.Infof("KVM CAP MAX VCPUS: %d", r)
	if r > 0 {
		h.kvmMaxCpus = uint(r)
	}
	r, _, errno = ioctl(uintptr(kvm), KVM_CHECK_EXTENSION, uintptr(KVM_CAP_NR_VCPUS))
	if errno != 0 {
		return errors.Errorf("kvm check extension KVM_CAP_NR_VCPUS errno: %d", errno)
	}
	log.Infof("KVM CAP NR VCPUS: %d", r)
	if r > 0 && (h.kvmMaxCpus == 0 || h.kvmMaxCpus > uint(r)) {
		h.kvmMaxCpus = uint(r)
	}

	// kernel doc: If the KVM_CAP_NR_VCPUS does not exist
	// you should assume that max_vcpus is 4 cpus max.
	if h.kvmMaxCpus == 0 {
		h.kvmMaxCpus = 4
	}
	return nil
}

type QemuCaps struct {
	QemuVersion     string
	MachineInfoList []monitor.MachineInfo
}

func (h *SHostInfo) loadQemuCaps(capsPath string) (*QemuCaps, error) {
	if fileutils2.Exists(capsPath) {
		caps, err := fileutils2.FileGetContents(capsPath)
		if err != nil {
			log.Errorf("failed get qemu caps: %s", err)
			return nil, err
		}
		qemuCaps := new(QemuCaps)
		jCaps, err := jsonutils.ParseString(caps)
		if err != nil {
			log.Errorf("failed parse qemu caps: %s", err)
			return nil, err
		}
		err = jCaps.Unmarshal(qemuCaps)
		if err != nil {
			log.Errorf("failed unmarshal qemu caps: %s", err)
			return nil, err
		}
		return qemuCaps, nil
	}
	return nil, nil
}

func (h *SHostInfo) detectQemuCapabilities(version string) error {
	capsPath := path.Join(options.HostOptions.ServersPath, "qemu_caps")
	caps, err := h.loadQemuCaps(capsPath)
	if err == nil && caps != nil && caps.QemuVersion == version {
		h.qemuMachineInfoList = caps.MachineInfoList
		return nil
	}

	qmpCmds := fmt.Sprintf(`echo "{'execute': 'qmp_capabilities'}
       {'execute': 'query-machines'}
       {'execute': 'quit'}" | %s -qmp stdio  -vnc none -machine none -display none`, qemutils.GetQemu(version))
	log.Debugf("qemu caps cmdline %v", qmpCmds)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", qmpCmds).Output()
	if err != nil {
		log.Errorf("failed start qemu caps cmdline: %s", qmpCmds)
	}
	segs := bytes.Split(out, []byte{'\n'})
	if len(segs) < 6 {
		return errors.Errorf("unexpect qmp res %s", out)
	}
	res, err := jsonutils.Parse(bytes.TrimSpace(segs[2]))
	if err != nil {
		return errors.Errorf("Unmarshal %s error: %s", segs[2], err)
	}
	var machineInfoList = make([]monitor.MachineInfo, 0)
	err = res.Unmarshal(&machineInfoList, "return")
	if err != nil {
		return errors.Errorf("failed unmarshal machineinfo return %s: %s", segs[3], err)
	}
	h.qemuMachineInfoList = machineInfoList
	qemuCaps := &QemuCaps{
		QemuVersion:     version,
		MachineInfoList: machineInfoList,
	}
	return fileutils2.FilePutContents(capsPath, jsonutils.Marshal(qemuCaps).String(), false)
}

func (h *SHostInfo) GetQemuMachineInfoList() []monitor.MachineInfo {
	return h.qemuMachineInfoList
}

func (h *SHostInfo) GetKVMMaxCpus() uint {
	return h.kvmMaxCpus
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
	mask, _ := h.MasterNic.Mask.Size()
	return h.MasterNic.Addr, mask
}

func (h *SHostInfo) GetMasterIp() string {
	return h.MasterNic.Addr
}

func (h *SHostInfo) GetMasterMac() string {
	return h.getMasterMacWithRefresh(false)
}

func (h *SHostInfo) getMasterMacWithRefresh(refresh bool) string {
	if refresh {
		h.MasterNic.FetchConfig()
	}
	return h.MasterNic.GetMac()
}

func (h *SHostInfo) getMatchNic(mac string, vlanId int) *SNIC {
	for _, nic := range h.Nics {
		if nic.BridgeDev.GetMac() == mac && nic.BridgeDev.GetVlanId() == vlanId {
			return nic
		}
	}
	return nil
}

func (h *SHostInfo) StartRegister(delay int) {
	time.Sleep(time.Duration(delay) * time.Second)

	h.register()
}

func (h *SHostInfo) reportHostErrors() {
	var errs = []api.HostError{}
	for _, v := range h.SysError {
		errs = append(errs, v...)
	}
	data := jsonutils.NewDict()
	data.Set(api.HOSTMETA_HOST_ERRORS, jsonutils.Marshal(errs))
	_, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, data)
	if err != nil {
		log.Errorf("failed sync host errors %s", err)
	}
}

func (h *SHostInfo) register() {
	if h.isRegistered {
		return
	}

	hostInfo, err := h.initHostRecord()
	if err != nil {
		h.onFail(errors.Wrap(err, "initHostRecords"))
	}
	defer h.reportHostErrors()

	err = h.initCgroup()
	if err != nil {
		h.onFail(errors.Wrap(err, "initCgroup"))
	}
	err = h.initHostNetworks(hostInfo)
	if err != nil {
		h.onFail(errors.Wrap(err, "initHostNetworks"))
	}
	err = h.initIsolatedDevices()
	if err != nil {
		h.onFail(errors.Wrap(err, "initIsolatedDevices"))
	}
	err = h.initStorages()
	if err != nil {
		h.onFail(errors.Wrap(err, "initStorages"))
	}
	h.deployAdminAuthorizedKeys()
	h.onSucc()
}

func (h *SHostInfo) onFail(reason error) {
	if len(h.HostId) > 0 && !h.isLoged {
		logclient.AddSimpleActionLog(h, logclient.ACT_ONLINE, reason, hostutils.GetComputeSession(context.Background()).GetToken(), false)
		data := jsonutils.NewDict()
		data.Add(jsonutils.NewString(h.GetName()), "name")
		data.Add(jsonutils.NewString(fmt.Sprintf("register failed: %v", reason)), "message")
		notifyclient.SystemExceptionNotify(context.TODO(), napi.ActionSystemException, napi.TOPIC_RESOURCE_HOST, data)
		h.isLoged = true
	}
	log.Errorf("register failed: %s", reason)
	if h.kubeletConfig != nil {
		// run in container, exit
		panic("exit immediately for retry...")
	} else {
		// retry
		log.Errorf("register failed, try 30 seconds later...")
		h.StartRegister(30)
	}
}

func (h *SHostInfo) initHostRecord() (*api.HostDetails, error) {
	wireId, err := h.ensureMasterNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "initHostRecord")
	}
	err = h.waitMasterNicIp()
	if err != nil {
		return nil, errors.Wrap(err, "waitMasterNicIp")
	}

	h.ZoneId, err = h.getZoneByWire(wireId)
	if err != nil {
		return nil, errors.Wrap(err, "getZoneByWire")
	}

	err = h.initZoneInfo(h.ZoneId)
	if err != nil {
		return nil, errors.Wrap(err, "initZoneInfo")
	}

	hostInfo, err := h.ensureHostRecord(h.ZoneId)
	if err != nil {
		return nil, errors.Wrap(err, "ensureHostRecord")
	}

	h.HostId = hostInfo.Id
	hostInfo, err = h.updateHostMetadata(hostInfo.Name)
	if err != nil {
		return nil, errors.Wrap(err, "updateHostMetadata")
	}

	// set auto migrate on host down
	if hostInfo.AutoMigrateOnHostDown {
		if err = h.SetOnHostDown(hostconsts.SHUTDOWN_SERVERS); err != nil {
			return nil, errors.Wrap(err, "failed set on host down")
		}
	}
	log.Infof("host health manager on host down %s", h.onHostDown)

	// fetch host reserved cpus info
	err = h.parseReservedCpusInfo(hostInfo)
	if err != nil {
		return nil, errors.Wrap(err, "parse reserved cpus info")
	}

	// set host reserved memory
	if h.IsHugepagesEnabled() && h.getReservedMemMb() != hostInfo.MemReserved {
		if err = h.updateHostReservedMem(h.getReservedMemMb()); err != nil {
			return nil, errors.Wrap(err, "updateHostReservedMem")
		}
	}
	return hostInfo, nil
}

// try to create network on region.
func (h *SHostInfo) tryCreateNetworkOnWire() (string, error) {
	masterIp, mask := h.GetMasterNicIpAndMask()
	log.Infof("Get master ip %s and mask %d", masterIp, mask)
	if len(masterIp) == 0 || mask == 0 {
		return "", errors.Wrapf(httperrors.ErrInvalidStatus, "master ip %s mask %d", masterIp, mask)
	}
	params := jsonutils.NewDict()
	params.Set("ip", jsonutils.NewString(masterIp))
	params.Set("mask", jsonutils.NewInt(int64(mask)))
	params.Set("is_classic", jsonutils.JSONTrue)
	params.Set("server_type", jsonutils.NewString(api.NETWORK_TYPE_BAREMETAL))
	params.Set("is_on_premise", jsonutils.JSONTrue)
	ret, err := modules.Networks.PerformClassAction(h.GetSession(), "try-create-network", params)
	if err != nil {
		return "", errors.Wrap(err, "try create network")
	}
	if !jsonutils.QueryBoolean(ret, "find_matched", false) {
		return "", errors.Wrap(httperrors.ErrInvalidStatus, "try create network: find_matched == false")
	}
	wireId, err := ret.GetString("wire_id")
	if err != nil {
		return "", errors.Wrap(err, "try create network: get wire_id")
	}
	return wireId, nil
}

func (h *SHostInfo) ensureMasterNetworks() (string, error) {
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		return "", errors.Wrap(httperrors.ErrInvalidStatus, "master ip not found")
	}
	log.Infof("Master ip %s to fetch wire", masterIp)
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
		return "", errors.Wrap(err, "fetch network by master ip")
	}

	var wireId string
	if len(res.Data) == 0 {
		wireId, err = h.tryCreateNetworkOnWire()
	} else if len(res.Data) == 1 {
		wireId, _ = res.Data[0].GetString("wire_id")
	} else {
		err = errors.Wrapf(httperrors.ErrConflict, "find multiple match network (%d) for access network", len(res.Data))
	}

	return wireId, err
}

func (h *SHostInfo) getZoneByWire(wireId string) (string, error) {
	wire, err := hostutils.GetWireInfo(context.Background(), wireId)
	if err != nil {
		return "", errors.Wrap(err, "getWireInfo")
	}
	zoneId, err := wire.GetString("zone_id")
	if err != nil {
		return "", errors.Wrapf(err, "fail to get zone_id in wire info %s", wire)
	}
	return zoneId, nil
}

func (h *SHostInfo) GetSession() *mcclient.ClientSession {
	return hostutils.GetComputeSession(context.Background())
}

func (h *SHostInfo) initZoneInfo(zoneId string) error {
	log.Infof("Start GetZoneInfo %s", zoneId)
	var params = jsonutils.NewDict()
	params.Set("provider", jsonutils.NewString(api.CLOUD_PROVIDER_ONECLOUD))
	res, err := modules.Zones.Get(h.GetSession(), zoneId, params)
	if err != nil {
		return errors.Wrap(err, "Zones.Get")
	}
	zone := api.ZoneDetails{}
	jsonutils.Update(&zone, res)
	h.Zone = zone.Name
	h.ZoneId = zone.Id
	h.Cloudregion = zone.Cloudregion
	h.CloudregionId = zone.CloudregionId
	h.ZoneManagerUri = zone.ManagerUri
	if len(h.Zone) == 0 {
		return errors.Wrapf(httperrors.ErrInvalidStatus, "failed to found zone with id %s", zoneId)
	}

	consts.SetZone(zone.Name)
	return nil
}

func (h *SHostInfo) waitMasterNicIp() error {
	const maxWaitSeconds = 900
	waitSeconds := 0
	for h.MasterNic.Addr == "" && waitSeconds < maxWaitSeconds {
		time.Sleep(time.Second)
		waitSeconds++
		h.MasterNic.FetchConfig()
	}
	if h.MasterNic.Addr == "" {
		return errors.Wrap(httperrors.ErrInvalidStatus, "fail to fetch master nic IP address")
	}
	if h.MasterNic.GetMac() == "" {
		return errors.Wrap(httperrors.ErrInvalidStatus, "fail to fetch master nic MAC address")
	}

	return nil
}

func (h *SHostInfo) ensureHostRecord(zoneId string) (*api.HostDetails, error) {
	allMasterMacs := h.MasterNic.GetAllMacs()
	log.Infof("Master MAC: %s", strings.Join(allMasterMacs, ","))

	params := jsonutils.NewDict()
	params.Set("provider", jsonutils.NewString(api.CLOUD_PROVIDER_ONECLOUD))
	params.Set("details", jsonutils.JSONTrue)
	params.Set("scope", jsonutils.NewString("system"))

	hosts := []api.HostDetails{}
	for _, masterMac := range allMasterMacs {
		params.Set("any_mac", jsonutils.NewString(masterMac))
		res, err := modules.Hosts.List(h.GetSession(), params)
		if err != nil {
			return nil, errors.Wrap(err, "Hosts.List")
		}
		if len(res.Data) > 0 {
			jsonutils.Update(&hosts, res.Data)
			break
		}
	}

	if len(hosts) > 1 {
		for i := range hosts {
			h.HostId = hosts[i].Id
			logclient.AddSimpleActionLog(h, logclient.ACT_ONLINE, fmt.Errorf("duplicate host with %s", params), hostutils.GetComputeSession(context.Background()).GetToken(), false)
		}
		return nil, errors.Wrapf(httperrors.ErrConflict, "find multiple hosts match access mac %s", strings.Join(allMasterMacs, ","))
	}

	h.HostId = ""
	if len(hosts) == 1 {
		h.Domain_id = hosts[0].DomainId
		h.HostId = hosts[0].Id
		h.Project_domain = strings.ReplaceAll(hosts[0].ProjectDomain, " ", "+")

		// 上次未能正常offline, 补充一次健康日志
		if hosts[0].HostStatus == api.HOST_ONLINE {
			reason := "The host status is online when it staring. Maybe the control center was down earlier"
			logclient.AddSimpleActionLog(h, logclient.ACT_HEALTH_CHECK, map[string]string{"reason": reason}, hostutils.GetComputeSession(context.Background()).GetToken(), false)
			data := jsonutils.NewDict()
			data.Add(jsonutils.NewString(h.GetName()), "name")
			data.Add(jsonutils.NewString(reason), "message")
			notifyclient.SystemExceptionNotify(context.TODO(), napi.ActionSystemException, napi.TOPIC_RESOURCE_HOST, data)
		}
	}

	return h.updateOrCreateHost(h.HostId)
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

func (h *SHostInfo) updateOrCreateHost(hostId string) (*api.HostDetails, error) {
	if len(hostId) == 0 {
		h.isInit = true
	} else {
		h.isInit = false
	}
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "master ip is none")
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
	if h.IsHugepagesEnabled() {
		pageSizeKb := options.HostOptions.HugepageSizeMb * 1024
		input.PageSizeKB = &pageSizeKb
	} else {
		pageSizeKb := 4
		input.PageSizeKB = &pageSizeKb
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

	if !options.HostOptions.DisableLocalVpc {
		input.OvnVersion = ovnutils.MustGetOvnVersion()
	}

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
		return nil, errors.Wrapf(err, "host create or update with %s", jsonutils.Marshal(input))
	}

	hostDetails := api.HostDetails{}
	err = res.Unmarshal(&hostDetails)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal host details failed")
	}

	return &hostDetails, nil
}

func json2HostDetails(res jsonutils.JSONObject) (*api.HostDetails, error) {
	hostDetails := api.HostDetails{}
	err := res.Unmarshal(&hostDetails)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &hostDetails, nil
}

func (h *SHostInfo) updateHostMetadata(hostname string) (*api.HostDetails, error) {
	onK8s, _ := tokens.IsInsideKubernetesCluster()
	meta := api.HostRegisterMetadata{
		OnKubernetes: onK8s,
		Hostname:     hostname,
	}
	if len(h.SysError) > 0 {
		meta.SysError = jsonutils.Marshal(h.SysError).String()
	}
	meta.RootPartitionTotalCapacityMB = int64(storageman.GetRootPartTotalCapacity())
	meta.RootPartitionUsedCapacityMB = int64(storageman.GetRootPartUsedCapacity())
	data := meta.JSON(meta)
	res, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, data)
	if err != nil {
		return nil, errors.Wrap(err, "SetMetadata")
	}
	return json2HostDetails(res)
}

// func (h *SHostInfo) SyncRootPartitionUsedCapacity() error {
//	data := jsonutils.NewDict()
//	data.Set("root_partition_used_capacity_mb", jsonutils.NewInt(int64(storageman.GetRootPartUsedCapacity())))
//	_, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, data)
//	return err
// }

func (h *SHostInfo) SetOnHostDown(action string) error {
	h.onHostDown = action
	return fileutils2.FilePutContents(path.Join(options.HostOptions.ServersPath, hostconsts.HOST_HEALTH_FILENAME), h.onHostDown, false)
}

func (h *SHostInfo) parseReservedCpusInfo(hostInfo *api.HostDetails) error {
	reservedCpusStr := hostInfo.Metadata[api.HOSTMETA_RESERVED_CPUS_INFO]
	if reservedCpusStr != "" {
		reservedCpusJson, err := jsonutils.ParseString(reservedCpusStr)
		if err != nil {
			return errors.Wrap(err, "parse reserved cpus info failed")
		}
		reservedCpusInfo := api.HostReserveCpusInput{}
		err = reservedCpusJson.Unmarshal(&reservedCpusInfo)
		if err != nil {
			return errors.Wrap(err, "unmarshal host reserved cpus info failed")
		}
		h.reservedCpusInfo = &reservedCpusInfo
	}
	return nil
}

func (h *SHostInfo) updateHostReservedMem(reserved int) error {
	content := jsonutils.NewDict()
	content.Set("mem_reserved", jsonutils.NewInt(int64(reserved)))
	_, err := modules.Hosts.Update(h.GetSession(), h.HostId, content)
	if err != nil {
		return errors.Wrap(err, "Update mem_reserved")
	}
	return nil
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
	if h.IsHugepagesEnabled() {
		hp, _ := h.Mem.GetHugepages()
		return h.GetMemory() - int(hp.BytesMb())
	} else {
		return h.getOSReservedMemMb() + h.getKubeReservedMemMb()
	}
}

func (h *SHostInfo) PutHostOnline() error {
	if len(h.SysError) > 0 {
		log.Errorf("Host sys error: %v", h.SysError)
	}

	data := jsonutils.NewDict()
	_, err := modules.Hosts.PerformAction(
		h.GetSession(), h.HostId, api.HOST_ONLINE, data)
	if err != nil {
		logclient.AddSimpleActionLog(h, logclient.ACT_ONLINE, data, hostutils.GetComputeSession(context.Background()).GetToken(), false)
	}
	return err
}

func (h *SHostInfo) initHostNetworks(hostInfo *api.HostDetails) error {
	err := h.ensureNicsHostwires(hostInfo)
	if err != nil {
		return errors.Wrap(err, "ensureNicsHostwires")
	}
	err = h.uploadNetworkInfo()
	if err != nil {
		return errors.Wrap(err, "uploadNetworkInfo")
	}
	return nil
}

func (h *SHostInfo) ensureNicsHostwires(hostInfo *api.HostDetails) error {
	for _, nicInfo := range hostInfo.NicInfo {
		if len(nicInfo.WireId) == 0 {
			// no wire info, ignore
			continue
		}
		nic := h.getMatchNic(nicInfo.Mac, nicInfo.VlanId)
		if nic != nil {
			if nicInfo.Bandwidth < 1 {
				nicInfo.Bandwidth = 1000
			}
			err := nic.SetWireId(nicInfo.Wire, nicInfo.WireId, int64(nicInfo.Bandwidth))
			if err != nil {
				return errors.Wrap(err, "SetWireId")
			}
		} else {
			log.Warningf("NIC not present %s", jsonutils.Marshal(nic).String())
		}
	}
	return nil
}

func (h *SHostInfo) isVirtualFunction(nic string) bool {
	physPortName, err := fileutils2.FileGetContents(path.Join("/sys/class/net", nic, "phys_port_name"))
	if err != nil {
		log.Warningf("failed get nic %s phys_port_name: %s", nic, err)
		return false
	}
	if strings.Contains(physPortName, "vf") {
		log.Infof("nic %s is virtual function", nic)
		return true
	}
	log.Infof("nic %s is not virtual function", nic)
	return false
}

func (h *SHostInfo) uploadNetworkInfo() error {
	phyNics, err := sysutils.Nics()
	if err != nil {
		return errors.Wrap(err, "parse physical nics info")
	}
	for _, pnic := range phyNics {
		if h.isVirtualFunction(pnic.Dev) {
			continue
		}
		nic := h.getMatchNic(pnic.Mac.String(), 1)
		if nic != nil {
			// no need to report managed NIC
			continue
		}
		// only report unmanaged physical NIC
		err := h.doSendPhysicalNicInfo(pnic)
		if err != nil {
			return errors.Wrapf(err, "doSendPhysicalNicInfo %s", pnic.Dev)
		}
	}

	var hostDetails *api.HostDetails
	for _, nic := range h.Nics {
		if len(nic.WireId) == 0 {
			// nic info not uploaded yet
			if len(nic.Wire) == 0 {
				// no wire defined, find from region
				kwargs := jsonutils.NewDict()
				kwargs.Set("ip", jsonutils.NewString(nic.Ip))
				kwargs.Set("is_classic", jsonutils.JSONTrue)
				kwargs.Set("scope", jsonutils.NewString("system"))
				kwargs.Set("limit", jsonutils.NewInt(0))

				wireInfo, err := hostutils.GetWireOfIp(context.Background(), kwargs)
				if err != nil {
					return errors.Wrapf(err, "GetWireOfIp args: %s", kwargs.String())
				}
				nic.Wire, _ = wireInfo.GetString("name")
				hostDetails, err = h.doUploadNicInfo(nic)
				if err != nil {
					return errors.Wrapf(err, "doUploadNicInfo with ip %s", nic.Inter)
				}
			} else {
				// no ip on interface, wire defined
				hostDetails, err = h.doUploadNicInfo(nic)
				if err != nil {
					return errors.Wrapf(err, "doUploadNicInfo with wire %s", nic.Inter)
				}
			}
		} else {
			// already uploaded, redo add-nic
			hostDetails, err = h.doUploadNicInfo(nic)
			if err != nil {
				return errors.Wrapf(err, "doSyncNicInfo %s", nic.Inter)
			}
		}
	}

	if hostDetails != nil {
		err = h.ensureNicsHostwires(hostDetails)
		if err != nil {
			return errors.Wrap(err, "onGetHostNetworkInfo")
		}
	}

	return nil
}

func (h *SHostInfo) doSendPhysicalNicInfo(nic *types.SNicDevInfo) error {
	log.Infof("upload physical nic: %s(%s)", nic.Dev, nic.Mac)
	_, err := h.doUploadNicInfoInternal(nic.Dev, nic.Mac.String(), 1, "", "", "", nic.Up != nil && *nic.Up)
	if err != nil {
		return errors.Wrap(err, "doUploadNicInfoInternal")
	}
	return nil
}

func (h *SHostInfo) doUploadNicInfo(nic *SNIC) (*api.HostDetails, error) {
	hostDetails, err := h.doUploadNicInfoInternal(nic.Inter, nic.BridgeDev.GetMac(), nic.BridgeDev.GetVlanId(), nic.Wire, nic.Bridge, nic.Ip, true)
	if err != nil {
		return nil, errors.Wrap(err, "doUploadNicInfoInternal")
	}
	return hostDetails, nil
}

func (h *SHostInfo) doUploadNicInfoInternal(ifname, mac string, vlanId int, wire, bridge, ipaddr string, isUp bool) (*api.HostDetails, error) {
	log.Infof("Upload NIC br:%s if:%s", bridge, ifname)
	content := jsonutils.NewDict()
	content.Set("mac", jsonutils.NewString(mac))
	content.Set("vlan_id", jsonutils.NewInt(int64(vlanId)))
	content.Set("wire", jsonutils.NewString(wire))
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
			content.Set("nic_type", jsonutils.NewString(string(api.NIC_TYPE_ADMIN)))
		}
		// always try to allocate from reserved pool
		content.Set("reserve", jsonutils.JSONTrue)
	}
	res, err := modules.Hosts.PerformAction(h.GetSession(), h.HostId, "add-netif", content)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Hosts.PerformAction add-netif")
	}

	return json2HostDetails(res)
}

/*func (h *SHostInfo) doSyncNicInfo(nic *SNIC) error {
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
}*/

/*func (h *SHostInfo) onUploadNicInfoSucc(nic *SNIC) error {
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
}*/

func (h *SHostInfo) initStorages() error {
	err := h.initLocalStorageImageManager()
	if err != nil {
		return errors.Wrap(err, "init local storage image ")
	}
	hoststorages, err := h.getStorageInfo()
	if err != nil {
		return errors.Wrap(err, "get storage info")
	}
	h.initStoragesInternal(hoststorages)
	return nil
}

func (h *SHostInfo) initLocalStorageImageManager() error {
	localImageCachePath := storageman.GetManager().LocalStorageImagecacheManager.GetPath()
	params := jsonutils.NewDict()
	params.Set("external_id", jsonutils.NewString(h.HostId))
	params.Set("path", jsonutils.NewString(localImageCachePath))
	res, err := modules.Storagecaches.List(h.GetSession(), params)
	if err != nil {
		return errors.Wrap(err, "Storagecaches.List")
	}

	var scid string
	if len(res.Data) == 0 {
		// create local storage cache
		body := jsonutils.NewDict()
		body.Set("name", jsonutils.NewString(fmt.Sprintf(
			"local-%s-%s", h.FullName, time.Now().String())))
		body.Set("path", jsonutils.NewString(localImageCachePath))
		body.Set("external_id", jsonutils.NewString(h.HostId))
		sc, err := modules.Storagecaches.Create(h.GetSession(), body)
		if err != nil {
			return errors.Wrap(err, "Storagecaches.Create")
		}
		scid, _ = sc.GetString("id")
	} else {
		scid, _ = res.Data[0].GetString("id")

	}
	storageman.GetManager().LocalStorageImagecacheManager.SetStoragecacheId(scid)
	return nil
}

func (h *SHostInfo) getStorageInfo() ([]jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.Hoststorages.ListDescendent(h.GetSession(), h.HostId, params)
	if err != nil {
		return nil, errors.Wrap(err, "Hoststorages.ListDescendent")
	} else {
		return res.Data, nil
	}
}

func (h *SHostInfo) initStoragesInternal(hoststorages []jsonutils.JSONObject) {
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
					h.AppendError(err.Error(), "storages", storageId, storageName)
					continue
				}
				if err := storage.Accessible(); err != nil {
					h.AppendError(fmt.Sprintf("check storage accessible failed: %s", err.Error()),
						"storages", storageId, storageName)
					continue
				}
				storageManager.Storages = append(storageManager.Storages, storage)
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
						h.AppendError(
							fmt.Sprintf("Request update host storage %s with params %s: %s", storageId, params, err),
							"storages", storageId, storageName)
					}
				}
				if err := storage.SetStorageInfo(storageId, storageName, storageConf); err != nil {
					h.AppendError(
						fmt.Sprintf("Set storage info %s/%s/%s failed: %s", storageId, storageName, storageConf, err),
						"storages", storageId, storageName)
					continue
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

	for _, s := range storageman.GetManager().Storages {
		if !s.IsLocal() {
			// only local storage need to do the sync
			continue
		}
		storageId := s.GetId()
		storageName := s.GetStorageName()
		storageConf := s.GetStorageConf()
		if err := s.SetStorageInfo(s.GetId(), s.GetStorageName(), s.GetStorageConf()); err != nil {
			h.AppendError(fmt.Sprintf("Set storage info %s/%s/%s failed: %s", storageId, storageName, storageConf, err.Error()),
				"storages", storageId, storageName)
			continue
		}
		res, err := s.SyncStorageInfo()
		if err != nil {
			h.AppendError(fmt.Sprintf("sync storage %s failed: %s", s.GetStorageName(), err.Error()), "storages", storageId, storageName)
			continue
		}
		{
			err = h.onSyncStorageInfoSucc(s, res)
			if err != nil {
				h.AppendError(err.Error(), "storages", storageId, storageName)
				continue
			}
		}
	}
}

func (h *SHostInfo) onSyncStorageInfoSucc(storage storageman.IStorage, storageInfo jsonutils.JSONObject) error {
	log.Infof("storage id %s", storage.GetId())
	if len(storage.GetId()) == 0 {
		log.Errorf("storage config %s", storageInfo)
		id, _ := storageInfo.GetString("id")
		name, _ := storageInfo.GetString("name")
		storageConf, _ := storageInfo.Get("storage_conf")
		if err := storage.SetStorageInfo(id, name, storageConf); err != nil {
			return errors.Wrapf(err, "Set storage info %s/%s/%s failed", id, name, storageConf)
		}
		err := h.attachStorage(storage)
		if err != nil {
			return errors.Wrap(err, "attachStorage")
		}
	}
	return nil
}

func (h *SHostInfo) attachStorage(storage storageman.IStorage) error {
	content := jsonutils.NewDict()
	content.Set("mount_point", jsonutils.NewString(storage.GetPath()))
	content.Set("is_root_partition", jsonutils.NewBool(IsRootPartition(storage.GetPath())))
	_, err := modules.Hoststorages.Attach(h.GetSession(), h.HostId, storage.GetId(), content)
	if err != nil {
		return errors.Wrap(err, "Hoststorages.Attach")
	}
	return nil
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

func (h *SHostInfo) initIsolatedDevices() error {
	info, err := h.probeSyncIsolatedDevices()
	if err != nil {
		return errors.Wrap(err, "probeSyncIsolatedDevices")
	}
	log.Infof("probeSyncIsolatedDevices %s", info)
	return nil
}

func (h *SHostInfo) getNicsInterfaces(nics []string) []isolated_device.HostNic {
	if len(nics) == 0 {
		return nil
	}
	log.Infof("sriov input nics %v", nics)
	res := []isolated_device.HostNic{}
	for i := 0; i < len(nics); i++ {
		found := false
		for j := 0; j < len(h.Nics); j++ {
			if nics[i] == h.Nics[j].Inter {
				found = true
				res = append(res, isolated_device.HostNic{h.Nics[j].Bridge, h.Nics[j].Inter, h.Nics[j].WireId})
			}
		}
		if !found {
			res = append(res, isolated_device.HostNic{h.Nics[0].Bridge, nics[i], h.Nics[0].WireId})
		}
	}
	log.Infof("sriov output nics %v", res)
	return res
}

func (h *SHostInfo) getNicsOvsOffloadInterfaces(nics []string) ([]isolated_device.HostNic, error) {
	if len(nics) == 0 {
		return nil, nil
	}

	res := []isolated_device.HostNic{}
	for i := 0; i < len(h.Nics); i++ {
		if utils.IsInStringArray(h.Nics[i].Inter, nics) {
			if fileutils2.Exists(fmt.Sprintf("/sys/class/net/%s/bonding/slaves", h.Nics[i].Inter)) {
				interStr, err := fileutils2.FileGetContents(fmt.Sprintf("/sys/class/net/%s/bonding/slaves", h.Nics[i].Inter))
				if err != nil {
					return nil, err
				}
				inters := strings.Split(strings.TrimSpace(interStr), " ")
				for _, inter := range inters {
					res = append(res, isolated_device.HostNic{
						Bridge:    h.Nics[i].Bridge,
						Interface: inter,
						Wire:      h.Nics[i].WireId,
					})
				}
			} else {
				res = append(res, isolated_device.HostNic{
					Bridge:    h.Nics[i].Bridge,
					Interface: h.Nics[i].Inter,
					Wire:      h.Nics[i].WireId,
				})
			}
		}
	}
	return res, nil
}

func (h *SHostInfo) probeSyncIsolatedDevices() (*jsonutils.JSONArray, error) {
	if !h.IsKvmSupport() {
		// skip probe isolated device on kvm not supported
		log.Errorf("KVM is not supported, skip probe isolated devices")
		return nil, nil
	}

	for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
		if out, err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Output(); err != nil {
			log.Errorf("failed probe driver %s: %s %s", driver, out, err)
		}
	}

	enableDevWhitelist := options.HostOptions.EnableIsolatedDeviceWhitelist

	offloadNics, err := h.getNicsOvsOffloadInterfaces(options.HostOptions.OvsOffloadNics)
	if err != nil {
		return nil, err
	}
	sriovNics := h.getNicsInterfaces(options.HostOptions.SRIOVNics)
	h.IsolatedDeviceMan.ProbePCIDevices(
		options.HostOptions.DisableGPU, options.HostOptions.DisableUSB, options.HostOptions.DisableCustomDevice,
		sriovNics, offloadNics, options.HostOptions.PTNVMEConfigs, options.HostOptions.AMDVgpuPFs, options.HostOptions.NVIDIAVgpuPFs,
		enableDevWhitelist)

	objs, err := h.getRemoteIsolatedDevices()
	if err != nil {
		return nil, errors.Wrap(err, "getRemoteIsolatedDevices")
	}
	for _, obj := range objs {
		info := isolated_device.CloudDeviceInfo{}
		if err := obj.Unmarshal(&info); err != nil {
			return nil, errors.Wrapf(err, "unmarshal isolated device %s to cloud device info", obj)
		}
		dev := h.IsolatedDeviceMan.GetDeviceByIdent(info.VendorDeviceId, info.Addr, info.MdevId)
		if dev != nil {
			dev.SetDeviceInfo(info)
		} else {
			// detach device
			h.IsolatedDeviceMan.AppendDetachedDevice(&info)
		}
	}
	h.IsolatedDeviceMan.StartDetachTask()
	h.IsolatedDeviceMan.BatchCustomProbe()

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
	err := fsdriver.DeployAdminAuthorizedKeys(h.GetSession())
	if err != nil {
		h.AppendHostError(fmt.Sprintf("DeployAdminAuthorizedKeys: %s", err))
	}
}

func (h *SHostInfo) onSucc() {
	if !h.stopped && !h.isRegistered {
		log.Infof("Host registration process success....")
		if err := h.save(); err != nil {
			panic(err.Error())
		}
		h.StartPinger()
		// if h.registerCallback != nil {
		// 	h.registerCallback()
		// }
		h.isRegistered = true

		// Notify caller, host register is success
		// close(h.IsRegistered)
	}
}

func (h *SHostInfo) AppendHostError(content string) {
	h.AppendError(content, "hosts", h.HostId, h.GetName())
}

func (h *SHostInfo) AppendError(content, errType, id, name string) {
	if errType == "" {
		errType = "hosts"
		id = h.HostId
		name = h.GetName()
	}
	es, ok := h.SysError[errType]
	if !ok {
		h.SysError[errType] = make([]api.HostError, 0)
	}
	h.SysError[errType] = append(es, api.HostError{Type: errType, Id: id, Name: name, Content: content, Time: time.Now()})
}

func (h *SHostInfo) RemoveErrorType(errType string) {
	delete(h.SysError, errType)
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
		input := api.HostOfflineInput{
			Reason: "host stop",
		}
		_, err := modules.Hosts.PerformAction(h.GetSession(), h.HostId, api.HOST_OFFLINE, jsonutils.Marshal(input))
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
	tsdb, _ := tsdb.GetDefaultServiceSource(s, defaultEndpointType)
	if tsdb != nil && len(tsdb.URLs) > 0 {
		conf[apis.SERVICE_TYPE_INFLUXDB] = map[string]interface{}{
			"url":       tsdb.URLs,
			"database":  "telegraf",
			"tsdb_type": tsdb.Type,
		}
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
	if h.FullName == "" {
		h.FullName = h.fetchHostname()
	}
	return h.FullName
}

func (h *SHostInfo) GetCpuArchitecture() string {
	return h.Cpu.CpuArchitecture
}

func (h *SHostInfo) GetKernelVersion() string {
	return h.sysinfo.KernelVersion
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

func (h *SHostInfo) GetHostTopology() *hostapi.HostTopology {
	return h.sysinfo.Topology
}

func (h *SHostInfo) GetReservedCpusInfo() *cpuset.CPUSet {
	if h.reservedCpusInfo == nil {
		return nil
	}
	cpus, _ := cpuset.Parse(h.reservedCpusInfo.Cpus)
	return &cpus
}

func (h *SHostInfo) IsContainerdRuning() bool {
	return false
}

func (h *SHostInfo) IsContainerHost() bool {
	//return options.HostOptions.EnableContainerRuntime || options.HostOptions.HostType == api.HOST_TYPE_CONTAINER
	return options.HostOptions.HostType == api.HOST_TYPE_CONTAINER
}

func (h *SHostInfo) GetContainerRuntimeEndpoint() string {
	return options.HostOptions.ContainerRuntimeEndpoint
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
	// res.IsRegistered = make(chan struct{})
	res.SysError = map[string][]api.HostError{}

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
