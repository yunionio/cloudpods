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

package guestman

import (
	"fmt"
	"net"
	"path"
	"strings"
	"time"

	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
	"github.com/sergi/go-diff/diffmatchpatch"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/qemu"
	qemucerts "yunion.io/x/onecloud/pkg/hostman/guestman/qemu/certs"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	OS_NAME_LINUX   = qemu.OS_NAME_LINUX
	OS_NAME_WINDOWS = qemu.OS_NAME_WINDOWS
	OS_NAME_MACOS   = qemu.OS_NAME_MACOS
	OS_NAME_ANDROID = qemu.OS_NAME_ANDROID
	OS_NAME_VMWARE  = qemu.OS_NAME_VMWARE
	OS_NAME_CIRROS  = qemu.OS_NAME_CIRROS
	OS_NAME_OPENWRT = qemu.OS_NAME_OPENWRT

	MODE_READLINE = qemu.MODE_READLINE
	MODE_CONTROL  = qemu.MODE_CONTROL

	DISK_DRIVER_VIRTIO = qemu.DISK_DRIVER_VIRTIO
	DISK_DRIVER_SCSI   = qemu.DISK_DRIVER_SCSI
	DISK_DRIVER_PVSCSI = qemu.DISK_DRIVER_PVSCSI
	DISK_DRIVER_IDE    = qemu.DISK_DRIVER_IDE
	DISK_DRIVER_SATA   = qemu.DISK_DRIVER_SATA
)

func (s *SKVMGuestInstance) IsKvmSupport() bool {
	return guestManager.GetHost().IsKvmSupport()
}

func (s *SKVMGuestInstance) IsVdiSpice() bool {
	vdi, _ := s.Desc.GetString("vdi")
	return vdi == "spice"
}

func (s *SKVMGuestInstance) getOsname() string {
	osName, err := s.Desc.GetString("metadata", "os_name")
	if err != nil {
		return OS_NAME_LINUX
	}
	return osName
}

func (s *SKVMGuestInstance) disableUsbKbd() bool {
	val, _ := s.Desc.GetString("metadata", "disable_usb_kbd")
	return val == "true"
}

func (s *SKVMGuestInstance) getOsDistribution() string {
	osDis, _ := s.Desc.GetString("metadata", "os_distribution")
	return osDis
}

func (s *SKVMGuestInstance) getOsVersion() string {
	osVer, _ := s.Desc.GetString("metadata", "os_version")
	return osVer
}

// is windows prioer to windows server 2003
func (s *SKVMGuestInstance) isOldWindows() bool {
	if s.getOsname() == OS_NAME_WINDOWS {
		ver := s.getOsVersion()
		if len(ver) > 1 && ver[0:2] == "5." {
			return true
		}
	}
	return false
}

func (s *SKVMGuestInstance) isWindows10() bool {
	if s.getOsname() == OS_NAME_WINDOWS {
		distro := s.getOsDistribution()
		if strings.Contains(strings.ToLower(distro), "windows 10") {
			return true
		}
	}
	return false
}

func (s *SKVMGuestInstance) getMachine() string {
	machine, err := s.Desc.GetString("machine")
	if err != nil {
		machine = api.VM_MACHINE_TYPE_PC
	}
	return machine
}

func (s *SKVMGuestInstance) getBios() string {
	bios, err := s.Desc.GetString("bios")
	if err != nil {
		bios = "bios"
	}
	return bios
}

func (s *SKVMGuestInstance) isQ35() bool {
	return s.getMachine() == api.VM_MACHINE_TYPE_Q35
}

func (s *SKVMGuestInstance) GetVdiProtocol() string {
	vdi, err := s.Desc.GetString("vdi")
	if err != nil {
		vdi = "vnc"
	}
	return vdi
}

func (s *SKVMGuestInstance) GetPciBus() string {
	if s.isQ35() {
		return "pcie.0"
	} else {
		return "pci.0"
	}
}

func (s *SKVMGuestInstance) disableIsaSerialDev() bool {
	val, _ := s.Desc.GetString("metadata", "disable_isa_serial")
	return val == "true"
}

func (s *SKVMGuestInstance) disablePvpanicDev() bool {
	val, _ := s.Desc.GetString("metadata", "disable_pvpanic")
	return val == "true"
}

func (s *SKVMGuestInstance) GetDiskAddr(idx int) int {
	return qemu.GetDiskAddr(idx, s.IsVdiSpice())
}

func (s *SKVMGuestInstance) getNicUpScriptPath(nic jsonutils.JSONObject) string {
	ifname, _ := nic.GetString("ifname")
	bridge, _ := nic.GetString("bridge")
	return path.Join(s.HomeDir(), fmt.Sprintf("if-up-%s-%s.sh", bridge, ifname))
}

func (s *SKVMGuestInstance) getNicDownScriptPath(nic jsonutils.JSONObject) string {
	ifname, _ := nic.GetString("ifname")
	bridge, _ := nic.GetString("bridge")
	return path.Join(s.HomeDir(), fmt.Sprintf("if-down-%s-%s.sh", bridge, ifname))
}

func (s *SKVMGuestInstance) generateNicScripts(nic jsonutils.JSONObject) error {
	bridge, _ := nic.GetString("bridge")
	dev := guestManager.GetHost().GetBridgeDev(bridge)
	if dev == nil {
		return fmt.Errorf("Can't find bridge %s", bridge)
	}
	isSlave := s.IsSlave()
	if err := dev.GenerateIfupScripts(s.getNicUpScriptPath(nic), nic, isSlave); err != nil {
		return errors.Wrap(err, "GenerateIfupScripts")
	}
	if err := dev.GenerateIfdownScripts(s.getNicDownScriptPath(nic), nic, isSlave); err != nil {
		return errors.Wrap(err, "GenerateIfdownScripts")
	}
	return nil
}

func (s *SKVMGuestInstance) getNicDeviceModel(name string) string {
	return qemu.GetNicDeviceModel(name)
}

func (s *SKVMGuestInstance) getNicAddr(index int) int {
	disks, _ := s.Desc.GetArray("disks")
	isolatedDevices, _ := s.Desc.GetArray("isolated_devices")
	return qemu.GetNicAddr(index, len(disks), len(isolatedDevices), s.IsVdiSpice())
}

func (s *SKVMGuestInstance) extraOptions() string {
	cmd := " "
	extraOptions, _ := s.Desc.GetMap("extra_options")
	for k, v := range extraOptions {
		switch jsonV := v.(type) {
		case *jsonutils.JSONArray:
			for i := 0; i < jsonV.Size(); i++ {
				vAtI, _ := jsonV.GetAt(i)
				cmd += fmt.Sprintf(" -%s %s", k, vAtI.String())
			}
		default:
			cmd += fmt.Sprintf(" -%s %s", k, v.String())
		}
	}
	return cmd
}

func (s *SKVMGuestInstance) generateStartScript(data *jsonutils.JSONDict) (string, error) {
	// initial data
	var (
		uuid, _ = s.Desc.GetString("uuid")
		mem, _  = s.Desc.Int("mem")
		cpu, _  = s.Desc.Int("cpu")
		name, _ = s.Desc.GetString("name")
		nics, _ = s.Desc.GetArray("nics")
		osname  = s.getOsname()
		input   = &qemu.GenerateStartOptionsInput{
			UUID:                 uuid,
			Mem:                  uint64(mem),
			Cpu:                  uint(cpu),
			Name:                 name,
			OsName:               osname,
			Nics:                 nics,
			OVNIntegrationBridge: options.HostOptions.OvnIntegrationBridge,
			HomeDir:              s.HomeDir(),
			HugepagesEnabled:     s.manager.host.IsHugepagesEnabled(),
			PidFilePath:          s.GetPidFilePath(),
			BIOS:                 s.getBios(),
		}
	)

	if data.Contains("encrypt_key") {
		key, _ := data.GetString("encrypt_key")
		s.saveEncryptKeyFile(key)
		input.EncryptKeyPath = s.getEncryptKeyPath()
	}

	cmd := ""

	// inject disks
	disks := make([]api.GuestdiskJsonDesc, 0)
	s.Desc.Unmarshal(&disks, "disks")
	input.Disks = disks

	// inject machine and bios
	if input.OsName == OS_NAME_MACOS {
		s.Desc.Set("machine", jsonutils.NewString(api.VM_MACHINE_TYPE_Q35))
		input.Machine = api.VM_MACHINE_TYPE_Q35
		s.Desc.Set("bios", jsonutils.NewString(qemu.BIOS_UEFI))
		input.BIOS = qemu.BIOS_UEFI
	}

	// inject vncPort
	vncPort, _ := data.Int("vnc_port")
	input.VNCPort = uint(vncPort)

	// inject qemu version and arch
	qemuVersion := options.HostOptions.DefaultQemuVersion
	if data.Contains("qemu_version") {
		qemuVersion, _ = data.GetString("qemu_version")
	}
	if qemuVersion == "latest" {
		qemuVersion = ""
	}
	input.QemuVersion = qemu.Version(qemuVersion)
	// inject qemu arch
	if s.manager.host.IsAarch64() {
		input.QemuArch = qemu.Arch_aarch64
	} else {
		input.QemuArch = qemu.Arch_x86_64
	}

	// inject isolatedDevices
	var devAddrs = []string{}
	isolatedParams, _ := s.Desc.GetArray("isolated_devices")
	for _, params := range isolatedParams {
		devAddr, _ := params.GetString("addr")
		devAddrs = append(devAddrs, devAddr)
	}
	isolatedDevsParams := s.manager.GetHost().GetIsolatedDeviceManager().GetQemuParams(devAddrs)
	input.IsolatedDevicesParams = isolatedDevsParams

	for _, nic := range input.Nics {
		downscript := s.getNicDownScriptPath(nic)
		ifname, _ := nic.GetString("ifnam")
		cmd += fmt.Sprintf("%s %s\n", downscript, ifname)
	}

	if input.HugepagesEnabled {
		cmd += fmt.Sprintf("mkdir -p /dev/hugepages/%s\n", input.UUID)
		cmd += fmt.Sprintf("mount -t hugetlbfs -o pagesize=%dK,size=%dM hugetlbfs-%s /dev/hugepages/%s\n",
			s.manager.host.HugepageSizeKb(), input.Mem, input.UUID, input.UUID)
	}

	cmd += "sleep 1\n"
	cmd += fmt.Sprintf("echo %d > %s\n", input.VNCPort, s.GetVncFilePath())

	diskScripts, err := s.generateDiskSetupScripts(input.Disks)
	if err != nil {
		return "", errors.Wrap(err, "generateDiskSetupScripts")
	}
	cmd += diskScripts

	cmd += fmt.Sprintf("STATE_FILE=`ls -d %s* | head -n 1`\n", s.getStateFilePathRootPrefix())
	cmd += fmt.Sprintf("PID_FILE=%s\n", input.PidFilePath)

	var qemuCmd = qemutils.GetQemu(string(input.QemuVersion))
	if len(qemuCmd) == 0 {
		qemuCmd = qemutils.GetQemu("")
	}

	cmd += fmt.Sprintf("DEFAULT_QEMU_CMD='%s'\n", qemuCmd)
	/*
	 * cmd += "if [ -n \"$STATE_FILE\" ]; then\n"
	 * cmd += "    QEMU_VER=`echo $STATE_FILE" +
	 * 	` | grep -o '_[[:digit:]]\+\.[[:digit:]]\+.*'` + "`\n"
	 * cmd += "    QEMU_CMD=\"qemu-system-x86_64\"\n"
	 * cmd += "    QEMU_LOCAL_PATH=\"/usr/local/bin/$QEMU_CMD\"\n"
	 * cmd += "    QEMU_LOCAL_PATH_VER=\"/usr/local/qemu-$QEMU_VER/bin/$QEMU_CMD\"\n"
	 * cmd += "    QEMU_BIN_PATH=\"/usr/bin/$QEMU_CMD\"\n"
	 * cmd += "    if [ -f \"$QEMU_LOCAL_PATH_VER\" ]; then\n"
	 * cmd += "        QEMU_CMD=$QEMU_LOCAL_PATH_VER\n"
	 * cmd += "    elif [ -f \"$QEMU_LOCAL_PATH\" ]; then\n"
	 * cmd += "        QEMU_CMD=$QEMU_LOCAL_PATH\n"
	 * cmd += "    elif [ -f \"$QEMU_BIN_PATH\" ]; then\n"
	 * cmd += "        QEMU_CMD=$QEMU_BIN_PATH\n"
	 * cmd += "    fi\n"
	 * cmd += "else\n"
	 * cmd += "    QEMU_CMD=$DEFAULT_QEMU_CMD\n"
	 * cmd += "fi\n"
	 */
	cmd += "QEMU_CMD=$DEFAULT_QEMU_CMD\n"
	if s.IsKvmSupport() && !options.HostOptions.DisableKVM {
		cmd += "QEMU_CMD_KVM_ARG=-enable-kvm\n"
	} else if utils.IsInStringArray(s.manager.host.GetCpuArchitecture(), apis.ARCH_X86) {
		// -no-kvm仅x86适用，且将在qemu 5.2之后移除
		// https://gitlab.com/qemu-project/qemu/-/blob/master/docs/about/removed-features.rst
		cmd += "QEMU_CMD_KVM_ARG=-no-kvm\n"
	} else {
		cmd += "QEMU_CMD_KVM_ARG=\n"
	}
	// cmd += "fi\n"
	cmd += `
function nic_speed() {
    $QEMU_CMD $QEMU_CMD_KVM_ARG -device virtio-net-pci,help 2>&1 | grep -q "\<speed="
    if [ "$?" -eq "0" ]; then
        echo ",speed=$1"
    fi
}

function nic_mtu() {
    local bridge="$1"; shift

    $QEMU_CMD $QEMU_CMD_KVM_ARG -device virtio-net-pci,help 2>&1 | grep -q '\<host_mtu='
    if [ "$?" -eq "0" ]; then
        local origmtu="$(<"/sys/class/net/$bridge/mtu")"
        if [ -n "$origmtu" -a "$origmtu" -gt 576 ]; then
            echo ",host_mtu=$(($origmtu - ` + api.VpcOvnEncapCostStr() + `))"
        fi
    fi
}
`

	// Generate Start VM script
	cmd += `CMD="$QEMU_CMD $QEMU_CMD_KVM_ARG`

	// inject cpu info
	if s.IsKvmSupport() && !options.HostOptions.DisableKVM {
		input.EnableKVM = true
		input.HostCPUPassthrough = options.HostOptions.HostCpuPassthrough
		input.IsCPUIntel = sysutils.IsProcessorIntel()
		input.IsCPUAMD = sysutils.IsProcessorAmd()
		input.EnableNested = guestManager.GetHost().IsNestedVirtualization()
	}

	if options.HostOptions.LogLevel == "debug" {
		input.EnableLog = true
		input.LogPath = s.getQemuLogPath()
	}

	// inject monitor
	input.HMPMonitor = &qemu.Monitor{
		Id:   "hmqmon",
		Port: uint(s.GetHmpMonitorPort(int(input.VNCPort))),
		Mode: MODE_READLINE,
	}
	if options.HostOptions.EnableQmpMonitor {
		input.QMPMonitor = &qemu.Monitor{
			Id:   "qmqmon",
			Port: uint(s.GetQmpMonitorPort(int(input.VNCPort))),
			Mode: MODE_CONTROL,
		}
	}

	input.EnableUUID = options.HostOptions.EnableVmUuid
	// inject machine
	input.Machine = s.getMachine()

	// inject bootOrder and cdrom
	bootOrder, _ := s.Desc.GetString("boot_order")
	input.BootOrder = bootOrder
	cdrom, _ := s.Desc.Get("cdrom")
	if cdrom != nil && cdrom.Contains("path") {
		cdromPath, _ := cdrom.GetString("path")
		input.CdromPath = cdromPath
	}

	// UEFI ovmf file path
	if input.QemuArch == qemu.Arch_aarch64 && !strings.HasPrefix(input.BIOS, qemu.BIOS_UEFI) {
		input.BIOS = qemu.BIOS_UEFI
		if len(input.OVMFPath) == 0 {
			input.OVMFPath = options.HostOptions.OvmfPath
		}
	}

	// inject nic and disks
	if input.OsName == OS_NAME_MACOS {
		for i := 0; i < len(input.Disks); i++ {
			disks[i].Driver = DISK_DRIVER_SATA
		}
		for i := 0; i < len(input.Nics); i++ {
			nic := nics[i].(*jsonutils.JSONDict)
			nic.Set("vectors", jsonutils.NewInt(0))
			nic.Set("driver", jsonutils.NewString("e1000"))
		}
	} else if input.OsName == OS_NAME_ANDROID {
		if len(input.Nics) > 1 {
			s.Desc.Set("nics", jsonutils.NewArray(input.Nics[0]))
		}
		nics, _ = s.Desc.GetArray("nics")
		input.Nics = nics
	}

	// inject devices
	if input.QemuArch == qemu.Arch_aarch64 {
		input.Devices = append(input.Devices,
			"qemu-xhci,p2=8,p3=8,id=usb1",
			"usb-tablet,id=input0,bus=usb1.0,port=1",
			"usb-kbd,id=input1,bus=usb1.0,port=2",
			"virtio-gpu-pci,id=video1,max_outputs=1",
		)
	} else {
		if !utils.IsInStringArray(s.getOsDistribution(), []string{OS_NAME_OPENWRT, OS_NAME_CIRROS}) &&
			!s.isOldWindows() && !s.isWindows10() &&
			!s.disableUsbKbd() {
			input.Devices = append(input.Devices, "usb-kbd")
		}
		if osname == OS_NAME_ANDROID {
			input.Devices = append(input.Devices, "usb-mouse")
		} else if !s.isOldWindows() {
			input.Devices = append(input.Devices, "usb-tablet")
		}
	}

	// inject spice and vnc display
	input.IsVdiSpice = s.IsVdiSpice()
	input.SpicePort = uint(5900 + vncPort)
	input.PCIBus = s.GetPciBus()
	if input.QemuArch != qemu.Arch_aarch64 {
		vga, err := s.Desc.GetString("vga")
		if err != nil {
			vga = "std"
		}
		input.VGA = vga
	}
	input.VNCPassword = options.HostOptions.SetVncPassword

	// reinject nics
	input.IsKVMSupport = s.IsKvmSupport()
	for i := 0; i < len(input.Nics); i++ {
		if input.OsName == OS_NAME_VMWARE {
			input.Nics[i].(*jsonutils.JSONDict).Set("driver", jsonutils.NewString("vmxnet3"))
		}
		if err := s.generateNicScripts(input.Nics[i]); err != nil {
			return "", errors.Wrapf(err, "generateNicScripts for nic: %s", input.Nics[i])
		}
		upscript := s.getNicUpScriptPath(input.Nics[i])
		downscript := s.getNicDownScriptPath(input.Nics[i])
		input.Nics[i].(*jsonutils.JSONDict).Set("upscript_path", jsonutils.NewString(upscript))
		input.Nics[i].(*jsonutils.JSONDict).Set("downscript_path", jsonutils.NewString(downscript))
	}

	input.ExtraOptions = append(input.ExtraOptions, s.extraOptions())

	/*
		QIU Jian
		virtio-rng device may cause live migration failure
		qemu-system-x86_64: Unknown savevm section or instance '0000:00:05.0/virtio-rng' 0
		qemu-system-x86_64: load of migration failed: Invalid argument
	*/
	if options.HostOptions.EnableVirtioRngDevice {
		input.EnableRNGRandom = true
	}

	// add serial device
	if !s.disableIsaSerialDev() {
		input.EnableSerialDevice = true
	}

	if jsonutils.QueryBoolean(data, "need_migrate", false) {
		input.NeedMigrate = true
		migratePort := s.manager.GetFreePortByBase(LIVE_MIGRATE_PORT_BASE)
		s.Desc.Set("live_migrate_dest_port", jsonutils.NewInt(int64(migratePort)))
		input.LiveMigratePort = uint(migratePort)
		if jsonutils.QueryBoolean(data, "live_migrate_use_tls", false) {
			input.LiveMigrateUseTLS = true
			s.Desc.Set("live_migrate_use_tls", jsonutils.JSONTrue)
		}
	} else if jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
		input.IsSlave = true
		input.LiveMigratePort = uint(s.manager.GetFreePortByBase(LIVE_MIGRATE_PORT_BASE))
	} else if jsonutils.QueryBoolean(s.Desc, "is_master", false) {
		input.IsMaster = true
	}
	// cmd += fmt.Sprintf(" -D %s", path.Join(s.HomeDir(), "log"))
	if !s.disablePvpanicDev() {
		input.EnablePvpanic = true
	}

	qemuOpts, err := qemu.GenerateStartOptions(input)
	if err != nil {
		return "", errors.Wrap(err, "GenerateStartCommand")
	}
	cmd = fmt.Sprintf("%s %s", cmd, qemuOpts)
	cmd += "\"\n"

	cmd += `
if [ ! -z "$STATE_FILE" ] && [ -d "$STATE_FILE" ] && [ -f "$STATE_FILE/content" ]; then
    CMD="$CMD --incoming \"exec: cat $STATE_FILE/content\""
elif [ ! -z "$STATE_FILE" ] && [ -f "$STATE_FILE" ]; then
    CMD="$CMD --incoming \"exec: cat $STATE_FILE\""
fi
eval $CMD`

	return cmd, nil
}

func (s *SKVMGuestInstance) parseCmdline(input string) (*qemutils.Cmdline, []qemutils.Option, error) {
	cl, err := qemutils.NewCmdline(input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "NewCmdline %q", input)
	}
	filterOpts := make([]qemutils.Option, 0)
	// filter migrate and other option include dynamic port
	cl.FilterOption(func(o qemutils.Option) bool {
		switch o.Key {
		case "incoming":
			if strings.HasPrefix(o.Value, "tcp:") || strings.HasPrefix(o.Value, "defer") {
				filterOpts = append(filterOpts, o)
				return true
			}
		case "vnc":
			filterOpts = append(filterOpts, o)
			return true
		case "chardev":
			valsMatch := []string{
				"socket,id=hmqmondev",
				"socket,id=hmpmondev",
				"socket,id=qmqmondev",
				"socket,id=qmpmondev",
			}
			for _, valM := range valsMatch {
				if strings.HasPrefix(o.Value, valM) {
					filterOpts = append(filterOpts, o)
					return true
				}
			}
		}
		return false
	})
	return cl, filterOpts, nil
}

func (s *SKVMGuestInstance) _unifyMigrateQemuCmdline(cur string, src string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(cur, src, false)

	log.Debugf("unify migrate qemu cmdline diffs: %s", jsonutils.Marshal(diffs).PrettyString())

	// make patch
	patch := dmp.PatchMake(cur, diffs)

	// apply patch
	newStr, _ := dmp.PatchApply(patch, cur)
	return newStr
}

func (s *SKVMGuestInstance) unifyMigrateQemuCmdline(cur string, src string) (string, error) {
	curCl, curFilterOpts, err := s.parseCmdline(cur)
	if err != nil {
		return "", errors.Wrapf(err, "parseCmdline current %q", cur)
	}
	srcCl, _, err := s.parseCmdline(src)
	if err != nil {
		return "", errors.Wrapf(err, "parseCmdline source %q", src)
	}
	unifyStr := s._unifyMigrateQemuCmdline(curCl.ToString(), srcCl.ToString())
	unifyCl, _, err := s.parseCmdline(unifyStr)
	if err != nil {
		return "", errors.Wrapf(err, "parseCmdline unitfy %q", unifyStr)
	}
	unifyCl.AddOption(curFilterOpts...)
	return unifyCl.ToString(), nil
}

func (s *SKVMGuestInstance) generateStopScript(data *jsonutils.JSONDict) string {
	var (
		uuid, _ = s.Desc.GetString("uuid")
		nics, _ = s.Desc.GetArray("nics")
	)

	cmd := ""
	cmd += fmt.Sprintf("VNC_FILE=%s\n", s.GetVncFilePath())
	cmd += fmt.Sprintf("PID_FILE=%s\n", s.GetPidFilePath())
	cmd += "if [ -f $VNC_FILE ]; then\n"
	cmd += "  VNC=`cat $VNC_FILE`\n"

	// TODO, replace with qmp monitor
	cmd += fmt.Sprintf("  MON=$(($VNC + %d))\n", MONITOR_PORT_BASE)
	cmd += "  echo quit | nc -w 1 127.0.0.1 $MON > /dev/null\n"
	cmd += "  sleep 1\n"
	cmd += "  echo \"Remove VNC $VNC_FILE\"\n"
	cmd += "  rm -f $VNC_FILE\n"
	cmd += "fi\n"
	cmd += "if [ -f $PID_FILE ]; then\n"
	cmd += "  PID=`cat $PID_FILE`\n"
	cmd += "  ps -p $PID > /dev/null\n"
	cmd += "  if [ $? -eq 0 ]; then\n"
	cmd += "    echo \"Kill process $PID\"\n"
	cmd += "    kill -9 $PID > /dev/null 2>&1\n"
	cmd += "  fi\n"
	cmd += "  echo \"Remove PID $PID_FILE\"\n"
	cmd += "  rm -f $PID_FILE\n"
	cmd += "fi\n"

	cmd += fmt.Sprintf("for d in $(ls -d /dev/hugepages/%s*)\n", uuid)
	cmd += fmt.Sprintf("do\n")
	cmd += fmt.Sprintf("  if [ -d $d ]; then\n")
	cmd += fmt.Sprintf("    umount $d\n")
	cmd += fmt.Sprintf("    rm -rf $d\n")
	cmd += fmt.Sprintf("  fi\n")
	cmd += fmt.Sprintf("done\n")

	for _, nic := range nics {
		ifname, _ := nic.GetString("ifname")
		downscript := s.getNicDownScriptPath(nic)
		cmd += fmt.Sprintf("%s %s\n", downscript, ifname)
	}
	return cmd
}

func (s *SKVMGuestInstance) presendArpForNic(nic jsonutils.JSONObject) {
	ifname, _ := nic.GetString("ifname")
	ifi, err := net.InterfaceByName(ifname)
	if err != nil {
		log.Errorf("InterfaceByName error %s", ifname)
		return
	}

	cli, err := arp.Dial(ifi)
	if err != nil {
		log.Errorf("arp Dial error %s", err)
		return
	}
	defer cli.Close()

	var (
		sSrcMac, _ = nic.GetString("mac")
		sScrIp, _  = nic.GetString("ip")
		srcIp      = net.ParseIP(sScrIp)
		dstMac, _  = net.ParseMAC("00:00:00:00:00:00")
		dstIp      = net.ParseIP("255.255.255.255")
	)
	srcMac, err := net.ParseMAC(sSrcMac)
	if err != nil {
		log.Errorf("Send arp parse mac error: %s", err)
		return
	}

	pkt, err := arp.NewPacket(arp.OperationRequest, srcMac, srcIp, dstMac, dstIp)
	if err != nil {
		log.Errorf("New arp packet error %s", err)
		return
	}
	if err := cli.WriteTo(pkt, ethernet.Broadcast); err != nil {
		log.Errorf("Send arp packet error %s ", err)
		return
	}
}

func (s *SKVMGuestInstance) StartPresendArp() {
	go func() {
		for i := 0; i < 5; i++ {
			nics, _ := s.Desc.GetArray("nics")
			for _, nic := range nics {
				s.presendArpForNic(nic)
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func (s *SKVMGuestInstance) getPKIDirPath() string {
	return path.Join(s.HomeDir(), "pki")
}

func (s *SKVMGuestInstance) makePKIDir() error {
	output, err := procutils.NewCommand("mkdir", "-p", s.getPKIDirPath()).Output()
	if err != nil {
		return errors.Wrapf(err, "mkdir %s failed: %s", s.getPKIDirPath(), output)
	}
	return nil
}

func (s *SKVMGuestInstance) PrepareMigrateCerts() (map[string]string, error) {
	pkiDir := s.getPKIDirPath()
	if err := s.makePKIDir(); err != nil {
		return nil, errors.Wrap(err, "make pki dir")
	}
	tree, err := qemucerts.GetDefaultCertList().AsMap().CertTree()
	if err != nil {
		return nil, errors.Wrap(err, "construct cert tree")
	}
	if err := tree.CreateTree(pkiDir); err != nil {
		return nil, errors.Wrap(err, "create certs")
	}
	return qemucerts.FetchDefaultCerts(pkiDir)
}

func (s *SKVMGuestInstance) WriteMigrateCerts(certs map[string]string) error {
	pkiDir := s.getPKIDirPath()
	if err := s.makePKIDir(); err != nil {
		return errors.Wrap(err, "make pki dir")
	}
	if err := qemucerts.CreateByMap(pkiDir, certs); err != nil {
		return errors.Wrapf(err, "create by map %#v", certs)
	}
	return nil
}
