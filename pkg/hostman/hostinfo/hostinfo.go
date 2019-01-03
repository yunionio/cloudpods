package hostinfo

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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
	isRegistered bool

	kvmModuleSupport string
	nestStatus       string

	Cpu     *SCPUInfo
	Mem     *SMemory
	sysinfo *SSysInfo
	// storageManager *storageman.SStorageManager

	MasterNic *netutils2.SNetInterface

	Nics []*SNIC
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

func (h *SHostInfo) Start() error {
	if err := h.prepareEnv(); err != nil {
		return err
	}
	if err := h.parseConfig(); err != nil {
		return err
	}
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

	// if err := storageman.Init(); err != nil {
	// 	return err
	// }
	// h.storageManager, err = storageman.NewStorageManager()
	// if err != nil {
	// 	return err
	// }
	// TODO
	// h.IsolatedDeviceMan = IsolatedDeviceManager()

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
	// TODO: cgrouputils 还未实现
	// if !cgrouputils.Init() {
	// 	return fmt.Errorf("Cannot initialize control group subsystem")
	// }

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
		_, err := exec.Command(fmt.Sprintf("iptables -t %s -F", tbl)).Output()
		if err != nil {
			e := err.(*exec.ExitError)
			return fmt.Errorf("Fail to clean NAT iptable: %s", e.Stderr)
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

func (h *SHostInfo) StartRegister() {
	// TODO
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
	return res, nil
}

var hostInfo *SHostInfo

func Init() error {
	hostInfo, err := NewHostInfo()
	if err != nil {
		return err
	}
	return hostInfo.Start()
}

func Instance() *SHostInfo {
	if hostInfo == nil {
		panic("Get nil hostinfo, Init first")
	}
	return hostInfo
}
