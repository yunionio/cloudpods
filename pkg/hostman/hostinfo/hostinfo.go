package hostinfo

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type SHostInfo struct {
	isRegistered bool

	Cpu *SCPUInfo
	Mem *SMemory

	storageManager *storageman.SStorageManager
}

func (h *SHostInfo) Start() error {
	if err := h.prepareEnv(); err != nil {
		return err
	}
	if err := h.parseConfig(); err != nil {
		return err
	}
	if err := h.detactHostInfo(); err != nil {
		return err
	}
	return nil
}

func (h *SHostInfo) parseConfig() error {
	if h.GetMemory() < 64 { // MB
		return fmt.Errorf("Not enough memory!")
	}
	if len(options.HostOptions.ListenInterface) > 0 {
		// TODO netutils.NetInterface netutils未实现
		h.MasterNic = netutils2.NewNetInterface(options.HostOptions.ListenInterface)
	} else {
		h.MasterNic = nil
	}
	for _, n := range options.HostOptions.Networks {
		var nic = NewNIC(n) // TODO NIC 未实现
		h.Nics = append(h.Nics, nic)
		// XXX ???
		// if options.enable_tc_bwlimit:
		// tcman.init_manager(nic.interface, nic.ip)
	}
	for i := 0; i < len(h.Nics); i++ {
		// TODO ...
		h.Nics[i].SetupDhcpRelay(h.GetMasterIp())
	}
	h.Storages = StorageManager(h)
	h.IsolatedDeviceMan = IsolatedDeviceManager()
}

func (h *SHostInfo) prepareEnv() error {
	h.fixPathEnv()
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

	// TODO: BriggeDeriver 还未实现
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

func (h *SHostInfo) fixPathEnv() {
	var paths = []string{
		"/usr/local/sbin",
		"/usr/local/bin",
		"/sbin",
		"/bin",
		"/usr/sbin",
		"/usr/bin",
	}
	env := os.Getenv("PATH")
	os.Setenv("PATH", strings.Join(paths, ":"))
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
		preAllocPagesNum := h.GetMemory()/h.Memory.GetHugepagesizeMb() + 1
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

func (h *SHostInfo) StartRegister() {
	// TODO
}

func NewHostInfo() (*SHostInfo, error) {
	var res = new(SHostInfo)
	cpu, err := DetectCpuInfo()
	if err != nil {
		return nil, err
	} else {
		res.Cpu = cpu
	}
	mem, err := DetectMemoryInfo()
	if err != nil {

	}
}

var hostInfo *SHostInfo

func Init() error {
	hostInfo = NewHostInfo()
	return hostInfo.Start()
}

func Instance() *SHostInfo {
	return hostInfo
}

func GetStorageManager() *storageman.SStorageManager {
	return hostInfo.storageManager
}
