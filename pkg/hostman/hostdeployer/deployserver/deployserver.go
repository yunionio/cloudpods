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
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
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

const CENTOS_VGNAME = "centos"

type DeployerServer struct{}

func (*DeployerServer) DeployGuestFs(ctx context.Context, req *deployapi.DeployParams,
) (res *deployapi.DeployGuestFsResponse, err error) {
	// There will be some occasional unknown panic, so temporarily capture panic here.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DeployGuestFs: %s", r)
			debug.PrintStack()
			msg := "panic: "
			if str, ok := r.(fmt.Stringer); ok {
				msg += str.String()
			}
			res, err = nil, errors.Error(msg)
		}
	}()
	log.Infof("Deploy guest fs on %s", req.DiskPath)
	var disk = diskutils.GetIDisk(req)
	if len(req.GuestDesc.Hypervisor) == 0 {
		req.GuestDesc.Hypervisor = comapi.HYPERVISOR_KVM
	}
	defer disk.Disconnect()
	if !disk.Connect() {
		log.Infof("Failed to connect %s disk", req.GuestDesc.Hypervisor)
		return new(deployapi.DeployGuestFsResponse), nil
	}
	root := disk.MountRootfs()
	if root == nil {
		log.Infof("Failed mounting rootfs for %s disk", req.GuestDesc.Hypervisor)
		return new(deployapi.DeployGuestFsResponse), nil
	}
	defer disk.UmountRootfs(root)

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
		return new(deployapi.Empty), errors.Error("resize fs disk connect failed")
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
		return new(deployapi.ImageInfo), errors.Error("Disk connector failed to connect image")
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

var connectedEsxiDisks = map[string]*diskutils.VDDKDisk{}

func (*DeployerServer) ConnectEsxiDisks(
	ctx context.Context, req *deployapi.ConnectEsxiDisksParams,
) (*deployapi.EsxiDisksConnectionInfo, error) {
	log.Infof("Connect esxi disks ...")
	var (
		err          error
		flatFilePath string
		ret          = new(deployapi.EsxiDisksConnectionInfo)
	)
	ret.Disks = make([]*deployapi.EsxiDiskInfo, len(req.AccessInfo))
	for i := 0; i < len(req.AccessInfo); i++ {
		disk := diskutils.NewVDDKDisk(req.VddkInfo, req.AccessInfo[i].DiskPath)
		flatFilePath, err = disk.ConnectBlockDevice()
		if err != nil {
			err = errors.Wrapf(err, "disk %s connect block device", req.AccessInfo[i].DiskPath)
			break
		}
		connectedEsxiDisks[flatFilePath] = disk
		ret.Disks[i] = &deployapi.EsxiDiskInfo{DiskPath: flatFilePath}
	}
	if err != nil {
		for i := 0; i < len(req.AccessInfo); i++ {
			if disk, ok := connectedEsxiDisks[req.AccessInfo[i].DiskPath]; ok {
				if e := disk.DisconnectBlockDevice(); e != nil {
					log.Errorf("disconnect disk %s: %s", req.AccessInfo[i].DiskPath, e)
				} else {
					delete(connectedEsxiDisks, req.AccessInfo[i].DiskPath)
				}
			}
		}
		return ret, err
	}
	return ret, nil
}

func (*DeployerServer) DisconnectEsxiDisks(
	ctx context.Context, req *deployapi.EsxiDisksConnectionInfo,
) (*deployapi.Empty, error) {
	log.Infof("Disconnect esxi disks ...")
	for i := 0; i < len(req.Disks); i++ {
		if disk, ok := connectedEsxiDisks[req.Disks[i].DiskPath]; ok {
			if e := disk.DisconnectBlockDevice(); e != nil {
				return new(deployapi.Empty), errors.Wrapf(e, "disconnect disk %s", req.Disks[i].DiskPath)
			} else {
				delete(connectedEsxiDisks, req.Disks[i].DiskPath)
			}
		} else {
			log.Warningf("esxi disk %s not connected", req.Disks[i].DiskPath)
			continue
		}
	}
	return new(deployapi.Empty), nil
}

type SDeployService struct {
	*service.SServiceBase

	grpcServer *grpc.Server
}

func NewDeployService() *SDeployService {
	deployer := &SDeployService{}
	deployer.SServiceBase = service.NewBaseService(deployer)
	return deployer
}

func (s *SDeployService) RunService() {
	s.grpcServer = grpc.NewServer()
	deployapi.RegisterDeployAgentServer(s.grpcServer, &DeployerServer{})
	if fileutils2.Exists(DeployOption.DeployServerSocketPath) {
		if conn, err := net.Dial("unix", DeployOption.DeployServerSocketPath); err == nil {
			conn.Close()
			log.Fatalf("socket %s already listening", DeployOption.DeployServerSocketPath)
		}

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
	s.grpcServer.Serve(listener)
}

func (s *SDeployService) FixPathEnv() error {
	var paths = []string{
		"/usr/bin",
		"/usr/local/sbin",
		"/usr/local/bin",
		"/sbin",
		"/bin",
		"/usr/sbin",
	}
	return os.Setenv("PATH", strings.Join(paths, ":"))
}

func (s *SDeployService) PrepareEnv() error {
	if err := s.FixPathEnv(); err != nil {
		return err
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible("rmmod", "nbd").Output()
	if err != nil {
		log.Errorf("rmmod error: %s", output)
	}
	output, err = procutils.NewRemoteCommandAsFarAsPossible("modprobe", "nbd", "max_part=16").Output()
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
		if winutils.CheckTool("/usr/bin/chntpw.static") {
			winutils.SetChntpwPath("/usr/bin/chntpw.static")
		} else {
			log.Errorf("Failed to find chntpw tool")
		}
	} else {
		winutils.SetChntpwPath(DeployOption.ChntpwPath)
	}

	output, err = procutils.NewCommand("pvscan").Output()
	if err != nil {
		log.Errorf("Failed exec lvm command pvscan: %s", output)
	}
	output, err = procutils.NewCommand("vgdisplay").Output()
	if err == nil {
		re := regexp.MustCompile(`\s+`)
		for _, line := range strings.Split(string(output), "\n") {
			s := strings.TrimSpace(line)
			if strings.HasPrefix(s, "VG Name") {
				data := re.Split(s, -1)
				if len(data) == 3 {
					vgName := data[2]
					if vgName == CENTOS_VGNAME {
						vgNewName := stringutils.UUID4()
						output, err := procutils.NewCommand("vgrename", vgName, vgNewName).Output()
						if err != nil {
							log.Errorf("vg rename failed %s %s", err, output)
						}
						log.Infof("vg name %s rename to %s success", vgName, vgNewName)
					}
				}
			}
		}
	}
	return nil
}

func (s *SDeployService) InitService() {
	common_options.ParseOptions(&DeployOption, os.Args, "host.conf", "deploy-server")
	log.Infof("exec socket path: %s", DeployOption.ExecSocketPath)
	if DeployOption.EnableRemoteExecutor {
		execlient.Init(DeployOption.ExecSocketPath)
		procutils.SetRemoteExecutor()
	}

	if err := s.PrepareEnv(); err != nil {
		log.Fatalln(err)
	}
	if err := fsdriver.Init(DeployOption.PrivatePrefixes); err != nil {
		log.Fatalln(err)
	}
	s.O = &DeployOption.BaseOptions
	if len(DeployOption.DeployServerSocketPath) == 0 {
		log.Fatalf("missing deploy server socket path")
	}
	s.SignalTrap(func() {
		for {
			if len(connectedEsxiDisks) > 0 {
				log.Warningf("Waiting for esxi disks %d disconnect !!!", len(connectedEsxiDisks))
				time.Sleep(time.Second * 1)
			} else {
				if s.grpcServer != nil {
					s.grpcServer.Stop()
				} else {
					os.Exit(0)
				}
			}
		}
	})
}

func (s *SDeployService) OnExitService() {}
