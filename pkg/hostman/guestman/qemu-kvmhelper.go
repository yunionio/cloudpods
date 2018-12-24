package guestman

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

var (
	OS_NAME_LINUX   = "Linux"
	OS_NAME_WINDOWS = "Windows"
	OS_NAME_MACOS   = "macOS"
	OS_NAME_ANDROID = "Android"
	OS_NAME_VMWARE  = "VMWare"
)

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
}
