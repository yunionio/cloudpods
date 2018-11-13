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

	"yunion.io/x/onecloud/pkg/cloudcommon/httpclients"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

const (
	STATE_FILE_PREFIX = "STATEFILE"
	MONITOR_PORT_BASE = 55900
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

func (s *SKVMGuestInstance) AsyncScriptStart() {
	// TODO
	// s.manager.RequestStartGuest()
}

func (s *SKVMGuestInstance) ImportServer(pendingDelete bool) {
	if s.IsDirtyShotdown() && !pendingDelete {
		log.Infof("Server dirty shotdown %s", s.GetName())
		if jsonutils.QueryBoolean(s.Desc, "is_master", false) ||
			jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
			s.DirtyServerRequestStart()
		} else {
			s.AsyncScriptStart()
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

func (s *SKVMGuestInstance) StartMonitor(ctx context.Context) {
	// delay 100ms start monitor
	AddTimeout(100*time.Millisecond, func() { s.delayStartMonitor(ctx) })
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
	// TODO
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
