package guestman

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

const (
	STATE_FILE_PREFIX = "STATEFILE"
	MONITOR_PORT_BASE = 55900
	MAX_TRY           = 3
)

type SKVMGuestInstance struct {
	Id          string
	QemuVersion string

	Desc    *jsonutils.JSONDict
	monitor monitor.Monitor
	manager *SGuestManager
}

func NewKVMGuestInstance(id string, manager *SGuestManager) *SKVMGuestInstance {
	return &SKVMGuestInstance{
		Id:      id,
		manager: manager,
	}
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

func (s *SKVMGuestInstance) HomeDir() string {
	return path.Join(s.manager.ServersPath, s.Id)
}

func (s *SKVMGuestInstance) PrepareDir() error {
	_, err := exec.Command("mkdir", "-p", s.HomeDir()).Output()
	return err
}

func (s *SKVMGuestInstance) GetPidFilePath() string {
	return path.Join(s.HomeDir(), "pid")
}

func (s *SKVMGuestInstance) GetVncFilePath() string {
	return path.Join(s.HomeDir(), "vnc")
}

func (s *SKVMGuestInstance) GetPid() int {
	pidFile := s.GetPidFilePath()
	fi, err := os.Stat(pidFile)
	if os.IsNotExist(err) {
		return -1
	}
	if fi.Mode().IsRegular() {
		return -1
	}
	content, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return -2
	}
	pid := s.findPid(strings.Split(string(content), "\n"))
	if len(pid) > 0 && regutils.MatchInteger(pid) {
		v, _ := strconv.ParseInt(pid, 10, 0)
		return int(v)
	}
	return -2
}

func (s *SKVMGuestInstance) findPid(pids []string) string {
	if len(pids) == 0 {
		return ""
	}
	for _, pid := range pids {
		pid := strings.TrimSpace(pid)
		if s.isSelfQemuPid(pid) {
			return pid
		}
	}
	return ""
}

func (s *SKVMGuestInstance) isSelfQemuPid(pid string) bool {
	if len(pid) == 0 {
		return false
	}
	cmdlineFile := fmt.Sprintf("/proc/%s/cmdline", pid)
	fi, err := os.Stat(cmdlineFile)
	if err != nil {
		log.Warningf("IsSelfQemuPid Stat File %s error %s", cmdlineFile, err)
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
	return bytes.Index(cmdline, []byte("qemu-system")) >= 0 &&
		bytes.Index(cmdline, []byte(s.Id)) >= 0
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
		log.Errorf("Dirty server request start error: %s", err)
	}
}

// Delay Process
func (s *SKVMGuestInstance) asyncScriptStart(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		log.Errorln("asyncScriptStart params error")
		return nil, fmt.Errorf("Unknown params")
	}

	// TODO hostinof.instace().clean_deleted_ports

	time.Sleep(100 * time.Millisecond)
	var isStarted, tried = false, 0
	var err error
	for !isStarted && tried < MAX_TRY {
		tried += 1

		vncPort := s.manager.GetFreeVncPort()
		if err = s.saveVncPort(vncPort); err != nil {
			goto finally
		} else {
			data.Set("vnc_port", jsonutils.NewInt(vncPort))
		}

		if err = s.saveScripts(data); err != nil {
			goto finally
		} else {
			isStarted, err = s.scriptStart()
		}

	finally:
		if !isStarted {
			log.Errorf("Start VM failed: %s", err)
			time.Sleep((1 << (tried - 1)) * time.Seconde)
		} else {
			log.Infof("VM started ...")
		}
	}

	// is on_async_script_start
	if isStarted {
		log.Infof("Async start server %s success!", s.GetName())
		s.StartMonitor(ctx)
		return nil, nil
	} else {
		log.Infof("Async start server %s failed: %s!!!", s.GetName(), err)
		timeutils2.AddTimeout(100*time.Millisecond, s.SyncStatus())
		return nil, err
	}
}

func (s *SKVMGuestInstance) saveScripts(data *jsonutils.JSONDict) error {
	startScript, err := s.generateStartScript(data)
	if err != nil {
		return err
	}
	if err := fileutils2.FilePutContents(s.GetStartScriptPath, startScript, false); err != nil {
		return err
	}
	stopScript, err := s.generateStopScript(data)
	if err != nil {
		return err
	}
	return fileutils2.FilePutContents(s.GetStopScriptPath, stopScript, false)
}

func (s *SKVMGuestInstance) GetStartScriptPath() string {
	return path.Join(s.HomeDir(), "startvm")
}

func (s *SKVMGuestInstance) GetStopScriptPath() string {
	return path.Join(s.HomeDir(), "stopvm")
}

func (s *SKVMGuestInstance) ImportServer(pendingDelete bool) {
	if s.IsDirtyShotdown() && !pendingDelete {
		log.Infof("Server dirty shotdown %s", s.GetName())
		if jsonutils.QueryBoolean(s.Desc, "is_master", false) ||
			jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
			s.DirtyServerRequestStart()
		} else {
			s.StartGuest(nil, nil)
		}
		return
	}
	if s.IsRunning() {
		log.Infof("%s is running, pending_delete=%s", s.GetName(), pendingDelete)
		if !pendingDelete {
			s.StartMonitor(nil)
		}
	} else {
		var action = "stopped"
		if s.IsSuspend() {
			action = "suspend"
		}
		log.Infof("%s is %s, pending_delete=%s", s.GetName(), action, pendingDelete)
		s.SyncStatus()
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
	return s.monitor != nil && s.monitor.IsConnected()
}

func (s *SKVMGuestInstance) ListStateFilePaths() []string {
	files, err := ioutil.ReadDir(s.HomeDir())
	if err == nil {
		var ret = make([]string, 0)
		for i := 0; i < len(files); i++ {
			if strings.HasPrefix(files[i].Name(), STATE_FILE_PREFIX) {
				ret = append(ret, files[i].Name())
			}
		}
		return ret
	}
	return nil
}

func (s *SKVMGuestInstance) StartMonitor(ctx context.Context) {
	timeutils2.AddTimeout(100*time.Millisecond, func() { s.delayStartMonitor(ctx) })
}

func (s *SKVMGuestInstance) delayStartMonitor(ctx context.Context) {
	if options.HostOptions.EnableQmpMonitor && s.GetQmpMonitorPort() > 0 {
		s.monitor = monitor.NewQmpMonitor(s.onMonitorDisConnect, s.onMonitorTimeout,
			func() { s.onMonitorConnected(ctx) })
		s.monitor.Connect("127.0.0.1", s.GetQmpMonitorPort())
	}
}

func (s *SKVMGuestInstance) onMonitorConnected(ctx context.Context) {
	log.Infof("Monitor connected ...")
	s.monitor.GetVersion(func(v string) {
		s.onGetQemuVersion(ctx, v)
	})
}

func (s *SKVMGuestInstance) onGetQemuVersion(ctx context.Context, version string) {
	s.QemuVersion = version
	log.Infof("Guest(%s) qemu version %s", s.Id, s.QemuVersion)
	if s.Desc.Contains("live_migrate_dest_port") && ctx != nil {
		migratePort, _ := s.Desc.Get("live_migrate_dest_port")
		body := jsonutils.NewDict(
			jsonutils.JSONPair{"live_migrate_dest_port", migratePort})
		hostutils.TaskComplete(ctx, body)
	} else if jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
		// TODO
	} else if jsonutils.QueryBoolean(s.Desc, "is_master", false) && ctx == nil {
		// TODO
	} else {
		// TODO
		s.DoResumeTask(ctx)
	}
}

func (s *SKVMGuestInstance) onMonitorDisConnect(err error) {
	// TODO
}

func (s *SKVMGuestInstance) onMonitorTimeout(err error) {
	// TODO
}

func (s *SKVMGuestInstance) GetQmpMonitorPort() int {
	var vncPort = s.GetVncPort()
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

func (s *SKVMGuestInstance) saveVncPort(port int64) error {
	return fileutils2.FilePutContents(s.GetVncFilePath(), fmt.Sprintf("%d", port), false)
}

func (s *SKVMGuestInstance) SyncStatus() {
	if s.IsRunning() {
		s.monitor.GetBlockJobs(s.CheckBlockOrRunning)
		return
	}
	var status = "ready"
	if s.IsSuspend() {
		status = "suspend"
	}

	hostutils.UpdateServerStatus(context.Background(), s.GetId(), status)
}

func (s *SKVMGuestInstance) SaveDesc(desc jsonutils.JSONObject) error {
	// TODO
	// bw_info = self._get_bw_info()
	// netmon_info = self._get_netmon_info()
	s.Desc = desc.(*jsonutils.JSONDict)
	if err := fileutils2.FilePutContents(s.GetDescFilePath(), desc.String()); err != nil {
		log.Errorln(err)
	}
	// TODO
	// self._update_bw_limit(bw_info)
	// self._update_netmon_nic(netmon_info)
}

func (s *SKVMGuestInstance) StartGuest(ctx context.Context, params jsonutils.JSONObject) {
	hostutils.DelayTask(ctx, s.asyncScriptStart, params)
}

func (s *SKVMGuestInstance) DeployFs(deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error) {
	disks, _ := s.Desc.GetArray("disks")
	if len(disks) > 0 {
		storageId, _ := disks[0].GetString("storage_id")
		diskId, _ := disks[0].GetString("disk_id")

		disk := storageman.GetManager().GetStorageDisk(storageId, diskId)
		return disk.DeployGuestFs(disk.GetPath, s.Desc, deployInfo)
	} else {
		return nil, fmt.Errorf("Guest dosen't have disk ??")
	}
}

// Delay process
func (s *SKVMGuestInstance) CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migrated, ok := params.(bool)
	if !ok {
		return nil, fmt.Errorf("Unknown params")
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
		err := exec.Command("kill", "-9", fmt.Sprintf("%d", s.GetPid())).Run()
		if err != nil {
			log.Errorln(err)
			return false
		}
		for _, f := range s.GetCleanFiles() {
			err := exec.Command("rm", "-f", f).Run()
			if err != nil {
				log.Errorln(err)
				return false
			}
		}
		return true
	}
	return false
}

func (s *SKVMGuestInstance) ExitCleanup(clearCgroup bool) {
	if clearCgroup {
		pid := s.GetPid()
		if pid > 0 {
			// TODO: ClearCgroup
			s.ClearCgroup(pid)
		}
	}
	if s.monitor != nil {
		s.monitor.Disconnect()
		s.monitor = nil
	}
}

func (s *SKVMGuestInstance) GetCleanFiles() []string {
	return []string{s.GetPidFilePath(), s.GetVncFilePath()}
}

func (s *SKVMGuestInstance) delTmpDisks(ctx context.Context, migrated bool) {
	disks, _ := s.Desc.GetArray("disks")
	for _, disk := range disks {
		if disk.Contains("path") {
			diskPath, _ := disk.GetString("path")
			// TODO GetDisksByPath, storagetypes, deleteallsnapshot, delete
			d := storageman.GetManager().GetDiskByPath(diskPath)
			if d != nil && d.GetType == storagetypes.STORAGE_LOCAL && migrated {
				if err := d.DeleteAllSnapshot(); err != nil {
					log.Errorln(err)
				}
				if err := d.Delete(ctx); err != nil {
					log.Errorln(err)
				}
			}
		}
	}
}

func (s *SKVMGuestInstance) Delete(ctx context.Context, migrated bool) error {
	// self._del_bw_limit()
	// self._del_netmon_nic() ?? 需要开发？
	s.delTmpDisks(ctx, migrated)
	return exec.Command("rm", "-rf", s.HomeDir()).Run()
}

func (s *SKVMGuestInstance) Stop() bool {
	s.ExitCleanup(true)
	if s.scriptStop() {
		return true
	} else {
		return false
	}
}

func (s *SKVMGuestInstance) scriptStop() bool {
	err := exec.Command("sh", s.GetStopScriptPath()).Run()
	if err != nil {
		log.Errorln(err)
		return false
	}
	return true
}

func (s *SKVMGuestInstance) ExecStopTask(ctx context.Context, timeout int64) {
	NewGuestStopTask(s, ctx, timeout).Start()
}

func (s *SKVMGuestInstance) ExecSuspendTask(ctx context.Context) {
	// TODO
}

func (s *SKVMGuestInstance) GetNicDescMatch(mac, ip, port, bridge string) jsonutils.JSONObject {
	nics, _ := s.Desc.GetArray("nics")
	for _, nic := range nics {
		nicMac, _ := nic.GetString("mac")
		nicIp, _ := nic.GetString("ip")
		nicPort, _ := nic.GetString("ifname")
		nicBridge, _ := nic.GetString("bridge")
		if (len(mac) == 0 || netutils2.MacEqual(nicMac, mac)) &&
			(len(ip) == 0 || nicIp == ip) &&
			(len(port) == 0 || nicPort == port) &&
			(len(bridge) == 0 || nicBridge == bridge) {
			return nic
		}
	}
	return nil
}

// 目测sync_cgroup没有要用到，先不写
func (s *SKVMGuestInstance) SyncConfig(ctx context.Context, desc jsonutils.JSONObject, fwOnly bool) (jsonutils.JSONObject, error) {

}
