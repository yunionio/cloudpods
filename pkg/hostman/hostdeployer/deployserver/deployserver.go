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
	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/diskutils"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/consts"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/sysutils"
	"yunion.io/x/onecloud/pkg/util/winutils"
)

type DeployerServer struct {
	deployapi.UnimplementedDeployAgentServer
}

var _ deployapi.DeployAgentServer = &DeployerServer{}

func apiDiskInfo(req *deployapi.DiskInfo) qemuimg.SImageInfo {
	if req == nil {
		return qemuimg.SImageInfo{}
	}
	info := qemuimg.SImageInfo{
		Path: req.GetPath(),
	}
	passwd := req.GetEncryptPassword()
	if len(passwd) > 0 {
		info.Password = passwd
		info.EncryptFormat = qemuimg.TEncryptFormat(req.GetEncryptFormat())
		info.EncryptAlg = seclib2.TSymEncAlg(req.GetEncryptAlg())
	}
	return info
}

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
	log.Infof("********* Deploy guest fs on %#v", apiDiskInfo(req.DiskInfo))
	disk, err := diskutils.GetIDisk(diskutils.DiskParams{
		Hypervisor: req.GuestDesc.Hypervisor,
		DiskInfo:   apiDiskInfo(req.GetDiskInfo()),
		VddkInfo:   req.VddkInfo,
	}, DeployOption.ImageDeployDriver, false)
	if err != nil {
		log.Errorf("diskutils.GetIDisk fail %s", err)
		return new(deployapi.DeployGuestFsResponse), errors.Wrap(err, "GetIDisk")
	}
	defer disk.Cleanup()
	if len(req.GuestDesc.Hypervisor) == 0 {
		req.GuestDesc.Hypervisor = comapi.HYPERVISOR_KVM
	}
	if err := disk.Connect(); err != nil {
		log.Errorf("Failed to connect %s disk: %s", req.GuestDesc.Hypervisor, err)
		return new(deployapi.DeployGuestFsResponse), errors.Wrap(err, "Connect")
	}
	defer disk.Disconnect()
	root, err := disk.MountRootfs()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound && req.DeployInfo.IsInit {
			// if init deploy, ignore no partition error
			return new(deployapi.DeployGuestFsResponse), nil
		}
		log.Errorf("Failed mounting rootfs for %s disk: %s", req.GuestDesc.Hypervisor, err)
		return new(deployapi.DeployGuestFsResponse), err
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

func (*DeployerServer) ResizeFs(ctx context.Context, req *deployapi.ResizeFsParams) (res *deployapi.Empty, err error) {
	// There will be some occasional unknown panic, so temporarily capture panic here.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("DeployGuestFs: %s, stack:\n %s", req.String(), debug.Stack())
			msg := "panic: "
			if str, ok := r.(fmt.Stringer); ok {
				msg += str.String()
			}
			res, err = nil, errors.Error(msg)
		}
	}()
	log.Infof("********* Resize fs on %#v", apiDiskInfo(req.DiskInfo))
	disk, err := diskutils.GetIDisk(diskutils.DiskParams{
		Hypervisor: req.Hypervisor,
		DiskInfo:   apiDiskInfo(req.GetDiskInfo()),
		VddkInfo:   req.VddkInfo,
	}, DeployOption.ImageDeployDriver, false)
	if err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "GetIDisk fail")
	}
	defer disk.Cleanup()
	if err := disk.Connect(); err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "disk connect failed")
	}
	defer disk.Disconnect()

	unmount := func(root fsdriver.IRootFsDriver) error {
		err := disk.UmountRootfs(root)
		if err != nil {
			return errors.Wrap(err, "unmount rootfs")
		}
		return nil
	}

	root, err := disk.MountRootfs()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return new(deployapi.Empty), nil
		}
		return new(deployapi.Empty), errors.Wrapf(err, "disk.MountRootfs")
	}
	if !root.IsResizeFsPartitionSupport() {
		err := unmount(root)
		if err != nil {
			return new(deployapi.Empty), err
		}
		return new(deployapi.Empty), errors.ErrNotSupported
	}

	// must umount rootfs before resize partition
	err = unmount(root)
	if err != nil {
		return new(deployapi.Empty), err
	}
	err = disk.ResizePartition()
	if err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "resize disk partition")
	}
	return new(deployapi.Empty), nil
}

func (*DeployerServer) FormatFs(ctx context.Context, req *deployapi.FormatFsParams) (*deployapi.Empty, error) {
	log.Infof("********* Format fs on %#v", apiDiskInfo(req.DiskInfo))
	gd, err := diskutils.NewKVMGuestDisk(apiDiskInfo(req.GetDiskInfo()), DeployOption.ImageDeployDriver, false)
	if err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "NewKVMGuestDisk")
	}
	defer gd.Cleanup()

	if err := gd.Connect(); err == nil {
		defer gd.Disconnect()
		if err := gd.MakePartition(req.FsFormat); err == nil {
			err = gd.FormatPartition(req.FsFormat, req.Uuid)
			if err != nil {
				return new(deployapi.Empty), errors.Wrap(err, "FormatPartition")
			}
		} else {
			return new(deployapi.Empty), errors.Wrap(err, "MakePartition")
		}
	} else {
		log.Errorf("failed connect kvm disk %#v: %s", apiDiskInfo(req.DiskInfo), err)
	}
	return new(deployapi.Empty), nil
}

func (*DeployerServer) SaveToGlance(ctx context.Context, req *deployapi.SaveToGlanceParams) (*deployapi.SaveToGlanceResponse, error) {
	log.Infof("********* %#v save to glance", apiDiskInfo(req.DiskInfo))
	var (
		osInfo  string
		relInfo *deployapi.ReleaseInfo
	)
	kvmDisk, err := diskutils.NewKVMGuestDisk(apiDiskInfo(req.GetDiskInfo()), DeployOption.ImageDeployDriver, false)
	if err != nil {
		return new(deployapi.SaveToGlanceResponse), errors.Wrap(err, "NewKVMGuestDisk")
	}
	defer kvmDisk.Cleanup()

	ret := &deployapi.SaveToGlanceResponse{
		OsInfo:      osInfo,
		ReleaseInfo: relInfo,
	}

	err = kvmDisk.Connect()
	if err != nil {
		return ret, errors.Wrapf(err, "kvmDisk.Connect %#v", apiDiskInfo(req.DiskInfo))
	}
	defer kvmDisk.Disconnect()

	root, err := kvmDisk.MountKvmRootfs()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return ret, nil
		}
		return ret, errors.Wrapf(err, "MountKvmRootfs")
	}
	defer kvmDisk.UmountKvmRootfs(root)

	osInfo = root.GetOs()
	relInfo = root.GetReleaseInfo(root.GetPartition())
	if req.Compress {
		err = root.PrepareFsForTemplate(root.GetPartition())
		if err != nil {
			log.Errorf("PrepareFsForTemplate %s", err)
		}
	}
	if req.Compress {
		kvmDisk.Zerofree()
	}
	return ret, err
}

func (*DeployerServer) getImageInfo(kvmDisk *diskutils.SKVMGuestDisk) (*deployapi.ImageInfo, error) {
	// Fsck is executed during mount
	rootfs, err := kvmDisk.MountKvmRootfs()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return new(deployapi.ImageInfo), nil
		}
		return new(deployapi.ImageInfo), errors.Wrapf(err, "kvmDisk.MountKvmRootfs")
	}
	partition := rootfs.GetPartition()
	imageInfo := &deployapi.ImageInfo{
		OsInfo:               rootfs.GetReleaseInfo(partition),
		OsType:               rootfs.GetOs(),
		IsLvmPartition:       kvmDisk.IsLVMPartition(),
		IsReadonly:           partition.IsReadonly(),
		IsInstalledCloudInit: rootfs.IsCloudinitInstall(),
	}
	kvmDisk.UmountKvmRootfs(rootfs)

	// In case of deploy driver is guestfish, we can't mount
	// multi partition concurrent, so we need umount rootfs first
	imageInfo.IsUefiSupport = kvmDisk.DetectIsUEFISupport(rootfs)
	imageInfo.PhysicalPartitionType = partition.GetPhysicalPartitionType()
	log.Infof("ProbeImageInfo response %s", imageInfo)
	return imageInfo, nil
}

func (s *DeployerServer) ProbeImageInfo(ctx context.Context, req *deployapi.ProbeImageInfoPramas) (*deployapi.ImageInfo, error) {
	log.Infof("********* %#v probe image info", apiDiskInfo(req.DiskInfo))
	kvmDisk, err := diskutils.NewKVMGuestDisk(apiDiskInfo(req.GetDiskInfo()), DeployOption.ImageDeployDriver, true)
	if err != nil {
		return new(deployapi.ImageInfo), errors.Wrap(err, "NewKVMGuestDisk")
	}
	defer kvmDisk.Cleanup()

	if err := kvmDisk.Connect(); err != nil {
		log.Errorf("Failed to connect kvm disk %#v: %s", apiDiskInfo(req.DiskInfo), err)
		return new(deployapi.ImageInfo), errors.Wrap(err, "Disk connector failed to connect image")
	}
	defer kvmDisk.Disconnect()

	return s.getImageInfo(kvmDisk)
}

var connectedEsxiDisks = map[string]*diskutils.VDDKDisk{}

func (*DeployerServer) ConnectEsxiDisks(
	ctx context.Context, req *deployapi.ConnectEsxiDisksParams,
) (*deployapi.EsxiDisksConnectionInfo, error) {
	log.Infof("********* Connect esxi disks ...")
	var (
		err          error
		flatFilePath string
		ret          = new(deployapi.EsxiDisksConnectionInfo)
	)
	ret.Disks = make([]*deployapi.EsxiDiskInfo, len(req.AccessInfo))
	for i := 0; i < len(req.AccessInfo); i++ {
		disk, _ := diskutils.NewVDDKDisk(req.VddkInfo, req.AccessInfo[i].DiskPath, DeployOption.ImageDeployDriver, false)
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
	log.Infof("********* Disconnect esxi disks ...")
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

	// create /dev/lvm_remote
	err = s.checkLvmRemote()
	if err != nil {
		return errors.Wrap(err, "unable to checkLvmRemote")
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
	return nil
}

func (s *SDeployService) checkLvmRemote() error {
	_, err := os.Stat("/dev/lvm_remote")
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		err := os.Mkdir("/dev/lvm_remote", os.ModePerm)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

func (s *SDeployService) InitService() {
	log.Infof("exec socket path: %s", DeployOption.ExecutorSocketPath)
	if DeployOption.EnableRemoteExecutor {
		execlient.Init(DeployOption.ExecutorSocketPath)
		procutils.SetRemoteExecutor()
	}

	if err := s.PrepareEnv(); err != nil {
		log.Fatalln(err)
	}
	if err := fsdriver.Init(DeployOption.PrivatePrefixes, DeployOption.CloudrootDir); err != nil {
		log.Fatalln(err)
	}
	if DeployOption.ImageDeployDriver == consts.DEPLOY_DRIVER_LIBGUESTFS {
		if err := libguestfs.Init(3); err != nil {
			log.Fatalln(err)
		}
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
