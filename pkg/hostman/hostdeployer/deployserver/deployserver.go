package deployserver

import (
	"context"

	"yunion.io/x/log"

	diskutils "yunion.io/x/onecloud/pkg/hostman/diskutils"
	guestfs "yunion.io/x/onecloud/pkg/hostman/guestfs"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type DeployerServer struct{}

func (*DeployerServer) DeployGuestFs(ctx context.Context, req *deployapi.DeployParams) (*deployapi.DeployGuestFsResponse, error) {
	var kvmDisk = diskutils.NewKVMGuestDisk(req.DiskPath)
	defer kvmDisk.Disconnect()
	if !kvmDisk.Connect() {
		log.Infof("Failed to connect kvm disk")
		return nil, nil
	}

	root := kvmDisk.MountKvmRootfs()
	if root == nil {
		log.Infof("Failed mounting rootfs for kvm disk")
		return nil, nil
	}
	defer kvmDisk.UmountKvmRootfs(root)

	ret, err := guestfs.DeployGuestFs(root, guestDesc, deployInfo)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
