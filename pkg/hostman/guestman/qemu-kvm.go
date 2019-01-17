package guestman

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostbridge"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
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
	Id          string
	cgroupId    string
	QemuVersion string
	VncPassword string

	Desc    *jsonutils.JSONDict
	Monitor monitor.Monitor
	manager *SGuestManager

	startupTask *SGuestResumeTask
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

func (s *SKVMGuestInstance) getStateFilePathRootPrefix() string {
	return path.Join(s.HomeDir(), STATE_FILE_PREFIX)
}

func (s *SKVMGuestInstance) IsLoaded() bool {
	return s.Desc != nil
}

func (s *SKVMGuestInstance) HomeDir() string {
	return path.Join(s.manager.ServersPath, s.Id)
}

func (s *SKVMGuestInstance) PrepareDir() error {
	_, err := procutils.NewCommand("mkdir", "-p", s.HomeDir()).Run()
	return err
}

func (s *SKVMGuestInstance) GetPidFilePath() string {
	return path.Join(s.HomeDir(), "pid")
}

func (s *SKVMGuestInstance) GetVncFilePath() string {
	return path.Join(s.HomeDir(), "vnc")
}

func (s *SKVMGuestInstance) GetPid() int {
	return s.getPid(s.GetPidFilePath(), s.Id)
}

func (s *SKVMGuestInstance) getPid(pidFile, uuid string) int {
	_, err := os.Stat(pidFile)
	if os.IsNotExist(err) {
		return -1
	}
	pidStr, err := fileutils2.FileGetContents(pidFile)
	if err != nil {
		log.Errorln(err)
		return -2
	}
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
		bytes.Index(cmdline, []byte(uuid)) >= 0
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
		return nil, hostutils.ParamsError
	}

	hostbridge.CleanDeletedPorts()

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
			log.Errorf("Start VM failed: %s", err)
			time.Sleep(time.Duration(1<<uint(tried-1)) * time.Second)
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
		timeutils2.AddTimeout(100*time.Millisecond, s.SyncStatus)
		return nil, err
	}
}

func (s *SKVMGuestInstance) saveScripts(data *jsonutils.JSONDict) error {
	startScript, err := s.generateStartScript(data)
	if err != nil {
		return err
	}
	if fileutils2.Exists(s.GetStartScriptPath()) {
		os.Remove(s.GetStartScriptPath())
	}
	if err = fileutils2.FilePutContents(s.GetStartScriptPath(), startScript, false); err != nil {
		return err
	}
	stopScript := s.generateStopScript(data)
	if fileutils2.Exists(s.GetStartScriptPath()) {
		os.Remove(s.GetStopScriptPath())
	}
	return fileutils2.FilePutContents(s.GetStopScriptPath(), stopScript, false)
}

func (s *SKVMGuestInstance) GetStartScriptPath() string {
	return path.Join(s.HomeDir(), "startvm")
}

func (s *SKVMGuestInstance) GetStopScriptPath() string {
	return path.Join(s.HomeDir(), "stopvm")
}

func (s *SKVMGuestInstance) ImportServer(pendingDelete bool) {
	s.manager.Servers[s.Id] = s
	s.manager.RemoveCandidateServer(s)

	if s.IsDirtyShotdown() && !pendingDelete {
		log.Infof("Server dirty shotdown %s", s.GetName())
		if jsonutils.QueryBoolean(s.Desc, "is_master", false) ||
			jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
			s.DirtyServerRequestStart()
		} else {
			s.StartGuest(context.Background(), jsonutils.NewDict())
		}
		return
	}
	if s.IsRunning() {
		log.Infof("%s is running, pending_delete=%t", s.GetName(), pendingDelete)
		if !pendingDelete {
			s.StartMonitor(nil)
		}
	} else {
		var action = "stopped"
		if s.IsSuspend() {
			action = "suspend"
		}
		log.Infof("%s is %s, pending_delete=%t", s.GetName(), action, pendingDelete)
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

func (s *SKVMGuestInstance) StartMonitor(ctx context.Context) {
	timeutils2.AddTimeout(100*time.Millisecond, func() { s.delayStartMonitor(ctx) })
}

func (s *SKVMGuestInstance) delayStartMonitor(ctx context.Context) {
	if options.HostOptions.EnableQmpMonitor && s.GetQmpMonitorPort(-1) > 0 {
		s.Monitor = monitor.NewQmpMonitor(
			s.onMonitorDisConnect,
			func(err error) { s.onMonitorTimeout(ctx, err) },
			func() { s.onMonitorConnected(ctx) },
		)
		s.Monitor.Connect("127.0.0.1", s.GetQmpMonitorPort(-1))
	} else {
		// TODO HMP Monitor
	}
}

func (s *SKVMGuestInstance) onMonitorConnected(ctx context.Context) {
	log.Infof("Monitor connected ...")
	s.Monitor.GetVersion(func(v string) {
		s.onGetQemuVersion(ctx, v)
	})
}

func (s *SKVMGuestInstance) onGetQemuVersion(ctx context.Context, version string) {
	s.QemuVersion = version
	log.Infof("Guest(%s) qemu version %s", s.Id, s.QemuVersion)
	if s.Desc.Contains("live_migrate_dest_port") && ctx != nil {
		migratePort, _ := s.Desc.Get("live_migrate_dest_port")
		body := jsonutils.NewDict(jsonutils.NewPair("live_migrate_dest_port", migratePort))
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
	s.CleanStartupTask()
	s.scriptStop()
	if jsonutils.QueryBoolean(s.Desc, "is_slave", false) {
		// TODO
		// sync slave status
	} else {
		s.SyncStatus()
	}
	// TODO
	// s.clearCgroup()
	s.Monitor = nil
}

func (s *SKVMGuestInstance) CleanStartupTask() {
	log.Infof("Clean startup task ...")
	if s.startupTask != nil {
		s.startupTask.Stop()
		s.startupTask = nil
	}
}

func (s *SKVMGuestInstance) onMonitorTimeout(ctx context.Context, err error) {
	log.Errorf("Monitor connect timeout, VM %s frozen!! force restart!!!!", s.Id)
	s.ForceStop()
	timeutils2.AddTimeout(time.Second*3,
		func() { s.asyncScriptStart(ctx, jsonutils.NewDict()) })
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

func (s *SKVMGuestInstance) DoResumeTask(ctx context.Context) {
	s.startupTask = NewGuestResumeTask(ctx, s)
	s.startupTask.Start()
}

func (s *SKVMGuestInstance) SyncStatus() {
	if s.IsRunning() {
		s.Monitor.GetBlockJobs(s.CheckBlockOrRunning)
		return
	}
	var status = "ready"
	if s.IsSuspend() {
		status = "suspend"
	}

	hostutils.UpdateServerStatus(context.Background(), s.Id, status)
}

func (s *SKVMGuestInstance) CheckBlockOrRunning(jobs int) {
	var status = "running"
	if jobs > 0 {
		status = "block_stream"
	}
	_, err := hostutils.UpdateServerStatus(context.Background(), s.Id, status)
	if err != nil {
		log.Errorln(err)
	}
}

func (s *SKVMGuestInstance) SaveDesc(desc jsonutils.JSONObject) error {
	s.Desc = desc.(*jsonutils.JSONDict)
	if err := fileutils2.FilePutContents(s.GetDescFilePath(), desc.String(), false); err != nil {
		log.Errorln(err)
	}
	return nil
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
		_, err := procutils.NewCommand("kill", "-9", fmt.Sprintf("%d", s.GetPid())).Run()
		if err != nil {
			log.Errorln(err)
			return false
		}
		for _, f := range s.GetCleanFiles() {
			_, err := procutils.NewCommand("rm", "-f", f).Run()
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
			// s.ClearCgroup(pid)
		}
	}
	if s.Monitor != nil {
		s.Monitor.Disconnect()
		s.Monitor = nil
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
			d := storageman.GetManager().GetDiskByPath(diskPath)
			if d != nil && d.GetType() == storagetypes.STORAGE_LOCAL && migrated {
				if err := d.DeleteAllSnapshot(); err != nil {
					log.Errorln(err)
					return err
				}
				if _, err := d.Delete(ctx, nil); err != nil {
					log.Errorln(err)
					return err
				}
			}
		}
	}
	return nil
}

func (s *SKVMGuestInstance) Delete(ctx context.Context, migrated bool) error {
	// self._del_bw_limit()
	// self._del_netmon_nic() ?? 需要开发？
	if err := s.delTmpDisks(ctx, migrated); err != nil {
		return err
	}
	_, err := procutils.NewCommand("rm", "-rf", s.HomeDir()).Run()
	return err
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
	_, err := procutils.NewCommand("sh", s.GetStartScriptPath()).Run()
	if err != nil {
		s.scriptStop()
		return err
	}
	return nil
}

func (s *SKVMGuestInstance) scriptStop() bool {
	_, err := procutils.NewCommand("sh", s.GetStopScriptPath()).Run()
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

func (s *SKVMGuestInstance) compareDescNetworks(newDesc jsonutils.JSONObject) ([]jsonutils.JSONObject, []jsonutils.JSONObject) {
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
	nics, _ := newDesc.GetArray("nics")
	for _, n := range nics {
		if isValid(n) {
			addNics = append(addNics, n)
		}
	}

	nics, _ = newDesc.GetArray("nics")
	for _, n := range nics {
		if isValid(n) {
			idx := findNet(addNics, n)
			if idx >= 0 {
				// remove n
				addNics = append(addNics[:idx], addNics[idx+1:]...)
			} else {
				delNics = append(delNics, n)
			}
		}
	}
	return delNics, addNics
}

// 目测sync_cgroup没有要用到，先不写
func (s *SKVMGuestInstance) SyncConfig(ctx context.Context, desc jsonutils.JSONObject, fwOnly bool) (jsonutils.JSONObject, error) {
	var delDisks, addDisks, delNetworks, addNetworks []jsonutils.JSONObject
	var cdrom *string

	if !fwOnly {
		delDisks, addDisks = s.compareDescDisks(desc)
		cdrom = s.compareDescCdrom(desc)
		delNetworks, addNetworks = s.compareDescNetworks(desc)
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
	s.cgroupId = fmt.Sprintf("%d", s.GetPid())
	s.setCgroupIo()
	s.setCgroupCpu()
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
		cgrouputils.CgroupIoHardlimitSet(s.cgroupId, int(cpu), params, devId)
	}
}

func (s *SKVMGuestInstance) setCgroupCpu() {
	cpu, _ := s.Desc.Int("cpu")
	cgrouputils.CgroupSet(s.cgroupId, int(cpu))

	// TODO XXX
	/*
		var (
			cpuWeight = 1024
			cpuPeriod = 0
			cpuQuota  = 0
			appTags   = s.getApptags()
			meta, _   = s.Desc.Get("metadata")
		)

		if meta != nil {
			if meta.Contains("__cpu_weight") {
				cpuWeight, _ = meta.Int("__cpu_weight")
			}
			if meta.Contains("__cpu_period") {
				cpuPeriod, _ = meta.Int("__cpu_period")
			} else {
				cpuPeriod = -1
			}
			if meta.Contains("__cpu_quota") {
				cpuQuota, _ = meta.Int("__cpu_quota")
			} else {
				cpuQuota = -1
			}
		}
	*/
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
		d := storageman.GetManager().GetDiskByPath(diskpath)
		if d != nil {
			d.PostCreateFromImageFuse()
		}
		if jsonutils.QueryBoolean(disk, "merge_snapshot", false) {
			d := disks[i].(*jsonutils.JSONDict)
			d.Set("merge_snapshot", jsonutils.JSONFalse)
		}
	}
	s.SaveDesc(s.Desc)
	_, err := modules.Servers.PerformAction(hostutils.GetComputeSession(ctx),
		s.Id, "stream-disks-complete", nil)
	if err != nil {
		log.Infof("stream disks complete sync error %s", err)
	}
}

func (s *SKVMGuestInstance) GetQemuVersionStr() string {
	return s.QemuVersion
}

func (s *SKVMGuestInstance) SyncMetadata(meta *jsonutils.JSONDict) {
	_, err := modules.Servers.SetMetadata(hostutils.GetComputeSession(context.Background()),
		s.Id, meta)
	if err != nil {
		log.Errorln(err)
	}
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

// 好像不用了
func (s *SKVMGuestInstance) CleanStatefiles() {
	for _, stateFile := range s.ListStateFilePaths() {
		if _, err := procutils.NewCommand("mountpoint", stateFile).Run(); err == nil {
			if _, err = procutils.NewCommand("umount", stateFile).Run(); err != nil {
				log.Errorln(err)
			}
		}
		if _, err := procutils.NewCommand("rm", "-rf", stateFile).Run(); err != nil {
			log.Errorln(err)
		}
	}
	if _, err := procutils.NewCommand("rm", "-rf", s.GetFuseTmpPath()).Run(); err != nil {
		log.Errorln(err)
	}
}

func (s *SKVMGuestInstance) GetFuseTmpPath() string {
	return path.Join(s.HomeDir(), "tmp")
}

func (s *SKVMGuestInstance) StreamDisks(ctx context.Context, callback func(), disksIdx []int) {
	log.Infof("Start guest block stream task ...")
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
			task.WaitSnapshotReplaced(task.Start)
			return nil, nil
		} else {
			return nil, fmt.Errorf("Guest dosen't support reload disk")
		}
	} else {
		return jsonutils.NewDict(jsonutils.NewPair("reopen", jsonutils.JSONTrue)), nil
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
	location := path.Join(disk.GetSnapshotDir(), snapshotId)
	return jsonutils.NewDict(jsonutils.NewPair("localtion",
		jsonutils.NewString(location))), nil
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
	return jsonutils.NewDict(jsonutils.NewPair("deleted", jsonutils.JSONTrue)), nil
}

func (s *SKVMGuestInstance) PrepareMigrate(liveMigrage bool) (*jsonutils.JSONDict, error) {
	disksBackFile := jsonutils.NewDict()
	disks, _ := s.Desc.GetArray("disks")
	for _, disk := range disks {
		if disk.Contains("path") {
			diskPath, _ := disk.GetString("path")
			d := storageman.GetManager().GetDiskByPath(diskPath)
			if d.GetType() == storagetypes.STORAGE_LOCAL {
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
