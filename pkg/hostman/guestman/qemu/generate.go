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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type Monitor struct {
	Id   string
	Port uint
	Mode string
}

type GenerateStartOptionsInput struct {
	QemuVersion Version
	QemuArch    Arch

	CPUOption

	EnableUUID            bool
	UUID                  string
	Mem                   uint64
	Cpu                   uint
	Name                  string
	OsName                string
	HugepagesEnabled      bool
	IsQ35                 bool
	BootOrder             string
	CdromPath             string
	Nics                  []jsonutils.JSONObject
	OVNIntegrationBridge  string
	Disks                 []api.GuestdiskJsonDesc
	Devices               []string
	Machine               string
	BIOS                  string
	OVMFPath              string
	VNCPort               uint
	VNCPassword           bool
	IsolatedDevicesParams *isolated_device.QemuParams
	EnableLog             bool
	LogPath               string
	HMPMonitor            *Monitor
	QMPMonitor            *Monitor
	IsVdiSpice            bool
	SpicePort             uint
	PCIBus                string
	VGA                   string
	PidFilePath           string
	HomeDir               string
	ExtraOptions          []string
	EnableRNGRandom       bool
	EnableSerialDevice    bool
	NeedMigrate           bool
	LiveMigratePort       uint
	LiveMigrateUseTLS     bool
	IsSlave               bool
	IsMaster              bool
	EnablePvpanic         bool

	EncryptKeyPath string
}

func GenerateStartOptions(
	input *GenerateStartOptionsInput,
) (string, error) {
	drv, ok := GetCommand(input.QemuVersion, input.QemuArch)
	if !ok {
		return "", errors.Errorf("Qemu comand driver %s %s not registered", input.QemuVersion, input.QemuArch)
	}
	drvOpt := drv.GetOptions()

	opts := []string{}

	if input.IsolatedDevicesParams != nil && len(input.IsolatedDevicesParams.Cpu) > 0 {
		input.CPUOption.IsolatedDeviceCPU = input.IsolatedDevicesParams.Cpu
	}

	cpuOpt, accel, err := drvOpt.CPU(input.CPUOption, input.OsName)
	if err != nil {
		return "", errors.Wrap(err, "Get CPU option")
	}

	opts = append(opts, cpuOpt)

	if input.EnableLog {
		opts = append(opts, drvOpt.Log(input.EnableLog, input.LogPath))
	}

	// TODO hmp - -
	opts = append(opts, getMonitorOptions(drvOpt, input.HMPMonitor)...)
	if input.QMPMonitor != nil {
		opts = append(opts, getMonitorOptions(drvOpt, input.QMPMonitor)...)
	}

	opts = append(opts,
		drvOpt.RTC(),
		drvOpt.Daemonize(),
		drvOpt.Nodefaults(),
		drvOpt.Nodefconfig(),
		drvOpt.NoKVMPitReinjection(),
		drvOpt.Global(),
		drvOpt.Machine(input.Machine, accel),
		drvOpt.KeyboardLayoutLanguage("en-us"),
		drvOpt.SMP(input.Cpu),
		drvOpt.Name(input.Name),
		drvOpt.UUID(input.EnableUUID, input.UUID),
		drvOpt.Memory(input.Mem),
	)

	var memDev string
	if input.HugepagesEnabled {
		memDev = drvOpt.MemPath(input.Mem, fmt.Sprintf("/dev/hugepages/%s", input.UUID))
	} else {
		memDev = drvOpt.MemDev(input.Mem)
	}
	opts = append(opts, memDev)

	// bootOrder
	enableMenu := false
	if input.CdromPath != "" {
		enableMenu = true
	}
	opts = append(opts, drvOpt.Boot(input.BootOrder, enableMenu))

	// bios
	if input.BIOS == BIOS_UEFI {
		if input.OVMFPath == "" {
			return "", errors.Errorf("input OVMF path is empty")
		}
		opts = append(opts, drvOpt.BIOS(input.OVMFPath))
	}

	if input.OsName == OS_NAME_MACOS {
		opts = append(opts, drvOpt.Device("isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc"))
	}

	opts = append(opts, drvOpt.Device("virtio-serial"))
	// enable USB emulation
	opts = append(opts, drvOpt.USB())
	for _, device := range input.Devices {
		opts = append(opts, drvOpt.Device(device))
	}

	// vdi spice
	if input.IsVdiSpice {
		opts = append(opts, drvOpt.VdiSpice(input.SpicePort, input.PCIBus)...)
	} else {
		if input.IsolatedDevicesParams != nil && len(input.IsolatedDevicesParams.Vga) > 0 {
			opts = append(opts, drvOpt.VGA("", input.IsolatedDevicesParams.Vga))
		} else {
			if input.VGA != "" {
				opts = append(opts, drvOpt.VGA(input.VGA, ""))
			}
		}
		opts = append(opts, drvOpt.VNC(input.VNCPort, input.VNCPassword))
	}

	// iothread object
	opts = append(opts, drvOpt.Object("iothread", map[string]string{"id": "iothread0"}))

	isEncrypt := false
	if len(input.EncryptKeyPath) > 0 {
		opts = append(opts, drvOpt.Object("secret", map[string]string{"id": "sec0", "file": input.EncryptKeyPath, "format": "base64"}))
		isEncrypt = true
	}

	// genereate disk options
	opts = append(opts, generateDisksOptions(drvOpt, input.Disks, input.PCIBus, input.IsVdiSpice, isEncrypt)...)

	// cdrom
	opts = append(opts, drvOpt.Cdrom(input.CdromPath, input.OsName, input.IsQ35, len(input.Disks))...)

	// genereate nics
	nicOpts, err := generateNicOptions(drvOpt, input)
	if err != nil {
		return "", errors.Wrap(err, "generateNicOptions")
	}
	opts = append(opts, nicOpts...)

	// isolated devices
	// USB 3.0
	opts = append(opts, drvOpt.Device("qemu-xhci,id=usb"))
	if input.IsolatedDevicesParams != nil {
		for _, each := range input.IsolatedDevicesParams.Devices {
			opts = append(opts, each)
		}
	}

	// pidfile
	opts = append(opts, drvOpt.Pidfile(input.PidFilePath))

	// extra options
	if len(input.ExtraOptions) != 0 {
		opts = append(opts, input.ExtraOptions...)
	}

	// qga
	opts = append(opts, drvOpt.QGA(input.HomeDir)...)

	// random device
	if input.EnableRNGRandom {
		opts = append(opts, getRNGRandomOptions(drvOpt)...)
	}

	// serial device
	if input.EnableSerialDevice {
		opts = append(opts, drvOpt.SerialDevice()...)
	}

	// migrate options
	opts = append(opts, getMigrateOptions(drvOpt, input)...)

	// pvpanic device
	opts = append(opts, drvOpt.PvpanicDevice())

	return strings.Join(opts, " "), nil
}

func getMonitorOptions(drvOpt QemuOptions, input *Monitor) []string {
	if input == nil {
		return nil
	}
	idDev := input.Id + "dev"
	opts := []string{
		drvOpt.MonitorChardev(idDev, input.Port, "127.0.0.1"),
		drvOpt.Mon(idDev, input.Id, input.Mode),
	}
	return opts
}

func generateDisksOptions(drvOpt QemuOptions, disks []api.GuestdiskJsonDesc, pciBus string, isVdiSpice bool, isEncrypt bool) []string {
	opts := []string{}
	isArm := drvOpt.IsArm()
	firstDriver := make(map[string]bool)
	for _, disk := range disks {
		driver := disk.Driver
		if isArm && (driver == DISK_DRIVER_IDE || driver == DISK_DRIVER_SATA) {
			// unsupported configuration: IDE controllers are unsupported
			driver = DISK_DRIVER_SCSI
		}
		if driver == DISK_DRIVER_SCSI || driver == DISK_DRIVER_PVSCSI {
			if _, ok := firstDriver[driver]; !ok {
				switch driver {
				case DISK_DRIVER_SCSI:
					// FIXME: iothread will make qemu-monitor hang
					// REF: https://www.mail-archive.com/qemu-devel@nongnu.org/msg592729.html
					// cmd += " -device virtio-scsi-pci,id=scsi,iothread=iothread0,num_queues=4,vectors=5"
					opts = append(opts, drvOpt.Device("virtio-scsi-pci,id=scsi,num_queues=4,vectors=5"))
				case DISK_DRIVER_PVSCSI:
					opts = append(opts, drvOpt.Device("pvscsi,id=scsi"))
				}
				firstDriver[driver] = true
			}
		}
		opts = append(opts,
			getDiskDriveOption(drvOpt, disk, isArm, isEncrypt),
			getDiskDeviceOption(drvOpt, disk, isArm, pciBus, isVdiSpice),
		)
	}
	return opts
}

func getDiskDriveOption(drvOpt QemuOptions, disk api.GuestdiskJsonDesc, isArm bool, isEncrypt bool) string {
	format := disk.Format
	diskIndex := disk.Index
	cacheMode := disk.CacheMode
	aioMode := disk.AioMode

	opt := fmt.Sprintf("file=$DISK_%d", diskIndex)
	opt += ",if=none"
	opt += fmt.Sprintf(",id=drive_%d", diskIndex)
	if len(format) == 0 || format == "qcow2" {
		// pass    # qemu will automatically detect image format
	} else if format == "raw" {
		opt += ",format=raw"
	}
	opt += fmt.Sprintf(",cache=%s", cacheMode)
	if isLocalStorage(disk) {
		opt += fmt.Sprintf(",aio=%s", aioMode)
	}
	if len(disk.Url) > 0 { // # a remote file backed image
		opt += ",copy-on-read=on"
	}
	if isLocalStorage(disk) {
		opt += ",file.locking=off"
	}
	if isEncrypt {
		opt += ",encrypt.format=luks,encrypt.key-secret=sec0"
	}
	// #opt += ",media=disk"
	return drvOpt.Drive(opt)
}

func isLocalStorage(disk api.GuestdiskJsonDesc) bool {
	if disk.StorageType == api.STORAGE_LOCAL || len(disk.StorageType) == 0 {
		return true
	} else {
		return false
	}
}

func getDiskDeviceOption(optDrv QemuOptions, disk api.GuestdiskJsonDesc, isArm bool, pciBus string, isVdiSpice bool) string {
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

	var opt = ""
	opt += GetDiskDeviceModel(diskDriver)
	opt += fmt.Sprintf(",drive=drive_%d", diskIndex)
	if diskDriver == DISK_DRIVER_VIRTIO {
		// virtio-blk
		opt += fmt.Sprintf(",bus=%s,addr=0x%x", pciBus, GetDiskAddr(int(diskIndex), isVdiSpice))
		opt += fmt.Sprintf(",num-queues=%d,vectors=%d,iothread=iothread0", numQueues, numQueues+1)
	} else if utils.IsInStringArray(diskDriver, []string{DISK_DRIVER_SCSI, DISK_DRIVER_PVSCSI}) {
		opt += ",bus=scsi.0"
	} else if diskDriver == DISK_DRIVER_IDE {
		opt += fmt.Sprintf(",bus=ide.%d,unit=%d", diskIndex/2, diskIndex%2)
	} else if diskDriver == DISK_DRIVER_SATA {
		opt += fmt.Sprintf(",bus=ide.%d", diskIndex)
	}
	opt += fmt.Sprintf(",id=drive_%d", diskIndex)
	if isSsd {
		opt += ",rotation_rate=1"
	}
	return optDrv.Device(opt)

}

func GetNicAddr(index int, disksLen int, isoDevsLen int, isVdiSpice bool) int {
	var pciBase = 10
	if disksLen > 10 {
		pciBase = 20
	}
	if isoDevsLen > 0 {
		pciBase += 10
	}
	return GetDiskAddr(pciBase+index, isVdiSpice)
}

func GetDiskAddr(idx int, isVdiSpice bool) int {
	// host-bridge / isa-bridge / vga / serial / network / block / usb / rng
	var base = 7
	if isVdiSpice {
		base += 10
	}
	return base + idx
}

func GetDiskDeviceModel(driver string) string {
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

func generateNicOptions(drvOpt QemuOptions, input *GenerateStartOptionsInput) ([]string, error) {
	opts := []string{}
	nics := input.Nics
	/*
	 * withAddr := true
	 * if drvOpt.IsArm() {
	 * 	withAddr = false
	 * }
	 */
	withAddr := false
	for idx := range nics {
		netDevOpt, err := getNicNetdevOption(drvOpt, nics[idx], input.IsKVMSupport)
		if err != nil {
			return nil, errors.Wrapf(err, "getNicNetdevOption %s", nics[idx])
		}
		opts = append(opts,
			netDevOpt,
			// aarch64 with addr lead to:
			// virtio_net: probe of virtioN failed with error -22
			getNicDeviceOption(drvOpt, nics[idx], input, withAddr))
	}
	return opts, nil
}

func getNicNetdevOption(drvOpt QemuOptions, nic jsonutils.JSONObject, isKVMSupport bool) (string, error) {
	ifname, _ := nic.GetString("ifname")
	if ifname == "" {
		return "", errors.Error("ifname is empty")
	}
	driver, _ := nic.GetString("driver")
	upscript, _ := nic.GetString("upscript_path")
	if upscript == "" {
		return "", errors.Error("upscript_path is empty")
	}
	downscript, _ := nic.GetString("downscript_path")
	if downscript == "" {
		return "", errors.Error("downscript_path is empty")
	}
	opt := "-netdev type=tap"
	opt += fmt.Sprintf(",id=%s", ifname)
	opt += fmt.Sprintf(",ifname=%s", ifname)
	if driver == "virtio" && isKVMSupport {
		opt += ",vhost=on,vhostforce=off"
	}
	opt += fmt.Sprintf(",script=%s", upscript)
	opt += fmt.Sprintf(",downscript=%s", downscript)
	return opt, nil
}

func getNicDeviceOption(drvOpt QemuOptions, nic jsonutils.JSONObject, input *GenerateStartOptionsInput, withAddr bool) string {
	bridge, _ := nic.GetString("bridge")
	ifname, _ := nic.GetString("ifname")
	driver, _ := nic.GetString("driver")
	mac, _ := nic.GetString("mac")
	index, _ := nic.Int("index")
	vectors, _ := nic.Int("vectors")
	bw, _ := nic.Int("bw")

	cmd := fmt.Sprintf("-device %s", GetNicDeviceModel(driver))
	cmd += fmt.Sprintf(",id=netdev-%s", ifname)
	cmd += fmt.Sprintf(",netdev=%s", ifname)
	cmd += fmt.Sprintf(",mac=%s", mac)

	if withAddr {
		disksLen := len(input.Disks)
		isoDevsLen := 0
		if input.IsolatedDevicesParams != nil {
			isoDevsLen = len(input.IsolatedDevicesParams.Devices)
		}
		cmd += fmt.Sprintf(",addr=0x%x", GetNicAddr(int(index), disksLen, isoDevsLen, input.IsVdiSpice))
	}
	if driver == "virtio" {
		if nic.Contains("vectors") {
			cmd += fmt.Sprintf(",vectors=%d", vectors)
		}
		cmd += fmt.Sprintf("$(nic_speed %d)", bw)
		if bridge == input.OVNIntegrationBridge {
			cmd += fmt.Sprintf("$(nic_mtu %q)", bridge)
		}
	}
	return cmd
}

func GetNicDeviceModel(name string) string {
	if name == "virtio" {
		return "virtio-net-pci"
	} else if name == "e1000" {
		return "e1000-82545em"
	} else {
		return name
	}
}

func getRNGRandomOptions(drvOpt QemuOptions) []string {
	var randev string
	if fileutils2.Exists("/dev/urandom") {
		randev = "/dev/urandom"
	} else if fileutils2.Exists("/dev/random") {
		randev = "/dev/random"
	} else {
		return []string{}
	}
	return []string{
		drvOpt.Object("rng-random", map[string]string{
			"filename": randev,
			"id":       "rng0",
		}),
		drvOpt.Device("virtio-rng-pci,rng=rng0,max-bytes=1024,period=1000"),
	}
}

func getMigrateOptions(drvOpt QemuOptions, input *GenerateStartOptionsInput) []string {
	opts := []string{}
	if input.NeedMigrate {
		if input.LiveMigrateUseTLS {
			opts = append(opts, fmt.Sprintf("-incoming defer"))
		} else {
			opts = append(opts, fmt.Sprintf("-incoming tcp:0:%d", input.LiveMigratePort))
		}
	} else if input.IsSlave {
		opts = append(opts, fmt.Sprintf("-incoming tcp:0:%d", input.LiveMigratePort))
	} else if input.IsMaster {
		opts = append(opts, "-S")
	}
	return opts
}
