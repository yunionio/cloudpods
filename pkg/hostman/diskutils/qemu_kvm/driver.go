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

package qemu_kvm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

var BASE_SSH_PORT = 28000

var (
	QEMU_KVM_PATH = "/usr/libexec/qemu-kvm"

	OS_ARCH_AARCH64 = "aarch64"
	ARM_INITRD_PATH = "/yunionos/aarch64/initramfs"
	ARM_KERNEL_PATH = "/yunionos/aarch64/kernel"
	X86_INITRD_PATH = "/yunionos/x86_64/initramfs"
	X86_KERNEL_PATH = "/yunionos/x86_64/kernel"

	RUN_ON_HOST_ROOT_PATH = "/opt/cloud/host-deployer"

	DEPLOY_ISO = "/opt/cloud/host-deployer/host_deployer_v1.iso"
	// DEPLOYER_BIN    = "/opt/yunion/bin/host-deployer --common-config-file /opt/yunion/common.conf --config /opt/yunion/host.conf"
	DEPLOYER_BIN       = "/opt/yunion/bin/host-deployer --config /opt/yunion/host.conf"
	DEPLOY_PARAMS_FILE = "/deploy_params"
	YUNIONOS_PASSWD    = "mosbaremetal"
)

type QemuDeployManager struct {
	cpuArch         string
	hugepage        bool
	hugepageSizeKB  int
	portsInUse      *sync.Map
	sshPortLock     *sync.Mutex
	lastUsedSshPort int
	qemuCmd         string
	memSizeMb       int

	c chan struct{}
}

func (m *QemuDeployManager) runOnHost() bool {
	return m.qemuCmd != QEMU_KVM_PATH
}

func (m *QemuDeployManager) startQemu(cmd string) ([]byte, error) {
	if m.runOnHost() {
		return procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	} else {
		return procutils.NewCommand("bash", "-c", cmd).Output()
	}
}

func (m *QemuDeployManager) GetX86InitrdPath() string {
	if m.runOnHost() {
		return path.Join(RUN_ON_HOST_ROOT_PATH, X86_INITRD_PATH)
	} else {
		return X86_INITRD_PATH
	}
}

func (m *QemuDeployManager) GetARMInitrdPath() string {
	if m.runOnHost() {
		return path.Join(RUN_ON_HOST_ROOT_PATH, ARM_INITRD_PATH)
	} else {
		return ARM_INITRD_PATH
	}
}

func (m *QemuDeployManager) GetX86KernelPath() string {
	if m.runOnHost() {
		return path.Join(RUN_ON_HOST_ROOT_PATH, X86_KERNEL_PATH)
	} else {
		return X86_KERNEL_PATH
	}
}

func (m *QemuDeployManager) GetARMKernelPath() string {
	if m.runOnHost() {
		return path.Join(RUN_ON_HOST_ROOT_PATH, ARM_KERNEL_PATH)
	} else {
		return ARM_KERNEL_PATH
	}
}

func (m *QemuDeployManager) Acquire() {
	log.Infof("acquire QemuDeployManager")
	m.c <- struct{}{}
}

func (m *QemuDeployManager) Release() {
	log.Infof("release QemuDeployManager")
	<-m.c
}

func (m *QemuDeployManager) GetFreePortByBase(basePort int) int {
	var port = 1
	for {
		if netutils2.IsTcpPortUsed("0.0.0.0", basePort+port) {
			port += 1
		} else {
			if !m.checkAndSetPort(basePort + port) {
				continue
			}
			break
		}
	}
	return port + basePort
}

func (m *QemuDeployManager) checkAndSetPort(port int) bool {
	_, loaded := m.portsInUse.LoadOrStore(port, struct{}{})
	return !loaded
}

func (m *QemuDeployManager) unsetPort(port int) {
	m.portsInUse.Delete(port)
}

func (m *QemuDeployManager) GetSshFreePort() int {
	m.sshPortLock.Lock()
	defer m.sshPortLock.Unlock()
	port := m.GetFreePortByBase(BASE_SSH_PORT + m.lastUsedSshPort)
	m.lastUsedSshPort = port - BASE_SSH_PORT
	if m.lastUsedSshPort >= 2000 {
		m.lastUsedSshPort = 0
	}
	return port
}

func (m *QemuDeployManager) getMemSizeMb() int {
	return m.memSizeMb
}

var manager *QemuDeployManager

func tryCleanGuest(hmpPath string) {
	var c = make(chan error)
	onMonitorConnected := func() {
		log.Infof("%s monitor connected", hmpPath)
		c <- nil
	}
	onMonitorDisConnect := func(e error) {
		log.Errorf("%s monitor disconnect %s", hmpPath, e)
	}
	onMonitorConnectFailed := func(e error) {
		log.Errorf("%s monitor connect failed %s", hmpPath, e)
		c <- e
	}
	m := monitor.NewHmpMonitor("", "", onMonitorDisConnect, onMonitorConnectFailed, onMonitorConnected)
	if err := m.ConnectWithSocket(hmpPath); err != nil {
		log.Errorf("failed connect socket %s %s", hmpPath, err)
	} else {
		<-c
		m.Quit(func(string) {})
	}
}

func InitQemuDeployManager(
	cpuArch, qemuVersion string,
	enableRemoteExecutor, hugepage bool,
	hugepageSizeKB, memSizeMb, deployConcurrent int,
) error {
	if deployConcurrent <= 0 {
		deployConcurrent = 10
	}

	if cpuArch == OS_ARCH_AARCH64 {
		qemutils.UseAarch64()
	}

	var qemuCmd string
	if enableRemoteExecutor {
		qemuCmd = qemutils.GetQemu(qemuVersion)
	}
	if qemuCmd == "" {
		qemuCmd = QEMU_KVM_PATH
	}

	err := procutils.NewCommand("mkdir", "-p", "/etc/ceph").Run()
	if err != nil {
		log.Errorf("Failed to mkdir /etc/ceph: %s", err)
		return errors.Wrap(err, "Failed to mkdir /etc/ceph: %s")
	}
	err = procutils.NewCommand("test", "-f", "/etc/ceph/ceph.conf").Run()
	if err != nil {
		err = procutils.NewCommand("touch", "/etc/ceph/ceph.conf").Run()
		if err != nil {
			log.Errorf("failed to create /etc/ceph/ceph.conf: %s", err)
			return errors.Wrap(err, "failed to create /etc/ceph/ceph.conf")
		}
	}

	if manager == nil {
		manager = &QemuDeployManager{
			cpuArch:        cpuArch,
			hugepage:       hugepage,
			hugepageSizeKB: hugepageSizeKB,
			memSizeMb:      memSizeMb,
			portsInUse:     new(sync.Map),
			sshPortLock:    new(sync.Mutex),
			c:              make(chan struct{}, deployConcurrent),
			qemuCmd:        qemuCmd,
		}
	}

	if manager.runOnHost() {
		files, err := ioutil.ReadDir(RUN_ON_HOST_ROOT_PATH)
		if err != nil {
			return errors.Wrap(err, "readDir RUN_ON_HOST_ROOT_PATH")
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if strings.HasPrefix(file.Name(), "hmp_") && strings.HasSuffix(file.Name(), ".socket") {
				hmpPath := path.Join(RUN_ON_HOST_ROOT_PATH, file.Name())
				tryCleanGuest(hmpPath)
			}
		}

		err = procutils.NewCommand("cp", "-rf", "/yunionos", RUN_ON_HOST_ROOT_PATH).Run()
		if err != nil {
			log.Errorf("Failed to mkdir /opt/cloud/host-deployer: %s", err)
			return errors.Wrap(err, "Failed to mkdir /opt/cloud/host-deployer: %s")
		}
	}

	return nil
}

type QemuKvmDriver struct {
	imageInfo      qemuimg.SImageInfo
	qemuArchDriver IQemuArchDriver
	sshClient      *ssh.Client

	partitions    []fsdriver.IDiskPartition
	lvmPartitions []fsdriver.IDiskPartition
}

func NewQemuKvmDriver(imageInfo qemuimg.SImageInfo) *QemuKvmDriver {
	return &QemuKvmDriver{
		imageInfo: imageInfo,
	}
}

func (d *QemuKvmDriver) Connect(guestDesc *apis.GuestDesc) error {
	manager.Acquire()
	d.qemuArchDriver = NewCpuArchDriver(manager.cpuArch)
	err := d.connect(guestDesc)
	if err != nil {
		d.qemuArchDriver.CleanGuest()
		return err
	}

	return nil
}

func (d *QemuKvmDriver) connect(guestDesc *apis.GuestDesc) error {
	var (
		ncpu      = 2
		memSizeMB = manager.getMemSizeMb()
		disks     = make([]string, 0)
	)

	var sshport = manager.GetSshFreePort()
	defer manager.unsetPort(sshport)

	if guestDesc != nil && len(guestDesc.Disks) > 0 {
		for i := range guestDesc.Disks {
			disks = append(disks, guestDesc.Disks[i].Path)
		}
	} else {
		disks = append(disks, d.imageInfo.Path)
	}

	err := d.qemuArchDriver.StartGuest(
		sshport, ncpu, memSizeMB,
		manager.hugepage, manager.hugepageSizeKB,
		d.imageInfo, disks,
	)
	if err != nil {
		return err
	}
	log.Infof("guest started ....")

	for i := 0; i < 3; i++ {
		cli, e := ssh.NewClient("localhost", sshport, "root", YUNIONOS_PASSWD, "")
		if e == nil {
			d.sshClient = cli
			break
		}
		err = e
		log.Errorf("new ssh client failed: %s", err)
	}
	if d.sshClient == nil {
		return errors.Wrap(err, "new ssh client")
	}

	log.Infof("guest ssh connected")

	out, err := d.sshRun("mount /dev/sr0 /opt")
	if err != nil {
		return errors.Wrapf(err, "failed mount iso /dev/sr0: %v", out)
	}
	return nil
}

func (d *QemuKvmDriver) Disconnect() error {
	if d.sshClient != nil {
		d.sshClient.Close()
	}

	d.qemuArchDriver.CleanGuest()
	d.qemuArchDriver = nil
	return nil
}

func (d *QemuKvmDriver) GetPartitions() []fsdriver.IDiskPartition {
	return nil
}

func (d *QemuKvmDriver) IsLVMPartition() bool {
	return false
}

func (d *QemuKvmDriver) Zerofree() {}

func (d *QemuKvmDriver) ResizePartition() error {
	return nil
}

func (d *QemuKvmDriver) FormatPartition(fs, uuid string, features *apis.FsFeatures) error {
	return nil
}

func (d *QemuKvmDriver) MakePartition(fs string) error {
	return nil
}

func (d *QemuKvmDriver) DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool {
	return false
}

func (d *QemuKvmDriver) MountRootfs(readonly bool) (fsdriver.IRootFsDriver, error) {
	return nil, nil
}

func (d *QemuKvmDriver) UmountRootfs(fd fsdriver.IRootFsDriver) error {
	return nil
}

func (d *QemuKvmDriver) sshRun(cmd string) ([]string, error) {
	log.Infof("QemuKvmDriver start command %s", cmd)
	return d.sshClient.Run(cmd)
}

func (d *QemuKvmDriver) sshFilePutContent(params interface{}, filePath string) error {
	jcontent := jsonutils.Marshal(params).String()
	jcontent = strings.ReplaceAll(jcontent, "`", "\\`")
	cmd := fmt.Sprintf(`cat << EOF > %s
%s
EOF`, filePath, jcontent)
	out, err := d.sshRun(cmd)
	if err != nil {
		return errors.Wrapf(err, "sshFilePutContent %s", out)
	}
	return nil
}

func (d *QemuKvmDriver) DeployGuestfs(req *apis.DeployParams) (*apis.DeployGuestFsResponse, error) {
	defer func() {
		logStr, _ := d.sshRun("test -f /log && cat /log")
		log.Infof("DeployGuestfs log: %v", strings.Join(logStr, "\n"))
	}()

	err := d.sshFilePutContent(req, DEPLOY_PARAMS_FILE)
	if err != nil {
		return nil, errors.Wrap(err, "DeployGuestfs ssh copy deploy params")
	}

	cmd := fmt.Sprintf("%s --deploy-action deploy_guest_fs --deploy-params-file %s", DEPLOYER_BIN, DEPLOY_PARAMS_FILE)
	out, err := d.sshRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "run deploy_guest_fs failed %s", out)
	}
	log.Infof("DeployGuestfs log: %s", strings.Join(out, "\n"))

	errStrs, err := d.sshRun("test -f /error && cat /error || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	log.Infof("deploy error str %v", errStrs)
	var retErr error = nil
	if len(errStrs[0]) > 0 {
		retErr = errors.Errorf(errStrs[0])
	}

	responseStrs, err := d.sshRun("test -f /response && cat /response || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	log.Infof("deploy response str %v", responseStrs)
	var res = new(apis.DeployGuestFsResponse)
	if len(responseStrs[0]) > 0 {
		err := json.Unmarshal([]byte(responseStrs[0]), res)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshal deploy response %s", responseStrs[0])
		}
	}

	return res, retErr
}

func (d *QemuKvmDriver) ResizeFs() (*apis.Empty, error) {
	defer func() {
		logStr, _ := d.sshRun("test -f /log && cat /log")
		log.Infof("ResizeFs log: %v", strings.Join(logStr, "\n"))
	}()

	cmd := fmt.Sprintf("%s --deploy-action resize_fs", DEPLOYER_BIN)
	out, err := d.sshRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "run resize_fs failed %s", out)
	}
	log.Infof("ResizeFs log: %s", strings.Join(out, "\n"))

	errStrs, err := d.sshRun("test -f /error && cat /error || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	var retErr error = nil
	if len(errStrs[0]) > 0 {
		retErr = errors.Errorf(errStrs[0])
	}
	return new(apis.Empty), retErr
}

func (d *QemuKvmDriver) FormatFs(req *apis.FormatFsParams) (*apis.Empty, error) {
	defer func() {
		logStr, _ := d.sshRun("test -f /log && cat /log")
		log.Infof("FormatFs log: %v", strings.Join(logStr, "\n"))
	}()

	params, _ := json.Marshal(req)
	cmd := fmt.Sprintf("%s --deploy-action format_fs --deploy-params '%s'", DEPLOYER_BIN, params)
	out, err := d.sshRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "run format_fs failed %s", out)
	}
	log.Infof("FormatFs log: %s", strings.Join(out, "\n"))

	errStrs, err := d.sshRun("test -f /error && cat /error || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	var retErr error = nil
	if len(errStrs[0]) > 0 {
		retErr = errors.Errorf(errStrs[0])
	}
	return new(apis.Empty), retErr
}

func (d *QemuKvmDriver) SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	defer func() {
		logStr, _ := d.sshRun("test -f /log && cat /log")
		log.Infof("SaveToGlance log: %s", strings.Join(logStr, "\n"))
	}()

	params, _ := json.Marshal(req)
	cmd := fmt.Sprintf("%s --deploy-action save_to_glance --deploy-params '%s'", DEPLOYER_BIN, params)
	out, err := d.sshRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "run save_to_glance failed %s", out)
	}
	log.Infof("SaveToGlance log: %s", strings.Join(out, "\n"))

	responseStrs, err := d.sshRun("test -f /response && cat /response || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	var res = new(apis.SaveToGlanceResponse)
	if len(responseStrs[0]) > 0 {
		err := json.Unmarshal([]byte(responseStrs[0]), res)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshal deploy response %s", responseStrs[0])
		}
	}

	errStrs, err := d.sshRun("test -f /error && cat /error || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	var retErr error = nil
	if len(errStrs[0]) > 0 {
		retErr = errors.Errorf(errStrs[0])
	}
	return res, retErr
}

func (d *QemuKvmDriver) ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error) {
	defer func() {
		logStr, _ := d.sshRun("test -f /log && cat /log")
		log.Infof("ProbeImageInfo log: %v", strings.Join(logStr, "\n"))
	}()

	params, _ := json.Marshal(req)
	cmd := fmt.Sprintf("%s --deploy-action probe_image_info --deploy-params '%s'", DEPLOYER_BIN, params)
	out, err := d.sshRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "run probe_image_info failed %s", out)
	}
	log.Infof("ProbeImageInfo log: %s", strings.Join(out, "\n"))

	responseStrs, err := d.sshRun("test -f /response && cat /response || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	var res = new(apis.ImageInfo)
	if len(responseStrs[0]) > 0 {
		err := json.Unmarshal([]byte(responseStrs[0]), res)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshal deploy response %s", responseStrs[0])
		}
	}

	errStrs, err := d.sshRun("test -f /error && cat /error || true")
	if err != nil {
		return nil, errors.Wrapf(err, "ssh gather errors failed")
	}
	var retErr error = nil
	if len(errStrs[0]) > 0 {
		retErr = errors.Errorf(errStrs[0])
	}
	return res, retErr
}

// wrap strings
func __(v string, vs ...interface{}) string {
	return fmt.Sprintf(" "+v, vs...)
}

type QemuBaseDriver struct {
	hmp          *monitor.HmpMonitor
	hugepagePath string
	pidPath      string
	cleaned      uint32
}

func (d *QemuBaseDriver) CleanGuest() {
	if d.hmp != nil {
		d.hmp.Quit(func(string) {})
		d.hmp = nil
	}

	if d.pidPath != "" && fileutils2.Exists(d.pidPath) {
		pid, _ := fileutils2.FileGetContents(d.pidPath)
		if len(pid) > 0 {
			out, err := procutils.NewCommand("kill", pid).Output()
			log.Infof("kill  process %s %v", out, err)
		}
	}

	if d.hugepagePath != "" {
		err, out := procutils.NewCommand("umount", d.hugepagePath).Output()
		if err != nil {
			log.Errorf("failed umount %s %s : %s", d.hugepagePath, err, out)
		} else {
			procutils.NewCommand("rm", "-rf", d.hugepagePath).Run()
		}
		d.hugepagePath = ""
	}

	if atomic.LoadUint32(&d.cleaned) != 1 {
		manager.Release()
		atomic.StoreUint32(&d.cleaned, 1)
	}
}

func (d *QemuBaseDriver) startCmds(
	sshPort, ncpu, memSizeMB int, imageInfo qemuimg.SImageInfo, disksPath []string,
	machineOpts, cdromDeviceOpts, fwOpts, socketPath, initrdPath, kernelPath string,
) string {
	cmd := manager.qemuCmd

	if sysutils.IsKvmSupport() {
		cmd += __("-enable-kvm")
		cmd += __("-cpu host")
	} else {
		cmd += __("-cpu max")
	}
	cmd += machineOpts

	cmd += __("-nodefaults")
	cmd += __("-daemonize")
	cmd += __("-monitor unix:%s,server,nowait", socketPath)
	cmd += __("-pidfile %s", d.pidPath)
	cmd += __("-vnc none")
	cmd += __("-smp %d", ncpu)
	cmd += __("-m %dM", memSizeMB)
	cmd += __("-initrd %s", initrdPath)
	cmd += __("-kernel %s", kernelPath)
	cmd += __("-append rootfstype=ramfs")
	cmd += __("-append vsyscall=emulate")
	cmd += fwOpts
	cmd += __("-device virtio-serial-pci")
	cmd += __("-netdev user,id=hostnet0,hostfwd=tcp:127.0.0.1:%d-:22", sshPort)
	cmd += __("-device virtio-net-pci,netdev=hostnet0")
	cmd += __("-device virtio-scsi-pci,id=scsi")

	if imageInfo.Encrypted() {
		imageInfo.SetSecId("sec0")

		cmd += __("-object %s", imageInfo.SecretOptions())
	}
	if len(disksPath) == 0 {
		disksPath = []string{imageInfo.Path}
	}
	for i, diskPath := range disksPath {
		diskDrive := __("-drive file=%s,if=none,id=drive_%d,cache=none", diskPath, i)
		if imageInfo.Format != qemuimgfmt.RAW && imageInfo.Encrypted() {
			diskDrive += ",encrypt.format=luks,encrypt.key-secret=sec0"
		}

		cmd += diskDrive
		cmd += __("-device scsi-hd,drive=drive_%d,bus=scsi.0,id=drive_%d", i, i)
	}
	cmd += __("-drive id=cd0,if=none,media=cdrom,file=%s", DEPLOY_ISO)
	cmd += cdromDeviceOpts

	return cmd
}

type QemuX86Driver struct {
	QemuBaseDriver
}

func (d *QemuX86Driver) StartGuest(sshPort, ncpu, memSizeMB int, hugePage bool, pageSizeKB int, imageInfo qemuimg.SImageInfo, disksPath []string) error {
	uuid := stringutils.UUID4()
	socketPath := fmt.Sprintf("/opt/cloud/host-deployer/hmp_%s.socket", uuid)
	d.pidPath = fmt.Sprintf("/opt/cloud/host-deployer/%s.pid", uuid)

	machineOpts := __("-M pc")
	cdromDeviceOpts := __("-device ide-cd,drive=cd0,bus=ide.1")
	cmd := d.startCmds(
		sshPort,
		ncpu,
		memSizeMB,
		imageInfo,
		disksPath,
		machineOpts,
		cdromDeviceOpts,
		"",
		socketPath,
		manager.GetX86InitrdPath(),
		manager.GetX86KernelPath(),
	)

	log.Infof("start guest %s", cmd)
	out, err := manager.startQemu(cmd)
	if err != nil {
		log.Errorf("failed start guest %s: %s", out, err)
		return errors.Wrapf(err, "failed start guest %s", out)
	}

	var c = make(chan error)
	onMonitorConnected := func() {
		log.Infof("monitor connected")
		c <- nil
	}
	onMonitorDisConnect := func(e error) {
		log.Errorf("monitor disconnect %s", e)
	}
	onMonitorConnectFailed := func(e error) {
		log.Errorf("monitor connect failed %s", e)
		c <- e
	}
	m := monitor.NewHmpMonitor("", "", onMonitorDisConnect, onMonitorConnectFailed, onMonitorConnected)
	if err = m.ConnectWithSocket(socketPath); err != nil {
		return errors.Wrapf(err, "connect socket %s failed", socketPath)
	}
	if err = <-c; err != nil {
		return errors.Wrap(err, "monitor connect failed")
	}
	d.hmp = m
	return nil
}

type QemuARMDriver struct {
	QemuBaseDriver
}

func (d *QemuARMDriver) StartGuest(sshPort, ncpu, memSizeMB int, hugePage bool, pageSizeKB int, imageInfo qemuimg.SImageInfo, disksPath []string) error {
	uuid := stringutils.UUID4()
	socketPath := fmt.Sprintf("/opt/cloud/host-deployer/hmp_%s.socket", uuid)
	d.pidPath = fmt.Sprintf("/opt/cloud/host-deployer/%s.pid", uuid)

	machineOpts := __("-M virt,gic-version=max")
	cdromDeviceOpts := __("-device scsi-cd,drive=cd0,share-rw=true")
	fwOpts := ""
	if manager.runOnHost() {
		fwOpts = __("-drive if=pflash,format=raw,unit=0,file=/opt/cloud/contrib/OVMF.fd,readonly=on")
	} else {
		fwOpts = __("-drive if=pflash,format=raw,unit=0,file=/usr/share/AAVMF/AAVMF_CODE.fd,readonly=on")
	}

	cmd := d.startCmds(
		sshPort,
		ncpu,
		memSizeMB,
		imageInfo,
		disksPath,
		machineOpts,
		cdromDeviceOpts,
		fwOpts,
		socketPath,
		manager.GetARMInitrdPath(),
		manager.GetARMKernelPath(),
	)

	log.Infof("start guest %s", cmd)
	out, err := manager.startQemu(cmd)
	if err != nil {
		log.Errorf("failed start guest %s: %s", out, err)
		return errors.Wrapf(err, "failed start guest %s", out)
	}

	var c = make(chan error)
	onMonitorConnected := func() {
		log.Infof("monitor connected")
		c <- nil
	}
	onMonitorDisConnect := func(e error) {
		log.Errorf("monitor disconnect %s", e)
	}
	onMonitorConnectFailed := func(e error) {
		log.Errorf("monitor connect failed %s", e)
		c <- e
	}
	m := monitor.NewHmpMonitor("", "", onMonitorDisConnect, onMonitorConnectFailed, onMonitorConnected)
	if err = m.ConnectWithSocket(socketPath); err != nil {
		return errors.Wrapf(err, "connect socket %s failed", socketPath)
	}
	if err = <-c; err != nil {
		return errors.Wrap(err, "monitor connect failed")
	}
	d.hmp = m
	return nil
}

type IQemuArchDriver interface {
	StartGuest(sshPort, ncpu, memSizeMB int, hugePage bool, pageSizeKB int, imageInfo qemuimg.SImageInfo, disksPath []string) error
	CleanGuest()
}

func NewCpuArchDriver(cpuArch string) IQemuArchDriver {
	if cpuArch == OS_ARCH_AARCH64 {
		return &QemuARMDriver{}
	}

	return &QemuX86Driver{}
}
