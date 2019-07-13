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

package deployserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"google.golang.org/grpc"

	"yunion.io/x/log"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	execlient "yunion.io/x/onecloud/pkg/executor/client"
	"yunion.io/x/onecloud/pkg/hostman/diskutils"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
	"yunion.io/x/onecloud/pkg/util/winutils"
)

type DeployerServer struct{}

func (*DeployerServer) DeployGuestFs(ctx context.Context, req *deployapi.DeployParams,
) (*deployapi.DeployGuestFsResponse, error) {
	log.Infof("Deploy guest fs on %s", req.DiskPath)
	var kvmDisk = diskutils.NewKVMGuestDisk(req.DiskPath)
	defer kvmDisk.Disconnect()
	if !kvmDisk.Connect() {
		log.Infof("Failed to connect kvm disk")
		return new(deployapi.DeployGuestFsResponse), nil
	}

	root := kvmDisk.MountKvmRootfs()
	if root == nil {
		log.Infof("Failed mounting rootfs for kvm disk")
		return new(deployapi.DeployGuestFsResponse), nil
	}
	defer kvmDisk.UmountKvmRootfs(root)

	ret, err := guestfs.DoDeployGuestFs(root, req.GuestDesc, req.DeployInfo)
	if err != nil {
		return new(deployapi.DeployGuestFsResponse), err
	}
	if ret == nil {
		return new(deployapi.DeployGuestFsResponse), nil
	}
	return ret, nil
}

func (*DeployerServer) ResizeFs(ctx context.Context, req *deployapi.ResizeFsParams,
) (*deployapi.Empty, error) {
	log.Infof("Resize fs on %s", req.DiskPath)
	disk := diskutils.NewKVMGuestDisk(req.DiskPath)
	defer disk.Disconnect()
	if !disk.Connect() {
		return new(deployapi.Empty), errors.New("resize fs disk connect failed")
	}

	root := disk.MountKvmRootfs()
	if root == nil {
		err := disk.ResizePartition()
		return new(deployapi.Empty), err
	} else {
		if !root.IsResizeFsPartitionSupport() {
			disk.UmountKvmRootfs(root)
			return new(deployapi.Empty), nil
		} else {
			disk.UmountKvmRootfs(root)
			err := disk.ResizePartition()
			return new(deployapi.Empty), err
		}
	}
}

func (*DeployerServer) FormatFs(ctx context.Context, req *deployapi.FormatFsParams) (*deployapi.Empty, error) {
	log.Infof("Format fs on %s", req.DiskPath)
	gd := diskutils.NewKVMGuestDisk(req.DiskPath)
	defer gd.Disconnect()
	if gd.Connect() {
		if err := gd.MakePartition(req.FsFormat); err == nil {
			err = gd.FormatPartition(req.FsFormat, req.Uuid)
			if err != nil {
				return new(deployapi.Empty), err
			}
		} else {
			return new(deployapi.Empty), err
		}
	}
	return new(deployapi.Empty), nil
}

func (*DeployerServer) SaveToGlance(ctx context.Context, req *deployapi.SaveToGlanceParams) (*deployapi.SaveToGlanceResponse, error) {
	log.Infof("%s save to glance", req.DiskPath)
	var (
		kvmDisk = diskutils.NewKVMGuestDisk(req.DiskPath)
		osInfo  string
		relInfo *deployapi.ReleaseInfo
	)
	defer kvmDisk.Disconnect()
	if kvmDisk.Connect() {
		var err error
		func() {
			if root := kvmDisk.MountKvmRootfs(); root != nil {
				defer kvmDisk.UmountKvmRootfs(root)

				osInfo = root.GetOs()
				relInfo = root.GetReleaseInfo(root.GetPartition())
				if req.Compress {
					err = root.PrepareFsForTemplate(root.GetPartition())
				}
			}
		}()
		if err != nil {
			log.Errorln(err)
			return new(deployapi.SaveToGlanceResponse), err
		}

		if req.Compress {
			kvmDisk.Zerofree()
		}
	}
	return &deployapi.SaveToGlanceResponse{
		OsInfo:      osInfo,
		ReleaseInfo: relInfo,
	}, nil
}

func getImageInfo(kvmDisk *diskutils.SKVMGuestDisk, rootfs fsdriver.IRootFsDriver) *deployapi.ImageInfo {
	partition := rootfs.GetPartition()
	return &deployapi.ImageInfo{
		OsInfo:                rootfs.GetReleaseInfo(partition),
		OsType:                rootfs.GetOs(),
		IsUefiSupport:         kvmDisk.DetectIsUEFISupport(rootfs),
		IsLvmPartition:        kvmDisk.IsLVMPartition(),
		IsReadonly:            partition.IsReadonly(),
		PhysicalPartitionType: partition.GetPhysicalPartitionType(),
		IsInstalledCloudInit:  rootfs.IsCloudinitInstall(),
	}
}

func (*DeployerServer) ProbeImageInfo(ctx context.Context, req *deployapi.ProbeImageInfoPramas) (*deployapi.ImageInfo, error) {
	log.Infof("%s probe image info", req.DiskPath)
	kvmDisk := diskutils.NewKVMGuestDisk(req.DiskPath)
	defer kvmDisk.Disconnect()
	if !kvmDisk.Connect() {
		log.Infof("Failed to connect kvm disk")
		return new(deployapi.ImageInfo), errors.New("Disk connector failed to connect image")
	}

	// Fsck is executed during mount
	rootfs := kvmDisk.MountKvmRootfs()
	if rootfs == nil {
		return new(deployapi.ImageInfo), fmt.Errorf("Failed mounting rootfs for kvm disk")
	}
	defer kvmDisk.UmountKvmRootfs(rootfs)
	imageInfo := getImageInfo(kvmDisk, rootfs)
	log.Infof("ProbeImageInfo response %s", imageInfo)
	return imageInfo, nil
}

type SDeployService struct {
	*service.SServiceBase
}

func NewDeployService() *SDeployService {
	deployer := &SDeployService{}
	deployer.SServiceBase = service.NewBaseService(deployer)
	return deployer
}

func (s *SDeployService) RunService() {
	grpcServer := grpc.NewServer()
	deployapi.RegisterDeployAgentServer(grpcServer, &DeployerServer{})
	if fileutils2.Exists(DeployOption.DeployServerSocketPath) {
		if err := os.Remove(DeployOption.DeployServerSocketPath); err != nil {
			log.Fatalln(err)
		}
	}
	listener, err := net.Listen("unix", DeployOption.DeployServerSocketPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	log.Infof("Init net listener on %s succ", DeployOption.DeployServerSocketPath)
	grpcServer.Serve(listener)
}

func (s *SDeployService) FixPathEnv() error {
	var paths = []string{
		"/usr/local/sbin",
		"/usr/local/bin",
		"/sbin",
		"/bin",
		"/usr/sbin",
		"/usr/bin",
	}
	return os.Setenv("PATH", strings.Join(paths, ":"))
}

func (s *SDeployService) PrepareEnv() error {
	if err := s.FixPathEnv(); err != nil {
		return err
	}
	output, err := procutils.NewCommand("rmmod", "nbd").Output()
	if err != nil {
		log.Errorf("rmmod error: %s", output)
	}
	output, err = procutils.NewCommand("modprobe", "nbd", "max_part=16").Output()
	if err != nil {
		return fmt.Errorf("Failed to activate nbd device: %s", output)
	}
	err = nbd.Init()
	if err != nil {
		return err
	}

	// https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-bdi
	for i := 0; i < 16; i++ {
		nbdBdi := fmt.Sprintf("/sys/block/nbd%d/bdi/", i)
		sysutils.SetSysConfig(nbdBdi+"max_ratio", "0")
		sysutils.SetSysConfig(nbdBdi+"min_ratio", "0")
	}

	if !winutils.CheckTool(DeployOption.ChntpwPath) {
		log.Errorf("Failed to find chntpw tool")
	}

	output, err = procutils.NewCommand("pvscan").Output()
	if err != nil {
		log.Errorf("Failed exec lvm command pvscan: %s", output)
	}
	return nil
}

func (s *SDeployService) InitService() {
	common_options.ParseOptions(&DeployOption, os.Args, "host.conf", "deploy-server")
	log.Errorln(DeployOption.ExecSocketPath)
	execlient.Init(DeployOption.ExecSocketPath)
	procutils.SetSocketExecutor()
	if err := s.PrepareEnv(); err != nil {
		log.Fatalln(err)
	}
	fsdriver.Init(DeployOption.PrivatePrefixes)
	s.O = &DeployOption.BaseOptions
	if len(DeployOption.DeployServerSocketPath) == 0 {
		log.Fatalf("missing deploy server socket path")
	}
	// TODO implentment func onExit
	s.SignalTrap(nil)
}

func (s *SDeployService) OnExitService() {}
