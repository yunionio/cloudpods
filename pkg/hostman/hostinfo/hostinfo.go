package hostinfo

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	bare2 "yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	bare1 "yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

var (
	KVM_MODULE_INTEL     = "kvm-intel"
	KVM_MODULE_AMD       = "kvm-amd"
	KVM_MODULE_UNSUPPORT = "unsupport"

	HOST_NEST_UNSUPPORT = "0"
	HOST_NEST_SUPPORT   = "1"
	HOST_NEST_ENABLE    = "3"
)

type SHostInfo struct {
	isRegistered     bool
	IsRegistered     chan struct{}
	registerCallback func()
	saved            bool
	pinger           *SHostPingTask

	kvmModuleSupport string
	nestStatus       string

	Cpu     *SCPUInfo
	Mem     *SMemory
	sysinfo *SSysInfo
	// storageManager *storageman.SStorageManager
	IsolatedDeviceMan *isolated_device.IsolatedDeviceManager

	MasterNic *netutils2.SNetInterface
	Nics      []*SNIC

	HostId         string
	Zone           string
	ZoneId         string
	Cloudregion    string
	CloudregionId  string
	ZoneManagerUri string

	FullName string
}

func (h *SHostInfo) GetBridgeDev(bridge string) hostbridge.IBridgeDriver {
	for _, n := range h.Nics {
		if bridge == n.Bridge {
			return n.BridgeDev
		}
	}
	return nil
}

func (h *SHostInfo) GetHostId() string {
	return h.HostId
}

func (h *SHostInfo) GetZone() string {
	return h.Zone
}

func (h *SHostInfo) GetMediumType() string {
	if h.sysinfo != nil {
		return h.sysinfo.StorageType
	}
	return ""
}

func (h *SHostInfo) IsKvmSupport() bool {
	if h.kvmModuleSupport == KVM_MODULE_UNSUPPORT {
		return false
	}
	return true
}

func (h *SHostInfo) IsNestedVirtualization() bool {
	return utils.IsInStringArray("hypervisor", h.Cpu.cpuFeatures)
}

func (h *SHostInfo) Init() error {
	if err := h.prepareEnv(); err != nil {
		return err
	}
	log.Infof("Start parseConfig")
	if err := h.parseConfig(); err != nil {
		return err
	}
	log.Infof("Start detectHostInfo")
	if err := h.detectHostInfo(); err != nil {
		return err
	}
	return nil
}

func (h *SHostInfo) parseConfig() error {
	if h.GetMemory() < 64 { // MB
		return fmt.Errorf("Not enough memory!")
	}
	if len(options.HostOptions.ListenInterface) > 0 {
		h.MasterNic = netutils2.NewNetInterface(options.HostOptions.ListenInterface)
	} else {
		h.MasterNic = nil
	}
	for _, n := range options.HostOptions.Networks {
		nic, err := NewNIC(n)
		if err != nil {
			return err
		}
		h.Nics = append(h.Nics, nic)
		// XXX ???
		// if options.enable_tc_bwlimit:
		// tcman.init_manager(nic.interface, nic.ip)
	}
	for i := 0; i < len(h.Nics); i++ {
		h.Nics[i].SetupDhcpRelay()
	}

	// if err := storageman.Init(h); err != nil {
	// 	return err
	// }

	if man, err := isolated_device.NewManager(h); err != nil {
		return fmt.Errorf("NewIsolatedManager: %v", err)
	} else {
		h.IsolatedDeviceMan = man
	}

	return nil
}

func (h *SHostInfo) prepareEnv() error {
	if err := h.fixPathEnv(); err != nil {
		return err
	}
	if options.HostOptions.ReportInterval > 300 {
		return fmt.Errorf("Option report_interval must no longer than 5 min")
	}

	_, err := exec.Command("mkdir", "-p", options.HostOptions.ServersPath).Output()
	if err != nil {
		return fmt.Errorf("Failed to create path %s", options.HostOptions.ServersPath)
	}

	_, err = exec.Command(qemutils.GetQemu(""), "-version").Output()
	if err != nil {
		return fmt.Errorf("Qemu/Kvm not installed")
	}

	if _, err := os.Stat("/sbin/ethtool"); os.IsNotExist(err) {
		return fmt.Errorf("Ethtool not installed")
	}

	ioParams := make(map[string]string, 0)
	if options.HostOptions.BlockIoScheduler == "deadline" {
		ioParams["queue/scheduler"] = "deadline"
	} else {
		ioParams["queue/scheduler"] = "cfq"
		ioParams["queue/iosched/group_isolation"] = "1"
		ioParams["queue/iosched/slice_idle"] = "0"
		ioParams["queue/iosched/group_idle"] = "0"
		ioParams["queue/iosched/quantum"] = "32"
	}
	fileutils2.ChangeAllBlkdevsParams(ioParams)
	_, err = exec.Command("modprobe", "tun").Output()
	if err != nil {
		return fmt.Errorf("Failed to activate tun/tap device")
	}
	_, err = exec.Command("modprobe", "vhost_net").Output()
	if err != nil {
		e := err.(*exec.ExitError)
		log.Errorln(e.Stderr)
	}
	if !cgrouputils.Init() {
		return fmt.Errorf("Cannot initialize control group subsystem")
	}

	_, err = exec.Command("rmmod", "nbd").Output()
	if err != nil {
		e := err.(*exec.ExitError)
		log.Errorln(e.Stderr)
	}
	_, err = exec.Command("modprobe", "nbd", "max_part=16").Output()
	if err != nil {
		e := err.(*exec.ExitError)
		log.Errorf("Failed to activate nbd device: %s", e.Stderr)
	}

	// TODO: winRegTool还未实现
	// if not WinRegTool.check_tool(options.chntpw_path)...

	// TODO: BriggeDriver 还未实现
	// driver := GetBridgeDriverClass ..
	// driver.Prepareenv()...

	err = h.resetIptables()
	if err != nil {
		return err
	}
	if options.HostOptions.EnableKsm {
		h.EnableKsm(900)
	} else {
		h.DisableKsm()
	}
	switch options.HostOptions.HugepagesOption {
	case "disable":
		h.DisableHugepages()
	case "native":
		err := h.EnableNativeHugepages()
		if err != nil {
			return err
		}
	case "transparent":
		h.EnableTransparentHugepages()
	default:
		return fmt.Errorf("Invalid hugepages option")
	}
	for i := 0; i < 16; i++ {
		nbdBdi := fmt.Sprintf("/sys/block/nbd%d/bdi/", i)
		h.setSysConfig(nbdBdi+"max_ratio", "0")
		h.setSysConfig(nbdBdi+"min_ratio", "0")
	}
	h.PreventArpFlux()
	h.TuneSystem()
	return nil
}

func (h *SHostInfo) detectHostInfo() error {
	output, err := exec.Command("dmidecode", "-t", "1").Output()
	if err != nil {
		return err
	}

	sysinfo, err := sysutils.ParseDMISysinfo(strings.Split(string(output), "\n"))
	if err != nil {
		return err
	}
	h.sysinfo.DMISystemInfo = sysinfo

	h.detectiveKVMModuleSupport()
	h.detectiveNestSupport()
	h.tryEnableNest()

	if err := h.detectiveSyssoftwareInfo(); err != nil {
		return err
	}

	h.detectiveStorageSystem()

	if options.HostOptions.CheckSystemServices {
		if err := h.checkSystemServices(); err != nil {
			return err
		}
	}
	return nil
}

func (h *SHostInfo) checkSystemServices() error {
	// TOOD
	return fmt.Errorf("not implement so far")
}

func (h *SHostInfo) detectiveStorageSystem() {
	var stype = storagetypes.DISK_TYPE_ROTATE
	if options.HostOptions.DiskIsSsd {
		stype = storagetypes.DISK_TYPE_SSD
	}
	h.sysinfo.StorageType = stype
}

func (h *SHostInfo) fixPathEnv() error {
	var paths = []string{
		"/usr/local/sbin",
		"/usr/local/bin",
		"/sbin",
		"/bin",
		"/usr/sbin",
		"/usr/bin",
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
		h.setSysConfig(k, v)
	}
}

func (h *SHostInfo) EnableTransparentHugepages() {
	kv := map[string]string{
		"/proc/sys/vm/nr_hugepages":                   "0",
		"/sys/kernel/mm/transparent_hugepage/enabled": "always",
		"/sys/kernel/mm/transparent_hugepage/defrag":  "always",
	}
	for k, v := range kv {
		h.setSysConfig(k, v)
	}
}

func (h *SHostInfo) GetMemory() int {
	return h.Mem.Total // - options.reserved_memory
}

func (h *SHostInfo) EnableNativeHugepages() error {
	content, err := ioutil.ReadFile("/proc/sys/vm/nr_hugepages")
	if err != nil {
		return err
	}
	if string(content) == "0\n" {
		kv := map[string]string{
			"/sys/kernel/mm/transparent_hugepage/enabled": "never",
			"/sys/kernel/mm/transparent_hugepage/defrag":  "never",
		}
		for k, v := range kv {
			h.setSysConfig(k, v)
		}
		preAllocPagesNum := h.GetMemory()/h.Mem.GetHugepagesizeMb() + 1
		cmd := timeutils2.CommandWithTimeout(1, "sh", "-c", fmt.Sprintf("echo %d > /proc/sys/vm/nr_hugepages", preAllocPagesNum))
		_, err := cmd.Output()
		if err != nil {
			log.Errorln(err)
			_, err = exec.Command("sh", "-c", "echo 0 > /proc/sys/vm/nr_hugepages").Output()
			if err != nil {
				log.Warningf(err.Error())
			}
			return fmt.Errorf("Failed to set native hugepages, " +
				"the system might have run out of contiguous memory, fall back to 0")
		}
	}
	return nil
}

func (h *SHostInfo) setSysConfig(path, val string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		oval, _ := ioutil.ReadFile(path)
		if string(oval) != val {
			err = fileutils2.FilePutContents(path, val, false)
			if err == nil {
				return true
			}
			log.Errorln(err)
		}
	}
	return false
}

func (h *SHostInfo) EnableKsm(sleepSec int) {
	h.setSysConfig("/sys/kernel/mm/ksm/run", "1")
	h.setSysConfig("/sys/kernel/mm/ksm/sleep_millisecs",
		fmt.Sprintf("%d", sleepSec*1000))
}

func (h *SHostInfo) DisableKsm() {
	h.setSysConfig("/sys/kernel/mm/ksm/run", "0")
}

func (h *SHostInfo) PreventArpFlux() {
	h.setSysConfig("/proc/sys/net/ipv4/conf/all/arp_filter", "1")
}

// Any system wide optimizations
// set swappiness=0 to avoid swap
func (h *SHostInfo) TuneSystem() {
	kv := map[string]string{"/proc/sys/vm/swappiness": "0",
		"/sys/module/kvm/parameters/ignore_msrs": "1",
	}
	for k, v := range kv {
		h.setSysConfig(k, v)
	}
}

func (h *SHostInfo) resetIptables() error {
	for _, tbl := range []string{"filter", "nat", "mangle"} {
		err := exec.Command("iptables", "-t", tbl, "-F").Run()
		if err != nil {
			return fmt.Errorf("Fail to clean NAT iptable: %s", err)
		}
	}
	return nil
}

func (h *SHostInfo) detectiveKVMModuleSupport() {
	if len(h.kvmModuleSupport) == 0 {
		h.kvmModuleSupport = h._detectiveKVMModuleSupport()
	}
}

func (h *SHostInfo) _detectiveKVMModuleSupport() string {
	var km = KVM_MODULE_UNSUPPORT
	if h.modprobeKvmModule(KVM_MODULE_INTEL, false, false) {
		km = KVM_MODULE_INTEL
	} else if h.modprobeKvmModule(KVM_MODULE_AMD, false, false) {
		km = KVM_MODULE_AMD
	}
	return km
}

func (h *SHostInfo) modprobeKvmModule(name string, remove, nest bool) bool {
	var params = []string{"modprobe"}
	if remove {
		params = append(params, "-r")
	}
	params = append(params, name)
	if nest {
		params = append(params, "nested=1")
	}
	if err := exec.Command(params[0], params[1:]...).Run(); err != nil {
		return false
	}
	return true
}

func (h *SHostInfo) getKvmModuleSupport() string {
	if len(h.kvmModuleSupport) == 0 {
		h.detectiveKVMModuleSupport()
	}
	return h.kvmModuleSupport
}

func (h *SHostInfo) detectiveNestSupport() {
	if len(h.nestStatus) == 0 {
		h.nestStatus = h._detectiveNestSupport()
	}
	h.sysinfo.Nest = h.nestStatus2Str(h.nestStatus)
}

func (h *SHostInfo) _detectiveNestSupport() string {
	var (
		moduleName = h.getKvmModuleSupport()
		nestStatus = HOST_NEST_UNSUPPORT
	)

	if moduleName != KVM_MODULE_UNSUPPORT && h._isNestSupport(moduleName) {
		nestStatus = HOST_NEST_SUPPORT
	}
	return nestStatus
}

func (h *SHostInfo) _isNestSupport(name string) bool {
	output, err := exec.Command("modinfo", name).Output()
	if err != nil {
		log.Errorln(err)
		return false
	}

	// TODO Test
	var re = regexp.MustCompile(`parm:\s*nested:`)
	for _, line := range strings.Split(string(output), "\n") {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func (h *SHostInfo) nestStatus2Str(status string) string {
	if status == HOST_NEST_ENABLE {
		return "enabled"
	} else {
		return "disbaled"
	}
}

func (h *SHostInfo) tryEnableNest() {
	if h.nestStatus == HOST_NEST_SUPPORT {
		if h.loadKvmModuleWithNest(h.kvmModuleSupport) {
			h.nestStatus = HOST_NEST_ENABLE
		}
	}
	h.sysinfo.Nest = h.nestStatus2Str(h.nestStatus)
}

func (h *SHostInfo) loadKvmModuleWithNest(name string) bool {
	var notload = true
	if h.checkKvmModuleInstall(name) {
		nest := h.getModuleParameter(name, "nested")
		if nest == "Y" {
			return true
		}
		notload = h.unloadKvmModule(name)
	}
	if notload {
		if h.modprobeKvmModule(name, false, true) {
			return true
		}
	}
	return false
}

func (h *SHostInfo) unloadKvmModule(name string) bool {
	return h.modprobeKvmModule(name, true, false)
}

func (h *SHostInfo) getModuleParameter(name, moduel string) string {
	pa := path.Join("/sys/module/", strings.Replace(name, "-", "_", -1), "/parameters/", moduel)
	if f, err := os.Stat(pa); err != nil {
		if f.IsDir() {
			return ""
		}
		cont, err := fileutils2.FileGetContents(pa)
		if err != nil {
			log.Errorln(err)
			return ""
		}
		return strings.TrimSpace(cont)
	}
	return ""
}

func (h *SHostInfo) checkKvmModuleInstall(name string) bool {
	output, err := exec.Command("lsmod").Output()
	if err != nil {
		log.Errorln(err)
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		lm := strings.Split(line, " ")
		if len(lm) > 0 && utils.IsInStringArray(strings.Replace(name, "-", "_", -1), lm) {
			return true
		}
	}
	return false
}

func (h *SHostInfo) detectiveOsDist() {
	files, err := exec.Command("ls", "/etc/*elease").Output()
	if err != nil {
		log.Errorln(err)
		return
	}
	re := regexp.MustCompile(`(.+) release ([\d.]+)[^(]*(?:\((.+)\))?`)
	for _, file := range strings.Split(string(files), " ") {
		content, err := fileutils2.FileGetContents(path.Join("/etc", file))
		if err != nil {
			continue
		}
		m := re.FindStringSubmatch(content)
		if len(m) == 4 {
			h.sysinfo.OsDistribution = m[1]
			h.sysinfo.OsVersion = m[2]
			break
		}
	}
	if len(h.sysinfo.OsDistribution) == 0 {
		log.Errorln("Failed to detect distribution info")
	}
}

func (h *SHostInfo) detectiveKernelVersion() {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		log.Errorln(err)
	}
	h.sysinfo.KernelVersion = string(out)
}

func (h *SHostInfo) detectiveSyssoftwareInfo() error {
	h.detectiveOsDist()
	h.detectiveKernelVersion()
	if err := h.detectiveQemuVersion(); err != nil {
		return err
	}
	h.detectiveOvsVersion()
	return nil
}

func (h *SHostInfo) detectiveQemuVersion() error {
	cmd := qemutils.GetQemu(options.HostOptions.DefaultQemuVersion)
	version, err := exec.Command(cmd, "--version").Output()
	if err != nil {
		log.Errorln(err)
		return err
	} else {
		re := regexp.MustCompile(`(?i)(?<=version\s)[\d.]+`)
		v := re.FindStringSubmatch(string(version))
		if len(v) > 0 {
			h.sysinfo.QemuVersion = v[0]
		} else {
			return fmt.Errorf("Failed to detect qemu version")
		}
	}
	return nil
}

func (h *SHostInfo) detectiveOvsVersion() {
	version, err := exec.Command("ovs-vsctl", "--version").Output()
	if err != nil {
		log.Errorln(err)
	} else {
		re := regexp.MustCompile(`'(?i)(?<=\(open vswitch\)\s)[\d.]+'`)
		v := re.FindStringSubmatch(string(version))
		if len(v) > 0 {
			h.sysinfo.OvsVersion = v[0]
		} else {
			log.Errorln("Failed to detect ovs version")
		}
	}
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
	if h.MasterNic != nil {
		return h.MasterNic.Mac
	}
	for _, n := range h.Nics {
		if len(n.Ip) > 0 {
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

func (h *SHostInfo) onFail() {
	h.StartRegister(30, nil)
	panic("register failed, try 30 seconds later...")
}

func (h *SHostInfo) fetchAccessNetworkInfo() {
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		panic("master ip not found")
	}
	params := jsonutils.NewDict()
	params.Set("ip", jsonutils.NewString(masterIp))
	params.Set("limit", jsonutils.NewInt(0))
	wire, err := hostutils.GetWireOfIp(context.Background(), params)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	} else {
		h.ZoneId, err = wire.GetString("zone_id")
		if err != nil {
			log.Errorln(err)
			h.onFail()
		} else {
			h.getZoneInfo(h.ZoneId, false)
		}
	}
}

func (h *SHostInfo) GetSession() *mcclient.ClientSession {
	return hostutils.GetComputeSession(context.Background())
}

func (h *SHostInfo) getZoneInfo(zoneId string, standalone bool) {
	var params = jsonutils.NewDict()
	params.Set("standalone", jsonutils.NewBool(standalone))
	res, err := modules.Zones.Get(h.GetSession(),
		zoneId, params)
	if err != nil {
		log.Errorln(err)
		h.onFail()
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
	masterMac := h.GetMasterMac()
	if len(masterMac) == 0 {
		panic("master mac not found")
	}
	params := jsonutils.NewDict()
	params.Set("any_mac", jsonutils.NewString(masterMac))
	res, err := modules.Hosts.List(h.GetSession(), params)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	}
	if len(res.Data) == 0 {
		h.updateHostRecord("")
	} else {
		host := res.Data[0]
		name, _ := host.GetString("name")
		id, _ := host.GetString("id")
		h.setHostname(name)
		h.updateHostRecord(id)
	}
}

func (h *SHostInfo) setHostname(name string) {
	h.FullName = name
	err := exec.Command("hostnamectl", "set-hostname", name).Run()
	if err != nil {
		log.Errorln("Fail to set system hostname: %s", err)
	}
}

func (h *SHostInfo) fetchHostname() string {
	if len(options.HostOptions.Hostname) > 0 {
		return options.HostOptions.Hostname
	} else {
		masterIp := h.GetMasterIp()
		return "host-" + masterIp
	}
}

func (h *SHostInfo) getSysInfo() *SSysInfo {
	return h.sysinfo
}

func (h *SHostInfo) updateHostRecord(hostId string) {
	var method, url string
	if len(hostId) > 0 {
		method = "POST"
		url = fmt.Sprintf("/zones/%s/hosts", h.ZoneId)
	} else {
		method = "PUT"
		url = fmt.Sprintf("/hosts/%s", hostId)
	}
	content := jsonutils.NewDict()
	masterIp := h.GetMasterIp()
	if len(masterIp) == 0 {
		panic("master ip is none")
	}

	if len(hostId) == 0 {
		content.Set("name", jsonutils.NewString(h.fetchHostname()))
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
	content.Set("node_count", jsonutils.NewInt(int64(h.Cpu.cpuInfoDmi.Nodes)))
	content.Set("cpu_desc", jsonutils.NewString(h.Cpu.cpuInfoProc.Model))
	content.Set("cpu_mhz", jsonutils.NewInt(int64(h.Cpu.cpuInfoProc.Freq)))
	content.Set("cpu_cache", jsonutils.NewInt(int64(h.Cpu.cpuInfoProc.Cache)))
	content.Set("mem_size", jsonutils.NewInt(int64(h.Mem.MemInfo.Total)))
	content.Set("storage_driver", jsonutils.NewString(bare1.DISK_DRIVER_LINUX))
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
	// content.Set("version", GetVersion())
	body := jsonutils.NewDict()
	body.Set("host", content)
	session := h.GetSession()
	_, res, err := session.JSONVersionRequest("compute",
		session.GetEndpointType(), httputils.THttpMethod(method), url, nil, body, "v2")
	if err != nil {
		log.Errorln(err)
		h.onFail()
	} else {
		h.onUpdateHostInfoSucc(res)
	}
}

func (h *SHostInfo) onUpdateHostInfoSucc(body jsonutils.JSONObject) {
	hostbody, _ := body.Get("host")
	h.HostId, _ = hostbody.GetString("id")
	hostname, _ := hostbody.GetString("name")
	h.setHostname(hostname)
	if memReserved, _ := hostbody.Int("mem_reserved"); memReserved == 0 {
		h.updateHostReservedMem()
	} else {
		h.putHostOffline()
	}
}

func (h *SHostInfo) updateHostReservedMem() {
	content := jsonutils.NewDict()
	content.Set("mem_reserved", jsonutils.NewInt(h.getReservedMem()))
	res, err := modules.Hosts.Update(h.GetSession(),
		h.HostId, content)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	} else {
		h.onUpdateHostInfoSucc(res)
	}
}

func (h *SHostInfo) getReservedMem() int64 {
	reserved := h.Mem.MemInfo.Total / 10
	if reserved > 4096 {
		return 4096
	}
	return int64(reserved)
}

func (h *SHostInfo) putHostOffline() {
	_, err := modules.Hosts.PerformAction(
		h.GetSession(), h.HostId, "offline", nil)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	} else {
		h.getNetworkInfo()
	}
}

func (h *SHostInfo) PutHostOnline() error {
	_, err := modules.Hosts.PerformAction(
		h.GetSession(),
		h.HostId, "online", nil)
	return err
}

func (h *SHostInfo) getNetworkInfo() {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	res, err := modules.Hostwires.ListDescendent(
		h.GetSession(),
		h.HostId, params)
	if err != nil {
		log.Errorln(err)
		h.onFail()
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
				kwargs.Set("limit", jsonutils.NewInt(0))

				wireInfo, err := hostutils.GetWireOfIp(context.Background(), kwargs)
				if err != nil {
					log.Errorln(err)
					h.onFail()
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
			content.Set("nic_type", jsonutils.NewString(bare2.NIC_TYPE_ADMIN))
		}
	}
	_, err := modules.Hosts.PerformAction(h.GetSession(),
		h.HostId, "add-netif", content)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	} else {
		h.onUploadNicInfoSucc(nic)
	}
}

func (h *SHostInfo) doSyncNicInfo(nic *SNIC) {
	content := jsonutils.NewDict()
	content.Set("bridge", jsonutils.NewString(nic.Bridge))
	content.Set("interface", jsonutils.NewString(nic.Inter))
	_, err := modules.Hostwires.Patch(h.GetSession(),
		h.HostId, nic.Network, content)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	}
}

func (h *SHostInfo) onUploadNicInfoSucc(nic *SNIC) {
	res, err := modules.Hostwires.Get(h.GetSession(), h.HostId, nic.Network, nil)
	if err != nil {
		log.Errorln(err)
		h.onFail()
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
			log.Errorln("GetMatchNic failed!!!")
			h.onFail()
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
		log.Errorln(err)
		h.onFail()
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
				log.Errorln(err)
				h.onFail()
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
	res, err := modules.Hoststorages.ListAscendent(
		h.GetSession(),
		h.HostId, params)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	} else {
		h.onGetStorageInfoSucc(res.Data)
	}
}

func (h *SHostInfo) onGetStorageInfoSucc(hoststorages []jsonutils.JSONObject) {
	var detachStorages = []jsonutils.JSONObject{}
	storageManager := storageman.GetManager()
	for _, hs := range hoststorages {
		mountPoint, _ := hs.GetString("mount_point")
		storagecacheId, _ := hs.GetString("storagecache_id")
		storagetype, _ := hs.GetString("storage_type")
		imagecachePath, _ := hs.GetString("imagecache_path")
		storageId, _ := hs.GetString("storage_id")
		storageName, _ := hs.GetString("storage")
		storageConf, _ := hs.Get("storage_conf")

		storage := storageManager.NewSharedStorageInstance(mountPoint, storagetype)
		if storage != nil {
			storageManager.Storages = append(storageManager.Storages, storage)
		}
		storageManager.InitSharedStorageImageCache(storagetype,
			storagecacheId, imagecachePath, storage)
		storage = storageManager.GetStorageByPath(mountPoint)
		if storage != nil {
			storage.SetStorageInfo(storageId, storageName, storageConf)
		} else if storagetype != storagetypes.STORAGE_BAREMETAL {
			detachStorages = append(detachStorages, hs)
		}
	}

	if len(detachStorages) > 0 {
		go StartDetachStorages(detachStorages)
	}

	h.uploadStorageInfo()
}

func (h *SHostInfo) uploadStorageInfo() {
	for _, s := range storageman.GetManager().Storages {
		res, err := s.SyncStorageInfo()
		if err != nil {
			log.Errorln(err)
			h.onFail()
		} else {
			h.onSyncStorageInfoSucc(s, res)
		}
	}
	h.getIsolatedDevices()
}

func (h *SHostInfo) onSyncStorageInfoSucc(storage storageman.IStorage, storageInfo jsonutils.JSONObject) {
	if len(storage.GetId()) > 0 {
		id, _ := storageInfo.GetString("id")
		name, _ := storageInfo.GetString("name")
		storageConf, _ := storageInfo.Get("storage_conf")
		storage.SetStorageInfo(id, name, storageConf)
		h.attachStorage(storage)
	}
}

func (h *SHostInfo) attachStorage(storage storageman.IStorage) {
	content := jsonutils.NewDict()
	content.Set("mount_point", jsonutils.NewString(storage.GetPath()))
	_, err := modules.Hoststorages.Attach(h.GetSession(),
		h.HostId, storage.GetId(), content)
	if err != nil {
		log.Errorln(err)
		h.onFail()
	}
}

func (h *SHostInfo) getIsolatedDevices() {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("host", jsonutils.NewString(h.GetHostId()))
	res, err := modules.IsolatedDevices.List(h.GetSession(), params)
	if err != nil {
		log.Errorf("getIsolatedDevices: %v", err)
		h.onFail()
		return
	}
	h.onGetIsolatedDeviceSucc(res.Data)
}

func (h *SHostInfo) onGetIsolatedDeviceSucc(objs []jsonutils.JSONObject) {
	for _, obj := range objs {
		info := isolated_device.CloudDeviceInfo{}
		obj.Unmarshal(&info)
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
		log.Errorf("Device probe error: %v", err)
		h.onFail()
		return
	}
	h.uploadIsolatedDevices()
}

func (h *SHostInfo) uploadIsolatedDevices() {
	for _, dev := range h.IsolatedDeviceMan.Devices {
		if err := dev.SyncDeviceInfo(h); err != nil {
			log.Errorf("Sync device %s: %v", dev.String(), err)
			h.onFail()
			return
		}
	}
	h.onSucc()
}

func (h *SHostInfo) onSucc() {
	if !h.isRegistered {
		log.Infof("Host registration process success....")
		h.isRegistered = true

		if err := h.save(); err != nil {
			panic(err.Error())
		}

		// TODO
		h.StartPinger()

		if h.registerCallback != nil {
			h.registerCallback()
		}

		// To notify caller, host register is success
		close(h.IsRegistered)
	}
}

func (h *SHostInfo) StartPinger() {
	h.pinger = NewHostPingTask(options.HostOptions.PingRegionInterval)
	go h.pinger.Start()
}

func (h *SHostInfo) save() error {
	if h.saved {
		return nil
	}
	h.saved = true
	if err := h.registerHostlocalServer(); err != nil {
		return err
	}
	// TODO XXX >>> ???
	// file put content
	return h.setupBridges()
}

func (h *SHostInfo) setupBridges() error {
	for _, n := range h.Nics {
		if err := n.BridgeDev.WarmupConfig(); err != nil {
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
			log.Errorln(err)
			return err
		}
	}
	return nil
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

	mem, err := DetectMemoryInfo()
	if err != nil {
		return nil, err
	} else {
		res.Mem = mem
	}

	res.Nics = make([]*SNIC, 0)
	res.IsRegistered = make(chan struct{})
	return res, nil
}

var hostInfo *SHostInfo

func Instance() *SHostInfo {
	if hostInfo == nil {
		var err error
		hostInfo, err = NewHostInfo()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}
	return hostInfo
}
