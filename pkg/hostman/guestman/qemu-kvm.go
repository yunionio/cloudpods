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

package guestman

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	noapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/hostman/guestman/arch"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/monitor/qga"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fuseutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/regutils2"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

const (
	STATE_FILE_PREFIX             = "STATEFILE"
	MONITOR_PORT_BASE             = 55900
	QMP_MONITOR_PORT_BASE         = 56100
	LIVE_MIGRATE_PORT_BASE        = 4396
	BUILT_IN_NBD_SERVER_PORT_BASE = 7777
	MAX_TRY                       = 3
)

type SKVMInstanceRuntime struct {
	QemuVersion string
	VncPassword string

	LiveMigrateDestPort *int64
	LiveMigrateUseTls   bool

	syncMeta *jsonutils.JSONDict

	cgroupPid  int
	cgroupName string

	stopping            bool
	needSyncStreamDisks bool
	blockJobTigger      map[string]chan struct{}
	quorumFailed        int32

	StartupTask *SGuestResumeTask
	MigrateTask *SGuestLiveMigrateTask

	pciUninitialized bool
	pciAddrs         *desc.SGuestPCIAddresses
}

type SKVMGuestInstance struct {
	SKVMInstanceRuntime
	*sBaseGuestInstance

	Monitor    monitor.Monitor
	guestAgent *qga.QemuGuestAgent

	archMan arch.Arch
}

func NewKVMGuestInstance(id string, manager *SGuestManager) *SKVMGuestInstance {
	qemuArch := arch.Arch_x86_64
	if manager.host.IsAarch64() {
		qemuArch = arch.Arch_aarch64
	}
	return &SKVMGuestInstance{
		SKVMInstanceRuntime: SKVMInstanceRuntime{
			blockJobTigger: make(map[string]chan struct{}),
		},
		sBaseGuestInstance: newBaseGuestInstance(id, manager, api.HYPERVISOR_KVM),
		archMan:            arch.NewArch(qemuArch),
	}
}

// update guest runtime desc from source desc
// and check is need regenerate runtime desc
// these property can't be upate in running guest
func (s *SKVMGuestInstance) updateGuestDesc() error {
	s.Desc = new(desc.SGuestDesc)
	err := jsonutils.Marshal(s.SourceDesc).Unmarshal(s.Desc)
	if err != nil {
		return errors.Wrap(err, "unmarshal source desc")
	}

	if s.isPcie() {
		s.setPcieExtendBus()
	}

	err = s.initGuestDesc()
	if err != nil {
		return err
	}

	return SaveLiveDesc(s, s.Desc)
}

func (s *SKVMGuestInstance) initLiveDescFromSourceGuest(srcDesc *desc.SGuestDesc) error {
	srcDesc.SGuestProjectDesc = s.SourceDesc.SGuestProjectDesc
	srcDesc.SGuestRegionDesc = s.SourceDesc.SGuestRegionDesc
	srcDesc.SGuestControlDesc = s.SourceDesc.SGuestControlDesc
	srcDesc.SGuestMetaDesc = s.SourceDesc.SGuestMetaDesc
	for i := 0; i < len(srcDesc.Cdroms); i++ {
		if i == len(s.SourceDesc.Cdroms) {
			break
		}
		srcDesc.Cdroms[i].Path = s.SourceDesc.Cdroms[i].Path
	}
	for i := 0; i < len(srcDesc.Disks); i++ {
		for j := 0; j < len(s.SourceDesc.Disks); j++ {
			if srcDesc.Disks[i].Index == s.SourceDesc.Disks[j].Index {
				numQueues := srcDesc.Disks[i].NumQueues
				srcDesc.Disks[i].GuestdiskJsonDesc = s.SourceDesc.Disks[j].GuestdiskJsonDesc
				srcDesc.Disks[i].NumQueues = numQueues
			}
		}
	}
	for i := 0; i < len(srcDesc.Nics); i++ {
		var numQueues, vectors int
		for j := 0; j < len(s.SourceDesc.Nics); j++ {
			if srcDesc.Nics[i].Index == s.SourceDesc.Nics[j].Index {
				numQueues = srcDesc.Nics[i].NumQueues
				if srcDesc.Nics[i].Vectors != nil {
					vectors = *srcDesc.Nics[i].Vectors
				}

				srcDesc.Nics[i].GuestnetworkJsonDesc = s.SourceDesc.Nics[j].GuestnetworkJsonDesc
				break
			}
		}

		if err := s.generateNicScripts(srcDesc.Nics[i]); err != nil {
			return errors.Wrapf(err, "generateNicScripts for nic: %v", srcDesc.Nics[i])
		}
		srcDesc.Nics[i].NumQueues = numQueues
		if vectors > 0 {
			srcDesc.Nics[i].Vectors = &vectors
		}
		srcDesc.Nics[i].UpscriptPath = s.getNicUpScriptPath(srcDesc.Nics[i])
		srcDesc.Nics[i].DownscriptPath = s.getNicDownScriptPath(srcDesc.Nics[i])
	}

	s.Desc = srcDesc
	err := s.loadGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "initLiveDescFromSourceGuest")
	}
	return SaveLiveDesc(s, srcDesc)
}

func (s *SKVMGuestInstance) IsStopping() bool {
	return s.stopping
}

func (s *SKVMGuestInstance) IsValid() bool {
	return s.Desc != nil && s.Desc.Uuid != ""
}

func (s *SKVMGuestInstance) getStateFilePathRootPrefix() string {
	return path.Join(s.HomeDir(), STATE_FILE_PREFIX)
}

func (s *SKVMGuestInstance) GetStateFilePath(version string) string {
	p := s.getStateFilePathRootPrefix()
	if version != "" {
		p = fmt.Sprintf("%s_%s", p, version)
	}
	return p
}

func (s *SKVMGuestInstance) getQemuLogPath() string {
	return path.Join(s.HomeDir(), "qemu.log")
}

func (s *SKVMGuestInstance) GetPidFilePath() string {
	return path.Join(s.HomeDir(), "pid")
}

func (s *SKVMGuestInstance) GetVncFilePath() string {
	return path.Join(s.HomeDir(), "vnc")
}

func (s *SKVMGuestInstance) getEncryptKeyPath() string {
	return path.Join(s.HomeDir(), "key")
}

func (s *SKVMGuestInstance) getEncryptKeyId() string {
	return s.Desc.EncryptKeyId
}

func (s *SKVMGuestInstance) isEncrypted() bool {
	return len(s.getEncryptKeyId()) > 0
}

func (s *SKVMGuestInstance) getEncryptKey(ctx context.Context, userCred mcclient.TokenCredential) (apis.SEncryptInfo, error) {
	ret := apis.SEncryptInfo{}
	encKeyId := s.getEncryptKeyId()
	if len(encKeyId) > 0 {
		if userCred == nil {
			return ret, errors.Wrap(httperrors.ErrUnauthorized, "no credential to fetch encrypt key")
		}
		session := auth.GetSession(ctx, userCred, consts.GetRegion())
		secKey, err := identity_modules.Credentials.GetEncryptKey(session, encKeyId)
		if err != nil {
			return ret, errors.Wrap(err, "GetEncryptKey")
		}
		ret.Id = secKey.KeyId
		ret.Name = secKey.KeyName
		ret.Key = secKey.Key
		ret.Alg = secKey.Alg
		return ret, nil
	}
	return ret, nil
}

func (s *SKVMGuestInstance) saveEncryptKeyFile(key string) error {
	return fileutils2.FilePutContents(s.getEncryptKeyPath(), key, false)
}

func (s *SKVMGuestInstance) getOriginId() string {
	if s.Desc == nil {
		return ""
	}
	return s.Desc.Metadata["__origin_id"]
}

func (s *SKVMGuestInstance) isImportFromLibvirt() bool {
	return s.getOriginId() != ""
}

func (s *SKVMGuestInstance) GetPid() int {
	return s.getPid(s.GetPidFilePath(), s.getOriginId())
}

/*
pid -> running qemu's pid
-1 -> pid file does not exists
-2 -> pid file ok but content does not match any qemu process
*/
func (s *SKVMGuestInstance) getPid(pidFile, uuid string) int {
	if !fileutils2.Exists(pidFile) {
		return -1
	}
	pidStr, err := fileutils2.FileGetContents(pidFile)
	if err != nil {
		log.Errorf("Get pid file %s error %s: %s", pidFile, s.GetName(), err)
		return -2
	}
	pidStr = strings.TrimSpace(pidStr)
	pid := s.findPid(strings.Split(pidStr, "\n"), uuid)
	if len(pid) > 0 && regutils.MatchInteger(pid) {
		v, _ := strconv.ParseInt(pid, 10, 0)
		return int(v)
	}
	return -2
}

func (s *SKVMGuestInstance) findPid(pids []string, uuid string) string {
	if len(pids) == 0 {
		return ""
	}
	for _, pid := range pids {
		pid := strings.TrimSpace(pid)
		if s.isSelfQemuPid(pid, uuid) {
			return pid
		}
	}
	return ""
}

func (s *SKVMGuestInstance) isSelfQemuPid(pid, uuid string) bool {
	if len(pid) == 0 {
		return false
	}
	cmdlineFile := fmt.Sprintf("/proc/%s/cmdline", pid)
	fi, err := os.Stat(cmdlineFile)
	if err != nil {
		return false
	}
	if !fi.Mode().IsRegular() {
		return false
	}
	cmdline, err := ioutil.ReadFile(cmdlineFile)
	if err != nil {
		log.Warningf("IsSelfQemuPid Read File %s error %s", cmdlineFile, err)
		return false
	}
	return s.isSelfCmdline(string(cmdline), uuid)
}

func (s *SKVMGuestInstance) isSelfCmdline(cmdline, uuid string) bool {
	return (strings.Index(cmdline, "qemu-system") >= 0 ||
		strings.Index(cmdline, "qemu-kvm") >= 0) &&
		strings.Index(cmdline, uuid) >= 0
}

func (s *SKVMGuestInstance) GetRescueDirPath() string {
	return path.Join(s.HomeDir(), api.GUEST_RESCUE_RELATIVE_PATH)
}

func (s *SKVMGuestInstance) CreateRescueDirPath() (string, error) {
	rescueDir := path.Join(s.HomeDir(), api.GUEST_RESCUE_RELATIVE_PATH)

	// Check if rescue dir exists
	output, err := procutils.NewCommand("mkdir", "-p", rescueDir).Output()
	if err != nil {
		return "", errors.Wrapf(err, "mkdir %s failed: %s", s.HomeDir(), output)
	}

	return rescueDir, nil
}

func (s *SKVMGuestInstance) LoadDesc() error {
	if err := LoadDesc(s); err != nil {
		return errors.Wrap(err, "LoadDesc")
	}

	if s.IsRunning() {
		if len(s.Desc.PCIControllers) > 0 {
			if err := s.loadGuestPciAddresses(); err != nil {
				log.Errorf("failed load guest %s pci addresses %s", s.GetName(), err)
				if len(s.Desc.AnonymousPCIDevs) > 0 {
					s.Desc = s.SourceDesc
					s.pciUninitialized = true
				} else {
					return err
				}
			}
		} else {
			s.pciUninitialized = true
		}
	}

	return nil
}

func (s *SKVMGuestInstance) PostLoad(m *SGuestManager) error {
	if s.needSyncStreamDisks {
		go s.sendStreamDisksComplete(context.Background())
	}
	return s.loadGuestCpuset(m)
}

func (s *SKVMGuestInstance) loadGuestCpuset(m *SGuestManager) error {
	if s.GetPid() > 0 {
		for _, vcpuPin := range s.GetDesc().VcpuPin {
			pcpuSet, err := cpuset.Parse(vcpuPin.Pcpus)
			if err != nil {
				log.Errorf("failed parse %s pcpus: %s", s.GetName(), vcpuPin.Pcpus)
				continue
			}
			vcpuSet, err := cpuset.Parse(vcpuPin.Vcpus)
			if err != nil {
				log.Errorf("failed parse %s vcpus: %s", s.GetName(), vcpuPin.Vcpus)
				continue
			}
			m.cpuSet.LoadCpus(pcpuSet.ToSlice(), vcpuSet.Size())
		}
	}
	return nil
}

func (s *SKVMGuestInstance) IsDirtyShotdown() bool {
	return s.GetPid() == -2
}

func (s *SKVMGuestInstance) IsDaemon() bool {
	return s.Desc.IsDaemon
}

func (s *SKVMGuestInstance) DirtyServerRequestStart() {
	var body = jsonutils.NewDict()
	body.Set("guest_id", jsonutils.NewString(s.Id))
	body.Set("host_id", jsonutils.NewString(s.Desc.HostId))
	_, err := modules.Servers.PerformClassAction(
		hostutils.GetComputeSession(context.Background()), "dirty-server-start", body)
	if err != nil {
		log.Errorf("Dirty server request start error %s: %s", s.GetName(), err)
	}
}

func (s *SKVMGuestInstance) fuseMount(encryptInfo *apis.SEncryptInfo) error {
	disks := s.Desc.Disks
	for i := 0; i < len(disks); i++ {
		if disks[i].MergeSnapshot && len(disks[i].Url) > 0 {
			disk, err := storageman.GetManager().GetDiskByPath(disks[i].Path)
			if err != nil {
				return errors.Wrapf(err, "GetDiskByPath(%s)", disks[i].Path)
			}
			storage := disk.GetStorage()
			mntPath := path.Join(storage.GetFuseMountPath(), disk.GetId())
			if err := procutils.NewCommand("mountpoint", mntPath).Run(); err == nil {
				// fetcherfs is mounted
				continue
			}
			tmpdir := storage.GetFuseTmpPath()
			err = fuseutils.MountFusefs(
				options.HostOptions.FetcherfsPath, disks[i].Url, tmpdir,
				auth.GetTokenString(), mntPath,
				options.HostOptions.FetcherfsBlockSize,
				encryptInfo,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SKVMGuestInstance) asyncScriptStart(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	var err error
	var encryptInfo *apis.SEncryptInfo
	if data.Contains("encrypt_info") {
		encryptInfo = new(apis.SEncryptInfo)
		data.Unmarshal(encryptInfo, "encrypt_info")
	}
	err = s.fuseMount(encryptInfo)
	if err != nil {
		return nil, errors.Wrap(err, "fuse mount")
	}

	if jsonutils.QueryBoolean(data, "need_migrate", false) {
		var sourceDesc = new(desc.SGuestDesc)
		err = data.Unmarshal(sourceDesc, "src_desc")
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal src desc")
		}
		err = s.initLiveDescFromSourceGuest(sourceDesc)
	} else {
		err = s.updateGuestDesc()
	}

	// init live migrate listen port
	if jsonutils.QueryBoolean(data, "need_migrate", false) || s.Desc.IsSlave {
		log.Infof("backup guest alloc dest port %v", s.LiveMigrateDestPort)
		migratePort := s.manager.GetLiveMigrateFreePort()
		defer s.manager.unsetPort(migratePort)
		migratePortInt64 := int64(migratePort)
		s.LiveMigrateDestPort = &migratePortInt64
	}

	if err != nil {
		if ctx != nil && len(appctx.AppContextTaskId(ctx)) >= 0 {
			hostutils.TaskFailed(ctx, fmt.Sprintf("Async start server failed: %s", err))
		}
		return nil, errors.Wrap(err, "asyncScriptStart init desc")
	}

	hostbridge.CleanDeletedPorts(options.HostOptions.BridgeDriver)
	time.Sleep(100 * time.Millisecond)

	var isStarted, tried = false, 0
	for !isStarted && tried < MAX_TRY {
		tried += 1

		vncPort := s.manager.GetFreeVncPort()
		defer s.manager.unsetPort(vncPort)
		log.Infof("Use vnc port %d", vncPort)
		if err = s.saveVncPort(vncPort); err != nil {
			goto finally
		} else {
			data.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
		}

		err = s.saveScripts(data)
		if err != nil {
			goto finally
		} else {
			err = s.scriptStart(ctx)
			if err == nil {
				isStarted = true
			} else {
				// call script stop if guest start failed
				s.scriptStop()
			}
		}

	finally:
		if !isStarted {
			log.Errorf("Start VM failed %s: %s", s.GetName(), err)
			time.Sleep(time.Duration(1<<uint(tried-1)) * time.Second)
		} else {
			log.Infof("VM started %s ...", s.GetName())
		}
	}

	// is on_async_script_start
	if isStarted {
		log.Infof("Async start server %s success!", s.GetName())
		s.syncMeta = s.CleanImportMetadata()
		return nil, nil
	}
	log.Errorf("Async start server %s failed: %s!!!", s.GetName(), err)
	if ctx != nil && len(appctx.AppContextTaskId(ctx)) >= 0 {
		hostutils.TaskFailed(ctx, fmt.Sprintf("Async start server failed: %s", err))
	}
	needMigrate := jsonutils.QueryBoolean(data, "need_migrate", false)
	// do not syncstatus if need_migrate
	if !needMigrate {
		s.SyncStatus("")
	}
	return nil, err
}

func (s *SKVMGuestInstance) saveScripts(data *jsonutils.JSONDict) error {
	startScript, err := s.generateStartScript(data)
	if err != nil {
		return err
	}

	if err = fileutils2.FilePutContents(s.GetStartScriptPath(), startScript, false); err != nil {
		return err
	}

	if jsonutils.QueryBoolean(data, "sync_qemu_cmdline", false) {
		cmdline, err := s.getQemuCmdlineFromContent(startScript)
		if err != nil {
			log.Errorf("failed parse cmdline from start script: %s", err)
		} else {
			s.SyncQemuCmdline(cmdline)
		}
	}

	launcher := fmt.Sprintf(guestLauncher, s.GetStartScriptPath(), s.GetStateFilePath(""), s.LogFilePath())
	if err := fileutils2.FilePutContents(s.pyLauncherPath(), launcher, false); err != nil {
		return errors.Wrap(err, "generate guest launcher")
	}

	stopScript := s.generateStopScript(data)
	return fileutils2.FilePutContents(s.GetStopScriptPath(), stopScript, false)
}

func (s *SKVMGuestInstance) GetStartScriptPath() string {
	return path.Join(s.HomeDir(), "startvm")
}

func (s *SKVMGuestInstance) GetStopScriptPath() string {
	return path.Join(s.HomeDir(), "stopvm")
}

func (s *SKVMGuestInstance) ImportServer(pendingDelete bool) {
	// verify host_id consistency
	if s.Desc.HostId != hostinfo.Instance().HostId {
		// fix host_id
		s.Desc.HostId = hostinfo.Instance().HostId
		SaveLiveDesc(s, s.Desc)
	}

	s.manager.SaveServer(s.Id, s)
	s.manager.RemoveCandidateServer(s)

	if s.IsDirtyShotdown() && !pendingDelete {
		log.Infof("Server dirty shutdown or a daemon %s", s.GetName())

		if s.Desc.IsMaster || s.Desc.IsSlave ||
			len(s.GetNeedMergeBackingFileDiskIndexs()) > 0 {
			go s.DirtyServerRequestStart()
		} else {
			s.StartGuest(context.Background(), nil, jsonutils.NewDict())
		}
		return
	}
	if s.IsRunning() {
		log.Infof("%s is running, pending_delete=%t", s.GetName(), pendingDelete)
		if !pendingDelete {
			s.StartMonitor(context.Background(), nil)
		}
	} else if s.IsDaemon() {
		s.StartGuest(context.Background(), nil, jsonutils.NewDict())
	} else {
		var action = "stopped"
		if s.IsSuspend() {
			action = "suspend"
		}
		log.Infof("%s is %s, pending_delete=%t", s.GetName(), action, pendingDelete)
		s.SyncStatus("")
	}
}

func (s *SKVMGuestInstance) IsRunning() bool {
	return s.GetPid() > 0
}

func (s *SKVMGuestInstance) IsStopped() bool {
	return !s.IsRunning()
}

func (s *SKVMGuestInstance) IsSuspend() bool {
	if !s.IsRunning() && len(s.ListStateFilePaths()) > 0 {
		return true
	}
	return false
}

func (s *SKVMGuestInstance) IsMonitorAlive() bool {
	return s.Monitor != nil && s.Monitor.IsConnected()
}

// func (s *SKVMGuestInstance) ListStateFilePaths() []string {
// 	files, err := ioutil.ReadDir(s.HomeDir())
// 	if err == nil {
// 		var ret = make([]string, 0)
// 		for i := 0; i < len(files); i++ {
// 			if strings.HasPrefix(files[i].Name(), STATE_FILE_PREFIX) {
// 				ret = append(ret, files[i].Name())
// 			}
// 		}
// 		return ret
// 	}
// 	return nil
// }

func (s *SKVMGuestInstance) onImportGuestMonitorDisConnect(err error) {
	log.Infof("Import Guest %s monitor disconnect reason: %v", s.Id, err)
	s.SyncStatus(fmt.Sprintf("import guest monitor disconnect %v", err))

	// clean import pid file
	if s.GetPid() == -2 {
		spath := s.GetPidFilePath()
		if fileutils2.Exists(spath) {
			os.Remove(spath)
		}
	}
}

func (s *SKVMGuestInstance) onImportGuestMonitorTimeout(ctx context.Context, err error) {
	log.Errorf("Import guest %s monitor connect timeout: %s", s.Id, err)
	// clean import pid file
	if s.GetPid() == -2 {
		spath := s.GetPidFilePath()
		if fileutils2.Exists(spath) {
			os.Remove(spath)
		}
	}
}

func (s *SKVMGuestInstance) onImportGuestMonitorConnected(ctx context.Context) {
	log.Infof("Guest %s Monitor connect success", s.Id)
	s.Monitor.GetVersion(func(version string) {
		log.Infof("Guest %s qemu version %s", s.Id, version)
		s.QemuVersion = version
		meta := jsonutils.NewDict()
		meta.Set(api.VM_METADATA_HOTPLUG_CPU_MEM, jsonutils.NewString("disable"))
		meta.Set(api.VM_METADATA_HOT_REMOVE_NIC, jsonutils.NewString("disable"))
		meta.Set("__qemu_version", jsonutils.NewString(s.GetQemuVersionStr()))
		s.SyncMetadata(meta)
		s.SyncStatus("")
	})
}

func (s *SKVMGuestInstance) GetMonitorPath() string {
	return s.Desc.Metadata["__monitor_path"]
}

func (s *SKVMGuestInstance) StartMonitorWithImportGuestSocketFile(ctx context.Context, socketFile string, cb func()) error {
	var mon monitor.Monitor
	mon = monitor.NewQmpMonitor(
		s.GetName(),
		s.Id,
		s.onImportGuestMonitorDisConnect, // on monitor disconnect
		func(err error) { s.onImportGuestMonitorTimeout(ctx, err) }, // on monitor timeout
		func() {
			s.Monitor = mon
			s.onImportGuestMonitorConnected(ctx)
			if cb != nil {
				cb()
			}
		}, // on monitor connected
		s.onReceiveQMPEvent, // on reveive qmp event
	)
	return mon.ConnectWithSocket(socketFile, 0)
}

func (s *SKVMGuestInstance) StartMonitor(ctx context.Context, cb func()) error {
	if s.GetQmpMonitorPort(-1) > 0 {
		var mon monitor.Monitor
		var onMonitorTimeout = func(err error) { s.onMonitorTimeout(ctx, err) }
		var onMonitorConnected = func() {
			s.Monitor = mon
			s.onMonitorConnected(ctx)
			if cb != nil {
				cb()
			}
		}
		mon = monitor.NewQmpMonitor(
			s.GetName(), s.Id,
			s.onMonitorDisConnect, // on monitor disconnect
			onMonitorTimeout,      // on monitor timeout
			onMonitorConnected,    // on monitor connected
			s.onReceiveQMPEvent,   // on receive qmp event
		)
		err := mon.Connect("127.0.0.1", s.GetQmpMonitorPort(-1))
		if err != nil {
			mon = nil
			log.Errorf("Guest %s qmp monitor connect failed %s, something wrong", s.GetName(), err)
			return errors.Errorf("connect qmp monitor: %s", err)
		}
		return nil
	} else if monitorPath := s.GetMonitorPath(); len(monitorPath) > 0 {
		return s.StartMonitorWithImportGuestSocketFile(ctx, monitorPath, cb)
	} else {
		log.Errorf("Guest %s start monitor failed, can't get qmp monitor port or monitor path", s.Id)
		return errors.Errorf("Guest %s start monitor failed, can't get qmp monitor port or monitor path", s.Id)
	}
}

func (s *SKVMGuestInstance) onReceiveQMPEvent(event *monitor.Event) {
	switch event.Event {
	case `"BLOCK_JOB_READY"`, `"BLOCK_JOB_COMPLETED"`:
		s.eventBlockJobReady(event)
	case `"BLOCK_JOB_ERROR"`:
		s.eventBlockJobError(event)
	case `"GUEST_PANICKED"`:
		s.eventGuestPaniced(event)
	case `"STOP"`:
		s.eventGuestStop()
	case `"QUORUM_REPORT_BAD"`:
		s.eventQuorumReportBad(event)
	case `"MIGRATION"`:
		s.eventMigration(event)
	}
}

func (s *SKVMGuestInstance) eventMigration(event *monitor.Event) {
	if s.MigrateTask == nil {
		return
	}

	status, ok := event.Data["status"]
	if !ok {
		return
	}
	if status == "pre-switchover" {
		// migrating complete
		s.MigrateTask.onMigrateReceivedPreSwitchoverEvent()
		hostutils.UpdateServerProgress(context.Background(), s.Id, 0.0, 0)
	} else if status == "completed" {
		s.MigrateTask.migrateComplete(nil)
	}
}

func (s *SKVMGuestInstance) eventBlockJobError(event *monitor.Event) {
	if s.MigrateTask != nil {
		s.MigrateTask.onMigrateReceivedBlockJobError(event.String())
	} else {
		s.SyncMirrorJobFailed(event.String())
	}
}

func (s *SKVMGuestInstance) eventGuestStop() {
	// do nothing
}

func (s *SKVMGuestInstance) eventQuorumReportBad(event *monitor.Event) {
	if !atomic.CompareAndSwapInt32(&s.quorumFailed, 0, 1) {
		return
	}

	disks := s.Desc.Disks
	for i := 0; i < len(disks); i++ {
		diskIndex := disks[i].Index
		drive := fmt.Sprintf("drive_%d", diskIndex)
		node := fmt.Sprintf("node_%d", diskIndex)
		child := fmt.Sprintf("children.%d", s.getQuorumChildIndex())
		s.Monitor.XBlockdevChange(drive, "", child, func(res string) {
			if len(res) > 0 {
				log.Errorf("On QUORUM_REPORT_BAD failed remove child %s for parent %s: %s", drive, node, res)
				return
			}
			s.Monitor.DriveDel(node, func(res string) {
				if len(res) > 0 {
					log.Errorf("On QUORUM_REPORT_BAD failed remove drive %s: %s", node, res)
					return
				}
			})
		})
	}

	s.SyncMirrorJobFailed(event.String())
}

func (s *SKVMGuestInstance) eventGuestPaniced(event *monitor.Event) {
	// qemu runc state event source qemu/src/qapi/run-state.json
	params := jsonutils.NewDict()
	if action, ok := event.Data["action"]; ok {
		sAction, _ := action.(string)
		params.Set("action", jsonutils.NewString(sAction))
	}
	if info, ok := event.Data["info"]; ok {
		params.Set("info", jsonutils.Marshal(info))
	}
	params.Set("event", jsonutils.NewString(strings.Trim(event.Event, "\"")))
	_, err := modules.Servers.PerformAction(
		hostutils.GetComputeSession(context.Background()),
		s.GetId(), "event", params)
	if err != nil {
		log.Errorf("Server %s send event guest paniced got error %s", s.GetId(), err)
	}
}

func (s *SKVMGuestInstance) eventBlockJobReady(event *monitor.Event) {
	if !s.IsSlave() {
		return
	}

	itype, ok := event.Data["type"]
	if !ok {
		log.Errorf("block job missing event type")
		return
	}
	stype, _ := itype.(string)
	if stype != "mirror" && stype != "stream" {
		return
	}
	iDevice, ok := event.Data["device"]
	if !ok {
		return
	}
	device := iDevice.(string)
	if !strings.HasPrefix(device, "drive_") {
		return
	}
	disks := s.Desc.Disks
	diskIndex, err := strconv.Atoi(device[len("drive_"):])
	if err != nil || diskIndex < 0 || diskIndex >= len(disks) {
		log.Errorf("failed get disk from index %d", diskIndex)
		return
	}
	var diskId, diskPath string
	for i := 0; i < len(disks); i++ {
		index := disks[i].Index
		if index == int8(diskIndex) {
			diskId = disks[i].DiskId
			diskPath = disks[i].Path
		}
	}
	if len(diskId) == 0 {
		log.Errorf("failed find disk %s", device)
		return
	}

	disk, err := storageman.GetManager().GetDiskByPath(diskPath)
	if err != nil {
		log.Errorf("eventBlockJobReady failed get disk %s", diskPath)
		return
	}
	disk.PostCreateFromImageFuse()
	blockJobCount := s.BlockJobsCount()
	if blockJobCount == 0 {
		for {
			_, err := modules.Servers.PerformAction(
				hostutils.GetComputeSession(context.Background()), s.GetId(), "slave-block-stream-ready", nil,
			)
			if err != nil {
				log.Errorf("onReceiveQMPEvent sync slave block stream ready error: %s", err)
				time.Sleep(3 * time.Second)
			} else {
				break
			}
		}
	}
}

func (s *SKVMGuestInstance) QgaPath() string {
	return path.Join(s.HomeDir(), "qga.sock")
}

func (s *SKVMGuestInstance) InitQga() error {
	guestAgent, err := qga.NewQemuGuestAgent(s.Id, s.QgaPath())
	if err != nil {
		return err
	}
	s.guestAgent = guestAgent
	return nil
}

func (s *SKVMGuestInstance) SyncMirrorJobFailed(reason string) {
	params := jsonutils.NewDict()
	params.Set("reason", jsonutils.NewString(reason))
	_, err := modules.Servers.PerformAction(
		hostutils.GetComputeSession(context.Background()),
		s.GetId(), "block-stream-failed", params,
	)
	if err != nil {
		log.Errorf("Server %s perform block-stream-failed got error %s", s.GetId(), err)
	}
}

func (s *SKVMGuestInstance) onMonitorConnected(ctx context.Context) {
	log.Infof("Monitor connected ...")
	s.Monitor.GetVersion(func(v string) {
		s.onGetQemuVersion(ctx, v)
	})
}

func (s *SKVMGuestInstance) setDestMigrateTLS(ctx context.Context, data *jsonutils.JSONDict) {
	port, _ := data.Int("live_migrate_dest_port")
	s.Monitor.ObjectAdd("tls-creds-x509", map[string]string{
		"dir":         s.getPKIDirPath(),
		"endpoint":    "server",
		"id":          "tls0",
		"verify-peer": "no",
	}, func(res string) {
		if strings.Contains(strings.ToLower(res), "error") {
			hostutils.TaskFailed(ctx, fmt.Sprintf("Migrate add tls-creds-x509 object server tls0 error: %s", res))
			return
		}
		s.Monitor.MigrateSetParameter("tls-creds", "tls0", func(res string) {
			if strings.Contains(strings.ToLower(res), "error") {
				hostutils.TaskFailed(ctx, fmt.Sprintf("Migrate set tls-creds tls0 error: %s", res))
				return
			}
			address := fmt.Sprintf("tcp:0:%d", port)
			s.Monitor.MigrateIncoming(address, func(res string) {
				if strings.Contains(strings.ToLower(res), "error") {
					hostutils.TaskFailed(ctx, fmt.Sprintf("Migrate set incoming %q error: %s", address, res))
					return
				}
				hostutils.TaskComplete(ctx, data)
			})
		})
	})
}

func (s *SKVMGuestInstance) migrateEnableMultifd() error {
	// https://bugzilla.redhat.com/show_bug.cgi?id=1899442
	// Migration fails if both multifd and tls are enabled
	if version.LT(s.QemuVersion, "4.0.0") || s.LiveMigrateUseTls {
		return nil
	}
	var err = make(chan error)
	cb := func(res string) {
		if len(res) > 0 {
			err <- errors.Errorf("failed enable multifd %s", res)
		} else {
			err <- nil
		}
	}
	log.Infof("migrate dest guest enable multifd")
	s.Monitor.MigrateSetCapability("multifd", "on", cb)
	return <-err
}

func (s *SKVMGuestInstance) migrateStartNbdServer(nbdServerPort int) error {
	var err = make(chan error)
	onNbdServerStarted := func(res string) {
		if len(res) > 0 {
			err <- errors.Errorf("failed enable multifd %s", res)
		} else {
			err <- nil
		}
	}
	log.Infof("migrate dest guest start nbd server on %d", nbdServerPort)
	s.Monitor.StartNbdServer(nbdServerPort, true, true, onNbdServerStarted)
	return <-err
}

func (s *SKVMGuestInstance) onGetQemuVersion(ctx context.Context, version string) {
	s.QemuVersion = version
	log.Infof("Guest(%s) qemu version %s", s.Id, s.QemuVersion)
	if s.pciUninitialized {
		if err := s.collectGuestDescription(); err != nil {
			log.Errorf("failed init desc from existing guest: %s", err)
			s.syncStatusUnsync(fmt.Sprintf("failed init desc from existing guest: %s", err))
			return
		}
		s.pciUninitialized = false
	}
	s.guestRun(ctx)
}

func (s *SKVMGuestInstance) syncStatusUnsync(reason string) {
	statusInput := &apis.PerformStatusInput{
		Status:      api.VM_UNSYNC,
		Reason:      reason,
		PowerStates: GetPowerStates(s),
	}
	if _, err := hostutils.UpdateServerStatus(context.Background(), s.Id, statusInput); err != nil {
		log.Errorf("failed update guest status %s", err)
	}
}

func (s *SKVMGuestInstance) collectGuestDescription() error {
	var cpuList []monitor.HotpluggableCPU
	var err error
	if !s.manager.host.IsAarch64() {
		cpuList, err = s.getHotpluggableCPUList()
		if err != nil {
			return errors.Wrap(err, "get hotpluggable cpus")
		}
	}

	pciInfoList, err := s.getPciDevices()
	if err != nil {
		return errors.Wrap(err, "get pci devices")
	}
	memoryDevicesInfoList, err := s.getMemoryDevices()
	if err != nil {
		return errors.Wrap(err, "get memory devices")
	}
	memDevs, err := s.getMemoryDevs()
	if err != nil {
		return errors.Wrap(err, "query mem devs")
	}

	if s.Desc.Machine == "" {
		s.Desc.Machine = s.getMachine()
	}
	qtree := s.infoQtree()
	scsiNumQueues := s.getScsiNumQueues(qtree)
	for i := range s.Desc.Disks {
		// fix virtio disk driver num-queues
		if s.Desc.Disks[i].Driver == "virtio" {
			diskDriver := fmt.Sprintf("drive_%d", s.Desc.Disks[i].Index)
			numQueues := s.getDiskDriverNumQueues(qtree, diskDriver)
			log.Infof("disk driver %s num queues %d", diskDriver, numQueues)
			if numQueues > 0 {
				s.Desc.Disks[i].NumQueues = uint8(numQueues)
			}
		}
	}

	err = s.initGuestDescFromExistingGuest(
		cpuList, pciInfoList, memoryDevicesInfoList, memDevs, scsiNumQueues, qtree)
	if err != nil {
		return errors.Wrap(err, "failed init guest devices")
	}

	if err := SaveLiveDesc(s, s.Desc); err != nil {
		return errors.Wrap(err, "failed save live desc")
	}
	return nil
}

func (s *SKVMGuestInstance) syncVirtioDiskNumQueues() error {
	qtree := s.infoQtree()
	for i := range s.Desc.Disks {
		// fix virtio disk driver num-queues
		if s.Desc.Disks[i].Driver == "virtio" {
			diskDriver := fmt.Sprintf("drive_%d", s.Desc.Disks[i].Index)
			numQueues := s.getDiskDriverNumQueues(qtree, diskDriver)
			log.Infof("disk driver %s num queues %d", diskDriver, numQueues)
			if numQueues > 0 {
				s.Desc.Disks[i].NumQueues = uint8(numQueues)
			}
		}
	}
	return SaveLiveDesc(s, s.Desc)
}

func (s *SKVMGuestInstance) getHotpluggableCPUList() ([]monitor.HotpluggableCPU, error) {
	var res []monitor.HotpluggableCPU
	var errChan = make(chan error)
	cb := func(cpuList []monitor.HotpluggableCPU, err string) {
		if err != "" {
			errChan <- errors.Errorf(err)
		} else {
			res = cpuList
			errChan <- nil
		}
	}
	s.Monitor.GetHotPluggableCpus(cb)
	err := <-errChan
	return res, err
}

func (s *SKVMGuestInstance) infoQtree() string {
	var qtreeChan = make(chan string)
	cb := func(qtree string) {
		qtreeChan <- qtree
	}
	s.Monitor.InfoQtree(cb)
	return <-qtreeChan
}

func getScsiNumQueuesQmp(output string) int64 {
	// if output is response from hmp, sep should use \r\n
	var lines = strings.Split(strings.TrimSuffix(output, "\r\n"), "\\r\\n")
	for i, line := range lines {
		line := strings.TrimSpace(line)
		if strings.HasPrefix(line, "dev: virtio-scsi-device") {
			if len(lines) <= i+1 {
				log.Errorf("failed parse num queues")
				return -1
			}
			line = strings.TrimSpace(lines[i+1])
			segs := strings.Split(line, " ")
			numQueue, err := strconv.ParseInt(segs[2], 10, 0)
			if err != nil {
				log.Errorf("failed parse num queue %s: %s", line, err)
				return -1
			} else {
				return numQueue
			}
		}
	}
	return -1
}

func (s *SKVMGuestInstance) getScsiNumQueues(qtree string) int64 {
	return getScsiNumQueuesQmp(qtree)
}

func (s *SKVMGuestInstance) getDiskDriverNumQueues(qtree, driver string) int64 {
	var lines = strings.Split(strings.TrimSuffix(qtree, "\r\n"), "\\r\\n")
	var currentIndentLevel = -1
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if currentIndentLevel > 0 {
			// disk has been found
			newIndentLevel := len(line) - len(trimmedLine)
			if newIndentLevel <= currentIndentLevel { // run out disk driver block
				break
			}
			if strings.HasPrefix(trimmedLine, "num-queues =") {
				segs := strings.Split(trimmedLine, " ")
				numQueue, err := strconv.ParseInt(segs[2], 10, 0)
				if err != nil {
					log.Errorf("failed parse num queue %s: %s", trimmedLine, err)
					return -1
				} else {
					return numQueue
				}
			} else {
				continue
			}
		}

		if strings.HasPrefix(trimmedLine, "dev: virtio-blk-pci") && strings.Contains(line, driver) {
			currentIndentLevel = len(line) - len(trimmedLine)
		} else {
			continue
		}
	}
	return -1
}

func (s *SKVMGuestInstance) getPciDevices() ([]monitor.PCIInfo, error) {
	var res []monitor.PCIInfo
	var errChan = make(chan error)
	cb := func(pciInfoList []monitor.PCIInfo, err string) {
		if err != "" {
			errChan <- errors.Errorf(err)
		} else {
			res = pciInfoList
			errChan <- nil
		}
	}
	s.Monitor.QueryPci(cb)
	err := <-errChan
	return res, err
}

func (s *SKVMGuestInstance) getMemoryDevs() ([]monitor.Memdev, error) {
	var res []monitor.Memdev
	var errChan = make(chan error)
	cb := func(memDevs []monitor.Memdev, err string) {
		if err != "" {
			errChan <- errors.Errorf(err)
		} else {
			res = memDevs
			errChan <- nil
		}
	}
	s.Monitor.GetMemdevList(cb)
	err := <-errChan
	return res, err
}

func (s *SKVMGuestInstance) getMemoryDevices() ([]monitor.MemoryDeviceInfo, error) {
	var res []monitor.MemoryDeviceInfo
	var errChan = make(chan error)
	cb := func(memoryDevicesInfoList []monitor.MemoryDeviceInfo, err string) {
		if err != "" {
			errChan <- errors.Errorf(err)
		} else {
			res = memoryDevicesInfoList
			errChan <- nil
		}
	}
	s.Monitor.GetMemoryDevicesInfo(cb)
	err := <-errChan
	return res, err
}

func (s *SKVMGuestInstance) hasVirtioBlkDriver() bool {
	for i := range s.Desc.Disks {
		if s.Desc.Disks[i].Driver == "virtio" {
			return true
		}
	}
	return false
}

func (s *SKVMGuestInstance) guestRun(ctx context.Context) {
	if s.LiveMigrateDestPort != nil && ctx != nil && !s.IsSlave() {
		// dest migrate guest
		body := jsonutils.NewDict()
		body.Set("live_migrate_dest_port", jsonutils.NewInt(*s.LiveMigrateDestPort))
		err := s.migrateEnableMultifd()
		if err != nil {
			hostutils.TaskFailed(ctx, err.Error())
			return
		}

		if s.hasVirtioBlkDriver() {
			// virtio driver bind iothread, need migrate use driver mirror
			nbdServerPort := s.manager.GetNBDServerFreePort()
			defer s.manager.unsetPort(nbdServerPort)
			err = s.migrateStartNbdServer(nbdServerPort)
			if err != nil {
				hostutils.TaskFailed(ctx, err.Error())
				return
			}
			body.Set("nbd_server_port", jsonutils.NewInt(int64(nbdServerPort)))
		} else {
			body.Set("nbd_server_disabled", jsonutils.JSONTrue)
		}

		if s.LiveMigrateUseTls {
			s.setDestMigrateTLS(ctx, body)
		} else {
			hostutils.TaskComplete(ctx, body)
		}
	} else if s.IsSlave() {
		s.startQemuBuiltInNbdServer(ctx)
	} else {
		if s.IsMaster() {
			s.startDiskBackupMirror(ctx)
		} else {
			s.DoResumeTask(ctx, true, false)
		}
		if err := s.InitQga(); err != nil {
			log.Errorf("Guest %s init qga failed %s", s.Id, err)
		}
	}
}

func (s *SKVMGuestInstance) onMonitorDisConnect(err error) {
	log.Errorf("Guest %s on Monitor Disconnect reason: %v", s.Id, err)
	s.CleanStartupTask()
	s.scriptStop()
	s.SyncStatus(fmt.Sprintf("monitor disconnect %v", err))
	if s.guestAgent != nil {
		s.guestAgent.Close()
		s.guestAgent = nil
	}
	s.clearCgroup(0)
	s.Monitor = nil
}

func (s *SKVMGuestInstance) startDiskBackupMirror(ctx context.Context) {
	if ctx == nil || len(appctx.AppContextTaskId(ctx)) == 0 {
		s.DoResumeTask(ctx, true, false)
	} else {
		nbdUri, ok := s.Desc.Metadata["backup_nbd_server_uri"]
		if !ok {
			hostutils.TaskFailed(ctx, "Missing dest nbd location")
			return
		}
		nbdOpts := strings.Split(nbdUri, ":")
		if len(nbdOpts) != 3 {
			hostutils.TaskFailed(ctx, fmt.Sprintf("Nbd uri is not vaild %s", nbdUri))
			return
		}
		s.quorumFailed = 0
		onSucc := func() {
			if err := s.updateChildIndex(); err != nil {
				hostutils.TaskFailed(ctx, err.Error())
				return
			}
			s.DoResumeTask(ctx, true, false)
		}
		onFail := func(res string) {
			s.SyncMirrorJobFailed(res)
			s.DoResumeTask(ctx, true, false)
		}
		NewGuestBlockReplicationTask(ctx, s, nbdOpts[1], nbdOpts[2], "full", onSucc, onFail).Start()
	}
}

func (s *SKVMGuestInstance) startQemuBuiltInNbdServer(ctx context.Context) {
	if ctx != nil && len(appctx.AppContextTaskId(ctx)) > 0 {
		nbdServerPort := s.manager.GetNBDServerFreePort()
		defer s.manager.unsetPort(nbdServerPort)
		var onNbdServerStarted = func(res string) {
			if len(res) > 0 {
				log.Errorf("Start Qemu Builtin nbd server error %s", res)
				hostutils.TaskFailed(ctx, res)
			} else {
				res := jsonutils.NewDict()
				res.Set("nbd_server_port", jsonutils.NewInt(int64(nbdServerPort)))
				hostutils.TaskComplete(ctx, res)
			}
		}
		s.Monitor.StartNbdServer(nbdServerPort, true, true, onNbdServerStarted)
	} else {
		s.SyncStatus("")
	}
}

func (s *SKVMGuestInstance) SlaveDisksBlockStream() error {
	errChan := make(chan string, 1)
	disks := s.Desc.Disks
	for i := 0; i < len(disks); i++ {
		diskIndex := disks[i].Index
		drive := fmt.Sprintf("drive_%d", diskIndex)
		s.Monitor.BlockStream(drive, func(res string) {
			errChan <- res
		})
		if errStr := <-errChan; len(errStr) > 0 {
			return fmt.Errorf("block stream disk %s: %s", drive, errStr)
		}
	}
	return nil
}

func (s *SKVMGuestInstance) releaseGuestCpuset() {
	for _, vcpuPin := range s.Desc.VcpuPin {
		pcpuSet, err := cpuset.Parse(vcpuPin.Pcpus)
		if err != nil {
			log.Errorf("failed parse %s pcpus: %s", s.GetName(), vcpuPin.Pcpus)
			continue
		}
		vcpuSet, err := cpuset.Parse(vcpuPin.Vcpus)
		if err != nil {
			log.Errorf("failed parse %s vcpus: %s", s.GetName(), vcpuPin.Vcpus)
			continue
		}
		s.manager.cpuSet.ReleaseCpus(pcpuSet.ToSlice(), vcpuSet.Size())
	}
	s.Desc.VcpuPin = nil
	SaveLiveDesc(s, s.Desc)
}

func (s *SKVMGuestInstance) clearCgroup(pid int) {
	s.releaseGuestCpuset()
	if pid == 0 && s.cgroupPid > 0 {
		pid = s.cgroupPid
	}
	cgrupName := s.GetCgroupName()
	log.Infof("cgroup destroy %d %s", pid, cgrupName)
	if pid > 0 && !options.HostOptions.DisableSetCgroup {
		cgrouputils.CgroupDestroy(strconv.Itoa(pid), cgrupName)
	}
}

func (s *SKVMGuestInstance) IsMaster() bool {
	return s.Desc.IsMaster
}

func (s *SKVMGuestInstance) IsSlave() bool {
	return s.Desc.IsSlave
}

func (s *SKVMGuestInstance) IsMigratingDestGuest() bool {
	return s.LiveMigrateDestPort != nil
}

func (s *SKVMGuestInstance) DiskCount() int {
	return len(s.Desc.Disks)
}

func (s *SKVMGuestInstance) BlockJobsCount() int {
	res := make(chan []monitor.BlockJob)
	log.Debugf("BlockJobsCount start...")
	s.Monitor.GetBlockJobs(func(jobs []monitor.BlockJob) {
		res <- jobs
	})
	select {
	case <-time.After(time.Second * 30):
		log.Debugf("BlockJobsCount timeout")
		return -1
	case v := <-res:
		log.Debugf("BlockJobsCount %d", len(v))
		return len(v)
	}
}

func (s *SKVMGuestInstance) detachStartupTask() {
	log.Infof("[%s] detachStartupTask", s.GetId())
	s.StartupTask = nil
}

func (s *SKVMGuestInstance) CleanStartupTask() {
	if s.StartupTask != nil {
		log.Infof("[%s] Clean startup task ... stop task ...", s.GetId())
		s.StartupTask.Stop()
		s.StartupTask = nil
	} else {
		log.Infof("[%s] Clean startup task ... no task", s.GetId())
	}
}

func (s *SKVMGuestInstance) onMonitorTimeout(ctx context.Context, err error) {
	log.Errorf("Monitor connect timeout, VM %s frozen: %s force restart!!!!", s.Id, err)
	s.ForceStop()
	timeutils2.AddTimeout(
		time.Second*3, func() { s.StartGuest(ctx, nil, jsonutils.NewDict()) })
}

func (s *SKVMGuestInstance) GetHmpMonitorPort(vncPort int) int {
	if vncPort <= 0 {
		vncPort = s.GetVncPort()
	}
	if vncPort > 0 {
		return vncPort + MONITOR_PORT_BASE
	} else {
		return -1
	}
}

func (s *SKVMGuestInstance) GetQmpMonitorPort(vncPort int) int {
	if vncPort <= 0 {
		vncPort = s.GetVncPort()
	}
	if vncPort > 0 {
		return vncPort + QMP_MONITOR_PORT_BASE
	} else {
		return -1
	}
}

func (s *SKVMGuestInstance) GetVncPort() int {
	if s.IsRunning() {
		vncPort, err := ioutil.ReadFile(s.GetVncFilePath())
		if err != nil {
			return -1
		}
		strPort := strings.TrimSpace(string(vncPort))
		if len(strPort) > 0 {
			port, err := strconv.Atoi(strPort)
			if err == nil {
				return port
			}
		}
	}
	return -1
}

func (s *SKVMGuestInstance) saveVncPort(port int) error {
	return fileutils2.FilePutContents(s.GetVncFilePath(), fmt.Sprintf("%d", port), false)
}

func (s *SKVMGuestInstance) DoResumeTask(ctx context.Context, isTimeout bool, cleanTls bool) {
	s.StartupTask = NewGuestResumeTask(ctx, s, isTimeout, false)
	s.StartupTask.Start()
}

func (s *SKVMGuestInstance) SyncStatus(reason string) {
	if s.IsRunning() {
		s.Monitor.GetBlockJobCounts(s.CheckBlockOrRunning)
		return
	}
	var status = api.VM_READY
	if s.IsSuspend() {
		status = api.VM_SUSPEND
	}
	statusInput := &apis.PerformStatusInput{
		Status:      status,
		Reason:      reason,
		PowerStates: GetPowerStates(s),
		HostId:      hostinfo.Instance().HostId,
	}

	if _, err := hostutils.UpdateServerStatus(context.Background(), s.Id, statusInput); err != nil {
		log.Errorf("failed update guest status %s", err)
	}
}

func (s *SKVMGuestInstance) CheckBlockOrRunning(jobs int) {
	var status = api.VM_RUNNING

	if jobs > 0 {
		// TODO: check block jobs ready
		status = api.VM_BLOCK_STREAM
	}
	var statusInput = &apis.PerformStatusInput{
		Status:         status,
		BlockJobsCount: jobs,
		PowerStates:    GetPowerStates(s),
		HostId:         hostinfo.Instance().HostId,
	}
	_, err := hostutils.UpdateServerStatus(context.Background(), s.Id, statusInput)
	if err != nil {
		log.Errorln(err)
	}
}

//func (s *SKVMGuestInstance) GetRescueDesc() error {
//	if !s.SourceDesc.LightMode {
//		return errors.Errorf("guest %s not in rescue mode", s.Id)
//	}
//
//	s.SourceDesc.RescueInitrdPath = path.Join(s.GetRescueDirPath(), api.GUEST_RESCUE_INITRAMFS)
//	s.SourceDesc.RescueKernelPath = path.Join(s.GetRescueDirPath(), api.GUEST_RESCUE_KERNEL)
//	s.SourceDesc.RescueDiskPath = path.Join(s.GetRescueDirPath(), api.GUEST_RESCUE_SYS_DISK_NAME)
//	if s.manager.GetHost().IsAarch64() {
//		s.SourceDesc.RescueInitrdPath = path.Join(s.GetRescueDirPath(), api.GUEST_RESCUE_INITRAMFS_ARM64)
//		s.SourceDesc.RescueKernelPath = path.Join(s.GetRescueDirPath(), api.GUEST_RESCUE_KERNEL_ARM64)
//	}
//
//	// Address
//	bus, slot, found := s.findUnusedSlotForController(desc.CONTROLLER_TYPE_PCI_ROOT, 0)
//	if !found {
//		return errors.Errorf("no valid pci address found ?")
//	}
//	s.SourceDesc.RescueDiskDeviceBus = uint(bus)
//	s.SourceDesc.RescueDiskDeviceSlot = uint(slot)
//
//	return nil
//}

type guestStartTask struct {
	s *SKVMGuestInstance

	ctx    context.Context
	params *jsonutils.JSONDict
}

func (t *guestStartTask) Run() {
	if _, err := t.s.asyncScriptStart(t.ctx, t.params); err != nil {
		log.Errorf("failed asyncScriptStart %s: %s", t.s.GetId(), err)
	}
}

func (t *guestStartTask) Dump() string {
	return fmt.Sprintf("guest %s params: %v", t.s.Id, t.params)
}

func (s *SKVMGuestInstance) prepareEncryptKeyForStart(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if s.isEncrypted() {
		ekey, err := s.getEncryptKey(ctx, userCred)
		if err != nil {
			log.Errorf("fail to fetch encrypt key: %s", err)
			return nil, errors.Wrap(err, "getEncryptKey")
		}
		if params == nil {
			params = jsonutils.NewDict()
		}
		params.Add(jsonutils.NewString(ekey.Key), "encrypt_key")
		params.Add(jsonutils.Marshal(ekey), "encrypt_info")
	}
	return params, nil
}

func (s *SKVMGuestInstance) HandleGuestStart(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if s.IsStopped() {
		data, err := body.Get("params")
		if err != nil {
			data = jsonutils.NewDict()
		}
		err = s.StartGuest(ctx, userCred, data.(*jsonutils.JSONDict))
		if err != nil {
			return nil, errors.Wrap(err, "StartGuest")
		}
		res := jsonutils.NewDict()
		res.Set("vnc_port", jsonutils.NewInt(0))
		return res, nil
	} else {
		vncPort := s.GetVncPort()
		if vncPort > 0 {
			res := jsonutils.NewDict()
			res.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
			res.Set("is_running", jsonutils.JSONTrue)
			return res, nil
		} else {
			return nil, httperrors.NewBadRequestError("Seems started, but no VNC info")
		}
	}
}

func (s *SKVMGuestInstance) StartGuest(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict) error {
	var err error
	params, err = s.prepareEncryptKeyForStart(ctx, userCred, params)
	if err != nil {
		return errors.Wrap(err, "prepareEncryptKeyForStart")
	}
	task := &guestStartTask{
		s:      s,
		ctx:    ctx,
		params: params,
	}
	s.manager.GuestStartWorker.Run(task, nil, nil)
	return nil
}

func (s *SKVMGuestInstance) HandleStop(ctx context.Context, timeout int64) error {
	hostutils.DelayTaskWithoutReqctx(ctx, s.ExecStopTask, timeout)
	return nil
}

func (s *SKVMGuestInstance) DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	diskInfo := deployapi.DiskInfo{}
	if s.isEncrypted() {
		ekey, err := s.getEncryptKey(ctx, userCred)
		if err != nil {
			log.Errorf("fail to fetch encrypt key: %s", err)
			return nil, errors.Wrap(err, "getEncryptKey")
		}
		diskInfo.EncryptPassword = ekey.Key
		diskInfo.EncryptAlg = string(ekey.Alg)
	}
	disks := s.Desc.Disks
	if len(disks) > 0 {
		diskPath := disks[0].Path
		disk, err := storageman.GetManager().GetDiskByPath(diskPath)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDiskByPath(%s)", diskPath)
		}
		diskInfo.Path = disk.GetPath()
		return disk.DeployGuestFs(&diskInfo, s.Desc, deployInfo)
	} else {
		return nil, fmt.Errorf("Guest dosen't have disk ??")
	}
}

// Delay process
func (s *SKVMGuestInstance) CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migrated, ok := params.(bool)
	if !ok {
		return nil, hostutils.ParamsError
	}
	if err := s.StartDelete(ctx, migrated); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *SKVMGuestInstance) StartDelete(ctx context.Context, migrated bool) error {
	for s.IsRunning() {
		s.ForceStop()
		time.Sleep(time.Second * 1)
	}
	return s.Delete(ctx, migrated)
}

func (s *SKVMGuestInstance) ForceStop() bool {
	s.ExitCleanup(true)
	if s.IsRunning() {
		output, err := procutils.NewCommand("kill", "-9", fmt.Sprintf("%d", s.GetPid())).Output()
		if err != nil {
			log.Errorf("kill process %d failed: %s, %s", s.GetPid(), err, output)
			return false
		}
		for _, f := range s.GetCleanFiles() {
			output, err := procutils.NewCommand("rm", "-f", f).Output()
			if err != nil {
				log.Errorf("rm %s failed: %s, %s", f, err, output)
				return false
			}
		}
		return true
	}
	return false
}

func (s *SKVMGuestInstance) ExitCleanup(clear bool) {
	if clear {
		pid := s.GetPid()
		if pid > 0 {
			s.clearCgroup(pid)
		} else {
			s.clearCgroup(0)
		}
	}
	if s.Monitor != nil {
		s.Monitor.Disconnect()
		s.Monitor = nil
	}
	s.LiveMigrateDestPort = nil
}

func (s *SKVMGuestInstance) CleanupCpuset() {
	task := cgrouputils.NewCGroupCPUSetTask(strconv.Itoa(s.GetPid()), s.GetCgroupName(), 0, "")
	if !task.RemoveTask() {
		log.Warningf("remove cpuset cgroup error: %s, pid: %d", s.Id, s.GetPid())
	}
}

func (s *SKVMGuestInstance) GetCleanFiles() []string {
	return []string{s.GetPidFilePath(), s.GetVncFilePath(), s.getEncryptKeyPath()}
}

func (s *SKVMGuestInstance) delTmpDisks(ctx context.Context, migrated bool) error {
	disks := s.Desc.Disks
	for _, disk := range disks {
		if disk.Path != "" {
			diskPath := disk.Path
			d, _ := storageman.GetManager().GetDiskByPath(diskPath)
			if d != nil && d.GetType() == api.STORAGE_LOCAL && migrated {
				skipRecycle := true
				if err := d.DeleteAllSnapshot(skipRecycle); err != nil {
					log.Errorln(err)
					return err
				}
				if _, err := d.Delete(ctx, api.DiskDeleteInput{SkipRecycle: &skipRecycle}); err != nil {
					log.Errorln(err)
					return err
				}
			}
			if migrated {
				// remove memory snapshot files
				dir := GetMemorySnapshotPath(s.GetId(), "")
				if err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-rf", dir).Run(); err != nil {
					return errors.Wrapf(err, "remove dir %q", dir)
				}
			}
		}
	}
	return nil
}

func (s *SKVMGuestInstance) delFlatFiles(ctx context.Context) error {
	if eid, ok := s.Desc.Metadata["__server_convert_from_esxi"]; ok && len(eid) > 0 {
		disks := s.Desc.Disks
		connections := new(deployapi.EsxiDisksConnectionInfo)
		connections.Disks = make([]*deployapi.EsxiDiskInfo, len(disks))
		for i := 0; i < len(disks); i++ {
			connections.Disks[i] = &deployapi.EsxiDiskInfo{DiskPath: disks[i].EsxiFlatFilePath}
		}
		_, err := deployclient.GetDeployClient().DisconnectEsxiDisks(ctx, connections)
		if err != nil {
			log.Errorf("Disconnect %s esxi disks failed %s", s.GetName(), err)
			return err
		}
	}
	return nil
}

func (s *SKVMGuestInstance) Delete(ctx context.Context, migrated bool) error {
	if err := s.delTmpDisks(ctx, migrated); err != nil {
		return errors.Wrap(err, "delTmpDisks")
	}
	return DeleteHomeDir(s)
}

func (s *SKVMGuestInstance) Stop() bool {
	s.ExitCleanup(true)
	if s.scriptStop() {
		return true
	} else {
		return false
	}
}

func (s *SKVMGuestInstance) LogFilePath() string {
	return path.Join(s.manager.QemuLogDir(), s.Id)
}

func (s *SKVMGuestInstance) readQemuLogFileEnd(size int64) string {
	fname := s.LogFilePath()
	file, err := os.Open(fname)
	if err != nil {
		return fmt.Sprintf("failed open log file %s: %s", fname, err)
	}
	defer file.Close()

	buf := make([]byte, size)
	stat, err := os.Stat(fname)
	if err != nil {
		return fmt.Sprintf("failed stat file %s: %s", fname, err)
	}
	start := stat.Size() - size
	_, err = file.ReadAt(buf, start)
	if err != nil {
		return fmt.Sprintf("failed read logfile %s: %s", fname, err)
	}
	return string(buf)
}

func (s *SKVMGuestInstance) pyLauncherPath() string {
	return path.Join(s.HomeDir(), "startvm.py")
}

func (s *SKVMGuestInstance) scriptStart(ctx context.Context) error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible(s.manager.getPythonPath(), s.pyLauncherPath()).Output()
	if err != nil {
		return fmt.Errorf("Start VM Failed %s %s", output, err)
	}
	pid, err := strconv.Atoi(string(output))
	if err != nil {
		return errors.Wrapf(err, "failed parse pid %s", output)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrapf(err, "find process pid(%d)", pid)
	}
	for {
		err = proc.Signal(syscall.Signal(0))
		if err != nil { // qemu process exited
			log.Errorf("Guest %s check qemu(%d) process failed: %s", s.Id, pid, err)
			return errors.Errorf(s.readQemuLogFileEnd(64))
		}
		if err = s.StartMonitor(ctx, nil); err == nil {
			return nil
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func (s *SKVMGuestInstance) scriptStop() bool {
	_, err := procutils.NewRemoteCommandAsFarAsPossible("bash", s.GetStopScriptPath()).Output()
	if err != nil {
		log.Errorln(err)
		return false
	}
	return true
}

func (s *SKVMGuestInstance) forceScriptStop() bool {
	_, err := procutils.NewRemoteCommandAsFarAsPossible("bash", s.GetStopScriptPath(), "--force").Output()
	if err != nil {
		log.Errorln(err)
		return false
	}
	return true
}

func (s *SKVMGuestInstance) ExecStopTask(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	timeout, ok := params.(int64)
	if !ok {
		return nil, hostutils.ParamsError
	}
	NewGuestStopTask(s, ctx, timeout).Start()
	return nil, nil
}

func (s *SKVMGuestInstance) ExecStartRescueTask(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	baremetalManagerUri, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	NewGuestStartRescueTask(s, ctx, baremetalManagerUri).Start()
	return nil, nil
}

func (s *SKVMGuestInstance) ExecStopRescueTask(ctx context.Context, params interface{}) {
	NewGuestStopRescueTask(s, ctx).Start()
}

func (s *SKVMGuestInstance) ExecSuspendTask(ctx context.Context) {
	NewGuestSuspendTask(s, ctx, nil).Start()
}

func pathEqual(disk, ndisk *desc.SGuestDisk) bool {
	if disk.Path != "" && ndisk.Path != "" {
		return disk.Path == ndisk.Path
	} else if disk.Url != "" && ndisk.Url != "" {
		//path1 := disk.AssumedPath
		//path2 := ndisk.AssumedPath
		//return path1 == path2
		// not assumed path found
		return true
	} else {
		return false
	}
}

func (s *SKVMGuestInstance) compareDescDisks(newDesc *desc.SGuestDesc) ([]*desc.SGuestDisk, []*desc.SGuestDisk) {
	var delDisks, addDisks = make([]*desc.SGuestDisk, 0), make([]*desc.SGuestDisk, 0)
	newDisks := newDesc.Disks
	for _, disk := range newDisks {
		if utils.IsInStringArray(disk.Driver, []string{"virtio", "scsi"}) {
			newDisk := *disk
			addDisks = append(addDisks, &newDisk)
		}
	}
	oldDisks := s.Desc.Disks
	for _, disk := range oldDisks {
		if utils.IsInStringArray(disk.Driver, []string{"virtio", "scsi"}) {
			var find = false
			for idx, ndisk := range addDisks {
				diskIndex := disk.Index
				nDiskIndex := ndisk.Index
				if diskIndex == nDiskIndex && pathEqual(disk, ndisk) {
					addDisks = append(addDisks[:idx], addDisks[idx+1:]...)
					find = true
					break
				}
			}
			if !find {
				delDisks = append(delDisks, disk)
			}
		}
	}
	return delDisks, addDisks
}

func (s *SKVMGuestInstance) compareDescIsolatedDevices(newDesc *desc.SGuestDesc,
) ([]*desc.SGuestIsolatedDevice, []*desc.SGuestIsolatedDevice) {
	var delDevs, addDevs = []*desc.SGuestIsolatedDevice{}, []*desc.SGuestIsolatedDevice{}
	newDevs := newDesc.IsolatedDevices
	for _, dev := range newDevs {
		newDev := *dev
		addDevs = append(addDevs, &newDev)
	}
	oldDevs := s.Desc.IsolatedDevices
	for _, oldDev := range oldDevs {
		var find = false
		oVendorDevId := oldDev.VendorDeviceId
		oAddr := oldDev.Addr
		for idx, addDev := range addDevs {
			nVendorDevId := addDev.VendorDeviceId
			nAddr := addDev.Addr
			if oVendorDevId == nVendorDevId && oAddr == nAddr {
				addDevs = append(addDevs[:idx], addDevs[idx+1:]...)
				find = true
				break
			}
		}
		if !find {
			delDevs = append(delDevs, oldDev)
		}
	}
	return delDevs, addDevs
}

func (s *SKVMGuestInstance) compareDescCdroms(newDesc *desc.SGuestDesc) []*desc.SGuestCdrom {
	var changeCdroms []*desc.SGuestCdrom
	newCdroms := newDesc.Cdroms
	for i := 0; i < options.HostOptions.CdromCount; i++ {
		changeCdrom := new(desc.SGuestCdrom)
		changeCdrom.Ordinal = int64(i)
		changeCdrom.Path = ""
		s.archMan.GenerateCdromDesc(s.GetOsName(), changeCdrom)
		changeCdroms = append(changeCdroms, changeCdrom)
	}

	for _, newCdrom := range newCdroms {
		ordinal := newCdrom.Ordinal
		path := newCdrom.Path
		changeCdrom := new(desc.SGuestCdrom)
		changeCdrom.Ordinal = ordinal
		changeCdrom.Path = path
		s.archMan.GenerateCdromDesc(s.GetOsName(), changeCdrom)
		for i, tmp := range changeCdroms {
			if tmp.Ordinal == ordinal {
				changeCdroms[i] = changeCdrom
			}
		}
	}

	return changeCdroms
}

func (s *SKVMGuestInstance) compareDescFloppys(newDesc *desc.SGuestDesc) []*desc.SGuestFloppy {
	var changeFloppys []*desc.SGuestFloppy
	newFloppys := newDesc.Floppys
	for i := 0; i < options.HostOptions.FloppyCount; i++ {
		changeFloppy := new(desc.SGuestFloppy)
		changeFloppy.Ordinal = int64(i)
		changeFloppy.Path = ""
		s.archMan.GenerateFloppyDesc(s.GetOsName(), changeFloppy)
		changeFloppys = append(changeFloppys, changeFloppy)
	}

	for _, newFloppy := range newFloppys {
		ordinal := newFloppy.Ordinal
		path := newFloppy.Path
		changeFloppy := new(desc.SGuestFloppy)
		changeFloppy.Ordinal = ordinal
		changeFloppy.Path = path
		s.archMan.GenerateFloppyDesc(s.GetOsName(), changeFloppy)
		for i, tmp := range changeFloppys { //如果新增的跟changeCdrom重叠，则以新增为准
			if tmp.Ordinal == ordinal {
				changeFloppys[i] = changeFloppy
			}
		}
	}

	return changeFloppys
}

func (s *SKVMGuestInstance) compareDescNetworks(newDesc *desc.SGuestDesc,
) ([]*desc.SGuestNetwork, []*desc.SGuestNetwork, [][2]*desc.SGuestNetwork) {
	var isValid = func(net *desc.SGuestNetwork) bool {
		return net.Driver == "virtio" || net.Driver == "vfio-pci"
	}
	var isChangeNetworkValid = func(net *desc.SGuestNetwork) bool {
		return net.Driver == "virtio" || net.Driver == "vfio-pci"
	}

	var findNet = func(nets []*desc.SGuestNetwork, net *desc.SGuestNetwork) int {
		for i := 0; i < len(nets); i++ {
			if nets[i].Mac == net.Mac {
				return i
			}
		}
		return -1
	}

	var delNics, addNics = []*desc.SGuestNetwork{}, []*desc.SGuestNetwork{}
	var changedNics = [][2]*desc.SGuestNetwork{}
	for _, n := range newDesc.Nics {
		if isValid(n) {
			newNic := *n
			// assume all nics in new desc are new
			addNics = append(addNics, &newNic)
		}
	}

	for _, n := range s.Desc.Nics {
		if isValid(n) {
			idx := findNet(addNics, n)
			if idx >= 0 {
				if isChangeNetworkValid(n) {
					// check if bridge changed
					changedNics = append(changedNics, [2]*desc.SGuestNetwork{
						n,            // old
						addNics[idx], // new
					})
				}

				// remove existing nic from new
				addNics = append(addNics[:idx], addNics[idx+1:]...)
			} else {
				// not found, remove the nic
				delNics = append(delNics, n)
			}
		}
	}
	return delNics, addNics, changedNics
}

func getNicBridge(nic *desc.SGuestNetwork) string {
	if nic.Bridge == "" && nic.Vpc.Provider == api.VPC_PROVIDER_OVN {
		return options.HostOptions.OvnIntegrationBridge
	} else if nic.Bridge == api.HostTapBridge {
		return options.HostOptions.TapBridgeName
	} else if nic.Bridge == api.HostVpcBridge {
		return options.HostOptions.OvnIntegrationBridge
	} else {
		return nic.Bridge
	}
}

func (s *SKVMGuestInstance) onNicChange(oldNic, newNic *desc.SGuestNetwork) error {
	log.Infof("nic changed old: %s new: %s", jsonutils.Marshal(oldNic), jsonutils.Marshal(newNic))
	// override network base desc
	oldNic.GuestnetworkBaseDesc = newNic.GuestnetworkBaseDesc

	if oldNic.Driver == "vfio-pci" {
		err := s.reconfigureVfioNicsBandwidth(oldNic)
		if err != nil {
			return err
		}
		return nil
	}

	oldbr := getNicBridge(oldNic)
	oldifname := oldNic.Ifname
	newbr := getNicBridge(newNic)
	newifname := newNic.Ifname
	newvlan := newNic.Vlan
	if oldbr != newbr {
		// bridge changed
		if oldifname == newifname {
			args := []string{
				"--", "del-port", oldbr, oldifname,
				"--", "add-port", newbr, newifname,
			}
			if newvlan > 1 {
				args = append(args, fmt.Sprintf("tag=%d", newvlan))
			}
			output, err := procutils.NewRemoteCommandAsFarAsPossible("ovs-vsctl", args...).Output()
			log.Infof("ovs-vsctl %v: %s", args, output)
			if err != nil {
				return errors.Wrap(err, "NewRemoteCommandAsFarAsPossible")
			}
		} else {
			log.Errorf("cannot change both bridge(%s!=%s) and ifname(%s!=%s)!!!!!", oldbr, newbr, oldifname, newifname)
		}
	} else {
		// bridge not changed
		if oldifname == newifname {
			if newvlan > 1 {
				output, err := procutils.NewRemoteCommandAsFarAsPossible("ovs-vsctl", "set", "port", newifname, fmt.Sprintf("tag=%d", newvlan)).Output()
				if err != nil {
					return errors.Wrapf(err, "NewRemoteCommandAsFarAsPossible change vlan tag to %d: %s", newvlan, output)
				}
			} else {
				// clear vlan
				output, err := procutils.NewRemoteCommandAsFarAsPossible("ovs-vsctl", "get", "port", newifname, "tag").Output()
				if err != nil {
					return errors.Wrapf(err, "NewRemoteCommandAsFarAsPossible get vlan tag: %s", output)
				}
				tagStr := strings.TrimSpace(string(output))
				if tag, err := strconv.Atoi(tagStr); err == nil && tag > 1 {
					if output, err := procutils.NewRemoteCommandAsFarAsPossible("ovs-vsctl", "remove", "port", newifname, "tag", tagStr).Output(); err != nil {
						return errors.Wrapf(err, "NewRemoteCommandAsFarAsPossible remove vlan tag %s: %s", tagStr, output)
					}
				}
			}
		}
	}
	return nil
}

func (s *SKVMGuestInstance) SyncConfig(
	ctx context.Context, guestDesc *desc.SGuestDesc, fwOnly bool,
) (jsonutils.JSONObject, error) {
	var delDisks, addDisks []*desc.SGuestDisk
	var delNetworks, addNetworks []*desc.SGuestNetwork
	var changedNetworks [][2]*desc.SGuestNetwork
	var delDevs, addDevs []*desc.SGuestIsolatedDevice
	var cdroms []*desc.SGuestCdrom
	var floppys []*desc.SGuestFloppy

	if err := SaveDesc(s, guestDesc); err != nil {
		return nil, err
	}

	if !fwOnly && !s.isImportFromLibvirt() {
		delDisks, addDisks = s.compareDescDisks(guestDesc)
		cdroms = s.compareDescCdroms(guestDesc)
		floppys = s.compareDescFloppys(guestDesc)
		delNetworks, addNetworks, changedNetworks = s.compareDescNetworks(guestDesc)
		delDevs, addDevs = s.compareDescIsolatedDevices(guestDesc)
	}

	if len(changedNetworks) > 0 && s.IsRunning() {
		// process changed networks
		for i := range changedNetworks {
			err := s.onNicChange(changedNetworks[i][0], changedNetworks[i][1])
			if err != nil {
				return nil, errors.Wrap(err, "onNicChange")
			}
		}
	}

	if !s.IsRunning() {
		return nil, nil
	}

	// update guest live desc, don't be here update cpu and mem
	// cpu and memory should update from SGuestHotplugCpuMemTask
	s.Desc.SGuestControlDesc = guestDesc.SGuestControlDesc
	s.Desc.SGuestProjectDesc = guestDesc.SGuestProjectDesc
	s.Desc.SGuestRegionDesc = guestDesc.SGuestRegionDesc
	s.Desc.SGuestMetaDesc = guestDesc.SGuestMetaDesc

	SaveLiveDesc(s, s.Desc)

	if fwOnly {
		res := jsonutils.NewDict()
		res.Set("task", jsonutils.NewArray(jsonutils.NewString("secgroupsync")))
		return res, nil
	}
	var runTaskNames = []jsonutils.JSONObject{}
	var tasks = []IGuestTasks{}

	if len(delDisks)+len(addDisks) > 0 || cdroms != nil || floppys != nil {
		task := NewGuestDiskSyncTask(s, delDisks, addDisks, cdroms, floppys)
		runTaskNames = append(runTaskNames, jsonutils.NewString("disksync"))
		tasks = append(tasks, task)
	}

	if len(delDevs)+len(addDevs) > 0 {
		task := NewGuestIsolatedDeviceSyncTask(s, delDevs, addDevs)
		runTaskNames = append(runTaskNames, jsonutils.NewString("isolated_device_sync"))
		tasks = append(tasks, task)
	}

	// make sure network sync before isolated device
	if len(delNetworks)+len(addNetworks) > 0 {
		task := NewGuestNetworkSyncTask(s, delNetworks, addNetworks)
		runTaskNames = append(runTaskNames, jsonutils.NewString("networksync"))
		tasks = append(tasks, task)
	}

	lenTasks := len(tasks)
	var callBack = func(errs []error) {
		SaveLiveDesc(s, s.Desc)
		if lenTasks > 0 { // devices updated, regenerate start script
			vncPort := s.GetVncPort()
			data := jsonutils.NewDict()
			data.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
			data.Set("sync_qemu_cmdline", jsonutils.JSONTrue)
			if err := s.saveScripts(data); err != nil {
				log.Errorf("failed save script: %s", err)
			}
		}

		if len(errs) == 0 {
			hostutils.TaskComplete(ctx, nil)
		} else {
			var reason string
			for _, err := range errs {
				reason += "; " + err.Error()
			}
			hostutils.TaskFailed(ctx, reason[2:])
		}
	}

	NewGuestSyncConfigTaskExecutor(ctx, s, tasks, callBack).Start(1)
	res := jsonutils.NewDict()
	res.Set("task", jsonutils.NewArray(runTaskNames...))
	return res, nil
}

func (s *SKVMGuestInstance) getApptags() []string {
	if tagsStr, ok := s.Desc.Metadata["app_tags"]; ok {
		if len(tagsStr) > 0 {
			return strings.Split(tagsStr, ",")
		}
	}
	return nil
}

func (s *SKVMGuestInstance) getStorageDeviceId() string {
	disks := s.Desc.Disks
	if len(disks) > 0 {
		if len(disks[0].Path) > 0 {
			return fileutils2.GetDevId(disks[0].Path)
		}
	}
	return ""
}

func (s *SKVMGuestInstance) GetCgroupName() string {
	if s.cgroupPid == 0 {
		return ""
	}

	if val, _ := s.Desc.Metadata["__enable_cgroup_cpuset"]; val == "true" {
		return fmt.Sprintf("%s/server_%s_%d", hostconsts.HOST_CGROUP, s.Id, s.cgroupPid)
	}

	return ""
}

func (s *SKVMGuestInstance) GuestPrelaunchSetCgroup() {
	s.cgroupPid = s.GetPid()
	s.setCgroupIo()
	s.setCgroupCpu()
	s.setCgroupCPUSet()
}

func (s *SKVMGuestInstance) setCgroupPid() {
	s.cgroupPid = s.GetPid()
}

func (s *SKVMGuestInstance) setCgroupIo() {
	appTags := s.getApptags()
	params := map[string]int{}
	if utils.IsInStringArray("io_hardlimit", appTags) {
		devId := s.getStorageDeviceId()
		if len(devId) == 0 {
			log.Errorln("failed to get device ID (MAJOR:MINOR)")
			return
		}
		params["blkio.throttle.read_bps_device"] = options.HostOptions.DefaultReadBpsPerCpu
		params["blkio.throttle.read_iops_device"] = options.HostOptions.DefaultReadIopsPerCpu
		params["blkio.throttle.write_bps_device"] = options.HostOptions.DefaultWriteBpsPerCpu
		params["blkio.throttle.write_iops_device"] = options.HostOptions.DefaultWriteIopsPerCpu
		cgrouputils.CgroupIoHardlimitSet(
			strconv.Itoa(s.cgroupPid), s.GetCgroupName(), int(s.Desc.Cpu), params, devId,
		)
	}
}

func (s *SKVMGuestInstance) setCgroupCpu() {
	var (
		cpu       = s.Desc.Cpu
		cpuWeight = 1024
	)

	cgrouputils.CgroupSet(strconv.Itoa(s.cgroupPid), s.GetCgroupName(), int(cpu)*cpuWeight)
}

func (s *SKVMGuestInstance) setCgroupCPUSet() {
	var cpus []int
	if cpuset, ok := s.Desc.Metadata[api.VM_METADATA_CGROUP_CPUSET]; ok {
		cpusetJson, err := jsonutils.ParseString(cpuset)
		if err != nil {
			log.Errorf("failed parse server %s cpuset %s: %s", s.Id, cpuset, err)
			return
		}
		input := new(api.ServerCPUSetInput)
		err = cpusetJson.Unmarshal(input)
		if err != nil {
			log.Errorf("failed unmarshal server %s cpuset %s", s.Id, err)
			return
		}
		cpus = input.CPUS
	} else {
		cpus = s.allocGuestCpuset()
	}
	if _, err := s.CPUSet(context.Background(), cpus); err != nil {
		log.Errorf("Do CPUSet error: %v", err)
		return
	}
	s.Desc.VcpuPin = []desc.CpuPin{
		{
			Vcpus: fmt.Sprintf("0-%d", s.Desc.Cpu-1),
			Pcpus: cpuset.NewCPUSet(cpus...).String(),
		},
	}
	SaveLiveDesc(s, s.Desc)
}

func (s *SKVMGuestInstance) allocGuestCpuset() []int {
	var cpuset = []int{}
	numaCpus := s.manager.cpuSet.AllocCpuset(int(s.Desc.Cpu))
	for _, cpus := range numaCpus {
		cpuset = append(cpuset, cpus...)
	}
	return cpuset
}

func (s *SKVMGuestInstance) GetNeedMergeBackingFileDiskIndexs() []int {
	res := make([]int, 0)
	for _, disk := range s.Desc.Disks {
		if disk.MergeSnapshot {
			res = append(res, int(disk.Index))
		}
	}
	return res
}

func (s *SKVMGuestInstance) streamDisksComplete(ctx context.Context) {
	disks := s.Desc.Disks
	for i, _ := range disks {
		d, _ := storageman.GetManager().GetDiskByPath(disks[i].Path)
		if d != nil {
			log.Infof("Disk %s do post create from fuse", d.GetId())
			d.PostCreateFromImageFuse()
		}
		if disks[i].MergeSnapshot {
			disks[i].MergeSnapshot = false
			s.needSyncStreamDisks = true
		}
	}
	if err := SaveLiveDesc(s, s.Desc); err != nil {
		log.Errorf("save guest desc failed %s", err)
	}
	if err := s.delFlatFiles(ctx); err != nil {
		log.Errorf("del flat files failed %s", err)
	}
	go s.sendStreamDisksComplete(ctx)
}

func (s *SKVMGuestInstance) sendStreamDisksComplete(ctx context.Context) {
	for {
		_, err := modules.Servers.PerformAction(hostutils.GetComputeSession(ctx),
			s.Id, "stream-disks-complete", nil)
		if err != nil {
			log.Errorf("stream disks complete sync error %s", err)
			time.Sleep(30 * time.Second)
			continue
		} else {
			break
		}
	}

	s.needSyncStreamDisks = false
	if err := SaveLiveDesc(s, s.Desc); err != nil {
		log.Errorf("save guest desc failed %s", err)
	}
}

func (s *SKVMGuestInstance) GetQemuVersionStr() string {
	return s.QemuVersion
}

func (s *SKVMGuestInstance) optimizeOom() error {
	pid := s.GetPid()
	if pid > 0 {
		return fileutils2.FilePutContents(fmt.Sprintf("/proc/%d/oom_adj", pid), "-17", false)
	}
	return fmt.Errorf("Guest %s not running?", s.GetId())
}

func (s *SKVMGuestInstance) SyncMetadata(meta *jsonutils.JSONDict) error {
	if s.Desc.Metadata == nil {
		s.Desc.Metadata = make(map[string]string)
	}
	meta.Unmarshal(&s.Desc.Metadata)
	_, err := modules.Servers.SetMetadata(hostutils.GetComputeSession(context.Background()),
		s.Id, meta)
	if err != nil {
		log.Errorf("sync metadata error: %v", err)
		return errors.Wrap(err, "set metadata")
	}
	return nil
}

func (s *SKVMGuestInstance) updateChildIndex() error {
	idx := s.getQuorumChildIndex() + 1
	s.Desc.Metadata[api.QUORUM_CHILD_INDEX] = strconv.Itoa(int(idx))
	SaveLiveDesc(s, s.Desc)
	meta := jsonutils.NewDict()
	meta.Set(api.QUORUM_CHILD_INDEX, jsonutils.NewInt(idx))
	return s.SyncMetadata(meta)
}

func (s *SKVMGuestInstance) SetVncPassword() {
	password := seclib.RandomPassword(8)
	s.VncPassword = password
	var callback = func(res string) {
		if len(res) > 0 {
			log.Errorf("Set vnc password failed: %s", res)
		}
	}
	timeutils2.AddTimeout(time.Second*3,
		func() { s.Monitor.SetVncPassword(s.GetVdiProtocol(), password, callback) })
}

func (s *SKVMGuestInstance) OnResumeSyncMetadataInfo() {
	meta := jsonutils.NewDict()
	meta.Set("__qemu_version", jsonutils.NewString(s.GetQemuVersionStr()))
	meta.Set("__vnc_port", jsonutils.NewInt(int64(s.GetVncPort())))
	meta.Set("__enable_cgroup_cpuset", jsonutils.JSONTrue)
	meta.Set(api.VM_METADATA_HOTPLUG_CPU_MEM, jsonutils.NewString("enable"))
	meta.Set(api.VM_METADATA_HOT_REMOVE_NIC, jsonutils.NewString("enable"))
	if len(s.VncPassword) > 0 {
		meta.Set("__vnc_password", jsonutils.NewString(s.VncPassword))
	}
	if s.manager.host.IsHugepagesEnabled() {
		meta.Set("__hugepage", jsonutils.NewString("native"))
	}
	// not exactly
	if !options.HostOptions.HostCpuPassthrough || s.GetOsName() == OS_NAME_MACOS {
		meta.Set("__cpu_mode", jsonutils.NewString(api.CPU_MODE_QEMU))
	} else {
		meta.Set("__cpu_mode", jsonutils.NewString(api.CPU_MODE_HOST))
	}
	if s.hasPcieExtendBus() {
		meta.Set("__pcie_extend_bus", jsonutils.JSONTrue)
	}
	cmdline, err := s.getQemuCmdline()
	if err != nil {
		log.Errorf("Get qemu process cmdline error: %v", err)
	} else {
		meta.Set("__qemu_cmdline", jsonutils.NewString(cmdline))
	}
	if s.syncMeta != nil {
		meta.Update(s.syncMeta)
	}
	s.SyncMetadata(meta)
}

func (s *SKVMGuestInstance) SyncQemuCmdline(cmdline string) {
	meta := jsonutils.NewDict()
	meta.Set("__qemu_cmdline", jsonutils.NewString(cmdline))
	s.SyncMetadata(meta)
}

func (s *SKVMGuestInstance) doBlockIoThrottle() {
	disks := s.Desc.Disks
	if len(disks) > 0 {
		s.BlockIoThrottle(context.Background())
	}
}

func (s *SKVMGuestInstance) onGuestPrelaunch() error {
	s.LiveMigrateDestPort = nil
	if !s.Desc.IsSlave {
		if options.HostOptions.SetVncPassword {
			s.SetVncPassword()
		}
		if s.isMemcleanEnabled() {
			if err := s.startMemCleaner(); err != nil {
				return err
			}
		}
		s.OnResumeSyncMetadataInfo()
		s.GuestPrelaunchSetCgroup()
		s.optimizeOom()
		s.doBlockIoThrottle()
	}
	return nil
}

func (s *SKVMGuestInstance) CleanImportMetadata() *jsonutils.JSONDict {
	meta := jsonutils.NewDict()
	if len(s.Desc.Metadata["__origin_id"]) > 0 {
		meta.Set("__origin_id", jsonutils.NewString(""))
		delete(s.Desc.Metadata, "__origin_id")
	}
	if len(s.Desc.Metadata["__monitor_path"]) > 0 {
		meta.Set("__monitor_path", jsonutils.NewString(""))
		delete(s.Desc.Metadata, "__monitor_path")
	}

	if meta.Length() > 0 {
		// update local metadata record, after monitor started updata region record
		SaveLiveDesc(s, s.Desc)
		return meta
	}
	return nil
}

func (s *SKVMGuestInstance) ListStateFilePaths() []string {
	var ret = []string{}
	if fileutils2.Exists(s.HomeDir()) {
		files, err := ioutil.ReadDir(s.HomeDir())
		if err != nil {
			log.Errorln(err)
			return nil
		}
		for _, f := range files {
			if strings.HasPrefix(f.Name(), STATE_FILE_PREFIX) {
				ret = append(ret, path.Join(s.HomeDir(), f.Name()))
			}
		}
	}
	return ret
}

func (s *SKVMGuestInstance) CleanStatefiles() {
	for _, stateFile := range s.ListStateFilePaths() {
		log.Infof("Server %s remove statefile %q", s.GetName(), stateFile)
		if _, err := procutils.NewCommand("mountpoint", stateFile).Output(); err == nil {
			if output, err := procutils.NewCommand("umount", stateFile).Output(); err != nil {
				log.Errorf("umount %s failed: %s, %s", stateFile, err, output)
			}
		}
		if output, err := procutils.NewCommand("rm", "-rf", stateFile).Output(); err != nil {
			log.Errorf("failed rm %s: %s, %s", stateFile, err, output)
		}
	}
	if output, err := procutils.NewCommand("rm", "-rf", s.GetFuseTmpPath()).Output(); err != nil {
		log.Errorf("failed rm %s: %s, %s", s.GetFuseTmpPath(), err, output)
	}
}

func (s *SKVMGuestInstance) GetFuseTmpPath() string {
	return path.Join(s.HomeDir(), "tmp")
}

func (s *SKVMGuestInstance) StreamDisks(ctx context.Context, callback func(), disksIdx []int) {
	log.Infof("Start guest block stream task %s ...", s.GetName())
	task := NewGuestStreamDisksTask(ctx, s, callback, disksIdx)
	task.Start()
}

func (s *SKVMGuestInstance) isLiveSnapshotEnabled() bool {
	if version.GE(s.QemuVersion, "2.12.1") {
		return true
	} else {
		return false
	}
}

func (s *SKVMGuestInstance) ExecReloadDiskTask(ctx context.Context, disk storageman.IDisk) (jsonutils.JSONObject, error) {
	if s.IsRunning() {
		if s.isLiveSnapshotEnabled() {
			task := NewGuestReloadDiskTask(ctx, s, disk)
			return nil, task.WaitSnapshotReplaced(task.Start)
		} else {
			return nil, fmt.Errorf("Guest dosen't support reload disk")
		}
	} else {
		res := jsonutils.NewDict()
		res.Set("reopen", jsonutils.JSONTrue)
		return res, nil
	}
}

func (s *SKVMGuestInstance) ExecDiskSnapshotTask(
	ctx context.Context, userCred mcclient.TokenCredential, disk storageman.IDisk, snapshotId string,
) (jsonutils.JSONObject, error) {
	var (
		encryptKey = ""
		encFormat  qemuimg.TEncryptFormat
		encAlg     seclib2.TSymEncAlg
	)
	if s.isEncrypted() {
		key, err := s.getEncryptKey(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "getEncryptKey")
		}
		encryptKey = key.Key
		encFormat = qemuimg.EncryptFormatLuks
		encAlg = key.Alg
	}
	if s.IsRunning() {
		if !s.isLiveSnapshotEnabled() {
			return nil, fmt.Errorf("Guest dosen't support live snapshot")
		}
		err := disk.CreateSnapshot(snapshotId, encryptKey, encFormat, encAlg)
		if err != nil {
			return nil, errors.Wrap(err, "disk.CreateSnapshot")
		}
		task := NewGuestDiskSnapshotTask(ctx, s, disk, snapshotId)
		task.Start()
		return nil, nil
	} else {
		return s.StaticSaveSnapshot(ctx, disk, snapshotId, encryptKey, encFormat, encAlg)
	}
}

func (s *SKVMGuestInstance) StaticSaveSnapshot(
	ctx context.Context, disk storageman.IDisk, snapshotId string,
	encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg,
) (jsonutils.JSONObject, error) {
	err := disk.CreateSnapshot(snapshotId, encryptKey, encFormat, encAlg)
	if err != nil {
		return nil, errors.Wrap(err, "disk.CreateSnapshot")
	}
	location := path.Join(disk.GetSnapshotLocation(), snapshotId)
	res := jsonutils.NewDict()
	res.Set("location", jsonutils.NewString(location))
	return res, nil
}

func (s *SKVMGuestInstance) ExecDeleteSnapshotTask(
	ctx context.Context, disk storageman.IDisk,
	deleteSnapshot string, convertSnapshot string, pendingDelete bool,
) (jsonutils.JSONObject, error) {
	if s.IsRunning() {
		if s.isLiveSnapshotEnabled() {
			task := NewGuestSnapshotDeleteTask(ctx, s, disk,
				deleteSnapshot, convertSnapshot, pendingDelete)
			task.Start()
			return nil, nil
		} else {
			return nil, fmt.Errorf("Guest dosen't support live snapshot delete")
		}
	} else {
		return s.deleteStaticSnapshotFile(ctx, disk, deleteSnapshot,
			convertSnapshot, pendingDelete)
	}
}

func (s *SKVMGuestInstance) deleteStaticSnapshotFile(
	ctx context.Context, disk storageman.IDisk,
	deleteSnapshot string, convertSnapshot string, pendingDelete bool,
) (jsonutils.JSONObject, error) {
	if err := disk.DeleteSnapshot(deleteSnapshot, convertSnapshot, pendingDelete); err != nil {
		log.Errorln(err)
		return nil, err
	}
	res := jsonutils.NewDict()
	res.Set("deleted", jsonutils.JSONTrue)
	return res, nil
}

func GetMemorySnapshotPath(serverId, instanceSnapshotId string) string {
	dir := options.HostOptions.MemorySnapshotsPath
	memSnapPath := filepath.Join(dir, serverId, instanceSnapshotId)
	return memSnapPath
}

func (s *SKVMGuestInstance) ExecMemorySnapshotTask(ctx context.Context, input *hostapi.GuestMemorySnapshotRequest) (jsonutils.JSONObject, error) {
	if !s.IsRunning() {
		return nil, errors.Errorf("Server is not running status")
	}
	if s.IsSuspend() {
		return nil, errors.Errorf("Server is suspend status")
	}
	memSnapPath := GetMemorySnapshotPath(s.GetId(), input.InstanceSnapshotId)
	dir := filepath.Dir(memSnapPath)
	if err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Run(); err != nil {
		return nil, errors.Wrapf(err, "mkdir -p %q", dir)
	}
	NewGuestSuspendTask(s, ctx, func(_ *SGuestSuspendTask, memStatPath string) {
		log.Infof("Memory state file %q saved, move it to %q", memStatPath, memSnapPath)
		sizeBytes := fileutils2.FileSize(memStatPath)
		sizeMB := sizeBytes / 1024
		checksum, err := fileutils2.FastCheckSum(memStatPath)
		if err != nil {
			hostutils.TaskFailed(ctx, fmt.Sprintf("calculate statefile %q checksum: %v", memStatPath, err))
			return
		}
		if err := procutils.NewRemoteCommandAsFarAsPossible("mv", memStatPath, memSnapPath).Run(); err != nil {
			hostutils.TaskFailed(ctx, fmt.Sprintf("move statefile %q to memory snapshot %q: %v", memStatPath, memSnapPath, err))
			return
		}
		resumeTask := NewGuestResumeTask(ctx, s, false, false)
		resumeTask.SetGetTaskData(func() (jsonutils.JSONObject, error) {
			resp := &hostapi.GuestMemorySnapshotResponse{
				MemorySnapshotPath: memSnapPath,
				SizeMB:             sizeMB,
				Checksum:           checksum,
			}
			return jsonutils.Marshal(resp), nil
		})
		resumeTask.Start()
	}).Start()
	return nil, nil
}

func (s *SKVMGuestInstance) ExecMemorySnapshotResetTask(ctx context.Context, input *hostapi.GuestMemorySnapshotResetRequest) (jsonutils.JSONObject, error) {
	if !s.IsStopped() {
		return nil, errors.Errorf("Server is not stopped status")
	}
	memStatPath := s.GetStateFilePath("")
	if err := procutils.NewRemoteCommandAsFarAsPossible("ln", "-s", input.Path, memStatPath).Run(); err != nil {
		hostutils.TaskFailed(ctx, fmt.Sprintf("move %q to %q: %v", input.Path, memStatPath, err))
		return nil, err
	}
	if input.Checksum != "" {
		handleErr := func(msg string) error {
			// remove linked snapshot path
			if err := procutils.NewRemoteCommandAsFarAsPossible("unlink", memStatPath).Run(); err != nil {
				msg = errors.Wrapf(err, "unlink statfile cause %s", msg).Error()
			}
			hostutils.TaskFailed(ctx, msg)
			return errors.Error(msg)
		}
		checksum, err := fileutils2.FastCheckSum(memStatPath)
		if err != nil {
			return nil, handleErr(fmt.Sprintf("calculate statefile %s checksum: %v", memStatPath, err))
		}
		if checksum != input.Checksum {
			data := jsonutils.NewDict()
			data.Set("name", jsonutils.NewString(input.InstanceSnapshotId))
			notifyclient.SystemExceptionNotifyWithResult(context.Background(), noapi.ActionChecksumTest, noapi.TOPIC_RESOURCE_SNAPSHOT, noapi.ResultFailed, data)
			return nil, handleErr(fmt.Sprintf("calculate checksum %s != %s", checksum, input.Checksum))
		}
	}
	hostutils.TaskComplete(ctx, nil)
	return nil, nil
}

func (s *SKVMGuestInstance) PrepareDisksMigrate(liveMigrage bool) (*jsonutils.JSONDict, *jsonutils.JSONDict, bool, error) {
	disksBackFile := jsonutils.NewDict()
	diskSnapsChain := jsonutils.NewDict()
	sysDiskHasTemplate := false
	for _, disk := range s.Desc.Disks {
		if disk.Path != "" {
			d, err := storageman.GetManager().GetDiskByPath(disk.Path)
			if err != nil {
				return nil, nil, false, errors.Wrapf(err, "GetDiskByPath(%s)", disk.Path)
			}
			if d.GetType() == api.STORAGE_LOCAL {
				snaps, back, hasTemplate, err := d.PrepareMigrate(liveMigrage)
				if err != nil {
					return nil, nil, false, err
				}
				if len(back) > 0 {
					disksBackFile.Set(disk.DiskId, jsonutils.NewString(back))
				}
				if len(snaps) > 0 {
					diskSnapsChain.Set(disk.DiskId, jsonutils.NewStringArray(snaps))
				}
				if hasTemplate {
					sysDiskHasTemplate = hasTemplate
				}
			}
		}
	}
	return disksBackFile, diskSnapsChain, sysDiskHasTemplate, nil
}

func (s *SKVMGuestInstance) prepareNicsForVolatileGuestResume() error {
	for _, nic := range s.Desc.Nics {
		bridge := nic.Bridge
		dev := s.manager.GetHost().GetBridgeDev(bridge)
		if dev == nil {
			return fmt.Errorf("Can't find bridge %s", bridge)
		}
		if err := dev.OnVolatileGuestResume(nic); err != nil {
			return errors.Wrap(err, "dev.OnVolatileGuestResume")
		}
	}
	return nil
}

func (s *SKVMGuestInstance) onlineResizeDisk(ctx context.Context, diskId string, sizeMB int64) {
	task := NewGuestOnlineResizeDiskTask(ctx, s, diskId, sizeMB)
	task.Start()
}

func (s *SKVMGuestInstance) BlockIoThrottle(ctx context.Context) {
	task := SGuestBlockIoThrottleTask{s, ctx}
	task.Start()
}

func (s *SKVMGuestInstance) IsSharedStorage() bool {
	disks := s.Desc.Disks
	for i := 0; i < len(disks); i++ {
		disk, err := storageman.GetManager().GetDiskByPath(disks[i].Path)
		if err != nil {
			log.Errorf("failed find disk by path %s", disks[i].Path)
			return false
		}
		if !utils.IsInStringArray(disk.GetType(), api.SHARED_STORAGE) {
			return false
		}
	}
	return true
}

func (s *SKVMGuestInstance) generateDiskSetupScripts(disks []*desc.SGuestDisk) (string, error) {
	cmd := " "
	for i := range disks {
		diskPath := disks[i].Path
		d, err := storageman.GetManager().GetDiskByPath(diskPath)
		if err != nil {
			return "", errors.Wrapf(err, "GetDiskByPath(%s)", diskPath)
		}
		if len(disks[i].StorageType) == 0 {
			disks[i].StorageType = d.GetType()
		}
		diskIndex := disks[i].Index
		cmd += d.GetDiskSetupScripts(int(diskIndex))
	}
	return cmd, nil
}

func (s *SKVMGuestInstance) GetSriovDeviceByNetworkIndex(networkIndex int8) (isolated_device.IDevice, error) {
	manager := s.manager.GetHost().GetIsolatedDeviceManager()
	for i := 0; i < len(s.Desc.IsolatedDevices); i++ {
		if s.Desc.IsolatedDevices[i].DevType == api.NIC_TYPE &&
			s.Desc.IsolatedDevices[i].NetworkIndex == networkIndex {
			if dev := manager.GetDeviceByAddr(s.Desc.IsolatedDevices[i].Addr); dev == nil {
				return nil, errors.Errorf("get device by addr failed")
			} else {
				return dev, nil
			}
		}
	}
	return nil, errors.Errorf("network index %d no sriov device found", networkIndex)
}

var sriovInitFunc = `
function sriov_vf_init() {
	ip link set dev $1 vf $2 mac $3 vlan $4
	ip link set dev $1 vf $2 spoofchk $5
	ip link set dev $1 vf $2 max_tx_rate $6
}`

func srcMacCheckFunc(srcMacCheck bool) string {
	if srcMacCheck {
		return "on"
	}
	return "off"
}

func getVfVlan(vlan int) int {
	if vlan == 1 {
		return 0
	}
	return vlan
}

func getIbNodeMac(mac string) string {
	return "00:01:" + mac
}

func getIbPortMac(mac string) string {
	return "00:10:" + mac
}

func (s *SKVMGuestInstance) sriovNicAttachInitScript(networkIndex int8, dev isolated_device.IDevice) (string, error) {
	for i := range s.Desc.Nics {
		if s.Desc.Nics[i].Driver == "vfio-pci" && s.Desc.Nics[i].Index == networkIndex {
			cmd := s.generateSriovInitCmd(i, dev)
			return sriovInitFunc + " && " + cmd, nil
		}
	}
	return "", errors.Errorf("no nic found for index %d", networkIndex)
}

func (s *SKVMGuestInstance) generateSRIOVInitScripts() (string, error) {
	var cmd = ""

	for i := range s.Desc.Nics {
		if s.Desc.Nics[i].Driver == "vfio-pci" {
			dev, err := s.GetSriovDeviceByNetworkIndex(s.Desc.Nics[i].Index)
			if err != nil {
				return "", err
			}
			cmd += s.generateSriovInitCmd(i, dev)
		}
	}
	if len(cmd) > 0 {
		cmd = sriovInitFunc + "\n" + cmd
	}
	return cmd, nil
}

func (s *SKVMGuestInstance) generateSriovInitCmd(i int, dev isolated_device.IDevice) string {
	var cmd = ""
	if dev.GetOvsOffloadInterfaceName() != "" {
		cmd += fmt.Sprintf("ip link set dev %s vf %d mac %s max_tx_rate %d\n",
			dev.GetPfName(), dev.GetVirtfn(), s.Desc.Nics[i].Mac, s.Desc.Nics[i].Bw)
		cmd += s.getNicUpScriptPath(s.Desc.Nics[i]) + "\n"
	} else if dev.IsInfinibandNic() {
		sriovPath := path.Join("/sys/class/net", dev.GetPfName(), "device/sriov", strconv.Itoa(dev.GetVirtfn()))
		cmd = fmt.Sprintf("echo Follow > %s/policy\n", sriovPath)
		cmd += fmt.Sprintf("echo %s > %s/node\n", getIbNodeMac(s.Desc.Nics[i].Mac), sriovPath)
		cmd += fmt.Sprintf("echo %s > %s/port\n", getIbPortMac(s.Desc.Nics[i].Mac), sriovPath)
	} else {
		cmd += fmt.Sprintf(
			"sriov_vf_init %s %d %s %d %s %d\n",
			dev.GetPfName(), dev.GetVirtfn(), s.Desc.Nics[i].Mac,
			getVfVlan(s.Desc.Nics[i].Vlan), srcMacCheckFunc(s.Desc.SrcMacCheck), s.Desc.Nics[i].Bw,
		)
	}
	return cmd
}

func (s *SKVMGuestInstance) reconfigureVfioNicsBandwidth(nicDesc *desc.SGuestNetwork) error {
	if nicDesc.Driver == "vfio-pci" {
		dev, err := s.GetSriovDeviceByNetworkIndex(nicDesc.Index)
		if err != nil {
			return errors.Wrap(err, "reconfigureVfioNicsBandwidth")
		}
		out, err := procutils.NewRemoteCommandAsFarAsPossible("ip", "link", "set", "dev", dev.GetPfName(),
			"vf", strconv.Itoa(dev.GetVirtfn()), "max_tx_rate", strconv.Itoa(nicDesc.Bw)).Output()
		if err != nil {
			return errors.Wrapf(err, "reconfigureVfioNicsBandwidth set max_tx_rate %s", out)
		}
	}
	return nil
}

func (s *SKVMGuestInstance) getQemuCmdline() (string, error) {
	content, err := fileutils2.FileGetContents(s.GetStartScriptPath())
	if err != nil {
		return "", errors.Wrap(err, "get startvm content")
	}
	return s.getQemuCmdlineFromContent(content)
}

func (s *SKVMGuestInstance) getQemuCmdlineFromContent(content string) (string, error) {
	cmdReg := regexp.MustCompile(`CMD="(?P<cmd>.*)"`)
	cmdStr := regutils2.GetParams(cmdReg, content)["cmd"]
	if cmdStr == "" {
		return "", errors.Errorf("Not found CMD content")
	}
	return cmdStr, nil
}

func (s *SKVMGuestInstance) CPUSet(ctx context.Context, input []int) (*api.ServerCPUSetResp, error) {
	if !s.IsRunning() {
		return nil, nil
	}

	var cpusetStr string
	if input != nil {
		cpus := []string{}
		for _, id := range input {
			cpus = append(cpus, fmt.Sprintf("%d", id))
		}
		cpusetStr = strings.Join(cpus, ",")
	}

	task := cgrouputils.NewCGroupCPUSetTask(
		strconv.Itoa(s.GetPid()), s.GetCgroupName(), 0, cpusetStr,
	)
	if !task.SetTask() {
		return nil, errors.Errorf("Cgroup cpuset task failed")
	}
	return new(api.ServerCPUSetResp), nil
}

func (s *SKVMGuestInstance) CPUSetRemove(ctx context.Context) error {
	delete(s.Desc.Metadata, api.VM_METADATA_CGROUP_CPUSET)
	if err := SaveLiveDesc(s, s.Desc); err != nil {
		return errors.Wrap(err, "save desc after update metadata")
	}
	if !s.IsRunning() {
		return nil
	}
	task := cgrouputils.NewCGroupCPUSetTask(
		strconv.Itoa(s.GetPid()), s.GetCgroupName(), 0, "",
	)
	if !task.RemoveTask() {
		return errors.Errorf("Remove task error happened, please lookup host log")
	}
	return nil
}

func (s *SKVMGuestInstance) clearRescue(ctx context.Context) error {
	rescueDir := s.GetRescueDirPath()
	if err := fileutils2.Cleandir(rescueDir, false); err != nil {
		return errors.Wrap(err, "clear rescue dir failed")
	}

	return nil
}

func (s *SKVMGuestInstance) prepareRescue(ctx context.Context, baremetalManagerUri string) error {
	files := []string{
		api.GUEST_RESCUE_INITRAMFS,
		api.GUEST_RESCUE_KERNEL,
	}

	// Support arm64
	if s.manager.host.IsAarch64() {
		files = []string{
			api.GUEST_RESCUE_INITRAMFS_ARM64,
			api.GUEST_RESCUE_KERNEL_ARM64,
		}
	}

	// Prepare files
	for _, file := range files {
		err := s.downloadFromBaremetal(file, baremetalManagerUri)
		if err != nil {
			return errors.Wrapf(err, "download %s from baremetal failed", file)
		}
	}

	// Create disk
	diskPath := path.Join(s.GetRescueDirPath(), api.GUEST_RESCUE_SYS_DISK_NAME)
	err := s.createTempDisk(diskPath, api.GUEST_RESCUE_SYS_DISK_SIZE, qemuimgfmt.QCOW2.String(), nil, "")
	if err != nil {
		return errors.Wrapf(err, "create disk %s failed", diskPath)
	}

	return nil
}

func (s *SKVMGuestInstance) downloadFromBaremetal(filename, baremetalManagerUri string) error {
	rescueDir, err := s.CreateRescueDirPath()
	if err != nil {
		return errors.Wrap(err, "SKVMGuestInstance.GetRescueDirPath")
	}

	// Check file is exist
	filePath := path.Join(rescueDir, filename)
	if fileutils2.Exists(filePath) {
		// File already exist
		log.Debugf("File %s already exist", filePath)

		return nil
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return errors.Wrapf(err, "create file %s failed", filePath)
	}
	defer file.Close()

	// Get filepath
	fileURL, err := s.getTftpFileUrl(filename, baremetalManagerUri)
	if err != nil {
		return errors.Wrapf(err, "getTftpFileUrl")
	}

	// Request for file
	resp, err := httputils.Request(
		httputils.GetDefaultClient(),
		context.Background(),
		"GET",
		fileURL,
		nil,
		nil,
		false)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err, "request %s failed", fileURL)
	}

	// Write file
	if _, err := io.Copy(file, resp.Body); err != nil {
		return errors.Wrapf(err, "write file %s failed", filePath)
	}

	return nil
}

func (s *SKVMGuestInstance) createTempDisk(path string, sizeMB int, diskFormat string, encryptInfo *apis.SEncryptInfo, back string) error {
	if fileutils2.Exists(path) {
		os.Remove(path)
	}

	img, err := qemuimg.NewQemuImage(path)
	if err != nil {
		log.Errorln(err)
		return err
	}

	switch diskFormat {
	case "qcow2":
		if encryptInfo != nil {
			err = img.CreateQcow2(sizeMB, false, back, encryptInfo.Key, qemuimg.EncryptFormatLuks, encryptInfo.Alg)
		} else {
			err = img.CreateQcow2(sizeMB, false, back, "", "", "")
		}
	case "vmdk":
		err = img.CreateVmdk(sizeMB, false)
	default:
		err = img.CreateRaw(sizeMB)
	}
	if err != nil {
		return errors.Wrapf(err, "create_raw: Fail to create disk")
	}

	return nil
}

func (s *SKVMGuestInstance) getTftpFileUrl(filename, baremetalManagerUri string) (string, error) {
	endpoint, err := s.getTftpEndpoint(baremetalManagerUri)
	if err != nil {
		log.Errorf("Get http file server endpoint: %v", err)
		return filename, err
	}
	return fmt.Sprintf("http://%s/tftp/%s", endpoint, filename), nil
}

func (s *SKVMGuestInstance) getTftpEndpoint(baremetalManagerUri string) (string, error) {
	// Split with :
	addrs := strings.Split(baremetalManagerUri, "//")
	if len(addrs) < 2 {
		return "", errors.Errorf("baremetal manager uri is invalid")
	}
	endpoints := strings.Split(addrs[1], ":")
	if len(endpoints) < 2 {
		return "", errors.Errorf("baremetal manager uri is invalid")
	}

	// Plus baremetal agent port with 1000
	port, err := strconv.Atoi(endpoints[1])
	if err != nil {
		return "", errors.Wrapf(err, "convert port failed")
	}

	// Concat new endpoint url
	endpoint := fmt.Sprintf("%s:%d", endpoints[0], port+1000)

	return endpoint, nil
}

func (s *SKVMGuestInstance) HandleGuestStatus(ctx context.Context, status string, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	if status == GUEST_RUNNING && s.pciUninitialized {
		status = api.VM_UNSYNC
	} else if status == GUEST_RUNNING {
		var runCb = func() {
			body := jsonutils.NewDict()
			blockJobsCount := s.BlockJobsCount()
			if blockJobsCount > 0 {
				status = GUEST_BLOCK_STREAM
			}
			body.Set("block_jobs_count", jsonutils.NewInt(int64(blockJobsCount)))
			body.Set("status", jsonutils.NewString(status))
			hostutils.TaskComplete(ctx, body)
		}
		if s.Monitor == nil && !s.IsStopping() {
			if err := s.StartMonitor(context.Background(), runCb); err != nil {
				log.Errorf("guest %s failed start monitor %s", s.GetName(), err)
				body.Set("status", jsonutils.NewString(status))
				hostutils.TaskComplete(ctx, body)
			}
		} else {
			runCb()
		}
		return nil, nil
	}
	body.Set("status", jsonutils.NewString(status))
	hostutils.TaskComplete(ctx, body)
	return nil, nil
}
