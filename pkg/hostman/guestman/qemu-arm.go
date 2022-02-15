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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	options "yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

func (s *SKVMGuestInstance) getArmMachine() string {
	return "virt-2.12"
}

// arm cpu unsupport hackintosh
func (s *SKVMGuestInstance) generateArmStartScript(data *jsonutils.JSONDict) (string, error) {
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
	if s.IsKvmSupport() && !options.HostOptions.DisableKVM {
		cmd += "QEMU_CMD_KVM_ARG=-enable-kvm\n"
	} else {
		cmd += "QEMU_CMD_KVM_ARG=\n"
	}
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
		if options.HostOptions.HostCpuPassthrough {
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

	cmd += fmt.Sprintf(" -cpu %s", cpuType)
	if options.HostOptions.LogLevel == "debug" {
		cmd += fmt.Sprintf(" -D %s -d all", s.getQemuLogPath())
	}

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
	cmd += fmt.Sprintf(" -smp cpus=%d,sockets=2,cores=32,maxcpus=64", cpu)
	cmd += fmt.Sprintf(" -name %s", name)
	if options.HostOptions.EnableVmUuid {
		cmd += fmt.Sprintf(" -uuid %s", uuid)
	}
	cmd += fmt.Sprintf(" -m %dM,slots=4,maxmem=262144M", mem)

	if options.HostOptions.HugepagesOption == "native" {
		cmd += fmt.Sprintf(" -object memory-backend-file,id=mem,size=%dM,mem-path=/dev/hugepages/%s,share=on,prealloc=on -numa node,memdev=mem", mem, uuid)
	} else {
		cmd += fmt.Sprintf(" -object memory-backend-ram,id=mem,size=%dM -numa node,memdev=mem", mem)
	}

	bootOrder, _ := s.Desc.GetString("boot_order")
	cmd += fmt.Sprintf(" -boot order=%s", bootOrder)

	// arm unsupport bios
	cmd += fmt.Sprintf(" -bios %s", options.HostOptions.OvmfPath)

	cmd += " -device qemu-xhci,p2=8,p3=8,id=usb1"
	cmd += " -device usb-tablet,id=input0,bus=usb1.0,port=1"
	cmd += " -device usb-kbd,id=input1,bus=usb1.0,port=2"
	cmd += " -device virtio-gpu-pci,id=video0,max_outputs=1"

	if s.IsVdiSpice() {
		cmd += s.generateSpiceArgs(vncPort)
	} else {
		cmd += fmt.Sprintf(" -vnc :%d", vncPort)
		if options.HostOptions.SetVncPassword {
			cmd += ",password"
		}
	}

	cmd += " -object iothread,id=iothread0"

	cmd += s.generateDiskParams(disks, true)

	cdrom, _ := s.Desc.Get("cdrom")
	if cdrom != nil && cdrom.Contains("path") {
		cdromPath, _ := cdrom.GetString("path")
		if len(cdromPath) > 0 {
			cmd += " -device virtio-scsi-device -device scsi-cd,drive=cd0,share-rw=true"
			cmd += fmt.Sprintf(" -drive if=none,file=%s,id=cd0,media=cdrom", cdromPath)
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
		// aarch64 with addr lead to:
		// virtio_net: probe of virtioN failed with error -22
		cmd += s.getVnicDesc(nics[i], false)
	}

	if isolatedDevsParams != nil {
		for _, each := range isolatedDevsParams.Devices {
			cmd += each
		}
	}

	cmd += fmt.Sprintf(" -pidfile %s", s.GetPidFilePath())
	cmd += s.extraOptions()

	// cmd += s.getQgaDesc()
	if options.HostOptions.EnableVirtioRngDevice && fileutils2.Exists("/dev/urandom") {
		cmd += " -object rng-random,filename=/dev/urandom,id=rng0"
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

func (s *SKVMGuestInstance) generateSpiceArgs(vncPort int64) string {
	cmd := ""
	cmd += " -device qxl-vga,id=video0,ram_size=141557760,vram_size=141557760"
	cmd += " -device intel-hda,id=sound0"
	cmd += " -device hda-duplex,id=sound0-codec0,bus=sound0.0,cad=0"
	cmd += fmt.Sprintf(" -spice port=%d,password=87654312,seamless-migration=on", 5900+vncPort)
	// # ,streaming-video=all,playback-compression=on,jpeg-wan-compression=always,zlib-glz-wan-compression=always,image-compression=glz" % (5900+vnc_port)
	cmd += fmt.Sprintf(" -device virtio-serial-pci,id=virtio-serial0,max_ports=16,bus=%s", s.GetPciBus())
	cmd += " -chardev spicevmc,name=vdagent,id=vdagent"
	cmd += " -device virtserialport,nr=1,bus=virtio-serial0.0,chardev=vdagent,name=com.redhat.spice.0"

	// # usb redirect
	cmd += " -device ich9-usb-ehci1,id=usbspice"
	cmd += " -device ich9-usb-uhci1,masterbus=usbspice.0,firstport=0,multifunction=on"
	cmd += " -device ich9-usb-uhci2,masterbus=usbspice.0,firstport=2"
	cmd += " -device ich9-usb-uhci3,masterbus=usbspice.0,firstport=4"
	cmd += " -chardev spicevmc,name=usbredir,id=usbredirchardev1"
	cmd += " -device usb-redir,chardev=usbredirchardev1,id=usbredirdev1"
	cmd += " -chardev spicevmc,name=usbredir,id=usbredirchardev2"
	cmd += " -device usb-redir,chardev=usbredirchardev2,id=usbredirdev2"
	return cmd
}
