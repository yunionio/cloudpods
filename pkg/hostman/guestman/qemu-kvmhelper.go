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
	qemucerts "yunion.io/x/onecloud/pkg/hostman/guestman/qemu/certs"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	OS_NAME_LINUX   = "Linux"
	OS_NAME_WINDOWS = "Windows"
	OS_NAME_MACOS   = "macOS"
	OS_NAME_ANDROID = "Android"
	OS_NAME_VMWARE  = "VMWare"
	OS_NAME_CIRROS  = "Cirros"
	OS_NAME_OPENWRT = "OpenWrt"

	MODE_READLINE = "readline"
	MODE_CONTROL  = "control"

	DISK_DRIVER_VIRTIO = "virtio"
	DISK_DRIVER_SCSI   = "scsi"
	DISK_DRIVER_PVSCSI = "pvscsi"
	DISK_DRIVER_IDE    = "ide"
	DISK_DRIVER_SATA   = "sata"
)

func (s *SKVMGuestInstance) IsKvmSupport() bool {
	return guestManager.GetHost().IsKvmSupport()
}

func (s *SKVMGuestInstance) IsVdiSpice() bool {
	vdi, _ := s.Desc.GetString("vdi")
	return vdi == "spice"
}

func (s *SKVMGuestInstance) getMonitorDesc(idstr string, port int, mode string) string {
	var cmd = ""
	cmd += fmt.Sprintf(" -chardev socket,id=%sdev", idstr)
	cmd += fmt.Sprintf(",port=%d", port)
	cmd += ",host=127.0.0.1,nodelay,server,nowait"
	cmd += fmt.Sprintf(" -mon chardev=%sdev,id=%s,mode=%s", idstr, idstr, mode)
	return cmd
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

func (s *SKVMGuestInstance) getUsbControllerType() string {
	usbContType, _ := s.Desc.GetString("metadata", "usb_controller_type")
	if usbContType == "usb-ehci" {
		return usbContType
	} else {
		return "qemu-xhci"
	}
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

func isLocalStorage(disk api.GuestdiskJsonDesc) bool {
	if disk.StorageType == api.STORAGE_LOCAL || len(disk.StorageType) == 0 {
		return true
	} else {
		return false
	}
}

func (s *SKVMGuestInstance) getDriveDesc(disk api.GuestdiskJsonDesc, isArm bool) string {
	format := disk.Format
	diskIndex := disk.Index
	cacheMode := disk.CacheMode
	aioMode := disk.AioMode

	cmd := " -drive"
	cmd += fmt.Sprintf(" file=$DISK_%d", diskIndex)
	cmd += ",if=none"
	cmd += fmt.Sprintf(",id=drive_%d", diskIndex)
	if len(format) == 0 || format == "qcow2" {
		// pass    # qemu will automatically detect image format
	} else if format == "raw" {
		cmd += ",format=raw"
	}
	cmd += fmt.Sprintf(",cache=%s", cacheMode)
	if isLocalStorage(disk) {
		cmd += fmt.Sprintf(",aio=%s", aioMode)
	}
	if len(disk.Url) > 0 { // # a remote file backed image
		cmd += ",copy-on-read=on"
	}
	if isLocalStorage(disk) {
		cmd += ",file.locking=off"
	}
	// #cmd += ",media=disk"
	return cmd
}

func (s *SKVMGuestInstance) GetDiskAddr(idx int) int {
	// host-bridge / isa-bridge / vga / serial / network / block / usb / rng
	var base = 7
	if s.IsVdiSpice() {
		base += 10
	}
	return base + idx
}

func (s *SKVMGuestInstance) GetDiskDeviceModel(driver string) string {
	if driver == DISK_DRIVER_VIRTIO {
		return "virtio-blk-pci"
	} else if utils.IsInStringArray(driver, []string{DISK_DRIVER_SCSI, DISK_DRIVER_PVSCSI}) {
		return "scsi-hd"
	} else if driver == DISK_DRIVER_IDE {
		return "ide-hd"
	} else if driver == DISK_DRIVER_SATA {
		return "ide-drive"
	} else {
		return "None"
	}
}

func (s *SKVMGuestInstance) getVdiskDesc(disk api.GuestdiskJsonDesc, isArm bool) string {
	diskIndex := disk.Index
	diskDriver := disk.Driver
	numQueues := disk.NumQueues
	isSsd := disk.IsSSD

	if numQueues == 0 {
		numQueues = 4
	}

	if isArm && (diskDriver == DISK_DRIVER_IDE || diskDriver == DISK_DRIVER_SATA) {
		// unsupported configuration: IDE controllers are unsupported
		// for this QEMU binary or machine type
		// replace with scsi
		diskDriver = DISK_DRIVER_SCSI
	}

	var cmd = ""
	cmd += fmt.Sprintf(" -device %s", s.GetDiskDeviceModel(diskDriver))
	cmd += fmt.Sprintf(",drive=drive_%d", diskIndex)
	if diskDriver == DISK_DRIVER_VIRTIO {
		// virtio-blk
		cmd += fmt.Sprintf(",bus=%s,addr=0x%x", s.GetPciBus(), s.GetDiskAddr(int(diskIndex)))
		// cmd += fmt.Sprintf(",num-queues=%d,vectors=%d,iothread=iothread0", numQueues, numQueues+1)
		cmd += ",iothread=iothread0"
	} else if utils.IsInStringArray(diskDriver, []string{DISK_DRIVER_SCSI, DISK_DRIVER_PVSCSI}) {
		cmd += ",bus=scsi.0"
	} else if diskDriver == DISK_DRIVER_IDE {
		cmd += fmt.Sprintf(",bus=ide.%d,unit=%d", diskIndex/2, diskIndex%2)
	} else if diskDriver == DISK_DRIVER_SATA {
		cmd += fmt.Sprintf(",bus=ide.%d", diskIndex)
	}
	cmd += fmt.Sprintf(",id=drive_%d", diskIndex)
	if isSsd {
		if diskDriver == DISK_DRIVER_SCSI {
			cmd += ",rotation_rate=1"
		}
	}
	return cmd
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
		log.Errorln(err)
		return err
	}
	if err := dev.GenerateIfdownScripts(s.getNicDownScriptPath(nic), nic, isSlave); err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func (s *SKVMGuestInstance) getNetdevDesc(nic jsonutils.JSONObject) (string, error) {
	ifname, _ := nic.GetString("ifname")
	driver, _ := nic.GetString("driver")

	if err := s.generateNicScripts(nic); err != nil {
		return "", err
	}
	upscript := s.getNicUpScriptPath(nic)
	downscript := s.getNicDownScriptPath(nic)

	numQueues, _ := nic.Int("num_queues")

	cmd := " -netdev type=tap"
	cmd += fmt.Sprintf(",id=%s", ifname)
	cmd += fmt.Sprintf(",ifname=%s", ifname)
	if driver == "virtio" && s.IsKvmSupport() {
		cmd += ",vhost=on,vhostforce=off"
		if numQueues > 1 {
			cmd += fmt.Sprintf(",queues=%d", numQueues)
		}
	}
	cmd += fmt.Sprintf(",script=%s", upscript)
	cmd += fmt.Sprintf(",downscript=%s", downscript)
	return cmd, nil
}

func (s *SKVMGuestInstance) getNicDeviceModel(name string) string {
	if name == "virtio" {
		return "virtio-net-pci"
	} else if name == "e1000" {
		return "e1000-82545em"
	} else {
		return name
	}
}

func (s *SKVMGuestInstance) getNicAddr(index int) int {
	var pciBase = 10
	disks, _ := s.Desc.GetArray("disks")
	if len(disks) > 10 {
		pciBase = 20
	}
	isolatedDevices, _ := s.Desc.GetArray("isolated_devices")
	if len(isolatedDevices) > 0 {
		pciBase += 10
	}
	return s.GetDiskAddr(pciBase + index)
}

func (s *SKVMGuestInstance) getVnicDesc(nic jsonutils.JSONObject, withAddr bool) string {
	bridge, _ := nic.GetString("bridge")
	ifname, _ := nic.GetString("ifname")
	driver, _ := nic.GetString("driver")
	mac, _ := nic.GetString("mac")
	index, _ := nic.Int("index")
	vectors, _ := nic.Int("vectors")
	bw, _ := nic.Int("bw")
	numQueues, _ := nic.Int("num_queues")

	cmd := fmt.Sprintf(" -device %s", s.getNicDeviceModel(driver))
	cmd += fmt.Sprintf(",id=netdev-%s", ifname)
	cmd += fmt.Sprintf(",netdev=%s", ifname)
	cmd += fmt.Sprintf(",mac=%s", mac)

	if withAddr {
		cmd += fmt.Sprintf(",addr=0x%x", s.getNicAddr(int(index)))
	}
	if driver == "virtio" {
		if numQueues > 1 {
			cmd += fmt.Sprintf(",mq=on")
		}
		if nic.Contains("vectors") {
			cmd += fmt.Sprintf(",vectors=%d", vectors)
		}
		cmd += fmt.Sprintf("$(nic_speed %d)", bw)
		if bridge == options.HostOptions.OvnIntegrationBridge {
			cmd += fmt.Sprintf("$(nic_mtu %q)", bridge)
		}
	}
	return cmd
}

func (s *SKVMGuestInstance) getQgaDesc() string {
	cmd := " -chardev socket,path="
	cmd += path.Join(s.HomeDir(), "qga.sock")
	cmd += ",server,nowait,id=qga0"
	cmd += " -device virtserialport,chardev=qga0,name=org.qemu.guest_agent.0"
	return cmd
}

func (s *SKVMGuestInstance) generateStartScript(data *jsonutils.JSONDict) (string, error) {
	if s.manager.host.IsAarch64() {
		return s.generateArmStartScript(data)
	} else {
		return s._generateStartScript(data)
	}
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

func (s *SKVMGuestInstance) hasGpu() bool {
	manager := s.manager.GetHost().GetIsolatedDeviceManager()
	isolatedDevices, _ := s.Desc.GetArray("isolated_devices")
	for i := range isolatedDevices {
		vendorDevId, _ := isolatedDevices[i].GetString("vendor_device_id")
		addr, _ := isolatedDevices[i].GetString("addr")
		dev := manager.GetDeviceByIdent(vendorDevId, addr)
		if dev == nil {
			continue
		}
		if dev.GetDeviceType() == api.GPU_VGA_TYPE || dev.GetDeviceType() == api.GPU_HPC_TYPE {
			return true
		}
	}
	return false
}

func (s *SKVMGuestInstance) _generateStartScript(data *jsonutils.JSONDict) (string, error) {
	var (
		uuid, _ = s.Desc.GetString("uuid")
		mem, _  = s.Desc.Int("mem")
		cpu, _  = s.Desc.Int("cpu")
		name, _ = s.Desc.GetString("name")
		nics, _ = s.Desc.GetArray("nics")
		osname  = s.getOsname()
		cmd     = ""
	)
	disks := make([]api.GuestdiskJsonDesc, 0)
	s.Desc.Unmarshal(&disks, "disks")

	if osname == OS_NAME_MACOS {
		s.Desc.Set("machine", jsonutils.NewString("q35"))
		s.Desc.Set("bios", jsonutils.NewString("UEFI"))
	}

	vncPort, _ := data.Int("vnc_port")

	qemuVersion := options.HostOptions.DefaultQemuVersion
	if data.Contains("qemu_version") {
		qemuVersion, _ = data.GetString("qemu_version")
	}
	if qemuVersion == "latest" {
		qemuVersion = ""
	}

	var devAddrs = []string{}
	isolatedParams, _ := s.Desc.GetArray("isolated_devices")
	for _, params := range isolatedParams {
		devAddr, _ := params.GetString("addr")
		devAddrs = append(devAddrs, devAddr)
	}
	isolatedDevsParams := s.manager.GetHost().GetIsolatedDeviceManager().GetQemuParams(devAddrs)

	for _, nic := range nics {
		downscript := s.getNicDownScriptPath(nic)
		ifname, _ := nic.GetString("ifnam")
		cmd += fmt.Sprintf("%s %s\n", downscript, ifname)
	}

	if s.manager.host.IsHugepagesEnabled() {
		cmd += fmt.Sprintf("mkdir -p /dev/hugepages/%s\n", uuid)
		cmd += fmt.Sprintf("mount -t hugetlbfs -o pagesize=%dK,size=%dM hugetlbfs-%s /dev/hugepages/%s\n",
			s.manager.host.HugepageSizeKb(), mem, uuid, uuid)
	}

	cmd += "sleep 1\n"
	cmd += fmt.Sprintf("echo %d > %s\n", vncPort, s.GetVncFilePath())

	diskScripts, err := s.generateDiskSetupScripts(disks)
	if err != nil {
		return "", errors.Wrap(err, "generateDiskSetupScripts")
	}
	cmd += diskScripts

	// cmd += fmt.Sprintf("STATE_FILE=`ls -d %s* | head -n 1`\n", s.getStateFilePathRootPrefix())
	cmd += fmt.Sprintf("PID_FILE=%s\n", s.GetPidFilePath())

	var qemuCmd = qemutils.GetQemu(qemuVersion)
	if len(qemuCmd) == 0 {
		qemuCmd = qemutils.GetQemu("")
	}

	cmd += fmt.Sprintf("DEFAULT_QEMU_CMD='%s'\n", qemuCmd)
	// cmd += "if [ -n \"$STATE_FILE\" ]; then\n"
	// cmd += "    QEMU_VER=`echo $STATE_FILE" +
	// 	` | grep -o '_[[:digit:]]\+\.[[:digit:]]\+.*'` + "`\n"
	// cmd += "    QEMU_CMD=\"qemu-system-x86_64\"\n"
	// cmd += "    QEMU_LOCAL_PATH=\"/usr/local/bin/$QEMU_CMD\"\n"
	// cmd += "    QEMU_LOCAL_PATH_VER=\"/usr/local/qemu-$QEMU_VER/bin/$QEMU_CMD\"\n"
	// cmd += "    QEMU_BIN_PATH=\"/usr/bin/$QEMU_CMD\"\n"
	// cmd += "    if [ -f \"$QEMU_LOCAL_PATH_VER\" ]; then\n"
	// cmd += "        QEMU_CMD=$QEMU_LOCAL_PATH_VER\n"
	// cmd += "    elif [ -f \"$QEMU_LOCAL_PATH\" ]; then\n"
	// cmd += "        QEMU_CMD=$QEMU_LOCAL_PATH\n"
	// cmd += "    elif [ -f \"$QEMU_BIN_PATH\" ]; then\n"
	// cmd += "        QEMU_CMD=$QEMU_BIN_PATH\n"
	// cmd += "    fi\n"
	// cmd += "else\n"
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
	var accel, cpuType string
	if s.IsKvmSupport() && !options.HostOptions.DisableKVM {
		accel = "kvm"
		cpuType = ""
		if osname == OS_NAME_MACOS {
			cpuType = "Penryn,vendor=GenuineIntel"
		} else if options.HostOptions.HostCpuPassthrough {
			cpuType = "host"
			// https://unix.stackexchange.com/questions/216925/nmi-received-for-unknown-reason-20-do-you-have-a-strange-power-saving-mode-ena
			cpuType += ",+kvm_pv_eoi"
		} else {
			cpuType = "qemu64"
			cpuType += ",+kvm_pv_eoi"
			if sysutils.IsProcessorIntel() {
				cpuType += ",+vmx"
				cpuType += ",+ssse3,+sse4.1,+sse4.2,-x2apic,+aes,+avx"
				cpuType += ",+vme,+pat,+ss,+pclmulqdq,+xsave"
				cpuType += ",level=13"
			} else if sysutils.IsProcessorAmd() {
				cpuType += ",+svm"
			}
		}

		if s.hasGpu() {
			cpuType += ",kvm=off"
		}

		if isolatedDevsParams != nil && len(isolatedDevsParams.Cpu) > 0 {
			cpuType = isolatedDevsParams.Cpu
		}
	} else {
		accel = "tcg"
		cpuType = "qemu64"
	}

	cmd += fmt.Sprintf(" -cpu %s", cpuType)

	if options.HostOptions.LogLevel == "debug" {
		cmd += fmt.Sprintf(" -D %s -d all", s.getQemuLogPath())
	}

	// TODO hmp - -
	cmd += s.getMonitorDesc("hmqmon", s.GetHmpMonitorPort(int(vncPort)), MODE_READLINE)
	if options.HostOptions.EnableQmpMonitor {
		cmd += s.getMonitorDesc("qmqmon", s.GetQmpMonitorPort(int(vncPort)), MODE_CONTROL)
	}

	cmd += " -rtc base=utc,clock=host,driftfix=none"
	cmd += " -daemonize"
	cmd += " -nodefaults -nodefconfig"
	cmd += " -no-kvm-pit-reinjection"
	cmd += " -no-hpet"
	cmd += " -global kvm-pit.lost_tick_policy=discard"
	cmd += fmt.Sprintf(" -machine %s,accel=%s", s.getMachine(), accel)
	cmd += " -k en-us"
	// #cmd += " -g 800x600"
	cmd += fmt.Sprintf(" -smp cpus=%d,sockets=2,cores=64,maxcpus=128", cpu)
	cmd += fmt.Sprintf(" -name %s", name)
	if options.HostOptions.EnableVmUuid {
		cmd += fmt.Sprintf(" -uuid %s", uuid)
	}
	cmd += fmt.Sprintf(" -m %dM,slots=4,maxmem=524288M", mem)

	if options.HostOptions.HugepagesOption == "native" {
		cmd += fmt.Sprintf(" -object memory-backend-file,id=mem,size=%dM,mem-path=/dev/hugepages/%s,share=on,prealloc=on -numa node,memdev=mem", mem, uuid)
	} else {
		cmd += fmt.Sprintf(" -object memory-backend-ram,id=mem,size=%dM -numa node,memdev=mem", mem)
	}

	bootOrder, _ := s.Desc.GetString("boot_order")
	cmd += fmt.Sprintf(" -boot order=%s", bootOrder)
	cdrom, _ := s.Desc.Get("cdrom")
	if cdrom != nil && cdrom.Contains("path") {
		cmd += ",menu=on"
	}

	if s.getBios() == "UEFI" {
		// cmd += fmt.Sprintf(" -bios %s", options.HostOptions.OvmfPath)
		ovmfVarsPath := path.Join(s.HomeDir(), "OVMF_VARS.fd")
		if !fileutils2.Exists(ovmfVarsPath) {
			err := procutils.NewRemoteCommandAsFarAsPossible("cp", "-f", options.HostOptions.OvmfPath, ovmfVarsPath).Run()
			if err != nil {
				return "", errors.Wrap(err, "failed copy ovmf vars")
			}
		}
		cmd += fmt.Sprintf(" -drive if=pflash,format=raw,unit=0,file=%s,readonly=on", options.HostOptions.OvmfPath)
		cmd += fmt.Sprintf(" -drive if=pflash,format=raw,unit=1,file=%s", ovmfVarsPath)
	}

	for i := 0; i < len(nics); i++ {
		nic := nics[i].(*jsonutils.JSONDict)
		if numQueues, _ := nic.Int("num_queues"); numQueues > 1 {
			nic.Set("vectors", jsonutils.NewInt(2*numQueues+1))
		}
	}
	if osname == OS_NAME_MACOS {
		cmd += " -device isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc"
		for i := 0; i < len(disks); i++ {
			disks[i].Driver = DISK_DRIVER_SATA
		}
		for i := 0; i < len(nics); i++ {
			nic := nics[i].(*jsonutils.JSONDict)
			nic.Set("vectors", jsonutils.NewInt(0))
			nic.Set("driver", jsonutils.NewString("e1000"))
		}
	} else if osname == OS_NAME_ANDROID {
		if len(nics) > 1 {
			s.Desc.Set("nics", jsonutils.NewArray(nics[0]))
		}
		nics, _ = s.Desc.GetArray("nics")
	}

	cmd += " -device virtio-serial"
	cmd += " -usb"
	if !utils.IsInStringArray(s.getOsDistribution(), []string{OS_NAME_OPENWRT, OS_NAME_CIRROS}) &&
		!s.isOldWindows() && !s.isWindows10() &&
		!s.disableUsbKbd() {
		cmd += " -device usb-kbd"
	}
	if osname == OS_NAME_ANDROID {
		cmd += " -device usb-mouse"
	} else if !s.isOldWindows() {
		cmd += " -device usb-tablet"
	}

	if s.IsVdiSpice() {
		cmd += s.generateSpiceArgs(vncPort)
	} else {
		if isolatedDevsParams != nil && len(isolatedDevsParams.Vga) > 0 {
			cmd += isolatedDevsParams.Vga
		} else {
			vga, err := s.Desc.GetString("vga")
			if err != nil {
				vga = "std"
			}
			cmd += fmt.Sprintf(" -vga %s", vga)
		}
		cmd += fmt.Sprintf(" -vnc :%d", vncPort)
		if options.HostOptions.SetVncPassword {
			cmd += ",password"
		}
	}

	cmd += " -object iothread,id=iothread0"

	cmd += s.generateDiskParams(disks, false)

	if osname != OS_NAME_MACOS {
		cmd += " -device ide-cd,drive=ide0-cd0,bus=ide.1"
		if !s.isQ35() {
			cmd += ",unit=1"
		}
		cmd += " -drive id=ide0-cd0,media=cdrom,if=none"
	}

	if cdrom != nil && cdrom.Contains("path") {
		cdromPath, _ := cdrom.GetString("path")
		if len(cdromPath) > 0 {
			if osname != OS_NAME_MACOS {
				cmd += fmt.Sprintf(",file=%s", cdromPath)
			} else {
				cmd += " -device ide-drive,drive=MacDVD"
				cmd += fmt.Sprintf(",bus=ide.%d", len(disks))
				cmd += " -drive id=MacDVD,if=none,snapshot=on"
				cmd += fmt.Sprintf(",file=%s", cdromPath)
			}
		}
	}

	for i := 0; i < len(nics); i++ {
		if osname == OS_NAME_VMWARE {
			nics[i].(*jsonutils.JSONDict).Set("driver", jsonutils.NewString("vmxnet3"))
		}
		nicCmd, err := s.getNetdevDesc(nics[i])
		if err != nil {
			return "", err
		} else {
			cmd += nicCmd
		}
		cmd += s.getVnicDesc(nics[i], false)
	}

	// USB 3.0
	cmd += fmt.Sprintf(" -device %s,id=usb", s.getUsbControllerType())
	if isolatedDevsParams != nil {
		for _, each := range isolatedDevsParams.Devices {
			cmd += each
		}
	}

	cmd += fmt.Sprintf(" -pidfile %s", s.GetPidFilePath())
	cmd += s.extraOptions()

	cmd += s.getQgaDesc()
	/*
		QIU Jian
		virtio-rng device may cause live migration failure
		qemu-system-x86_64: Unknown savevm section or instance '0000:00:05.0/virtio-rng' 0
		qemu-system-x86_64: load of migration failed: Invalid argument
	*/
	if options.HostOptions.EnableVirtioRngDevice && fileutils2.Exists("/dev/urandom") {
		cmd += " -object rng-random,filename=/dev/urandom,id=rng0"
		cmd += " -device virtio-rng-pci,rng=rng0,max-bytes=1024,period=1000"
	}

	// add serial device
	if !s.disableIsaSerialDev() {
		cmd += " -chardev pty,id=charserial0"
		cmd += " -device isa-serial,chardev=charserial0,id=serial0"
	}

	if jsonutils.QueryBoolean(data, "need_migrate", false) {
		migratePort := s.manager.GetFreePortByBase(LIVE_MIGRATE_PORT_BASE)
		s.Desc.Set("live_migrate_dest_port", jsonutils.NewInt(int64(migratePort)))
		if jsonutils.QueryBoolean(data, "live_migrate_use_tls", false) {
			s.Desc.Set("live_migrate_use_tls", jsonutils.JSONTrue)
			cmd += fmt.Sprintf(" -incoming defer")
		} else {
			cmd += fmt.Sprintf(" -incoming tcp:0:%d", migratePort)
		}
	} else if jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
		cmd += fmt.Sprintf(" -incoming tcp:0:%d",
			s.manager.GetFreePortByBase(LIVE_MIGRATE_PORT_BASE))
	} else if jsonutils.QueryBoolean(s.Desc, "is_master", false) {
		cmd += " -S"
	}
	// cmd += fmt.Sprintf(" -D %s", path.Join(s.HomeDir(), "log"))
	if !s.disablePvpanicDev() {
		cmd += " -device pvpanic"
	}

	cmd += "\"\n"
	// cmd += "if [ ! -z \"$STATE_FILE\" ] && [ -d \"$STATE_FILE\" ] && [ -f \"$STATE_FILE/content\" ]; then\n"
	// cmd += "    $CMD --incoming \"exec: cat $STATE_FILE/content\"\n"
	// cmd += "elif [ ! -z \"$STATE_FILE\" ] && [ -f $STATE_FILE ]; then\n"
	// cmd += "    $CMD --incoming \"exec: cat $STATE_FILE\"\n"
	// cmd += "else\n"
	cmd += "eval $CMD\n"
	// cmd += "fi\n"

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
	cmd += "if [ \"$1\" != \"--force\" ] && [ -f $VNC_FILE ]; then\n"
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
