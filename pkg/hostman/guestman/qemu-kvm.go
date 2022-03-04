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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/appctx"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
	"yunion.io/x/onecloud/pkg/util/version"
)

const (
	STATE_FILE_PREFIX             = "STATEFILE"
	MONITOR_PORT_BASE             = 55900
	LIVE_MIGRATE_PORT_BASE        = 4396
	BUILT_IN_NBD_SERVER_PORT_BASE = 7777
	MAX_TRY                       = 3
)

type SKVMGuestInstance struct {
	Id string

	cgroupPid   int
	QemuVersion string
	VncPassword string

	Desc        *jsonutils.JSONDict
	Monitor     monitor.Monitor
	manager     *SGuestManager
	startupTask *SGuestResumeTask
	migrateTask *SGuestLiveMigrateTask
	stopping    bool
	syncMeta    *jsonutils.JSONDict
}

func NewKVMGuestInstance(id string, manager *SGuestManager) *SKVMGuestInstance {
	return &SKVMGuestInstance{
		Id:      id,
		manager: manager,
	}
}

func (s *SKVMGuestInstance) IsStopping() bool {
	return s.stopping
}

func (s *SKVMGuestInstance) GetId() string {
	id, _ := s.Desc.GetString("uuid")
	return id
}

func (s *SKVMGuestInstance) GetName() string {
	id, _ := s.Desc.GetString("uuid")
	name, _ := s.Desc.GetString("name")
	return fmt.Sprintf("%s(%s)", name, id)
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

func (s *SKVMGuestInstance) IsLoaded() bool {
	return s.Desc != nil
}

func (s *SKVMGuestInstance) HomeDir() string {
	return path.Join(s.manager.ServersPath, s.Id)
}

func (s *SKVMGuestInstance) PrepareDir() error {
	output, err := procutils.NewCommand("mkdir", "-p", s.HomeDir()).Output()
	if err != nil {
		return errors.Wrapf(err, "mkdir %s failed: %s", s.HomeDir(), output)
	}
	return nil
}

func (s *SKVMGuestInstance) GetPidFilePath() string {
	return path.Join(s.HomeDir(), "pid")
}

func (s *SKVMGuestInstance) GetVncFilePath() string {
	return path.Join(s.HomeDir(), "vnc")
}

func (s *SKVMGuestInstance) getOriginId() string {
	originId, _ := s.Desc.GetString("metadata", "__origin_id")
	if len(originId) == 0 {
		originId = s.Id
	}
	return originId
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

func (s *SKVMGuestInstance) GetDescFilePath() string {
	return path.Join(s.HomeDir(), "desc")
}

func (s *SKVMGuestInstance) LoadDesc() error {
	descPath := s.GetDescFilePath()
	descStr, err := ioutil.ReadFile(descPath)
	if err != nil {
		return err
	}
	desc, err := jsonutils.Parse(descStr)
	if err != nil {
		return err
	}
	dDesc, ok := desc.(*jsonutils.JSONDict)
	if !ok {
		return fmt.Errorf("Load and parse desc error")
	}
	s.Desc = dDesc
	return nil
}

func (s *SKVMGuestInstance) IsDirtyShotdown() bool {
	return s.GetPid() == -2
}

func (s *SKVMGuestInstance) DirtyServerRequestStart() {
	hostId, _ := s.Desc.GetString("host_id")
	var body = jsonutils.NewDict()
	body.Set("guest_id", jsonutils.NewString(s.Id))
	body.Set("host_id", jsonutils.NewString(hostId))
	_, err := modules.Servers.PerformClassAction(
		hostutils.GetComputeSession(context.Background()), "dirty-server-start", body)
	if err != nil {
		log.Errorf("Dirty server request start error %s: %s", s.GetName(), err)
	}
}

func (s *SKVMGuestInstance) asyncScriptStart(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	hostbridge.CleanDeletedPorts(options.HostOptions.BridgeDriver)

	time.Sleep(100 * time.Millisecond)
	var isStarted, tried = false, 0
	var err error
	for !isStarted && tried < MAX_TRY {
		tried += 1

		vncPort := s.manager.GetFreeVncPort()
		if err = s.saveVncPort(vncPort); err != nil {
			goto finally
		} else {
			data.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
		}

		if err = s.saveScripts(data); err != nil {
			goto finally
		} else {
			err = s.scriptStart()
			if err == nil {
				isStarted = true
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
		s.StartMonitor(ctx, nil)
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
	// diff qemu command options when migrating
	if jsonutils.QueryBoolean(data, "need_migrate", false) {
		srcCmdline, err := data.GetString("source_qemu_cmdline")
		if err != nil {
			return errors.Wrap(err, "Get source_qemu_cmdline")
		}
		currentCmd, err := s.getQemuCmdlineFromContent(startScript)
		if err != nil {
			return errors.Wrapf(err, "Get qemu cmdline from %q", startScript)
		}
		unifyCmd, err := s.unifyMigrateQemuCmdline(currentCmd, srcCmdline)
		if err != nil {
			return errors.Wrap(err, "Unify migrate qemu cmdline")
		}
		// reinject cmd
		startScript = strings.ReplaceAll(startScript, currentCmd, unifyCmd)
	}

	if err = fileutils2.FilePutContents(s.GetStartScriptPath(), startScript, false); err != nil {
		return err
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
	hostId, _ := s.Desc.GetString("host_id")
	if hostId != hostinfo.Instance().HostId {
		// fix host_id
		s.Desc.Set("host_id", jsonutils.NewString(hostinfo.Instance().HostId))
		s.SaveDesc(s.Desc)
	}

	s.manager.SaveServer(s.Id, s)
	s.manager.RemoveCandidateServer(s)

	if s.IsDirtyShotdown() && !pendingDelete {
		log.Infof("Server dirty shotdown %s", s.GetName())
		if jsonutils.QueryBoolean(s.Desc, "is_master", false) ||
			jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
			go s.DirtyServerRequestStart()
		} else {
			s.StartGuest(context.Background(), jsonutils.NewDict())
		}
		return
	}
	if s.IsRunning() {
		log.Infof("%s is running, pending_delete=%t", s.GetName(), pendingDelete)
		if !pendingDelete {
			s.StartMonitor(context.Background(), nil)
		}
	} else {
		var action = "stopped"
		if s.IsSuspend() {
			action = "suspend"
		}
		log.Infof("%s is %s, pending_delete=%t", s.GetName(), action, pendingDelete)
		if !s.IsSlave() {
			s.SyncStatus("")
		}
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
		meta.Set("hotplug_cpu_mem", jsonutils.NewString("disable"))
		meta.Set("hot_remove_nic", jsonutils.NewString("disable"))
		meta.Set("__qemu_version", jsonutils.NewString(s.GetQemuVersionStr()))
		s.SyncMetadata(meta)
		s.SyncStatus("")
	})
}

func (s *SKVMGuestInstance) GetMonitorPath() string {
	monitorPath, _ := s.Desc.GetString("metadata", "__monitor_path")
	return monitorPath
}

func (s *SKVMGuestInstance) StartMonitorWithImportGuestSocketFile(ctx context.Context, socketFile string, cb func()) {
	timeutils2.AddTimeout(100*time.Millisecond, func() {
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
		mon.ConnectWithSocket(socketFile)
	})
}

func (s *SKVMGuestInstance) StartMonitor(ctx context.Context, cb func()) {
	if options.HostOptions.EnableQmpMonitor && s.GetQmpMonitorPort(-1) > 0 {
		// try qmp first, if qmp connect failed, use hmp
		timeutils2.AddTimeout(100*time.Millisecond, func() {
			var mon monitor.Monitor
			mon = monitor.NewQmpMonitor(
				s.GetName(),
				s.Id,
				s.onMonitorDisConnect, // on monitor disconnect
				func(err error) { s.onMonitorTimeout(ctx, err) }, // on monitor timeout
				func() {
					s.Monitor = mon
					s.onMonitorConnected(ctx)
					if cb != nil {
						cb()
					}
				}, // on monitor connected
				s.onReceiveQMPEvent, // on reveive qmp event
			)
			err := mon.Connect("127.0.0.1", s.GetQmpMonitorPort(-1))
			if err != nil {
				log.Errorf("Guest %s qmp monitor connect failed %s, try hmp", s.GetName(), err)
				mon = monitor.NewHmpMonitor(
					s.GetName(),
					s.Id,
					s.onMonitorDisConnect, // on monitor disconnect
					func(err error) { s.onMonitorTimeout(ctx, err) }, // on monitor timeout
					func() {
						s.Monitor = mon
						s.onMonitorConnected(ctx)
						if cb != nil {
							cb()
						}
					}, // on monitor connected
				)
				err = mon.Connect("127.0.0.1", s.GetHmpMonitorPort(-1))
				if err != nil {
					mon = nil
					log.Errorf("Guest %s hmp monitor connect failed %s, something wrong", s.GetName(), err)
				}
			}
		})
	} else if monitorPath := s.GetMonitorPath(); len(monitorPath) > 0 {
		s.StartMonitorWithImportGuestSocketFile(ctx, monitorPath, cb)
	} else {
		log.Errorf("Guest start monitor failed, can't get qmp monitor port or monitor path")
	}
}

func (s *SKVMGuestInstance) onReceiveQMPEvent(event *monitor.Event) {
	switch {
	case event.Event == `"BLOCK_JOB_READY"` && s.IsMaster():
		if itype, ok := event.Data["type"]; ok {
			stype, _ := itype.(string)
			if stype == "mirror" {
				mirrorStatus := s.MirrorJobStatus()
				if mirrorStatus.IsSucc() {
					_, err := hostutils.UpdateServerStatus(context.Background(), s.GetId(), api.VM_RUNNING, "BLOCK_JOB_READY")
					if err != nil {
						log.Errorf("onReceiveQMPEvent update server status error: %s", err)
					}
				} else if mirrorStatus.IsFailed() {
					s.SyncMirrorJobFailed("Block job missing")
				}
			}
		}
	case event.Event == `"BLOCK_JOB_ERROR"`:
		s.SyncMirrorJobFailed("BLOCK_JOB_ERROR")
	case event.Event == `"GUEST_PANICKED"`:
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
		modules.Servers.PerformAction(
			hostutils.GetComputeSession(context.Background()),
			s.GetId(), "event", params)
		// case utils.IsInStringArray(event.Event, []string{`"SHUTDOWN"`, `"POWERDOWN"`, `"RESET"`}):
		// 	params := jsonutils.NewDict()
		// 	params.Set("event", jsonutils.NewString(strings.Trim(event.Event, "\"")))
		// 	modules.Servers.PerformAction(
		// 		hostutils.GetComputeSession(context.Background()),
		// 		s.GetId(), "event", params)
	case event.Event == `"STOP"`:
		if s.migrateTask != nil {
			// migrating complete
			s.migrateTask.migrateComplete()
		}
		hostutils.UpdateServerProgress(context.Background(), s.Id, 0.0, 0)
	}
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

func (s *SKVMGuestInstance) onGetQemuVersion(ctx context.Context, version string) {
	s.QemuVersion = version
	log.Infof("Guest(%s) qemu version %s", s.Id, s.QemuVersion)
	if s.Desc.Contains("live_migrate_dest_port") && ctx != nil {
		migratePort, _ := s.Desc.Get("live_migrate_dest_port")
		body := jsonutils.NewDict()
		body.Set("live_migrate_dest_port", migratePort)
		if jsonutils.QueryBoolean(s.Desc, "live_migrate_use_tls", false) {
			s.setDestMigrateTLS(ctx, body)
		} else {
			hostutils.TaskComplete(ctx, body)
		}
	} else if s.IsSlave() {
		s.startQemuBuiltInNbdServer(ctx)
	} else if s.IsMaster() {
		s.startDiskBackupMirror(ctx)
		if ctx != nil && len(appctx.AppContextTaskId(ctx)) > 0 {
			s.DoResumeTask(ctx, false)
		} else {
			if options.HostOptions.SetVncPassword {
				s.SetVncPassword()
			}
			s.OnResumeSyncMetadataInfo()
		}
	} else {
		s.DoResumeTask(ctx, true)
	}
}

func (s *SKVMGuestInstance) onMonitorDisConnect(err error) {
	log.Errorf("Guest %s on Monitor Disconnect reason: %v", s.Id, err)
	s.CleanStartupTask()
	s.scriptStop()
	if !jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
		s.SyncStatus(fmt.Sprintf("monitor disconnect %v", err))
	}
	s.clearCgroup(0)
	s.Monitor = nil
}

func (s *SKVMGuestInstance) startDiskBackupMirror(ctx context.Context) {
	if ctx == nil || len(appctx.AppContextTaskId(ctx)) == 0 {
		status := api.VM_RUNNING
		mirrorStatus := s.MirrorJobStatus()
		if mirrorStatus.InProcess() {
			status = api.VM_BLOCK_STREAM
		} else if mirrorStatus.IsFailed() {
			status = api.VM_BLOCK_STREAM_FAIL
			s.SyncMirrorJobFailed("mirror job missing")
		}
		hostutils.UpdateServerStatus(context.Background(), s.GetId(), status, "")
	} else {
		metadata, _ := s.Desc.Get("metadata")
		if metadata == nil || !metadata.Contains("backup_nbd_server_uri") {
			hostutils.TaskFailed(ctx, "Missing dest nbd location")
		}
		nbdUri, _ := metadata.GetString("backup_nbd_server_uri")

		onSucc := func() {
			cb := func(res string) { log.Infof("On backup mirror server(%s) resume start", s.Id) }
			s.Monitor.SimpleCommand("cont", cb)
		}
		NewDriveMirrorTask(ctx, s, nbdUri, "top", true, onSucc).Start()
	}
}

func (s *SKVMGuestInstance) startQemuBuiltInNbdServer(ctx context.Context) {
	if ctx != nil && len(appctx.AppContextTaskId(ctx)) > 0 {
		nbdServerPort := s.manager.GetFreePortByBase(BUILT_IN_NBD_SERVER_PORT_BASE)
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
	}
}

func (s *SKVMGuestInstance) clearCgroup(pid int) {
	if pid == 0 && s.cgroupPid > 0 {
		pid = s.cgroupPid
	}
	log.Infof("cgroup destroy %d", pid)
	if pid > 0 && !options.HostOptions.DisableSetCgroup {
		cgrouputils.CgroupDestroy(strconv.Itoa(pid))
	}
}

func (s *SKVMGuestInstance) IsMaster() bool {
	return jsonutils.QueryBoolean(s.Desc, "is_master", false)
}

func (s *SKVMGuestInstance) IsSlave() bool {
	return jsonutils.QueryBoolean(s.Desc, "is_slave", false)
}

func (s *SKVMGuestInstance) DiskCount() int {
	disks, _ := s.Desc.GetArray("disks")
	return len(disks)
}

type MirrorJob int

func (ms MirrorJob) IsSucc() bool {
	return ms == 1
}

func (ms MirrorJob) IsFailed() bool {
	return ms == -1
}

func (ms MirrorJob) InProcess() bool {
	return ms == 0
}

func (s *SKVMGuestInstance) MirrorJobStatus() MirrorJob {
	res := make(chan []monitor.BlockJob)
	s.Monitor.GetBlockJobs(func(jobs []monitor.BlockJob) {
		res <- jobs
	})
	select {
	case <-time.After(time.Second * 3):
		return 0
	case v := <-res:
		if len(v) >= s.DiskCount() {
			mirrorSuccCount := 0
			for _, job := range v {
				if job.Type == "mirror" && job.Status == "ready" {
					mirrorSuccCount += 1
				}
			}
			if mirrorSuccCount == s.DiskCount() {
				return 1
			}
			return 0
		}
		return -1
	}
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
	s.startupTask = nil
}

func (s *SKVMGuestInstance) CleanStartupTask() {
	if s.startupTask != nil {
		log.Infof("[%s] Clean startup task ... stop task ...", s.GetId())
		s.startupTask.Stop()
		s.startupTask = nil
	} else {
		log.Infof("[%s] Clean startup task ... no task", s.GetId())
	}
}

func (s *SKVMGuestInstance) onMonitorTimeout(ctx context.Context, err error) {
	log.Errorf("Monitor connect timeout, VM %s frozen: %s force restart!!!!", s.Id, err)
	s.ForceStop()
	timeutils2.AddTimeout(
		time.Second*3, func() { s.StartGuest(ctx, jsonutils.NewDict()) })
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
		return vncPort + MONITOR_PORT_BASE + 200
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

func (s *SKVMGuestInstance) DoResumeTask(ctx context.Context, isTimeout bool) {
	s.startupTask = NewGuestResumeTask(ctx, s, isTimeout, false)
	s.startupTask.Start()
}

func (s *SKVMGuestInstance) SyncStatus(reason string) {
	if s.IsRunning() {
		s.Monitor.GetBlockJobCounts(s.CheckBlockOrRunning)
		return
	}
	var status = "ready"
	if s.IsSuspend() {
		status = "suspend"
	}

	hostutils.UpdateServerStatus(context.Background(), s.Id, status, reason)
}

func (s *SKVMGuestInstance) CheckBlockOrRunning(jobs int) {
	var status = api.VM_RUNNING
	if jobs > 0 {
		if s.IsMaster() {
			mirrorStatus := s.MirrorJobStatus()
			if mirrorStatus.InProcess() {
				status = api.VM_BLOCK_STREAM
			} else if mirrorStatus.IsFailed() {
				status = api.VM_BLOCK_STREAM_FAIL
				s.SyncMirrorJobFailed("Block job missing")
			}
		} else {
			status = api.VM_BLOCK_STREAM
		}
	}
	_, err := hostutils.UpdateServerStatus(context.Background(), s.Id, status, "")
	if err != nil {
		log.Errorln(err)
	}
}

func (s *SKVMGuestInstance) SaveDesc(desc jsonutils.JSONObject) error {
	var ok bool
	s.Desc, ok = desc.(*jsonutils.JSONDict)
	if !ok {
		return fmt.Errorf("Unknown desc format, not JSONDict")
	}
	{
		// fill in ovn vpc nic bridge field
		nics, _ := s.Desc.GetArray("nics")
		for _, nic := range nics {
			if !nic.Contains("bridge") {
				nicjd := nic.(*jsonutils.JSONDict)
				nicjd.Set("bridge", jsonutils.NewString(getNicBridge(nic)))
			}
		}
	}
	if err := fileutils2.FilePutContents(s.GetDescFilePath(), desc.String(), false); err != nil {
		log.Errorln(err)
	}
	return nil
}

func (s *SKVMGuestInstance) GetVpcNIC() jsonutils.JSONObject {
	nics, _ := s.Desc.GetArray("nics")
	for _, nic := range nics {
		vpcProvider, _ := nic.GetString("vpc", "provider")
		if vpcProvider == api.VPC_PROVIDER_OVN {
			if ip, _ := nic.GetString("ip"); ip != "" {
				return nic
			}
		}
	}
	return nil
}

type guestStartTask struct {
	s *SKVMGuestInstance

	ctx    context.Context
	params *jsonutils.JSONDict
}

func (t *guestStartTask) Run() {
	t.s.asyncScriptStart(t.ctx, t.params)
}

func (t *guestStartTask) Dump() string {
	return fmt.Sprintf("guest %s params: %v", t.s.Id, t.params)
}

func (s *SKVMGuestInstance) StartGuest(ctx context.Context, params *jsonutils.JSONDict) {
	task := &guestStartTask{
		s:      s,
		ctx:    ctx,
		params: params,
	}
	s.manager.GuestStartWorker.Run(task, nil, nil)
}

func (s *SKVMGuestInstance) DeployFs(deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	disks, _ := s.Desc.GetArray("disks")
	if len(disks) > 0 {
		diskPath, _ := disks[0].GetString("path")
		disk, err := storageman.GetManager().GetDiskByPath(diskPath)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDiskByPath(%s)", diskPath)
		}
		return disk.DeployGuestFs(disk.GetPath(), s.Desc, deployInfo)
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
		}
	}
	if s.Monitor != nil {
		s.Monitor.Disconnect()
		s.Monitor = nil
	}
}

func (s *SKVMGuestInstance) CleanupCpuset() {
	task := cgrouputils.NewCGroupCPUSetTask(strconv.Itoa(s.GetPid()), 0, "")
	if !task.RemoveTask() {
		log.Warningf("remove cpuset cgroup error: %s, pid: %d", s.Id, s.GetPid())
	}
}

func (s *SKVMGuestInstance) GetCleanFiles() []string {
	return []string{s.GetPidFilePath(), s.GetVncFilePath()}
}

func (s *SKVMGuestInstance) delTmpDisks(ctx context.Context, migrated bool) error {
	disks, _ := s.Desc.GetArray("disks")
	for _, disk := range disks {
		if disk.Contains("path") {
			diskPath, _ := disk.GetString("path")
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
	if eid, _ := s.Desc.GetString("metadata", "__server_convert_from_esxi"); len(eid) > 0 {
		disks, _ := s.Desc.GetArray("disks")
		connections := new(deployapi.EsxiDisksConnectionInfo)
		connections.Disks = make([]*deployapi.EsxiDiskInfo, len(disks))
		for i := 0; i < len(disks); i++ {
			fpath, _ := disks[i].GetString("esxi_flat_file_path")
			connections.Disks[i] = &deployapi.EsxiDiskInfo{DiskPath: fpath}
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
	if err := s.delFlatFiles(ctx); err != nil {
		return errors.Wrap(err, "delFlatFiles")
	}
	if fileutils2.Exists(s.getQemuLogPath()) {
		procutils.NewRemoteCommandAsFarAsPossible("mv", s.getQemuLogPath(), fmt.Sprintf("/tmp/%s-qemu.log", s.GetId())).Run()
	}
	output, err := procutils.NewCommand("rm", "-rf", s.HomeDir()).Output()
	if err != nil {
		return errors.Wrapf(err, "rm %s failed: %s", s.HomeDir(), output)
	}
	return nil
}

func (s *SKVMGuestInstance) Stop() bool {
	s.ExitCleanup(true)
	if s.scriptStop() {
		return true
	} else {
		return false
	}
}

func (s *SKVMGuestInstance) scriptStart() error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", s.GetStartScriptPath()).Output()
	if err != nil {
		s.scriptStop()
		return fmt.Errorf("Start VM Failed %s %s", output, err)
	}
	return nil
}

func (s *SKVMGuestInstance) scriptStop() bool {
	_, err := procutils.NewRemoteCommandAsFarAsPossible("bash", s.GetStopScriptPath()).Output()
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

func (s *SKVMGuestInstance) ExecSuspendTask(ctx context.Context) {
	NewGuestSuspendTask(s, ctx, nil).Start()
}

func (s *SKVMGuestInstance) GetNicDescMatch(mac, ip, port, bridge string) jsonutils.JSONObject {
	nics, _ := s.Desc.GetArray("nics")
	for _, nic := range nics {
		nicBridge, _ := nic.GetString("bridge")
		if bridge == "" && nicBridge != "" && nicBridge == options.HostOptions.OvnIntegrationBridge {
			continue
		}
		nicMac, _ := nic.GetString("mac")
		nicIp, _ := nic.GetString("ip")
		nicPort, _ := nic.GetString("ifname")
		if (len(mac) == 0 || netutils2.MacEqual(nicMac, mac)) &&
			(len(ip) == 0 || nicIp == ip) &&
			(len(port) == 0 || nicPort == port) &&
			(len(bridge) == 0 || nicBridge == bridge) {
			return nic
		}
	}
	return nil
}

func pathEqual(disk, ndisk jsonutils.JSONObject) bool {
	if disk.Contains("path") && ndisk.Contains("path") {
		path1, _ := disk.GetString("path")
		path2, _ := ndisk.GetString("path")
		return path1 == path2
	} else if disk.Contains("url") && ndisk.Contains("url") {
		path1, _ := disk.GetString("assumed_path")
		path2, _ := ndisk.GetString("assumed_path")
		return path1 == path2
	} else {
		return false
	}

}

func (s *SKVMGuestInstance) compareDescDisks(newDesc jsonutils.JSONObject) ([]jsonutils.JSONObject, []jsonutils.JSONObject) {
	var delDisks, addDisks = []jsonutils.JSONObject{}, []jsonutils.JSONObject{}
	newDisks, _ := newDesc.GetArray("disks")
	for _, disk := range newDisks {
		driver, _ := disk.GetString("driver")
		if utils.IsInStringArray(driver, []string{"virtio", "scsi"}) {
			addDisks = append(addDisks, disk)
		}
	}
	oldDisks, _ := s.Desc.GetArray("disks")
	for _, disk := range oldDisks {
		driver, _ := disk.GetString("driver")
		if utils.IsInStringArray(driver, []string{"virtio", "scsi"}) {
			var find = false
			for idx, ndisk := range addDisks {
				diskIndex, _ := disk.Int("index")
				nDiskIndex, _ := ndisk.Int("index")
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

func (s *SKVMGuestInstance) compareDescIsolatedDevices(newDesc jsonutils.JSONObject) ([]jsonutils.JSONObject, []jsonutils.JSONObject) {
	var delDevs, addDevs = []jsonutils.JSONObject{}, []jsonutils.JSONObject{}
	newDevs, _ := newDesc.GetArray("isolated_devices")
	for _, dev := range newDevs {
		addDevs = append(addDevs, dev)
	}
	oldDevs, _ := s.Desc.GetArray("isolated_devices")
	for _, oldDev := range oldDevs {
		var find = false
		oVendorDevId, _ := oldDev.GetString("vendor_device_id")
		for idx, addDev := range addDevs {
			nVendorDevId, _ := addDev.GetString("vendor_device_id")
			if oVendorDevId == nVendorDevId {
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

func (s *SKVMGuestInstance) compareDescCdrom(newDesc jsonutils.JSONObject) *string {
	if !s.Desc.Contains("cdrom") && !newDesc.Contains("cdrom") {
		return nil
	} else if !s.Desc.Contains("cdrom") && newDesc.Contains("cdrom") {
		cdrom, _ := newDesc.GetString("cdrom", "path")
		return &cdrom
	} else if s.Desc.Contains("cdrom") && !newDesc.Contains("cdrom") {
		var res = ""
		return &res
	} else {
		cdrom, _ := s.Desc.GetString("cdrom", "path")
		ncdrom, _ := newDesc.GetString("cdrom", "path")
		if cdrom == ncdrom {
			return nil
		} else {
			return &ncdrom
		}
	}
}

func (s *SKVMGuestInstance) compareDescNetworks(newDesc jsonutils.JSONObject) ([]jsonutils.JSONObject, []jsonutils.JSONObject, [][]jsonutils.JSONObject) {
	var isValid = func(net jsonutils.JSONObject) bool {
		driver, _ := net.GetString("driver")
		return driver == "virtio"
	}

	var findNet = func(nets []jsonutils.JSONObject, net jsonutils.JSONObject) int {
		mac1, _ := net.GetString("mac")
		for i := 0; i < len(nets); i++ {
			mac2, _ := nets[i].GetString("mac")
			if mac1 == mac2 {
				return i
			}
		}
		return -1
	}

	var delNics, addNics = []jsonutils.JSONObject{}, []jsonutils.JSONObject{}
	var changedNics = [][]jsonutils.JSONObject{}
	nics, _ := newDesc.GetArray("nics")
	for _, n := range nics {
		if isValid(n) {
			// assume all nics in new desc are new
			addNics = append(addNics, n)
		}
	}

	nics, _ = s.Desc.GetArray("nics")
	for _, n := range nics {
		if isValid(n) {
			idx := findNet(addNics, n)
			if idx >= 0 {
				// check if bridge changed
				changedNics = append(changedNics, []jsonutils.JSONObject{
					n,            // old
					addNics[idx], // new
				})
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

func getNicBridge(nic jsonutils.JSONObject) string {
	bridge, _ := nic.GetString("bridge")
	if len(bridge) == 0 {
		vpcProvider, _ := nic.GetString("vpc", "provider")
		if vpcProvider == api.VPC_PROVIDER_OVN {
			bridge = options.HostOptions.OvnIntegrationBridge
		}
	}
	return bridge
}

func onNicChange(oldNic, newNic jsonutils.JSONObject) error {
	oldbr := getNicBridge(oldNic)
	oldifname, _ := oldNic.GetString("ifname")
	newbr := getNicBridge(newNic)
	newifname, _ := newNic.GetString("ifname")
	newvlan, _ := newNic.Int("vlan")
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

func (s *SKVMGuestInstance) SyncConfig(ctx context.Context, desc jsonutils.JSONObject, fwOnly bool) (jsonutils.JSONObject, error) {
	var delDisks, addDisks, delNetworks, addNetworks, delDevs, addDevs []jsonutils.JSONObject
	var changedNetworks [][]jsonutils.JSONObject
	var cdrom *string

	if !fwOnly {
		delDisks, addDisks = s.compareDescDisks(desc)
		cdrom = s.compareDescCdrom(desc)
		delNetworks, addNetworks, changedNetworks = s.compareDescNetworks(desc)
		delDevs, addDevs = s.compareDescIsolatedDevices(desc)
	}

	if len(changedNetworks) > 0 && s.IsRunning() {
		// process changed networks
		for i := range changedNetworks {
			err := onNicChange(changedNetworks[i][0], changedNetworks[i][1])
			if err != nil {
				return nil, errors.Wrap(err, "onNicChange")
			}
		}
	}

	if err := s.SaveDesc(desc); err != nil {
		return nil, err
	}

	if !s.IsRunning() {
		return nil, nil
	}

	vncPort := s.GetVncPort()
	data := jsonutils.NewDict()
	data.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
	s.saveScripts(data)

	// if options.enable_openflow_controller: 不写

	if fwOnly {
		res := jsonutils.NewDict()
		res.Set("task", jsonutils.NewArray(jsonutils.NewString("secgroupsync")))
		return res, nil
	}
	var runTaskNames = []jsonutils.JSONObject{}
	var tasks = []IGuestTasks{}

	var callBack = func(errs []error) {
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

	if len(delDisks)+len(addDisks) > 0 || cdrom != nil {
		task := NewGuestDiskSyncTask(s, delDisks, addDisks, cdrom)
		runTaskNames = append(runTaskNames, jsonutils.NewString("disksync"))
		tasks = append(tasks, task)
	}

	if len(delNetworks)+len(addNetworks) > 0 {
		task := NewGuestNetworkSyncTask(s, delNetworks, addNetworks)
		runTaskNames = append(runTaskNames, jsonutils.NewString("networksync"))
		tasks = append(tasks, task)
	}

	if len(delDevs)+len(addDevs) > 0 {
		task := NewGuestIsolatedDeviceSyncTask(s, delDevs, addDevs)
		runTaskNames = append(runTaskNames, jsonutils.NewString("isolated_device_sync"))
		tasks = append(tasks, task)
	}

	NewGuestSyncConfigTaskExecutor(ctx, s, tasks, callBack).Start(1)
	res := jsonutils.NewDict()
	res.Set("task", jsonutils.NewArray(runTaskNames...))
	return res, nil
}

func (s *SKVMGuestInstance) getApptags() []string {
	var tags []string
	meta, _ := s.Desc.Get("metadata")
	if meta != nil && meta.Contains("app_tags") {
		tagsStr, _ := meta.GetString("app_tags")
		if len(tagsStr) > 0 {
			return strings.Split(tagsStr, ",")
		}
	}
	return tags
}

func (s *SKVMGuestInstance) getStorageDeviceId() string {
	disks, _ := s.Desc.GetArray("disks")
	if len(disks) > 0 {
		diskPath, _ := disks[0].GetString("path")
		if len(diskPath) > 0 {
			return fileutils2.GetDevId(diskPath)
		}
	}
	return ""
}

func (s *SKVMGuestInstance) SetCgroup() {
	s.cgroupPid = s.GetPid()
	s.setCgroupIo()
	s.setCgroupCpu()
	s.setCgroupCPUSet()
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
		cpu, _ := s.Desc.Int("cpu")
		cgrouputils.CgroupIoHardlimitSet(strconv.Itoa(s.cgroupPid), int(cpu), params, devId)
	}
}

func (s *SKVMGuestInstance) setCgroupCpu() {
	var (
		cpu, _    = s.Desc.Int("cpu")
		cpuWeight = 1024
	)

	cgrouputils.CgroupSet(strconv.Itoa(s.cgroupPid), int(cpu)*cpuWeight)
}

func (s *SKVMGuestInstance) setCgroupCPUSet() {
	meta, _ := s.Desc.Get("metadata")
	if meta == nil {
		return
	}
	cpusetStr, _ := meta.GetString(api.VM_METADATA_CGROUP_CPUSET)
	if len(cpusetStr) == 0 {
		return
	}
	obj, err := jsonutils.ParseString(cpusetStr)
	if err != nil {
		log.Errorf("Parse cpusetStr %q error: %v", cpusetStr, err)
		return
	}
	input := new(api.ServerCPUSetInput)
	if err := obj.Unmarshal(input); err != nil {
		log.Errorf("Unmarshal %q to ServerCPUSetInput: %v", obj, err)
		return
	}
	if _, err := s.CPUSet(context.Background(), input); err != nil {
		log.Errorf("Do CPUSet error: %v", err)
		return
	}
}

func (s *SKVMGuestInstance) CreateFromDesc(desc jsonutils.JSONObject) error {
	if err := s.PrepareDir(); err != nil {
		uuid, _ := desc.GetString("uuid")
		return fmt.Errorf("Failed to create server dir %s", uuid)
	}
	return s.SaveDesc(desc)
}

func (s *SKVMGuestInstance) GetNeedMergeBackingFileDiskIndexs() []int {
	res := make([]int, 0)
	disks, _ := s.Desc.GetArray("disks")
	for _, disk := range disks {
		if jsonutils.QueryBoolean(disk, "merge_snapshot", false) {
			diskIdx, _ := disk.Int("index")
			res = append(res, int(diskIdx))
		}
	}
	return res
}

func (s *SKVMGuestInstance) streamDisksComplete(ctx context.Context) {
	disks, _ := s.Desc.GetArray("disks")
	for i, disk := range disks {
		diskpath, _ := disk.GetString("path")
		d, _ := storageman.GetManager().GetDiskByPath(diskpath)
		if d != nil {
			log.Infof("Disk %s do post create from fuse", d.GetId())
			d.PostCreateFromImageFuse()
		}
		if jsonutils.QueryBoolean(disk, "merge_snapshot", false) {
			d := disks[i].(*jsonutils.JSONDict)
			d.Set("merge_snapshot", jsonutils.JSONFalse)
			s.Desc.Set("need_sync_stream_disks", jsonutils.JSONTrue)
		}
	}
	if err := s.SaveDesc(s.Desc); err != nil {
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
	s.Desc.Remove("need_sync_stream_disks")
	if err := s.SaveDesc(s.Desc); err != nil {
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
	_, err := modules.Servers.SetMetadata(hostutils.GetComputeSession(context.Background()),
		s.Id, meta)
	if err != nil {
		log.Errorln("sync metadata error: %v", err)
		return errors.Wrap(err, "set metadata")
	}
	return nil
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
	meta.Set("hotplug_cpu_mem", jsonutils.NewString("enable"))
	meta.Set("hot_remove_nic", jsonutils.NewString("enable"))
	if len(s.VncPassword) > 0 {
		meta.Set("__vnc_password", jsonutils.NewString(s.VncPassword))
	}
	if options.HostOptions.HugepagesOption == "native" {
		meta.Set("__hugepage", jsonutils.NewString("native"))
	}
	if !options.HostOptions.HostCpuPassthrough || s.getOsname() == OS_NAME_MACOS {
		meta.Set("__cpu_mode", jsonutils.NewString(api.CPU_MODE_QEMU))
	} else {
		meta.Set("__cpu_mode", jsonutils.NewString(api.CPU_MODE_HOST))
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

func (s *SKVMGuestInstance) CleanImportMetadata() *jsonutils.JSONDict {
	meta := jsonutils.NewDict()
	if originId, _ := s.Desc.GetString("metadata", "__origin_id"); len(originId) > 0 {
		meta.Set("__origin_id", jsonutils.NewString(""))
	}
	if monitorPath, _ := s.Desc.GetString("metadata", "__monitor_path"); len(monitorPath) > 0 {
		meta.Set("__monitor_path", jsonutils.NewString(""))
	}

	if meta.Length() > 0 {
		// update local metadata record, after monitor started updata region record
		metadata, err := s.Desc.GetMap("metadata")
		if err == nil {
			updateMeta, _ := meta.GetMap()
			for k, v := range updateMeta {
				metadata[k] = v
			}
		}
		s.SaveDesc(s.Desc)
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
	ctx context.Context, disk storageman.IDisk, snapshotId string,
) (jsonutils.JSONObject, error) {
	if s.IsRunning() {
		if !s.isLiveSnapshotEnabled() {
			return nil, fmt.Errorf("Guest dosen't support live snapshot")
		}
		err := disk.CreateSnapshot(snapshotId)
		if err != nil {
			return nil, err
		}
		task := NewGuestDiskSnapshotTask(ctx, s, disk, snapshotId)
		task.Start()
		return nil, nil
	} else {
		return s.StaticSaveSnapshot(ctx, disk, snapshotId)
	}
}

func (s *SKVMGuestInstance) StaticSaveSnapshot(
	ctx context.Context, disk storageman.IDisk, snapshotId string,
) (jsonutils.JSONObject, error) {
	err := disk.CreateSnapshot(snapshotId)
	if err != nil {
		return nil, err
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
		if err := procutils.NewRemoteCommandAsFarAsPossible("mv", memStatPath, memSnapPath).Run(); err != nil {
			hostutils.TaskFailed(ctx, fmt.Sprintf("move statefile %q to memory snapshot %q: %v", memStatPath, memSnapPath, err))
			return
		}
		resumeTask := NewGuestResumeTask(ctx, s, false, false)
		resumeTask.SetGetTaskData(func() (jsonutils.JSONObject, error) {
			resp := &hostapi.GuestMemorySnapshotResponse{
				MemorySnapshotPath: memSnapPath,
				SizeMB:             sizeMB,
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
	hostutils.TaskComplete(ctx, nil)
	return nil, nil
}

func (s *SKVMGuestInstance) PrepareDisksMigrate(liveMigrage bool) (*jsonutils.JSONDict, error) {
	disksBackFile := jsonutils.NewDict()
	disks, _ := s.Desc.GetArray("disks")
	for _, disk := range disks {
		if disk.Contains("path") {
			diskPath, _ := disk.GetString("path")
			d, err := storageman.GetManager().GetDiskByPath(diskPath)
			if err != nil {
				return nil, errors.Wrapf(err, "GetDiskByPath(%s)", diskPath)
			}
			if d.GetType() == api.STORAGE_LOCAL {
				back, err := d.PrepareMigrate(liveMigrage)
				if err != nil {
					return nil, err
				}
				if len(back) > 0 {
					diskId, _ := disk.GetString("disk_id")
					disksBackFile.Set(diskId, jsonutils.NewString(back))
				}
			}
		}
	}
	return disksBackFile, nil
}

func (s *SKVMGuestInstance) onlineResizeDisk(ctx context.Context, diskId string, sizeMB int64) {
	task := NewGuestOnlineResizeDiskTask(ctx, s, diskId, sizeMB)
	task.Start()
}

func (s *SKVMGuestInstance) BlockIoThrottle(ctx context.Context, bps, iops int64) error {
	task := SGuestBlockIoThrottleTask{s, ctx, bps, iops}
	return task.Start()
}

func (s *SKVMGuestInstance) IsSharedStorage() bool {
	disks, _ := s.Desc.GetArray("disks")
	for i := 0; i < len(disks); i++ {
		diskPath, _ := disks[i].GetString("path")
		disk, err := storageman.GetManager().GetDiskByPath(diskPath)
		if err != nil {
			log.Errorf("failed find disk by path %s", diskPath)
			return false
		}
		if !utils.IsInStringArray(disk.GetType(), api.SHARED_STORAGE) {
			return false
		}
	}
	return true
}

func (s *SKVMGuestInstance) generateDiskSetupScripts(disks []api.GuestdiskJsonDesc) (string, error) {
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

func (s *SKVMGuestInstance) CPUSet(ctx context.Context, input *api.ServerCPUSetInput) (*api.ServerCPUSetResp, error) {
	if !s.IsRunning() {
		return nil, nil
	}
	cpus := []string{}
	for _, id := range input.CPUS {
		cpus = append(cpus, fmt.Sprintf("%d", id))
	}
	task := cgrouputils.NewCGroupCPUSetTask(strconv.Itoa(s.GetPid()), 0, strings.Join(cpus, ","))
	if !task.SetTask() {
		return nil, errors.Errorf("Cgroup cpuset task failed")
	}
	return new(api.ServerCPUSetResp), nil
}

func (s *SKVMGuestInstance) CPUSetRemove(ctx context.Context) error {
	metadata, err := s.Desc.Get("metadata")
	if err != nil {
		return errors.Wrap(err, "get metadata from desc")
	}
	metadata.(*jsonutils.JSONDict).Remove(api.VM_METADATA_CGROUP_CPUSET)
	s.Desc.Set("metadata", metadata)
	if err := s.SaveDesc(s.Desc); err != nil {
		return errors.Wrap(err, "save desc after update metadata")
	}
	if !s.IsRunning() {
		return nil
	}
	task := cgrouputils.NewCGroupCPUSetTask(strconv.Itoa(s.GetPid()), 0, "")
	if !task.RemoveTask() {
		return errors.Errorf("Remove task error happened, please lookup host log")
	}
	return nil
}
