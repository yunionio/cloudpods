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

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/httpclients"
	"yunion.io/x/onecloud/pkg/hostman"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
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
	var url = "/servers/dirty-server-start"
	hostId, _ := s.Desc.GetString("host_id")
	var body = jsonutils.NewDict()
	body.Set("guest_id", jsonutils.NewString(s.Id))
	body.Set("host_id", jsonutils.NewString(hostId))
	_, _, err := httpclients.GetDefaultComputeClient().
		Request(context.Background(), "POST", url, nil, body, false)
	if err != nil {
		log.Errorf("Dirty server request start error: %s", err)
	}
}

// Must called in new goroutine
func (s *SKVMGuestInstance) asyncScriptStart(ctx context.Context, params *jsonutils.JSONDict) {
	// TODO
	// hostinof.instace().clean_deleted_ports
	time.Sleep(100 * time.Millisecond)
	var isStarted, tried, err = false, 0, nil
	for !isStarted && tried < MAX_TRY {
		tried += 1
		vncPort := s.manager.GetFreeVncPort()
		s.saveVncPort(vncPort)
		params.Set("vnc_port", jsonutils.NewInt(vncPort))
		s.saveScripts(params)
		isStarted, err = s.scriptStart()
		if !isStarted {
			log.Errorf("Start VM failed: %s", err)
			time.Sleep((1 << (tried - 1)) * time.Seconde)
		} else {
			log.Infof("VM started ...")
		}
	}
	s.onAsyncScriptStart(ctx, isStarted, err)
}

func (s *SKVMGuestInstance) onAsyncScriptStart(ctx context.Context, isStarted bool, err error) {
	if isStarted {
		log.Infof("Async start server %s success!", s.GetName())
		s.StartMonitor(ctx)
	} else {
		log.Infof("Async start server %s failed: %s!!!", s.GetName(), err)
		if ctx != nil {
			s.TaskFailed(ctx, fmt.Sprintf("Async start server failed: %s", err))
		}
		s.SyncStatus()
	}
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
			go s.StartMonitor(nil)
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

// Must called in new goroutine
func (s *SKVMGuestInstance) StartMonitor(ctx context.Context) {
	// delay 100ms start monitor // hostman.AddTimeout(100*time.Millisecond, func() { s.delayStartMonitor(ctx) })
	time.Sleep(100 * time.Millisecond)
	s.delayStartMonitor(ctx)
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
		s.TaskComplete(ctx, body)
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

func (s *SKVMGuestInstance) SyncStatus() {
	if s.IsRunning() {
		s.monitor.GetBlockJobs(s.CheckBlockOrRunning)
		return
	}
	var status = "ready"
	if s.IsSuspend() {
		status = "suspend"
	}
	httpclients.GetDefaultComputeClient().UpdateServerStatus(s.GetId(), status)
}

func (s *SKVMGuestInstance) TaskFailed(ctx context.Context, reason string) error {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		httpclients.GetDefaultComputeClient().TaskFail(ctx, taskId.(string), reason)
		return nil
	} else {
		log.Errorln("Reqeuest task failed missing task id, with reason(%s)", reason)
		return fmt.Errorf("Reqeuest task failed missing task id")
	}
}

func (s *SKVMGuestInstance) TaskComplete(ctx context.Context, data jsonutils.JSONObject) error {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		httpclients.GetDefaultComputeClient().TaskComplete(ctx, taskId.(string), data, 0)
		return nil
	} else {
		log.Errorln("Reqeuest task complete missing task id")
		return fmt.Errorf("Reqeuest task complete missing task id")
	}
}

func (s *SKVMGuestInstance) SaveDesc(desc jsonutils.JSONObject) error {
	// TODO
	// bw_info = self._get_bw_info()
	// netmon_info = self._get_netmon_info()
	s.Desc = desc.(*jsonutils.JSONDict)
	if err := hostman.FilePutContents(s.GetDescFilePath(), desc.String()); err != nil {
		log.Errorln(err)
	}
	// TODO
	// self._update_bw_limit(bw_info)
	// self._update_netmon_nic(netmon_info)
}

func (s *SKVMGuestInstance) StartGuest(ctx context.Context, params jsonutils.JSONObject) {
	wm.RunTask(func() {
		s.asyncScriptStart(ctx, params)
	})
}
