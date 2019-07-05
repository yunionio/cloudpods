package deployserver

import (
	"context"
	"net"
	"os"

	"google.golang.org/grpc"
	"yunion.io/x/log"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	diskutils "yunion.io/x/onecloud/pkg/hostman/diskutils"
	guestfs "yunion.io/x/onecloud/pkg/hostman/guestfs"
	fsdriver "yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type DeployerServer struct{}

func (*DeployerServer) DeployGuestFs(ctx context.Context, req *deployapi.DeployParams,
) (*deployapi.DeployGuestFsResponse, error) {
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

	ret, err := guestfs.DoDeployGuestFs(root, req.GuestDesc, req.DeployInfo)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type SDeployService struct {
	*service.SServiceBase
}

func (s *SDeployService) RunService() {
	grpcServer := grpc.NewServer()
	deployapi.RegisterDeployAgentServer(grpcServer, &DeployerServer{})
	listener, err := net.Listen("unix", DeployOption.DeployServerSocketPath)
	if err != nil {
		log.Fatalln(err)
	}
	grpcServer.Serve(listener)
}

func (s *SDeployService) InitService() {
	common_options.ParseOptions(&DeployOption, os.Args, "host.conf", "deploy-server")
	fsdriver.Init(DeployOption.PrivatePrefixes)
	s.O = &DeployOption.BaseOptions
}

func (s *SDeployService) OnExitService() {}
