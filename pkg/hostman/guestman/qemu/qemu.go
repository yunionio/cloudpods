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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type Version string

const (
	Version_4_2_0  Version = "4.2.0"
	Version_4_0_1  Version = "4.0.1"
	Version_2_12_1 Version = "2.12.1"
	Version_9_0_1  Version = "9.0.1"
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
	Log(enable bool) string
	RTC() string
	FreezeCPU() string
	Daemonize() string
	Nodefaults() string
	Nodefconfig() string
	NoHpet() (bool, string)
	Global() string
	KeyboardLayoutLanguage(lang string) string
	Name(name string) string
	UUID(enable bool, uuid string) string
	MemPath(sizeMB uint64, p string) string
	MemDev(sizeMB uint64) string
	MemFd(sizeMB uint64) string
	Boot(order *string, enableMenu bool) string
	BIOS(ovmfPath, homedir string) (string, error)
	Device(devStr string) string
	Drive(driveStr string) string
	Chardev(backend string, id string, name string) string
	MonitorChardev(id string, port uint, host string) string
	Mon(chardev string, id string, mode string) string
	Object(typeName string, props map[string]string) string
	Pidfile(file string) string
	USB() string
	VNC(port uint, usePasswd bool) string
	Initrd(initrdPath string) string
	Kernel(kernelPath string) string
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

func (o baseOptions) Log(enable bool) string {
	if !enable {
		return ""
	}
	return "-d all"
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

func (o baseOptions) NoHpet() (bool, string) {
	return false, "-no-hpet"
}

func (o baseOptions) Nodefconfig() string {
	return "-nodefconfig"
}

/*
@subsection -no-kvm-pit-reinjection (since 1.3.0)

The “-no-kvm-pit-reinjection” argument is now a
synonym for setting “-global kvm-pit.lost_tick_policy=discard”.
*/
func (o baseOptions) Global() string {
	return "-global kvm-pit.lost_tick_policy=discard"
}

func (o baseOptions) KeyboardLayoutLanguage(lang string) string {
	return "-k " + lang
}

func (o baseOptions) Name(name string) string {
	return fmt.Sprintf(`-name '%s',debug-threads=on`, strings.ReplaceAll(name, " ", "_"))
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

func (o baseOptions) Boot(order *string, enableMenu bool) string {
	opts := []string{}
	if order != nil {
		opts = append(opts, *order)
	}
	if enableMenu {
		opts = append(opts, "menu=on")
	}
	if len(opts) == 0 {
		return ""
	}
	return fmt.Sprintf("-boot %s", strings.Join(opts, ","))
}

func (o baseOptions) BIOS(ovmfPath, homedir string) (string, error) {
	ovmfVarsPath := path.Join(homedir, "OVMF_VARS.fd")
	if !fileutils2.Exists(ovmfVarsPath) {
		err := procutils.NewRemoteCommandAsFarAsPossible("cp", "-f", ovmfPath, ovmfVarsPath).Run()
		if err != nil {
			return "", errors.Wrap(err, "failed copy ovmf vars")
		}
	}
	return fmt.Sprintf(
		"-drive if=pflash,format=raw,unit=0,file=%s,readonly=on -drive if=pflash,format=raw,unit=1,file=%s",
		ovmfPath, ovmfVarsPath,
	), nil
}

func (o baseOptions) Device(devStr string) string {
	return "-device " + devStr
}

func (o baseOptions) Drive(driveStr string) string {
	return "-drive " + driveStr
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

func (o baseOptions) Initrd(initrdPath string) string {
	return "-initrd " + initrdPath
}

func (o baseOptions) Kernel(kernelPath string) string {
	return "-kernel " + kernelPath
}

func (o baseOptions) VNC(port uint, usePasswd bool) string {
	opt := fmt.Sprintf("-vnc :%d", port)
	if usePasswd {
		opt += ",password"
	}
	return opt
}

type baseOptions_x86_64 struct {
	*baseOptions
}

func newBaseOptions_x86_64() *baseOptions_x86_64 {
	return &baseOptions_x86_64{
		baseOptions: newBaseOptions(Arch_x86_64),
	}
}

// qemu version grate or equal 8.0.0
type baseOptions_ge_800_x86_64 struct {
}

func (o baseOptions_ge_800_x86_64) NoHpet() (bool, string) {
	return true, "hpet=off"
}

func newBaseOptionsGE800_x86_64() *baseOptions_ge_800_x86_64 {
	return &baseOptions_ge_800_x86_64{}
}

// qemu version grate or equal 3.1.0
type baseOptions_ge_310 struct {
}

func (o baseOptions_ge_310) Nodefconfig() string {
	return "-no-user-config"
}

func newBaseOptionsGE310() *baseOptions_ge_310 {
	return &baseOptions_ge_310{}
}

type baseOptions_aarch64 struct {
	*baseOptions
}

func newBaseOptions_aarch64() *baseOptions_aarch64 {
	return &baseOptions_aarch64{
		baseOptions: newBaseOptions(Arch_aarch64),
	}
}

func (o baseOptions_aarch64) Global() string {
	return ""
}

func (o baseOptions_aarch64) NoHpet() (bool, string) {
	return false, ""
}
