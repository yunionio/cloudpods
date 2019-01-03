package guestman

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

var (
	OS_NAME_LINUX   = "Linux"
	OS_NAME_WINDOWS = "Windows"
	OS_NAME_MACOS   = "macOS"
	OS_NAME_ANDROID = "Android"
	OS_NAME_VMWARE  = "VMWare"
)

func IsKvmSupport() bool {
	return hostinfo.Instance().IsKvmSupport()
}

func (s *SKVMGuestInstance) getOsname() string {
	if s.Desc.Contains("metadata") {
		metadata, _ := s.Desc.Get("metadata")
		if metadata.Contains("os_name") {
			osname, _ := metadata.GetString("os_name")
			return osname
		}
	}
	return OS_NAME_LINUX
}

func (s *SKVMGuestInstance) generateStartScript(data *jsonutils.JSONDict) (string, error) {
	osname := s.GetOsname()
	if osname == OS_NAME_MACOS {
		s.Desc.Set("machine", jsonutils.NewString("q35"))
		s.Desc.Set("bios", jsonutils.NewString("UEFI"))
	}

	vncPort, _ := data.Int("vnc_port")

	qemuVersion := options.HostOptions.DefaultQemuVersion
	if data.Contains("qemu_version") {
		qemuVersion, _ := data.GetString("qemu_version")
	}
	if qemuVersion == "latest" {
		qemuVersion = ""
	}

	// TODO: isolatedDevsParams := hostinfo.Instance()...

	nics, _ := s.Desc.GetArray("nics")
	for _, nic := range nics {
		downscript := s.getNicDownScriptPath(nic)
		ifname, _ := nic.GetString("ifnam")
		cmd += fmt.Sprintf("%s %s\n", downscript, ifname)
	}

	if options.HostOptions.HugepagesOption == "native" {
		uuid, _ := s.Desc.GetString("uuid")
		mem, _ := s.Desc.Int("mem")
		cmd += fmt.Sprintf("mkdir -p /dev/hugepages/%s\n", uuid)
		cmd += fmt.Sprintf("mount -t hugetlbfs -o size=%dM hugetlbfs-%s /dev/hugepages/%s\n",
			mem, uuid, uuid)
	}

	cmd += "sleep 1\n"
	cmd += fmt.Sprintf("echo %d > %s\n", vncPort, s.GetVncFilePath())

	disks, _ := s.Desc.GetArray("disks")
	for _, disk := range disks {
		diskPath, _ := disk.GetString("path")
		d := storageman.GetManager().GetDiskByPath(diskPath)
		if d == nil {
			return fmt.Errorf("get disk %s by storage error", diskPath)
		}

		diskIndex, _ := disk.Int("index")
		// TODO
		cmd += d.GetDiskSetupScripts(diskIndex)
	}

	cmd += fmt.Sprintf("STATE_FILE=`ls -d %s* | head -n 1`\n", s.getStateFilePathRootPrefix())

	var qemuCmd = qemutils.GetQemu(qemuVersion)
	cmd += fmt.Sprintf("DEFAULT_QEMU_CMD='%s'\n", qemu_cmd)
	cmd += `if [ -n "$STATE_FILE" ]; then\n`
	cmd += "    QEMU_VER=`echo $STATE_FILE" +
		` | grep -o '_[[:digit:]]\+\.[[:digit:]]\+.*'` + "`\n"
	cmd += `    QEMU_CMD="qemu-system-x86_64"\n`
	cmd += `    QEMU_LOCAL_PATH="/usr/local/bin/$QEMU_CMD"\n`
	cmd += `    QEMU_LOCAL_PATH_VER="/usr/local/qemu-$QEMU_VER/bin/$QEMU_CMD"\n`
	cmd += `    QEMU_BIN_PATH="/usr/bin/$QEMU_CMD"\n`
	cmd += `    if [ -f "$QEMU_LOCAL_PATH_VER" ]; then\n`
	cmd += `        QEMU_CMD=$QEMU_LOCAL_PATH_VER\n`
	cmd += `    elif [ -f "$QEMU_LOCAL_PATH" ]; then\n`
	cmd += `        QEMU_CMD=$QEMU_LOCAL_PATH\n`
	cmd += `    elif [ -f "$QEMU_BIN_PATH" ]; then\n`
	cmd += `        QEMU_CMD=$QEMU_BIN_PATH\n`
	cmd += `    fi\n`
	cmd += `else\n`
	cmd += `    QEMU_CMD=$DEFAULT_QEMU_CMD\n`
	cmd += `fi\n`
	cmd += `function nic_speed() {\n`
	cmd += `    $QEMU_CMD `

	var accel, cpuType string
	if s.IsKvmSupport() {
		cmd += " -enable-kvm"
		accel = "kvm"
		cpuType = ""
		if osname == OS_NAME_MACOS {
			cpuType = "Penryn,vendor=GenuineIntel"
		} else {
			cpuType = "host"
		}

		if !hostinfo.Instance().IsNestedVirtualization() {
			cpu_type += ",kvm=off"
		}

		// TODO
		// if isolated_devs_params.get('cpu', None):
		//   cpu_type = isolated_devs_params['cpu']
	} else {
		cmd += " -no-kvm"
		accel = "tcg"
		cpu_type = "qemu64"
	}

	cmd += fmt.Sprintf(" -cpu %s", cpu_type)

	// TODO hmp - -

}
