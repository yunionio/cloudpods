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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	commonconsts "yunion.io/x/onecloud/pkg/cloudcommon/consts"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/diskutils"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/qemu_kvm"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/consts"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
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

	if err := disk.Connect(req.GuestDesc); err != nil {
		log.Errorf("Failed to connect %s disk: %s", req.GuestDesc.Hypervisor, err)
		return new(deployapi.DeployGuestFsResponse), errors.Wrap(err, "Connect")
	}
	defer disk.Disconnect()

	ret, err := disk.DeployGuestfs(req)
	if ret == nil {
		ret = new(deployapi.DeployGuestFsResponse)
	}
	if err != nil {
		log.Errorf("failed deploy guest fs: %s", err)
	}
	return ret, err
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

	if strings.HasPrefix(req.DiskInfo.Path, "/dev/loop") {
		// HACK: container loop device, do resize locally
		if err := fsutils.ResizeDiskFs(req.DiskInfo.Path, 0, true); err != nil {
			return new(deployapi.Empty), errors.Wrap(err, "fsutils.ResizeDiskFs")
		}
		return new(deployapi.Empty), nil
	}

	disk, err := diskutils.GetIDisk(diskutils.DiskParams{
		Hypervisor: req.Hypervisor,
		DiskInfo:   apiDiskInfo(req.GetDiskInfo()),
		VddkInfo:   req.VddkInfo,
	}, DeployOption.ImageDeployDriver, false)
	if err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "GetIDisk fail")
	}
	defer disk.Cleanup()

	if err := disk.Connect(nil); err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "disk connect failed")
	}
	defer disk.Disconnect()

	return disk.ResizeFs()
}

func (*DeployerServer) FormatFs(ctx context.Context, req *deployapi.FormatFsParams) (*deployapi.Empty, error) {
	log.Infof("********* Format fs on %#v", apiDiskInfo(req.DiskInfo))
	gd, err := diskutils.NewKVMGuestDisk(apiDiskInfo(req.GetDiskInfo()), DeployOption.ImageDeployDriver, false)
	if err != nil {
		return new(deployapi.Empty), errors.Wrap(err, "NewKVMGuestDisk")
	}
	defer gd.Cleanup()

	if err := gd.Connect(nil); err == nil {
		defer gd.Disconnect()
		return gd.FormatFs(req)
	} else {
		log.Errorf("failed connect kvm disk %#v: %s", apiDiskInfo(req.DiskInfo), err)
	}
	return new(deployapi.Empty), nil
}

func (*DeployerServer) SaveToGlance(ctx context.Context, req *deployapi.SaveToGlanceParams) (*deployapi.SaveToGlanceResponse, error) {
	log.Infof("********* %#v save to glance", apiDiskInfo(req.DiskInfo))

	kvmDisk, err := diskutils.NewKVMGuestDisk(apiDiskInfo(req.GetDiskInfo()), DeployOption.ImageDeployDriver, false)
	if err != nil {
		return new(deployapi.SaveToGlanceResponse), errors.Wrap(err, "NewKVMGuestDisk")
	}
	defer kvmDisk.Cleanup()

	err = kvmDisk.Connect(nil)
	if err != nil {
		return new(deployapi.SaveToGlanceResponse), errors.Wrapf(err, "kvmDisk.Connect %#v", apiDiskInfo(req.DiskInfo))
	}
	defer kvmDisk.Disconnect()

	return kvmDisk.SaveToGlance(req)
}

func (s *DeployerServer) ProbeImageInfo(ctx context.Context, req *deployapi.ProbeImageInfoPramas) (*deployapi.ImageInfo, error) {
	log.Infof("********* %#v probe image info", apiDiskInfo(req.DiskInfo))
	kvmDisk, err := diskutils.NewKVMGuestDisk(apiDiskInfo(req.GetDiskInfo()), DeployOption.ImageDeployDriver, true)
	if err != nil {
		return new(deployapi.ImageInfo), errors.Wrap(err, "NewKVMGuestDisk")
	}
	defer kvmDisk.Cleanup()

	if err := kvmDisk.Connect(nil); err != nil {
		log.Errorf("Failed to connect kvm disk %#v: %s", apiDiskInfo(req.DiskInfo), err)
		return new(deployapi.ImageInfo), errors.Wrap(err, "Disk connector failed to connect image")
	}
	defer kvmDisk.Disconnect()

	return kvmDisk.ProbeImageInfo(req)
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
	if DeployOption.DeployAction != "" {
		res, err := StartLocalDeploy(DeployOption.DeployAction)
		if err != nil {
			if e := fileutils2.FilePutContents("/error", err.Error(), false); e != nil {
				log.Errorf("failed put errors to file: %s", e)
			}
		}
		if res != nil {
			resStr, _ := json.Marshal(res)
			if e := fileutils2.FilePutContents("/response", string(resStr), false); e != nil {
				log.Errorf("failed put response to file: %s", e)
			}
		}
		return
	}

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

func InitEnvCommon() error {
	commonconsts.SetAllowVmSELinux(DeployOption.AllowVmSELinux)

	fsutils.SetExt4UsageTypeThresholds(
		int64(DeployOption.Ext4LargefileSizeGb)*1024*1024*1024,
		int64(DeployOption.Ext4HugefileSizeGb)*1024*1024*1024,
	)
	if err := fsdriver.Init(DeployOption.CloudrootDir); err != nil {
		return errors.Wrap(err, "init fsdriver")
	}

	netutils.SetPrivatePrefixes(DeployOption.CustomizedPrivatePrefixes)
	return nil
}

func (s *SDeployService) PrepareEnv() error {
	if !fileutils2.Exists(DeployOption.DeployTempDir) {
		err := os.MkdirAll(DeployOption.DeployTempDir, 0755)
		if err != nil {
			return errors.Errorf("fail to create %s: %s", DeployOption.DeployTempDir, err)
		}
	}
	commonconsts.SetDeployTempDir(DeployOption.DeployTempDir)

	if err := InitEnvCommon(); err != nil {
		return err
	}

	if err := s.FixPathEnv(); err != nil {
		return err
	}

	if DeployOption.ImageDeployDriver == consts.DEPLOY_DRIVER_LIBGUESTFS {
		if err := libguestfs.Init(3); err != nil {
			log.Fatalln(err)
		}
	}

	if DeployOption.ImageDeployDriver != consts.DEPLOY_DRIVER_QEMU_KVM {
		if err := nbd.Init(); err != nil {
			return errors.Wrap(err, "nbd.Init")
		}
	} else {
		cpuArch, err := procutils.NewCommand("uname", "-m").Output()
		if err != nil {
			return errors.Wrap(err, "get cpu architecture")
		}
		cpuArchStr := strings.TrimSpace(string(cpuArch))

		// prepare for yunionos don't have but necessary files
		out, err := procutils.NewCommand("mkdir", "-p", "/opt/yunion/bin/bundles").Output()
		if err != nil {
			return errors.Wrapf(err, "cp files failed %s", out)
		}
		copyFiles := map[string]string{
			"/usr/bin/chntpw.static":         "/opt/yunion/bin/chntpw.static",
			"/usr/bin/.chntpw.static.bin":    "/opt/yunion/bin/.chntpw.static.bin",
			"/usr/bin/bundles/chntpw.static": "/opt/yunion/bin/bundles/chntpw.static",
			"/usr/bin/growpart":              "/opt/yunion/bin/growpart",
			"/usr/sbin/zerofree":             "/opt/yunion/bin/zerofree",
		}
		// x86_64 or aarch64
		if cpuArchStr == qemu_kvm.OS_ARCH_AARCH64 {
			copyFiles["/yunionos/aarch64/qemu-ga"] = fsdriver.QGA_BINARY_PATH
		} else {
			copyFiles["/yunionos/x86_64/qemu-ga"] = fsdriver.QGA_BINARY_PATH
			copyFiles["/yunionos/x86_64/qemu-ga-x86_64.msi"] = fsdriver.QGA_WIN_MSI_INSTALLER_PATH
		}

		for k, v := range copyFiles {
			out, err = procutils.NewCommand("cp", "-rf", k, v).Output()
			if err != nil {
				return errors.Wrapf(err, "cp files failed %s", out)
			}
		}

		{
			newOptions := DeployOption
			newOptions.CommonConfigFile = ""
			// save runtime options to /opt/yunion/host.conf
			conf := jsonutils.Marshal(newOptions).YAMLString()
			log.Debugf("deploy options: %s", conf)
			err := ioutil.WriteFile("/opt/yunion/host.conf", []byte(conf), 0600)
			if err != nil {
				return errors.Wrapf(err, "save host.conf failed %s", out)
			}
		}

		err = procutils.NewCommand("mkdir", "-p", qemu_kvm.RUN_ON_HOST_ROOT_PATH).Run()
		if err != nil {
			return errors.Wrap(err, "Failed to mkdir RUN_ON_HOST_ROOT_PATH: %s")
		}

		cmd := fmt.Sprintf("mkisofs -l -J -L -R -r -v -hide-rr-moved -o %s -graft-points vmware-vddk=/opt/vmware-vddk yunion=/opt/yunion", qemu_kvm.DEPLOY_ISO)
		out, err = procutils.NewCommand("bash", "-c", cmd).Output()
		if err != nil {
			return errors.Wrapf(err, "mkisofs failed %s", out)
		}

		err = qemu_kvm.InitQemuDeployManager(
			cpuArchStr,
			DeployOption.DefaultQemuVersion,
			DeployOption.EnableRemoteExecutor,
			DeployOption.HugepagesOption == "native",
			DeployOption.HugepageSizeMb*1024,
			DeployOption.DeployGuestMemSizeMb,
			DeployOption.DeployConcurrent,
		)
		if err != nil {
			return err
		}
	}

	// create /dev/lvm_remote
	if err := s.checkLvmRemote(); err != nil {
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

	optionsInit()

	if len(DeployOption.DeployAction) > 0 {
		if err := LocalInitEnv(); err != nil {
			log.Fatalf("local init env %s", err)
		}
		return
	}

	if DeployOption.EnableRemoteExecutor {
		log.Infof("exec socket path: %s", DeployOption.ExecutorSocketPath)
		execlient.Init(DeployOption.ExecutorSocketPath)
		execlient.SetTimeoutSeconds(DeployOption.ExecutorConnectTimeoutSeconds)
		procutils.SetRemoteExecutor()
	}

	app_common.InitAuth(&DeployOption.CommonOptions, func() {
		log.Infof("Auth complete!!")
	})
	common_options.StartOptionManager(&DeployOption.CommonOptions, DeployOption.ConfigSyncPeriodSeconds, "", "", common_options.OnCommonOptionsChange)

	if err := s.PrepareEnv(); err != nil {
		log.Fatalln(err)
	}

	s.O = &DeployOption.BaseOptions
	if len(DeployOption.DeployServerSocketPath) == 0 {
		log.Fatalf("missing deploy server socket path")
	}
}

func (s *SDeployService) OnExitService() {}
