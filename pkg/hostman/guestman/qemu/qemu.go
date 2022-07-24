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

package qemu

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
)

type Version string

const (
	Version_4_2_0  Version = "4.2.0"
	Version_4_0_1  Version = "4.0.1"
	Version_2_12_1 Version = "2.12.1"
)

type Arch string

const (
	Arch_x86_64  Arch = "x86_64"
	Arch_aarch64 Arch = "aarch64"
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

	BIOS_UEFI = "UEFI"
)

type QemuCommand interface {
	GetVersion() Version
	GetArch() Arch
	GetOptions() QemuOptions
}

type CPUOption struct {
	EnableKVM          bool
	IsKVMSupport       bool
	HostCPUPassthrough bool
	IsCPUIntel         bool
	IsCPUAMD           bool
	EnableNested       bool
	IsolatedDeviceCPU  string
}

type QemuOptions interface {
	IsArm() bool
	CPU(opt CPUOption, osName string) (string, string, error)
	Log(enable bool, qemuLogPath string) string
	RTC() string
	FreezeCPU() string
	Daemonize() string
	Nodefaults() string
	Nodefconfig() string
	NoKVMPitReinjection() string
	Global() string
	Machine(machineType string, accel string) string
	KeyboardLayoutLanguage(lang string) string
	SMP(cpus uint) string
	Name(name string) string
	UUID(enable bool, uuid string) string
	Memory(sizeMB uint64) string
	MemPath(sizeMB uint64, p string) string
	MemDev(sizeMB uint64) string
	MemFd(sizeMB uint64) string
	Boot(order string, enableMenu bool) string
	BIOS(file string) string
	Device(devStr string) string
	Drive(driveStr string) string
	Spice(port uint, password string) string
	Chardev(backend string, id string, name string) string
	MonitorChardev(id string, port uint, host string) string
	Mon(chardev string, id string, mode string) string
	Object(typeName string, props map[string]string) string
	Pidfile(file string) string
	USB() string
	VdiSpice(spicePort uint, pciBus string) []string
	VNC(port uint, usePasswd bool) string
	VGA(vType string, alterOpt string) string
	Cdrom(cdromPath string, osName string, isQ35 bool, disksLen int) []string
	SerialDevice() []string
	QGA(homeDir string) []string
	PvpanicDevice() string
}

var (
	cmdDrivers = new(sync.Map)
)

func getCommandKey(version Version, arch Arch) string {
	return fmt.Sprintf("%s-%s", version, arch)
}

func RegisterCmd(cmds ...QemuCommand) {
	for _, cmd := range cmds {
		_, ok := GetCommand(cmd.GetVersion(), cmd.GetArch())
		if ok {
			log.Fatalf("cmd 'version %s' 'arch %s' already registered!", cmd.GetVersion(), cmd.GetArch())
		}
		cmdDrivers.Store(getCommandKey(cmd.GetVersion(), cmd.GetArch()), cmd)
	}
}

func GetCommand(version Version, arch Arch) (QemuCommand, bool) {
	key := getCommandKey(version, arch)
	val, ok := cmdDrivers.Load(key)
	if !ok {
		return nil, false
	}
	return val.(QemuCommand), true
}

func EnsureGetCommand(version Version, arch Arch) QemuCommand {
	cmd, ok := GetCommand(version, arch)
	if !ok {
		log.Fatalf("cmd %s %s not registered!", version, arch)
	}
	return cmd
}

type baseCommand struct {
	version Version
	arch    Arch
	options QemuOptions
}

func newBaseCommand(version Version, arch Arch, opts QemuOptions) QemuCommand {
	return &baseCommand{
		version: version,
		arch:    arch,
		options: opts,
	}
}

func (c *baseCommand) GetVersion() Version {
	return c.version
}

func (c *baseCommand) GetArch() Arch {
	return c.arch
}

func (c *baseCommand) GetOptions() QemuOptions {
	return c.options
}

type baseOptions struct {
	arch Arch
}

func newBaseOptions(arch Arch) *baseOptions {
	return &baseOptions{
		arch: arch,
	}
}

func (o baseOptions) IsArm() bool {
	return o.arch == Arch_aarch64
}

func (o baseOptions) Log(enable bool, qemuLogPath string) string {
	if !enable {
		return ""
	}
	return fmt.Sprintf("-D %s -d all", qemuLogPath)
}

func (o baseOptions) RTC() string {
	return "-rtc base=utc,clock=host,driftfix=none"
}

func (o baseOptions) Daemonize() string {
	return "-daemonize"
}

func (o baseOptions) FreezeCPU() string {
	return "-S"
}

func (o baseOptions) Nodefaults() string {
	return "-nodefaults"
}

func (o baseOptions) Nodefconfig() string {
	return "-nodefconfig"
}

func (o baseOptions) NoKVMPitReinjection() string {
	return "-no-kvm-pit-reinjection"
}

func (o baseOptions) Global() string {
	return "-global kvm-pit.lost_tick_policy=discard"
}

func (o baseOptions) KeyboardLayoutLanguage(lang string) string {
	return "-k " + lang
}

func (o baseOptions) Name(name string) string {
	return fmt.Sprintf(`-name '%s',debug-threads=on`, name)
}

func (o baseOptions) UUID(enable bool, uuid string) string {
	if !enable {
		return ""
	}
	return "-uuid " + uuid
}

func (o baseOptions) MemPrealloc() string {
	return "-mem-prealloc"
}

func (o baseOptions) MemPath(sizeMB uint64, p string) string {
	return fmt.Sprintf("-object memory-backend-file,id=mem,size=%dM,mem-path=%s,share=on,prealloc=on -numa node,memdev=mem", sizeMB, p)
}

func (o baseOptions) MemDev(sizeMB uint64) string {
	return fmt.Sprintf("-object memory-backend-ram,id=mem,size=%dM -numa node,memdev=mem", sizeMB)
}

func (o baseOptions) MemFd(sizeMB uint64) string {
	return fmt.Sprintf("-object memory-backend-memfd,id=mem,size=%dM,share=on,prealloc=on -numa node,memdev=mem", sizeMB)
}

func (o baseOptions) Boot(order string, enableMenu bool) string {
	opt := "-boot order=" + order
	if enableMenu {
		opt += ",menu=on"
	}
	return opt
}

func (o baseOptions) BIOS(file string) string {
	return "-bios " + file
}

func (o baseOptions) Device(devStr string) string {
	return "-device " + devStr
}

func (o baseOptions) Drive(driveStr string) string {
	return "-drive " + driveStr
}

func (o baseOptions) Spice(port uint, password string) string {
	return fmt.Sprintf("-spice port=%d,password=%s,seamless-migration=on", port, password)
}

func (o baseOptions) Chardev(backend string, id string, name string) string {
	opt := fmt.Sprintf("-chardev %s,id=%s", backend, id)
	if name != "" {
		opt = fmt.Sprintf("%s,name=%s", opt, name)
	}
	return opt
}

func (o baseOptions) MonitorChardev(id string, port uint, host string) string {
	opt := o.Chardev("socket", id, "")
	return fmt.Sprintf("%s,port=%d,host=%s,nodelay,server,nowait", opt, port, host)
}

func (o baseOptions) Mon(chardev string, id string, mode string) string {
	return fmt.Sprintf("-mon chardev=%s,id=%s,mode=%s", chardev, id, mode)
}

func (o baseOptions) Object(typeName string, props map[string]string) string {
	propStrs := []string{}
	for k, v := range props {
		propStrs = append(propStrs, fmt.Sprintf("%s=%s", k, v))
	}
	opt := fmt.Sprintf("-object %s", typeName)
	if len(propStrs) > 0 {
		opt = opt + "," + strings.Join(propStrs, ",")
	}
	return opt
}

func (o baseOptions) Pidfile(file string) string {
	return "-pidfile " + file
}

func (o baseOptions) USB() string {
	return "-usb"
}

func (o baseOptions) VdiSpice(spicePort uint, pciBus string) []string {
	return []string{
		o.Device("intel-hda,id=sound0"),
		o.Device("hda-duplex,id=sound0-codec0,bus=sound0.0,cad=0"),
		fmt.Sprintf("-spice port=%d,disable-ticketing=off,seamless-migration=on", spicePort),
		// # ,streaming-video=all,playback-compression=on,jpeg-wan-compression=always,zlib-glz-wan-compression=always,image-compression=glz" % (5900+vnc_port)
		o.Device(fmt.Sprintf("virtio-serial-pci,id=virtio-serial0,max_ports=16,bus=%s", pciBus)),
		o.Chardev("spicevmc", "vdagent", "vdagent"),
		o.Device("virtserialport,nr=1,bus=virtio-serial0.0,chardev=vdagent,name=com.redhat.spice.0"),

		// usb redirect
		o.Device("ich9-usb-ehci1,id=usbspice"),
		o.Device("ich9-usb-uhci1,masterbus=usbspice.0,firstport=0,multifunction=on"),
		o.Device("ich9-usb-uhci2,masterbus=usbspice.0,firstport=2"),
		o.Device("ich9-usb-uhci3,masterbus=usbspice.0,firstport=4"),
		o.Chardev("spicevmc", "usbredirchardev1", "usbredir"),
		o.Device("usb-redir,chardev=usbredirchardev1,id=usbredirdev1"),
		o.Chardev("spicevmc", "usbredirchardev2", "usbredir"),
		o.Device("usb-redir,chardev=usbredirchardev2,id=usbredirdev2"),
	}
}

func (o baseOptions) VNC(port uint, usePasswd bool) string {
	opt := fmt.Sprintf("-vnc :%d", port)
	if usePasswd {
		opt += ",password"
	}
	return opt
}

func (o baseOptions) VGA(vType string, alternativeOpt string) string {
	if alternativeOpt != "" {
		return alternativeOpt
	}
	return "-vga " + vType
}

type baseOptions_x86_64 struct {
	*baseOptions
}

func newBaseOptions_x86_64() *baseOptions_x86_64 {
	return &baseOptions_x86_64{
		baseOptions: newBaseOptions(Arch_x86_64),
	}
}

func (o *baseOptions_x86_64) CPU(input CPUOption, osName string) (string, string, error) {
	var accel, cpuType string
	if input.EnableKVM {
		accel = "kvm"
		cpuType = ""
		if osName == OS_NAME_MACOS {
			cpuType = "Penryn,vendor=GenuineIntel"
		} else if input.HostCPUPassthrough {
			cpuType = "host"
			// https://unix.stackexchange.com/questions/216925/nmi-received-for-unknown-reason-20-do-you-have-a-strange-power-saving-mode-ena
			cpuType += ",+kvm_pv_eoi"
		} else {
			cpuType = "qemu64"
			cpuType += ",+kvm_pv_eoi"
			if input.IsCPUIntel {
				cpuType += ",+vmx"
				cpuType += ",+ssse3,+sse4.1,+sse4.2,-x2apic,+aes,+avx"
				cpuType += ",+vme,+pat,+ss,+pclmulqdq,+xsave"
				cpuType += ",level=13"
			} else if input.IsCPUAMD {
				cpuType += ",+svm"
			}
		}

		if !input.EnableNested {
			cpuType += ",kvm=off"
		}

		if len(input.IsolatedDeviceCPU) > 0 {
			cpuType = input.IsolatedDeviceCPU
		}
	} else {
		accel = "tcg"
		cpuType = "qemu64"
	}
	return fmt.Sprintf("-cpu %s", cpuType), accel, nil
}

func (o baseOptions_x86_64) Machine(mType string, accel string) string {
	return fmt.Sprintf("-machine %s,accel=%s", mType, accel)
}

func (o baseOptions_x86_64) SMP(cpus uint) string {
	return fmt.Sprintf("-smp cpus=%d,sockets=2,cores=64,maxcpus=128", cpus)
}

func (o baseOptions_x86_64) Memory(sizeMB uint64) string {
	return fmt.Sprintf("-m %dM,slots=4,maxmem=524288M", sizeMB)
}

func (o baseOptions_x86_64) Cdrom(cdromPath string, osName string, isQ35 bool, disksLen int) []string {
	opts := []string{}
	cdromDrv := "id=ide0-cd0,media=cdrom,if=none"
	if osName != OS_NAME_MACOS {
		tmpOpt := "ide-cd,drive=ide0-cd0,bus=ide.1"
		if isQ35 {
			tmpOpt += ",unit=1"
		}
		opts = append(opts, o.Device(tmpOpt))
	}
	if cdromPath != "" {
		if osName != OS_NAME_MACOS {
			cdromDrv += fmt.Sprintf(",file=%s", cdromPath)
		} else {
			opts = append(opts,
				o.Device(fmt.Sprintf("ide-drive,drive=MacDVD,bus=ide.%d", disksLen)),
				o.Drive(fmt.Sprintf("id=MacDVD,if=none,snapshot=on,file=%s", cdromPath)))
		}
	}
	if osName != OS_NAME_MACOS {
		opts = append(opts, o.Drive(cdromDrv))
	}
	return opts
}

func (o baseOptions_x86_64) SerialDevice() []string {
	return []string{
		o.Chardev("pty", "charserial0", ""),
		o.Device("isa-serial,chardev=charserial0,id=serial0"),
	}
}

func (o baseOptions_x86_64) QGA(homeDir string) []string {
	return []string{
		fmt.Sprintf("-chardev socket,path=%s,server,nowait,id=qga0", path.Join(homeDir, "qga.sock")),
		o.Device("virtserialport,chardev=qga0,name=org.qemu.guest_agent.0"),
	}
}

func (o baseOptions_x86_64) PvpanicDevice() string {
	return o.Device("pvpanic")
}

func (o baseOptions_x86_64) VdiSpice(spicePort uint, pciBus string) []string {
	baseOpts := o.baseOptions.VdiSpice(spicePort, pciBus)
	vga := o.Device("qxl-vga,id=video0,ram_size=141557760,vram_size=141557760")
	return append([]string{vga}, baseOpts...)
}

type baseOptions_aarch64 struct {
	*baseOptions
}

func newBaseOptions_aarch64() *baseOptions_aarch64 {
	return &baseOptions_aarch64{
		baseOptions: newBaseOptions(Arch_aarch64),
	}
}

func (o *baseOptions_aarch64) CPU(input CPUOption, osName string) (string, string, error) {
	var accel, cpuType string
	if input.EnableKVM {
		accel = "kvm"
		if input.HostCPUPassthrough {
			cpuType = "host"
		} else {
			// * under KVM, -cpu max is the same as -cpu host
			// * under TCG, -cpu max means "emulate with as many features as possible"
			cpuType = "max"
		}
	} else {
		accel = "tcg"
		cpuType = "max"
	}
	return fmt.Sprintf("-cpu %s", cpuType), accel, nil
}

func (o baseOptions_aarch64) Machine(mType string, accel string) string {
	// TODO: fix machine type on region controller side
	if mType == "" || mType == compute.VM_MACHINE_TYPE_PC || mType == compute.VM_MACHINE_TYPE_Q35 {
		mType = "virt"
	}
	return fmt.Sprintf("-machine %s,accel=%s,gic-version=3", mType, accel)
}

func (o baseOptions_aarch64) NoKVMPitReinjection() string {
	return ""
}

func (o baseOptions_aarch64) Global() string {
	return ""
}

func (o baseOptions_aarch64) SMP(cpus uint) string {
	// warning: Number of hotpluggable cpus requested (128)
	// exceeds the recommended cpus supported by KVM (32)
	return fmt.Sprintf("-smp cpus=%d,sockets=2,cores=32,maxcpus=64", cpus)
}

func (o baseOptions_aarch64) Memory(sizeMB uint64) string {
	return fmt.Sprintf("-m %dM,slots=4,maxmem=262144M", sizeMB)
}

func (o baseOptions_aarch64) Cdrom(cdromPath string, osName string, isQ35 bool, disksLen int) []string {
	opts := []string{}
	if len(cdromPath) > 0 {
		opts = append(opts,
			o.Device("virtio-scsi-device -device scsi-cd,drive=cd0,share-rw=true"),
			o.Drive(fmt.Sprintf("if=none,file=%s,id=cd0,media=cdrom", cdromPath)))
	}
	return opts
}

func (o baseOptions_aarch64) SerialDevice() []string {
	return nil
}

func (o baseOptions_aarch64) QGA(_ string) []string {
	return nil
}

func (o baseOptions_aarch64) PvpanicDevice() string {
	// -device pvpanic: 'pvpanic' is not a valid device model name
	return ""
}

func (o baseOptions_aarch64) VdiSpice(spicePort uint, pciBus string) []string {
	return o.baseOptions.VdiSpice(spicePort, "pcie.0")
}
