package guestman

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	options "yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	fileutils2 "yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

func (s *SKVMGuestInstance) getArmMachine() string {
	return "virt-2.12"
}

func (s *SKVMGuestInstance) getArmVdiskDesc(disk jsonutils.JSONObject) string {
	diskIndex, _ := disk.Int("index")
	diskDriver, _ := disk.GetString("driver")

	if diskDriver == DISK_DRIVER_IDE || diskDriver == DISK_DRIVER_SATA {
		// unsupported configuration: IDE controllers are unsupported
		// for this QEMU binary or machine type
		// replace with scsi
		diskDriver = DISK_DRIVER_SCSI
	}

	var cmd = ""
	cmd += fmt.Sprintf(" -device %s", s.GetDiskDeviceModel(diskDriver))
	cmd += fmt.Sprintf(",drive=drive_%d", diskIndex)
	if diskDriver == DISK_DRIVER_VIRTIO {
		cmd += fmt.Sprintf(",bus=%s,addr=0x%x", s.GetPciBus(), s.GetDiskAddr(int(diskIndex)))
	} else if utils.IsInStringArray(diskDriver, []string{DISK_DRIVER_SCSI, DISK_DRIVER_PVSCSI}) {
		cmd += ",bus=scsi.0"
	}
	cmd += fmt.Sprintf(",id=drive_%d", diskIndex)
	return cmd
}

// arm cpu unsupport hackintosh
func (s *SKVMGuestInstance) generateArmStartScript(data *jsonutils.JSONDict) (string, error) {
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
		ifname, _ := nic.GetString("ifname")
		cmd += fmt.Sprintf("%s %s\n", downscript, ifname)
	}

	if options.HostOptions.HugepagesOption == "native" {
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

	cmd += fmt.Sprintf("STATE_FILE=`ls -d %s* | head -n 1`\n", s.getStateFilePathRootPrefix())
	cmd += fmt.Sprintf("PID_FILE=%s\n", s.GetPidFilePath())

	var qemuCmd = qemutils.GetQemu(qemuVersion)
	if len(qemuCmd) == 0 {
		qemuCmd = qemutils.GetQemu("")
	}

	cmd += fmt.Sprintf("DEFAULT_QEMU_CMD='%s'\n", qemuCmd)
	cmd += "if [ -n \"$STATE_FILE\" ]; then\n"
	cmd += "    QEMU_VER=`echo $STATE_FILE" +
		` | grep -o '_[[:digit:]]\+\.[[:digit:]]\+.*'` + "`\n"
	cmd += "    QEMU_CMD=\"qemu-system-x86_64\"\n"
	cmd += "    QEMU_LOCAL_PATH=\"/usr/local/bin/$QEMU_CMD\"\n"
	cmd += "    QEMU_LOCAL_PATH_VER=\"/usr/local/qemu-$QEMU_VER/bin/$QEMU_CMD\"\n"
	cmd += "    QEMU_BIN_PATH=\"/usr/bin/$QEMU_CMD\"\n"
	cmd += "    if [ -f \"$QEMU_LOCAL_PATH_VER\" ]; then\n"
	cmd += "        QEMU_CMD=$QEMU_LOCAL_PATH_VER\n"
	cmd += "    elif [ -f \"$QEMU_LOCAL_PATH\" ]; then\n"
	cmd += "        QEMU_CMD=$QEMU_LOCAL_PATH\n"
	cmd += "    elif [ -f \"$QEMU_BIN_PATH\" ]; then\n"
	cmd += "        QEMU_CMD=$QEMU_BIN_PATH\n"
	cmd += "    fi\n"
	cmd += "else\n"
	cmd += "    QEMU_CMD=$DEFAULT_QEMU_CMD\n"
	cmd += "fi\n"
	cmd += "function nic_speed() {\n"
	cmd += "    $QEMU_CMD "

	if s.IsKvmSupport() {
		cmd += "-enable-kvm"
	} else {
		cmd += "-no-kvm"
	}

	cmd += " -device virtio-net-pci,? 2>&1 | grep .speed= > /dev/null\n"
	cmd += "    if [ \"$?\" -eq \"0\" ]; then\n"
	cmd += "        echo \",speed=$1\"\n"
	cmd += "    fi\n"
	cmd += "}\n"

	// Generate Start VM script
	cmd += `CMD="$QEMU_CMD`
	var accel, cpuType string
	if s.IsKvmSupport() {
		cmd += " -enable-kvm"
		accel = "kvm"
		if options.HostOptions.HostCpuPassthrough {
			cpuType = "host"
		} else {
			// * under KVM, -cpu max is the same as -cpu host
			// * under TCG, -cpu max means "emulate with as many features as possible"
			cpuType = "max"
		}
	} else {
		cmd += " -no-kvm"
		accel = "tcg"
		cpuType = "max"
	}

	cmd += fmt.Sprintf(" -cpu %s", cpuType)

	cmd += s.getMonitorDesc("hmqmon", s.GetHmpMonitorPort(int(vncPort)), MODE_READLINE)
	if options.HostOptions.EnableQmpMonitor {
		cmd += s.getMonitorDesc("qmqmon", s.GetQmpMonitorPort(int(vncPort)), MODE_CONTROL)
	}

	cmd += " -rtc base=utc,clock=host,driftfix=none"
	cmd += " -daemonize"
	cmd += " -nodefaults -nodefconfig"
	cmd += fmt.Sprintf(" -machine %s,accel=%s,gic-version=3", s.getArmMachine(), accel)
	cmd += " -k en-us"

	// warning: Number of hotpluggable cpus requested (128)
	// exceeds the recommended cpus supported by KVM (32)
	cmd += fmt.Sprintf(" -smp %d,maxcpus=32", cpu)
	cmd += fmt.Sprintf(" -name %s", name)
	cmd += fmt.Sprintf(" -m %dM,slots=4,maxmem=262144M", mem)

	if options.HostOptions.HugepagesOption == "native" {
		cmd += fmt.Sprintf(" -mem-prealloc -mem-path %s", fmt.Sprintf("/dev/hugepages/%s", uuid))
	}

	bootOrder, _ := s.Desc.GetString("boot_order")
	cmd += fmt.Sprintf(" -boot order=%s", bootOrder)

	// arm unsupport bios
	cmd += fmt.Sprintf(" -bios %s", options.HostOptions.OvmfPath)

	cmd += " -device qemu-xhci,p2=8,p3=8,id=usb"
	cmd += " -device usb-tablet,id=input0,bus=usb.0,port=1"
	cmd += " -device usb-kbd,id=input1,bus=usb.0,port=2"
	cmd += " -device virtio-gpu-pci,id=video0,max_outputs=1"

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
		cmd += fmt.Sprintf(" -vnc :%d", vncPort)
		if options.HostOptions.SetVncPassword {
			cmd += ",password"
		}
	}

	var diskDrivers = []string{}
	for _, disk := range disks {
		diskDriver, _ := disk.GetString("driver")
		if diskDriver == DISK_DRIVER_IDE || diskDriver == DISK_DRIVER_SATA {
			// unsupported configuration: IDE controllers are unsupported
			// for this QEMU binary or machine type
			// replace with scsi
			diskDriver = DISK_DRIVER_SCSI
		}
		diskDrivers = append(diskDrivers, diskDriver)
	}

	if utils.IsInStringArray(DISK_DRIVER_SCSI, diskDrivers) {
		cmd += " -device virtio-scsi-pci,id=scsi"
	} else if utils.IsInStringArray(DISK_DRIVER_PVSCSI, diskDrivers) {
		cmd += " -device pvscsi,id=scsi"
	}

	for _, disk := range disks {
		format, _ := disk.GetString("format")
		cmd += s.getDriveDesc(disk, format)
		cmd += s.getArmVdiskDesc(disk)
	}

	cdrom, _ := s.Desc.Get("cdrom")
	if cdrom != nil && cdrom.Contains("path") {
		cdromPath, _ := cdrom.GetString("path")
		if len(cdromPath) > 0 {
			cmd += fmt.Sprintf("-cdrom %s", cdromPath)
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
		cmd += s.getVnicDesc(nics[i])
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

	// cmd += s.getQgaDesc()
	if fileutils2.Exists("/dev/random") {
		cmd += " -object rng-random,filename=/dev/random,id=rng0"
		cmd += " -device virtio-rng-pci,rng=rng0,max-bytes=1024,period=1000"
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

	// -device pvpanic: 'pvpanic' is not a valid device model name
	// if !s.disablePvpanicDev() {
	// 	cmd += " -device pvpanic"
	// }

	cmd += "\"\n"
	cmd += "if [ ! -z \"$STATE_FILE\" ] && [ -d \"$STATE_FILE\" ] && [ -f \"$STATE_FILE/content\" ]; then\n"
	cmd += "    $CMD --incoming \"exec: cat $STATE_FILE/content\"\n"
	cmd += "elif [ ! -z \"$STATE_FILE\" ] && [ -f $STATE_FILE ]; then\n"
	cmd += "    $CMD --incoming \"exec: cat $STATE_FILE\"\n"
	cmd += "else\n"
	cmd += "    $CMD\n"
	cmd += "fi\n"

	return cmd, nil
}
