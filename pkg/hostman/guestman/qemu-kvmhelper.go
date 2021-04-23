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
	"time"

	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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

func (s *SKVMGuestInstance) getMachine() string {
	machine, err := s.Desc.GetString("machine")
	if err != nil {
		machine = "pc"
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
	return s.getMachine() == "q35"
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

func (s *SKVMGuestInstance) getDriveDesc(disk jsonutils.JSONObject, format string) string {
	diskIndex, _ := disk.Int("index")
	cacheMode, _ := disk.GetString("cache_mode")
	aioMode, _ := disk.GetString("aio_mode")

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
	cmd += fmt.Sprintf(",aio=%s", aioMode)
	if disk.Contains("url") { // # a remote file backed image
		cmd += ",copy-on-read=on"
	}
	// #cmd += ",media=disk"
	return cmd
}

func (s *SKVMGuestInstance) GetDiskAddr(idx int) int {
	var base = 5
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

func (s *SKVMGuestInstance) getVdiskDesc(disk jsonutils.JSONObject) string {
	diskIndex, _ := disk.Int("index")
	diskDriver, _ := disk.GetString("driver")
	isSsd := jsonutils.QueryBoolean(disk, "is_ssd", false)

	var cmd = ""
	cmd += fmt.Sprintf(" -device %s", s.GetDiskDeviceModel(diskDriver))
	cmd += fmt.Sprintf(",drive=drive_%d", diskIndex)
	if diskDriver == DISK_DRIVER_VIRTIO {
		cmd += fmt.Sprintf(",bus=%s,addr=0x%x", s.GetPciBus(), s.GetDiskAddr(int(diskIndex)))
	} else if utils.IsInStringArray(diskDriver, []string{DISK_DRIVER_SCSI, DISK_DRIVER_PVSCSI}) {
		cmd += ",bus=scsi.0"
	} else if diskDriver == DISK_DRIVER_IDE {
		cmd += fmt.Sprintf(",bus=ide.%d,unit=%d", diskIndex/2, diskIndex%2)
	} else if diskDriver == DISK_DRIVER_SATA {
		cmd += fmt.Sprintf(",bus=ide.%d", diskIndex)
	}
	cmd += fmt.Sprintf(",id=drive_%d", diskIndex)
	if isSsd {
		cmd += ",rotation_rate=1"
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
	cmd := " -netdev type=tap"
	cmd += fmt.Sprintf(",id=%s", ifname)
	cmd += fmt.Sprintf(",ifname=%s", ifname)
	if driver == "virtio" && s.IsKvmSupport() {
		cmd += ",vhost=on,vhostforce=off"
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

	cmd := fmt.Sprintf(" -device %s", s.getNicDeviceModel(driver))
	cmd += fmt.Sprintf(",id=netdev-%s", ifname)
	cmd += fmt.Sprintf(",netdev=%s", ifname)
	cmd += fmt.Sprintf(",mac=%s", mac)

	if withAddr {
		cmd += fmt.Sprintf(",addr=0x%x", s.getNicAddr(int(index)))
	}
	if driver == "virtio" {
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

func (s *SKVMGuestInstance) _generateStartScript(data *jsonutils.JSONDict) (string, error) {
	var (
		uuid, _  = s.Desc.GetString("uuid")
		mem, _   = s.Desc.Int("mem")
		cpu, _   = s.Desc.Int("cpu")
		name, _  = s.Desc.GetString("name")
		nics, _  = s.Desc.GetArray("nics")
		disks, _ = s.Desc.GetArray("disks")
		osname   = s.getOsname()
		cmd      = ""
	)

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
		cmd += fmt.Sprintf("mount -t hugetlbfs -o size=%dM hugetlbfs-%s /dev/hugepages/%s\n",
			mem, uuid, uuid)
	}

	cmd += "sleep 1\n"
	cmd += fmt.Sprintf("echo %d > %s\n", vncPort, s.GetVncFilePath())

	for _, disk := range disks {
		diskPath, _ := disk.GetString("path")
		d := storageman.GetManager().GetDiskByPath(diskPath)
		if d == nil {
			return "", fmt.Errorf("get disk %s by storage error", diskPath)
		}

		diskIndex, _ := disk.Int("index")
		cmd += d.GetDiskSetupScripts(int(diskIndex))
	}

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
	if s.IsKvmSupport() {
		cmd += "QEMU_CMD_KVM_ARG=-enable-kvm\n"
	} else {
		cmd += "QEMU_CMD_KVM_ARG=-no-kvm\n"
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
                echo ",host_mtu=$(($origmtu - 58))"
	fi
    fi
}
`

	// Generate Start VM script
	cmd += `CMD="$QEMU_CMD`
	var accel, cpuType string
	if s.IsKvmSupport() {
		cmd += " -enable-kvm"
		accel = "kvm"
		cpuType = ""
		if osname == OS_NAME_MACOS {
			cpuType = "Penryn,vendor=GenuineIntel"
		} else if options.HostOptions.HostCpuPassthrough {
			cpuType = "host"
		} else {
			cpuType = "qemu64"
			if sysutils.IsProcessorIntel() {
				cpuType += ",+vmx"
				cpuType += ",+ssse3,+sse4.1,+sse4.2,-x2apic,+aes,+avx"
				cpuType += ",+vme,+pat,+ss,+pclmulqdq,+xsave"
				cpuType += ",level=13"
			} else if sysutils.IsProcessorAmd() {
				cpuType += ",+svm"
			}
		}

		if !guestManager.GetHost().IsNestedVirtualization() {
			cpuType += ",kvm=off"
		}

		if isolatedDevsParams != nil && len(isolatedDevsParams.Cpu) > 0 {
			cpuType = isolatedDevsParams.Cpu
		}
	} else {
		cmd += " -no-kvm"
		accel = "tcg"
		cpuType = "qemu64"
	}

	cmd += fmt.Sprintf(" -cpu %s", cpuType)

	// TODO hmp - -
	cmd += s.getMonitorDesc("hmqmon", s.GetHmpMonitorPort(int(vncPort)), MODE_READLINE)
	if options.HostOptions.EnableQmpMonitor {
		cmd += s.getMonitorDesc("qmqmon", s.GetQmpMonitorPort(int(vncPort)), MODE_CONTROL)
	}

	cmd += " -rtc base=utc,clock=host,driftfix=none"
	cmd += " -daemonize"
	cmd += " -nodefaults -nodefconfig"
	cmd += " -no-kvm-pit-reinjection"
	cmd += " -global kvm-pit.lost_tick_policy=discard"
	cmd += fmt.Sprintf(" -machine %s,accel=%s", s.getMachine(), accel)
	cmd += " -k en-us"
	// #cmd += " -g 800x600"
	cmd += fmt.Sprintf(" -smp %d,maxcpus=255", cpu)
	cmd += fmt.Sprintf(" -name %s", name)
	// #cmd += fmt.Sprintf(" -uuid %s", self.desc["uuid"])
	cmd += fmt.Sprintf(" -m %dM,slots=4,maxmem=524288M", mem)

	if s.manager.host.IsHugepagesEnabled() {
		cmd += fmt.Sprintf(" -mem-prealloc -mem-path %s", fmt.Sprintf("/dev/hugepages/%s", uuid))
	}

	bootOrder, _ := s.Desc.GetString("boot_order")
	cmd += fmt.Sprintf(" -boot order=%s", bootOrder)
	cdrom, _ := s.Desc.Get("cdrom")
	if cdrom != nil && cdrom.Contains("path") {
		cmd += ",menu=on"
	}

	if s.getBios() == "UEFI" {
		cmd += fmt.Sprintf(" -bios %s", options.HostOptions.OvmfPath)
	}

	if osname == OS_NAME_MACOS {
		cmd += " -device isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc"
		for i := 0; i < len(disks); i++ {
			disk := disks[i].(*jsonutils.JSONDict)
			disk.Set("driver", jsonutils.NewString(DISK_DRIVER_SATA))
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
	if !utils.IsInStringArray(s.getOsDistribution(), []string{OS_NAME_OPENWRT, OS_NAME_CIRROS}) && !s.disableUsbKbd() {
		cmd += " -device usb-kbd"
	}
	// # if osname == self.OS_NAME_ANDROID:
	// #     cmd += " -device usb-mouse"
	// # else:
	cmd += " -device usb-tablet"

	if s.IsVdiSpice() {
		cmd += " -device qxl-vga,id=video0,ram_size=141557760,vram_size=141557760"
		cmd += " -device intel-hda,id=sound0"
		cmd += " -device hda-duplex,id=sound0-codec0,bus=sound0.0,cad=0"
		cmd += fmt.Sprintf(" -spice port=%d,password=87654312,seamless-migration=on", 5900+vncPort)
		// # ,streaming-video=all,playback-compression=on,jpeg-wan-compression=always,zlib-glz-wan-compression=always,image-compression=glz" % (5900+vnc_port)
		cmd += fmt.Sprintf(" -device virtio-serial-pci,id=virtio-serial0,max_ports=16,bus=%s", s.GetPciBus())
		cmd += " -chardev spicevmc,name=vdagent,id=vdagent"
		cmd += " -device virtserialport,nr=1,bus=virtio-serial0.0,chardev=vdagent,name=com.redhat.spice.0"

		// # usb redirect
		cmd += " -device ich9-usb-ehci1,id=usb"
		cmd += " -device ich9-usb-uhci1,masterbus=usb.0,firstport=0,multifunction=on"
		cmd += " -device ich9-usb-uhci2,masterbus=usb.0,firstport=2"
		cmd += " -device ich9-usb-uhci3,masterbus=usb.0,firstport=4"
		cmd += " -chardev spicevmc,name=usbredir,id=usbredirchardev1"
		cmd += " -device usb-redir,chardev=usbredirchardev1,id=usbredirdev1"
		cmd += " -chardev spicevmc,name=usbredir,id=usbredirchardev2"
		cmd += " -device usb-redir,chardev=usbredirchardev2,id=usbredirdev2"
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

	var diskDrivers = []string{}
	for _, disk := range disks {
		driver, _ := disk.GetString("driver")
		diskDrivers = append(diskDrivers, driver)
	}

	if utils.IsInStringArray(DISK_DRIVER_SCSI, diskDrivers) {
		cmd += " -device virtio-scsi-pci,id=scsi"
	} else if utils.IsInStringArray(DISK_DRIVER_PVSCSI, diskDrivers) {
		cmd += " -device pvscsi,id=scsi"
	}

	for _, disk := range disks {
		format, _ := disk.GetString("format")
		cmd += s.getDriveDesc(disk, format)
		cmd += s.getVdiskDesc(disk)
	}

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
		cmd += s.getVnicDesc(nics[i], true)
	}

	if isolatedDevsParams != nil {
		for _, each := range isolatedDevsParams.Devices {
			cmd += each
		}
	}

	cmd += fmt.Sprintf(" -pidfile %s", s.GetPidFilePath())
	extraOptions, _ := s.Desc.GetMap("extra_options")
	for k, v := range extraOptions {
		cmd += fmt.Sprintf(" -%s %s", k, v.String())
	}

	cmd += s.getQgaDesc()
	if fileutils2.Exists("/dev/random") {
		cmd += " -object rng-random,filename=/dev/random,id=rng0"
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
		cmd += fmt.Sprintf(" -incoming tcp:0:%d", migratePort)
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
	cmd += "$CMD\n"
	// cmd += "fi\n"

	return cmd, nil
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

	if s.manager.host.IsHugepagesEnabled() {
		cmd += fmt.Sprintf("if [ -d /dev/hugepages/%s ]; then\n", uuid)
		cmd += fmt.Sprintf("  umount /dev/hugepages/%s\n", uuid)
		cmd += fmt.Sprintf("  rm -rf /dev/hugepages/%s\n", uuid)
		cmd += "fi\n"
	}
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
