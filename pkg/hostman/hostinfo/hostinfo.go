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
	"net"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/host_health"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/kubelet"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/k8s/tokens"
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

	saved  bool
	pinger *SHostPingTask

	Cpu     *SCPUInfo
	Mem     *SMemory
	sysinfo *SSysInfo

	kubeletConfig kubelet.KubeletConfig

	isInit          bool
	enableHugePages bool
	onHostDown      string

	IsolatedDeviceMan *isolated_device.IsolatedDeviceManager

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

func (h *SHostInfo) GetIsolatedDeviceManager() *isolated_device.IsolatedDeviceManager {
	return h.IsolatedDeviceMan
}

func (h *SHostInfo) GetBridgeDev(bridge string) hostbridge.IBridgeDriver {
	for _, n := range h.Nics {
		if bridge == n.Bridge {
			return n.BridgeDev
		}
	}
	if bridge == options.HostOptions.OvnIntegrationBridge {
		drv, err := hostbridge.NewOVSBridgeDriverByName(bridge)
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

func (h *SHostInfo) GetZoneName() string {
	return h.Zone
}

func (h *SHostInfo) GetMediumType() string {
	if h.sysinfo != nil {
		return h.sysinfo.StorageType
	}
	return ""
}

func (h *SHostInfo) IsKvmSupport() bool {
	return sysutils.IsKvmSupport()
}

func (h *SHostInfo) IsNestedVirtualization() bool {
	return utils.IsInStringArray("hypervisor", h.Cpu.cpuFeatures)
}

func (h *SHostInfo) IsHugepagesEnabled() bool {
	return h.enableHugePages || options.HostOptions.HugepagesOption == "native"
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
		return err
	}

	if err := hostbridge.Prepare(options.HostOptions.BridgeDriver); err != nil {
		log.Errorln(err)
		return err
	}

	log.Infof("Start parseConfig")
	if err := h.parseConfig(); err != nil {
		return err
	}
	if HasOvnSupport() {
		if err := h.setupOvnChassis(); err != nil {
			return err
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
	if mem, err := h.GetMemory(); err != nil {
		return err
	} else if mem < 64 { // MB
		return fmt.Errorf("Not enough memory!")
	}
	if len(options.HostOptions.Networks) == 0 {
		netConf, err := h.generateLocalNetworkConfig()
		if err != nil {
			return err
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
			return err
		}
		h.Nics = append(h.Nics, nic)
	}
	for i := 0; i < len(h.Nics); i++ {
		if err := h.Nics[i].SetupDhcpRelay(); err != nil {
			return err
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

	if man, err := isolated_device.NewManager(h); err != nil {
		return fmt.Errorf("NewIsolatedManager: %v", err)
	} else {
		h.IsolatedDeviceMan = man
	}

	return nil
}

func (h *SHostInfo) prepareEnv() error {
	if err := h.fixPathEnv(); err != nil {
		return errors.Wrap(err, "Fix path environment")
	}
	if options.HostOptions.ReportInterval > 300 {
		return fmt.Errorf("Option report_interval must no longer than 5 min")
	}

	output, err := procutils.NewCommand("mkdir", "-p", options.HostOptions.ServersPath).Output()
	if err != nil {
		return errors.Wrapf(err, "failed to create path %s: %s", options.HostOptions.ServersPath, output)
	}

	_, err = procutils.NewCommand("ethtool", "-h").Output()
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
	output, err = procutils.NewRemoteCommandAsFarAsPossible("modprobe", "vhost_net").Output()
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

	if options.HostOptions.EnableKsm {
		h.EnableKsm(900)
	} else {
		h.DisableKsm()
	}

	switch options.HostOptions.HugepagesOption {
	case "disable":
		h.DisableHugepages()
	case "native":
		size, err := h.Mem.GetHugepageTotal()
		if err != nil {
			return err
		}
		if size <= 0 {
			return errors.New("invalid hugepages total size")
		}
	case "transparent":
		h.EnableTransparentHugepages()
	default:
		return fmt.Errorf("Invalid hugepages option")
	}

	h.PreventArpFlux()
	h.TuneSystem()
	return nil
}

func (h *SHostInfo) detectHostInfo() error {
	output, err := procutils.NewCommand("dmidecode", "-t", "1").Output()
	if err != nil {
		return err
	}

	sysinfo, err := sysutils.ParseDMISysinfo(strings.Split(string(output), "\n"))
	if err != nil {
		return err
	}
	h.sysinfo.SSystemInfo = sysinfo

	h.detectKvmModuleSupport()
	h.detectNestSupport()

	if err := h.detectSyssoftwareInfo(); err != nil {
		return err
	}

	h.detectStorageSystem()

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
	for _, srv := range []string{"ntpd"} {
		srvinst := system_service.GetService(srv)
		funcEn(srv, srvinst)
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
	var stype = api.DISK_TYPE_ROTATE
	if options.HostOptions.DiskIsSsd {
		stype = api.DISK_TYPE_SSD
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

func (h *SHostInfo) GetMemory() (int, error) {
	if options.HostOptions.HugepagesOption == "native" {
		return h.Mem.GetHugepageTotal()
	}
	total := h.Mem.Total
	if h.kubeletConfig != nil {
		memThreshold := h.kubeletConfig.GetEvictionConfig().GetHard().GetMemoryAvailable()
		memBytes, _ := memThreshold.Value.Quantity.AsInt64()
		memMb := int(memBytes / 1024 / 1024)
		subMem := total - memMb
		log.Infof("Get total memory %d, kubelet memory threshold subtracted: (%d - %d)", subMem, total, memMb)
		total = subMem
	}
	return total, nil // - options.reserved_memory
}

func (h *SHostInfo) getCurrentHugepageNr() (int64, error) {
	nrStr, err := fileutils2.FileGetContents("/proc/sys/vm/nr_hugepages")
	if err != nil {
		return 0, errors.Wrap(err, "file get content nr hugepages")
	}
	nr, err := strconv.Atoi(strings.TrimSpace(nrStr))
	if err != nil {
		return 0, errors.Wrap(err, "nr str atoi")
	}
	return int64(nr), nil
}

func (h *SHostInfo) EnableNativeHugepages() error {
	kv := map[string]string{
		"/sys/kernel/mm/transparent_hugepage/enabled": "never",
		"/sys/kernel/mm/transparent_hugepage/defrag":  "never",
	}
	for k, v := range kv {
		sysutils.SetSysConfig(k, v)
	}
	nr, err := h.getCurrentHugepageNr()
	if err != nil {
		return err
	}
	mem, err := h.GetMemory()
	if err != nil {
		return err
	}
	mem -= h.getReservedMem()
	desiredNr := int64(mem/h.Mem.GetHugepagesizeMb() + 1)
	if nr < desiredNr {
		err = timeutils2.CommandWithTimeout(1, "sh", "-c",
			fmt.Sprintf("echo %d > /proc/sys/vm/nr_hugepages", desiredNr)).Run()
		if err != nil {
			return err
		}
	}
	currentNr, err := h.getCurrentHugepageNr()
	if err != nil {
		return err
	}
	if currentNr < desiredNr {
		err = timeutils2.CommandWithTimeout(1, "sh", "-c",
			fmt.Sprintf("echo %d > /proc/sys/vm/nr_hugepages", nr)).Run()
		return fmt.Errorf("no enough memory to resize hugepage, current nr %d, desired nr %d", currentNr, desiredNr)
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
func (h *SHostInfo) TuneSystem() {
	kv := map[string]string{"/proc/sys/vm/swappiness": "0",
		"/sys/module/kvm/parameters/ignore_msrs":         "1",
		"/sys/module/kvm/parameters/report_ignored_msrs": "0",
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
		system_service.SetOpenvswitchName("openvswitch-switch")
	}
}

func (h *SHostInfo) detectKernelVersion() {
	out, err := procutils.NewCommand("uname", "-r").Output()
	if err != nil {
		log.Errorln(err)
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
	}
	if !jsonutils.QueryBoolean(ret, "find_matched", false) {
		h.onFail("try create network: find_matched == false")
	}
	wireId, err := ret.GetString("wire_id")
	if err != nil {
		h.onFail(fmt.Sprintf("try create network: get wire_id: %v", err))
	}
	h.onGetWireId(wireId)
}

func (h *SHostInfo) fetchAccessNetworkInfo() {
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		h.onFail("master ip not found")
	}
	log.Debugf("Master ip %s to fetch wire", masterIp)
	params := jsonutils.NewDict()
	params.Set("ip", jsonutils.NewString(masterIp))
	params.Set("is_classic", jsonutils.JSONTrue)
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("limit", jsonutils.NewInt(0))
	// use default vpc
	params.Set("vpc", jsonutils.NewString(api.DEFAULT_VPC_ID))

	res, err := modules.Networks.List(h.GetSession(), params)
	if err != nil {
		h.onFail(err)
	}
	if len(res.Data) == 0 {
		h.tryCreateNetworkOnWire()
	} else if len(res.Data) == 1 {
		wireId, _ := res.Data[0].GetString("wire_id")
		h.onGetWireId(wireId)
	} else {
		h.onFail("Fail to get network info: no networks")
	}
}

func (h *SHostInfo) onGetWireId(wireId string) {
	wire, err := hostutils.GetWireInfo(context.Background(), wireId)
	if err != nil {
		h.onFail(err)
	}
	h.ZoneId, err = wire.GetString("zone_id")
	if err != nil {
		h.onFail(err)
	} else {
		h.getZoneInfo(h.ZoneId, false)
	}
}

func (h *SHostInfo) GetSession() *mcclient.ClientSession {
	return hostutils.GetComputeSession(context.Background())
}

func (h *SHostInfo) getZoneInfo(zoneId string, standalone bool) {
	log.Debugf("Start GetZoneInfo %s %v", zoneId, standalone)
	var params = jsonutils.NewDict()
	params.Set("standalone", jsonutils.NewBool(standalone))
	res, err := modules.Zones.Get(h.GetSession(),
		zoneId, params)
	if err != nil {
		h.onFail(err)
	}

	h.Zone, _ = res.GetString("name")
	h.ZoneId, _ = res.GetString("id")
	h.Cloudregion, _ = res.GetString("cloudregion")
	h.CloudregionId, _ = res.GetString("cloudregion_id")
	if res.Contains("manager_uri") {
		h.ZoneManagerUri, _ = res.GetString("manager_uri")
	}
	if !standalone {
		h.getHostInfo(h.ZoneId)
	}
}

func (h *SHostInfo) getHostInfo(zoneId string) {
	masterMac := h.getMasterMacWithRefresh(true)
	if len(masterMac) == 0 {
		h.onFail("master mac not found")
	}
	params := jsonutils.NewDict()
	params.Set("any_mac", jsonutils.NewString(masterMac))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.Hosts.List(h.GetSession(), params)
	if err != nil {
		h.onFail(err)
	}
	if len(res.Data) == 0 {
		h.updateHostRecord("")
	} else {
		host := res.Data[0]
		id, _ := host.GetString("id")
		h.getDomainInfo(id)

		h.updateHostRecord(id)
	}
}

func (h *SHostInfo) getDomainInfo(hostId string) {
	host, err := modules.Hosts.GetById(h.GetSession(), hostId, jsonutils.NewDict())
	if err != nil {
		h.onFail(err)
	}
	domain_id, _ := host.GetString("domain_id")
	project_domain, _ := host.GetString("project_domain")
	h.Domain_id = domain_id
	h.Project_domain = strings.ReplaceAll(project_domain, " ", "+")
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
	content := jsonutils.NewDict()
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		h.onFail("master ip is none")
	}

	if len(hostId) == 0 {
		content.Set("generate_name", jsonutils.NewString(h.fetchHostname()))
	}
	content.Set("access_ip", jsonutils.NewString(masterIp))
	content.Set("access_mac", jsonutils.NewString(h.GetMasterMac()))
	var schema = "http"
	if options.HostOptions.EnableSsl {
		schema = "https"
	}
	content.Set("manager_uri", jsonutils.NewString(fmt.Sprintf("%s://%s:%d",
		schema, masterIp, options.HostOptions.Port)))
	content.Set("cpu_count", jsonutils.NewInt(int64(h.Cpu.cpuInfoProc.Count)))
	if sysutils.IsHypervisor() {
		content.Set("node_count", jsonutils.NewInt(1))
	} else {
		content.Set("node_count", jsonutils.NewInt(int64(h.Cpu.cpuInfoDmi.Nodes)))
	}
	content.Set("cpu_desc", jsonutils.NewString(h.Cpu.cpuInfoProc.Model))
	content.Set("cpu_microcode", jsonutils.NewString(h.Cpu.cpuInfoProc.Microcode))
	content.Set("cpu_architecture", jsonutils.NewString(h.Cpu.CpuArchitecture))
	if h.Cpu.cpuInfoProc.Freq > 0 {
		content.Set("cpu_mhz", jsonutils.NewInt(int64(h.Cpu.cpuInfoProc.Freq)))
	}
	content.Set("cpu_cache", jsonutils.NewInt(int64(h.Cpu.cpuInfoProc.Cache)))
	memTotal, err := h.GetMemory()
	if err != nil {
		h.onFail(err)
	}
	content.Set("mem_size", jsonutils.NewInt(int64(memTotal)))
	if len(hostId) == 0 {
		content.Set("mem_reserved", jsonutils.NewInt(int64(h.getReservedMem())))
	}
	content.Set("storage_driver", jsonutils.NewString(api.DISK_DRIVER_LINUX))
	content.Set("storage_type", jsonutils.NewString(h.sysinfo.StorageType))
	content.Set("storage_size", jsonutils.NewInt(int64(storageman.GetManager().GetTotalCapacity())))

	// TODO optimize content data struct
	content.Set("sys_info", jsonutils.Marshal(h.sysinfo))
	content.Set("sn", jsonutils.NewString(h.sysinfo.SN))
	content.Set("host_type", jsonutils.NewString(options.HostOptions.HostType))
	if len(options.HostOptions.Rack) > 0 {
		content.Set("rack", jsonutils.NewString(options.HostOptions.Rack))
	}
	if len(options.HostOptions.Slots) > 0 {
		content.Set("slots", jsonutils.NewString(options.HostOptions.Slots))
	}
	content.Set("__meta__", jsonutils.Marshal(h.getSysInfo()))
	content.Set("version", jsonutils.NewString(version.GetShortString()))
	content.Set("ovn_version", jsonutils.NewString(MustGetOvnVersion()))

	var (
		res jsonutils.JSONObject
	)
	if !h.isInit {
		res, err = modules.Hosts.Update(h.GetSession(), hostId, content)
	} else {
		res, err = modules.Hosts.CreateInContext(h.GetSession(), content, &modules.Zones, h.ZoneId)
	}
	if err != nil {
		h.onFail(err)
	} else {
		h.onUpdateHostInfoSucc(res)
	}
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

func (h *SHostInfo) SyncRootPartitionUsedCapacity() error {
	data := jsonutils.NewDict()
	data.Set("root_partition_used_capacity_mb", jsonutils.NewInt(int64(storageman.GetRootPartUsedCapacity())))
	_, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, data)
	return err
}

func (h *SHostInfo) onUpdateHostInfoSucc(hostbody jsonutils.JSONObject) {
	h.HostId, _ = hostbody.GetString("id")
	hostname, _ := hostbody.GetString("name")
	if err := h.updateHostMetadata(hostname); err != nil {
		h.onFail(err)
		return
	}

	if options.HostOptions.HugepagesOption == "native" {
		if h.isInit && len(h.IsolatedDeviceMan.Devices) > 0 {
			meta := jsonutils.NewDict()
			meta.Set("__enable_hugepages", jsonutils.NewString("true"))
			_, err := modules.Hosts.SetMetadata(h.GetSession(), h.HostId, meta)
			if err != nil {
				h.onFail(fmt.Sprintf("failed "))
			}
			h.enableHugePages = true
		} else if hugepage, _ := hostbody.GetString("metadata", "__enable_hugepages"); hugepage == "true" {
			h.enableHugePages = true
		}
		if h.enableHugePages {
			err := h.EnableNativeHugepages()
			if err != nil {
				h.onFail(err)
			}
		}
	}
	h.onHostDown, _ = hostbody.GetString("metadata", "__on_host_down")

	if memReserved, _ := hostbody.Int("mem_reserved"); memReserved == 0 {
		h.updateHostReservedMem()
	} else {
		h.PutHostOffline()
	}
}

func (h *SHostInfo) updateHostReservedMem() {
	content := jsonutils.NewDict()
	content.Set("mem_reserved", jsonutils.NewInt(int64(h.getReservedMem())))
	res, err := modules.Hosts.Update(h.GetSession(),
		h.HostId, content)
	if err != nil {
		h.onFail(err)
	} else {
		h.onUpdateHostInfoSucc(res)
	}
}

func (h *SHostInfo) getReservedMem() int {
	reserved := h.Mem.MemInfo.Total / 10
	if reserved > options.HostOptions.MaxReservedMemory {
		return options.HostOptions.MaxReservedMemory
	}
	if reserved == 0 {
		panic("memory reserve value is 0, need help")
	}
	return reserved
}

func (h *SHostInfo) PutHostOffline() {
	data := jsonutils.NewDict()
	if options.HostOptions.EnableHealthChecker {
		data.Set("update_health_status", jsonutils.JSONTrue)
	}
	_, err := modules.Hosts.PerformAction(
		h.GetSession(), h.HostId, "offline", data)
	if err != nil {
		h.onFail(err)
	} else {
		h.getNetworkInfo()
	}
}

func (h *SHostInfo) PutHostOnline() error {
	if len(h.SysError) > 0 && !options.HostOptions.StartHostIgnoreSysError {
		log.Fatalf("Can't put host online, unless resolve these problem %v", h.SysError)
	} else if len(h.SysError) > 0 && options.HostOptions.StartHostIgnoreSysError {
		log.Errorf("Host sys error: %v", h.SysError)
	}

	if len(h.SysWarning) > 0 {
		log.Warningf("Host have some hidden problem %v", h.SysWarning)
	}

	data := jsonutils.NewDict()
	if options.HostOptions.EnableHealthChecker && len(options.HostOptions.EtcdEndpoints) > 0 {
		_, err := host_health.InitHostHealthManager(h.HostId, h.onHostDown)
		if err != nil {
			log.Fatalf("Init host health manager failed %s", err)
		}
		data.Set("update_health_status", jsonutils.JSONTrue)
	}

	_, err := modules.Hosts.PerformAction(
		h.GetSession(), h.HostId, "online", data)
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
		h.onFail(err)
	} else {
		for _, hostwire := range res.Data {
			bridge, _ := hostwire.GetString("bridge")
			iface, _ := hostwire.GetString("interface")
			macAddr, _ := hostwire.GetString("mac_addr")
			nic := h.GetMatchNic(bridge, iface, macAddr)
			if nic != nil {
				wire, _ := hostwire.GetString("wire")
				wireId, _ := hostwire.GetString("wire_id")
				bandwidth, err := hostwire.Int("bandwidth")
				if err != nil {
					bandwidth = 1000
				}
				nic.SetWireId(wire, wireId, bandwidth)
			} else {
				log.Warningf("NIC not present %s", hostwire.String())
			}
		}
		h.uploadNetworkInfo()
	}
}

func (h *SHostInfo) uploadNetworkInfo() {
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
					h.onFail(err)
				} else {
					nic.Network, _ = wireInfo.GetString("name")
					h.doUploadNicInfo(nic)
				}

			} else {
				h.doUploadNicInfo(nic)
			}
		} else {
			h.doSyncNicInfo(nic)
		}
	}
	h.getStoragecacheInfo()
}

func (h *SHostInfo) doUploadNicInfo(nic *SNIC) {
	log.Infof("Upload NIC br:%s if:%s", nic.Bridge, nic.Inter)
	content := jsonutils.NewDict()
	content.Set("mac", jsonutils.NewString(nic.BridgeDev.GetMac()))
	content.Set("wire", jsonutils.NewString(nic.Network))
	content.Set("bridge", jsonutils.NewString(nic.Bridge))
	content.Set("interface", jsonutils.NewString(nic.Inter))
	content.Set("link_up", jsonutils.JSONTrue)
	if len(nic.Ip) > 0 {
		content.Set("ip_addr", jsonutils.NewString(nic.Ip))
		if nic.Ip == h.GetMasterIp() {
			content.Set("nic_type", jsonutils.NewString(api.NIC_TYPE_ADMIN))
		}
		// always try to allocate from reserved pool
		content.Set("reserve", jsonutils.JSONTrue)
	}
	_, err := modules.Hosts.PerformAction(h.GetSession(),
		h.HostId, "add-netif", content)
	if err != nil {
		h.onFail(err)
	} else {
		h.onUploadNicInfoSucc(nic)
	}
}

func (h *SHostInfo) doSyncNicInfo(nic *SNIC) {
	content := jsonutils.NewDict()
	content.Set("bridge", jsonutils.NewString(nic.Bridge))
	content.Set("interface", jsonutils.NewString(nic.Inter))
	query := jsonutils.NewDict()
	query.Set("mac_addr", jsonutils.NewString(nic.BridgeDev.GetMac()))
	_, err := modules.Hostwires.Update(h.GetSession(),
		h.HostId, nic.WireId, query, content)
	if err != nil {
		h.onFail(err)
	}
}

func (h *SHostInfo) onUploadNicInfoSucc(nic *SNIC) {
	res, err := modules.Hostwires.Get(h.GetSession(), h.HostId, nic.Network, nil)
	if err != nil {
		h.onFail(err)
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
			h.onFail("GetMatchNic failed!!!")
		}
	}
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
				}
				storageManager.Storages = append(storageManager.Storages, storage)
				if err := storage.Accessible(); err != nil {
					h.onFail(err)
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
					}
				}
				if err := storage.SetStorageInfo(storageId, storageName, storageConf); err != nil {
					h.onFail(errors.Wrapf(err, "Set storage info %s/%s/%s", storageId, storageName, storageConf))
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
		}
		res, err := s.SyncStorageInfo()
		if err != nil {
			h.onFail(errors.Wrapf(err, "Sync storage %s info", s.GetStorageName()))
		} else {
			h.onSyncStorageInfoSucc(s, res)
		}
	}
	go storageman.StartSyncStorageSizeTask(
		time.Duration(options.HostOptions.SyncStorageInfoDurationSecond) * time.Second,
	)
	h.getIsolatedDevices()
}

func (h *SHostInfo) onSyncStorageInfoSucc(storage storageman.IStorage, storageInfo jsonutils.JSONObject) {
	if len(storage.GetId()) == 0 {
		id, _ := storageInfo.GetString("id")
		name, _ := storageInfo.GetString("name")
		storageConf, _ := storageInfo.Get("storage_conf")
		if err := storage.SetStorageInfo(id, name, storageConf); err != nil {
			h.onFail(err)
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
	}
}

func (h *SHostInfo) getIsolatedDevices() {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("host", jsonutils.NewString(h.GetHostId()))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.IsolatedDevices.List(h.GetSession(), params)
	if err != nil {
		h.onFail(fmt.Sprintf("getIsolatedDevices: %v", err))
	}
	h.onGetIsolatedDeviceSucc(res.Data)
}

func (h *SHostInfo) onGetIsolatedDeviceSucc(objs []jsonutils.JSONObject) {
	for _, obj := range objs {
		info := isolated_device.CloudDeviceInfo{}
		if err := obj.Unmarshal(&info); err != nil {
			h.onFail(fmt.Sprintf("unmarshal isolated device to cloud device info failed %s", err))
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
		h.onFail(fmt.Sprintf("Device probe error: %v", err))
	}
	h.uploadIsolatedDevices()
}

func (h *SHostInfo) uploadIsolatedDevices() {
	for _, dev := range h.IsolatedDeviceMan.Devices {
		if err := dev.SyncDeviceInfo(h.GetSession(), h.HostId); err != nil {
			h.onFail(fmt.Sprintf("Sync device %s: %v", dev.String(), err))
		}
	}

	h.deployAdminAuthorizedKeys()
}

func (h *SHostInfo) deployAdminAuthorizedKeys() {
	onErr := func(format string, args ...interface{}) {
		h.onFail(fmt.Sprintf(format, args...))
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
	for {
		_, err := modules.Hosts.PerformAction(
			h.GetSession(), h.HostId, "offline", nil)
		if err != nil {
			log.Errorf("put host offline failed: %s", err)
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}
	h.stopped = true
}

func (h *SHostInfo) OnCatalogChanged(catalog mcclient.KeystoneServiceCatalogV3) {
	// TODO: dynamic probe endpoint type
	defaultEndpointType := options.HostOptions.SessionEndpointType
	if len(defaultEndpointType) == 0 {
		defaultEndpointType = identityapi.EndpointInterfacePublic
	}
	if options.HostOptions.ManageNtpConfiguration {
		ntpd := system_service.GetService("ntpd")
		urls, _ := catalog.GetServiceURLs("ntp", options.HostOptions.Region, "", defaultEndpointType)
		if len(urls) > 0 {
			log.Infof("Get Ntp urls: %v", urls)
		} else {
			urls = []string{"ntp://cn.pool.ntp.org",
				"ntp://0.cn.pool.ntp.org",
				"ntp://1.cn.pool.ntp.org",
				"ntp://2.cn.pool.ntp.org",
				"ntp://3.cn.pool.ntp.org"}
		}
		if !reflect.DeepEqual(ntpd.GetConf(), urls) && !ntpd.IsActive() {
			ntpd.SetConf(urls)
			ntpd.BgReload(map[string]interface{}{"servers": urls})
		}
	}
	telegraf := system_service.GetService("telegraf")
	conf := map[string]interface{}{}
	conf["hostname"] = h.getHostname()
	conf["tags"] = map[string]string{
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
	urls, _ := catalog.GetServiceURLs("kafka", options.HostOptions.Region, "", defaultEndpointType)
	if len(urls) > 0 {
		conf["kafka"] = map[string]interface{}{"brokers": urls, "topic": "telegraf"}
	}
	urls, _ = catalog.GetServiceURLs("influxdb", options.HostOptions.Region, "", defaultEndpointType)
	if len(urls) > 0 {
		conf["influxdb"] = map[string]interface{}{"url": urls, "database": "telegraf"}
	}
	log.Debugf("telegraf config: %s", conf)
	if !reflect.DeepEqual(telegraf.GetConf(), conf) || !telegraf.IsActive() {
		telegraf.SetConf(conf)
		svcs := os.Getenv("HOST_SYSTEM_SERVICES_OFF")
		if !strings.Contains(svcs, "telegraf") {
			telegraf.BgReload(conf)
		} else {
			telegraf.BgReloadConf(conf)
		}
	}

	urls, _ = catalog.GetServiceURLs("elasticsearch",
		options.HostOptions.Region, "zone", defaultEndpointType)
	if len(urls) > 0 {
		conf["elasticsearch"] = map[string]interface{}{"url": urls[0]}
		fluentbit := system_service.GetService("fluentbit")
		if !reflect.DeepEqual(fluentbit.GetConf(), conf) || !fluentbit.IsActive() {
			fluentbit.SetConf(conf)
			fluentbit.BgReload(conf)
		}
	}
}

func (h *SHostInfo) getNicsTelegrafConf() []map[string]interface{} {
	var ret = make([]map[string]interface{}, 0)
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
