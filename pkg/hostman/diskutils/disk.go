package diskutils

import (
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type IDisk interface {
	Connect() bool
	Disconnect() bool
	MountRootfs() fsdriver.IRootFsDriver
	UmountRootfs(driver fsdriver.IRootFsDriver)
}

func GetIDisk(params *apis.DeployParams) IDisk {
	hypervisor := params.GuestDesc.Hypervisor
	switch hypervisor {
	case comapi.HYPERVISOR_KVM:
		return NewKVMGuestDisk(params.DiskPath)
	case comapi.HYPERVISOR_ESXI:
		return NewVDDKDisk(params.VddkInfo, params.DiskPath)
	default:
		return NewKVMGuestDisk(params.DiskPath)
	}
}
