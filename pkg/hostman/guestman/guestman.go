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
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"

	compute "yunion.io/x/onecloud/pkg/apis/compute"
	appsrv "yunion.io/x/onecloud/pkg/appsrv"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	hostutils "yunion.io/x/onecloud/pkg/hostman/hostutils"
	options "yunion.io/x/onecloud/pkg/hostman/options"
	storageman "yunion.io/x/onecloud/pkg/hostman/storageman"
	httperrors "yunion.io/x/onecloud/pkg/httperrors"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	cgrouputils "yunion.io/x/onecloud/pkg/util/cgrouputils"
	fileutils2 "yunion.io/x/onecloud/pkg/util/fileutils2"
	netutils2 "yunion.io/x/onecloud/pkg/util/netutils2"
	timeutils2 "yunion.io/x/onecloud/pkg/util/timeutils2"
)

var (
	LAST_USED_PORT = 0
	NbdWorker      = appsrv.NewWorkerManager("nbd_worker", 1, appsrv.DEFAULT_BACKLOG, false)
)

const (
	VNC_PORT_BASE           = 5900
	GUEST_RUNNING           = compute.VM_RUNNING
	GUEST_BLOCK_STREAM      = compute.VM_BLOCK_STREAM
	GUEST_BLOCK_STREAM_FAIL = compute.VM_BLOCK_STREAM_FAIL
	GUEST_SUSPEND           = compute.VM_SUSPEND
	GUSET_STOPPED           = "stopped"
	GUEST_NOT_FOUND         = "notfound"
)

type SGuestManager struct {
	host             hostutils.IHost
	ServersPath      string
	Servers          *sync.Map
	CandidateServers map[string]*SKVMGuestInstance
	UnknownServers   *sync.Map
	ServersLock      *sync.Mutex

	GuestStartWorker *appsrv.SWorkerManager

	isLoaded bool
}

func NewGuestManager(host hostutils.IHost, serversPath string) *SGuestManager {
	manager := &SGuestManager{}
	manager.host = host
	manager.ServersPath = serversPath
	manager.Servers = new(sync.Map)
	manager.CandidateServers = make(map[string]*SKVMGuestInstance, 0)
	manager.UnknownServers = new(sync.Map)
	manager.ServersLock = &sync.Mutex{}
	manager.GuestStartWorker = appsrv.NewWorkerManager("GuestStart", 1, appsrv.DEFAULT_BACKLOG, false)
	manager.StartCpusetBalancer()
	manager.LoadExistingGuests()
	manager.host.StartDHCPServer()
	return manager
}

func (m *SGuestManager) GetServer(sid string) (*SKVMGuestInstance, bool) {
	s, ok := m.Servers.Load(sid)
	if ok {
		return s.(*SKVMGuestInstance), ok
	} else {
		return nil, ok
	}
}

func (m *SGuestManager) GetUnknownServer(sid string) (*SKVMGuestInstance, bool) {
	s, ok := m.UnknownServers.Load(sid)
	if ok {
		return s.(*SKVMGuestInstance), ok
	} else {
		return nil, ok
	}
}

func (m *SGuestManager) SaveServer(sid string, s *SKVMGuestInstance) {
	m.Servers.Store(sid, s)
}

func (m *SGuestManager) Bootstrap() {
	if m.isLoaded || len(m.ServersPath) == 0 {
		log.Errorln("Guestman bootstrap has been called!!!!!")
	} else {
		m.isLoaded = true
		log.Infof("Loading existing guests ...")
		if len(m.CandidateServers) > 0 {
			m.VerifyExistingGuests(false)
		} else {
			m.OnLoadExistingGuestsComplete()
		}
	}
}

func (m *SGuestManager) VerifyExistingGuests(pendingDelete bool) {
	params := jsonutils.NewDict()
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("admin", jsonutils.JSONTrue)
	params.Set("system", jsonutils.JSONTrue)
	params.Set("host", jsonutils.NewString(m.host.GetHostId()))
	params.Set("pending_delete", jsonutils.NewBool(pendingDelete))
	params.Set("get_backup_guests_on_host", jsonutils.JSONTrue)
	if len(m.CandidateServers) > 0 {
		keys := make([]string, len(m.CandidateServers))
		var index = 0
		for k := range m.CandidateServers {
			keys[index] = k
			index++
		}
		params.Set("filter.1", jsonutils.NewString(fmt.Sprintf("id.in(%s)", strings.Join(keys, ","))))
	}
	res, err := modules.Servers.List(hostutils.GetComputeSession(context.Background()), params)
	if err != nil {
		m.OnVerifyExistingGuestsFail(err, pendingDelete)
	} else {
		m.OnVerifyExistingGuestsSucc(res.Data, pendingDelete)
	}
}

func (m *SGuestManager) OnVerifyExistingGuestsFail(err error, pendingDelete bool) {
	log.Errorf("OnVerifyExistingGuestFail: %s, try again 30 seconds later", err.Error())
	timeutils2.AddTimeout(30*time.Second, func() { m.VerifyExistingGuests(false) })
}

func (m *SGuestManager) OnVerifyExistingGuestsSucc(servers []jsonutils.JSONObject, pendingDelete bool) {
	for _, v := range servers {
		id, _ := v.GetString("id")
		server, ok := m.CandidateServers[id]
		if !ok {
			log.Errorf("verify_existing_guests return unknown server %s ???????", id)
		} else {
			server.ImportServer(pendingDelete)
		}
	}
	if !pendingDelete {
		m.VerifyExistingGuests(true)
	} else {
		for id, server := range m.CandidateServers {
			m.UnknownServers.Store(id, server)
			go m.RequestVerifyDirtyServer(server)
			log.Errorf("Server %s not found on this host", server.GetName())
			m.RemoveCandidateServer(server)
		}
	}
}

func (m *SGuestManager) RemoveCandidateServer(server *SKVMGuestInstance) {
	if _, ok := m.CandidateServers[server.Id]; ok {
		delete(m.CandidateServers, server.Id)
		if len(m.CandidateServers) == 0 {
			m.OnLoadExistingGuestsComplete()
		}
	}
}

func (m *SGuestManager) OnLoadExistingGuestsComplete() {
	log.Infof("Load existing guests complete...")
	err := m.host.PutHostOnline()
	if err != nil {
		log.Errorln(err)
	}

	if !options.HostOptions.EnableCpuBinding {
		m.ClenaupCpuset()
	}
}

func (m *SGuestManager) ClenaupCpuset() {
	m.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		guest.CleanupCpuset()
		return true
	})
}

func (m *SGuestManager) StartCpusetBalancer() {
	if !options.HostOptions.EnableCpuBinding {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
				log.Errorf("Cpuset balancer failed %s", r)
			}
		}()
		for {
			time.Sleep(time.Second * 120)

			if options.HostOptions.EnableCpuBinding {
				m.cpusetBalance()
			}
		}
	}()
}

func (m *SGuestManager) cpusetBalance() {
	cgrouputils.RebalanceProcesses(nil)
}

func (m *SGuestManager) IsGuestDir(f os.FileInfo) bool {
	if !regutils.MatchUUID(f.Name()) {
		return false
	}
	if !f.Mode().IsDir() {
		return false
	}
	descFile := path.Join(m.ServersPath, f.Name(), "desc")
	if !fileutils2.Exists(descFile) {
		return false
	}
	return true
}

func (m *SGuestManager) IsGuestExist(sid string) bool {
	if _, ok := m.GetServer(sid); !ok {
		return false
	} else {
		return true
	}
}

func (m *SGuestManager) LoadExistingGuests() {
	files, err := ioutil.ReadDir(m.ServersPath)
	if err != nil {
		log.Errorf("List servers path %s error %s", m.ServersPath, err)
	}
	for _, f := range files {
		if _, ok := m.GetServer(f.Name()); !ok && m.IsGuestDir(f) {
			log.Infof("Find existing guest %s", f.Name())
			m.LoadServer(f.Name())
		}
	}
}

func (m *SGuestManager) LoadServer(sid string) {
	guest := NewKVMGuestInstance(sid, m)
	err := guest.LoadDesc()
	if err != nil {
		log.Errorf("On load server error: %s", err)
		return
	}
	m.CandidateServers[sid] = guest
}

func (m *SGuestManager) ShutdownSharedStorageServers() {
	m.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		if guest.IsSharedStorage() {
			log.Infof("Start shutdown server %s", guest.GetName())
			if !guest.scriptStop() {
				log.Errorf("shutdown server %s failed", guest.GetName())
			}
		}
		return true
	})
}

func (m *SGuestManager) GetGuestNicDesc(mac, ip, port, bridge string, isCandidate bool) (jsonutils.JSONObject, jsonutils.JSONObject) {
	if isCandidate {
		return m.getGuestNicDescInCandidate(mac, ip, port, bridge)
	}
	var nic jsonutils.JSONObject
	var guestDesc jsonutils.JSONObject
	m.Servers.Range(func(k interface{}, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		if guest.IsLoaded() {
			nic = guest.GetNicDescMatch(mac, ip, port, bridge)
			if nic != nil {
				guestDesc = guest.Desc
				return false
			}
		}
		return true
	})
	return guestDesc, nic
}

func (m *SGuestManager) getGuestNicDescInCandidate(mac, ip, port, bridge string) (jsonutils.JSONObject, jsonutils.JSONObject) {
	for _, guest := range m.CandidateServers {
		if guest.IsLoaded() {
			nic := guest.GetNicDescMatch(mac, ip, port, bridge)
			if nic != nil {
				return guest.Desc, nic
			}
		}
	}
	return nil, nil
}

func (m *SGuestManager) PrepareCreate(sid string) error {
	m.ServersLock.Lock()
	defer m.ServersLock.Unlock()
	if _, ok := m.GetServer(sid); ok {
		return httperrors.NewBadRequestError("Guest %s exists", sid)
	}
	guest := NewKVMGuestInstance(sid, m)
	m.SaveServer(sid, guest)
	return guest.PrepareDir()
}

func (m *SGuestManager) PrepareDeploy(sid string) error {
	if guest, ok := m.GetServer(sid); !ok {
		return httperrors.NewBadRequestError("Guest %s not exists", sid)
	} else {
		if guest.IsRunning() || guest.IsSuspend() {
			return httperrors.NewBadRequestError("Cannot deploy on running/suspend guest")
		}
	}
	return nil
}

func (m *SGuestManager) Monitor(sid, cmd string, callback func(string)) error {
	if guest, ok := m.GetServer(sid); ok {
		if guest.IsRunning() {
			guest.Monitor.HumanMonitorCommand(cmd, callback)
			return nil
		} else {
			return httperrors.NewBadRequestError("Server stopped??")
		}
	} else {
		return httperrors.NewNotFoundError("Not found")
	}
}

func (m *SGuestManager) GuestCreate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	deployParams, ok := params.(*SGuestDeploy)
	if !ok {
		return nil, hostutils.ParamsError
	}

	var guest *SKVMGuestInstance
	err := func() error {
		m.ServersLock.Lock()
		defer m.ServersLock.Unlock()
		if _, ok := m.GetServer(deployParams.Sid); ok {
			return httperrors.NewBadRequestError("Guest %s exists", deployParams.Sid)
		}
		guest = NewKVMGuestInstance(deployParams.Sid, m)
		desc, _ := deployParams.Body.Get("desc")
		if desc != nil {
			err := guest.PrepareDir()
			if err != nil {
				return errors.Wrap(err, "guest prepare dir")
			}
			err = guest.SaveDesc(desc)
			if err != nil {
				return errors.Wrap(err, "save desc")
			}
		}
		m.SaveServer(deployParams.Sid, guest)
		return nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, "prepare guest")
	}
	return m.startDeploy(ctx, deployParams, guest)
}

func (m *SGuestManager) startDeploy(
	ctx context.Context, deployParams *SGuestDeploy, guest *SKVMGuestInstance) (jsonutils.JSONObject, error) {

	if jsonutils.QueryBoolean(deployParams.Body, "k8s_pod", false) {
		return nil, nil
	}
	publicKey := deployapi.GetKeys(deployParams.Body)
	deploys, _ := deployParams.Body.GetArray("deploys")
	password, _ := deployParams.Body.GetString("password")
	resetPassword := jsonutils.QueryBoolean(deployParams.Body, "reset_password", false)
	if resetPassword && len(password) == 0 {
		password = seclib.RandomPassword(12)
	}
	enableCloudInit := jsonutils.QueryBoolean(deployParams.Body, "enable_cloud_init", false)

	guestInfo, err := guest.DeployFs(deployapi.NewDeployInfo(
		publicKey, deployapi.JsonDeploysToStructs(deploys), password, deployParams.IsInit, false,
		options.HostOptions.LinuxDefaultRootUser, options.HostOptions.WindowsDefaultAdminUser, enableCloudInit))
	if err != nil {
		return nil, errors.Wrap(err, "Deploy guest fs")
	} else {
		return guestInfo, nil
	}
}

// Delay process
func (m *SGuestManager) GuestDeploy(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	deployParams, ok := params.(*SGuestDeploy)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, ok := m.GetServer(deployParams.Sid)
	if ok {
		desc, _ := deployParams.Body.Get("desc")
		if desc != nil {
			guest.SaveDesc(desc)
		}
		return m.startDeploy(ctx, deployParams, guest)
	} else {
		return nil, fmt.Errorf("Guest %s not found", deployParams.Sid)
	}
}

// delay cpuset balance
func (m *SGuestManager) CpusetBalance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	m.cpusetBalance()
	return nil, nil
}

func (m *SGuestManager) Status(sid string) string {
	status := m.GetStatus(sid)
	if status == GUEST_RUNNING {
		guest, _ := m.GetServer(sid)
		if guest.Monitor == nil && !guest.IsStopping() {
			guest.StartMonitor(context.Background())
		}
	}
	return status
}

func (m *SGuestManager) GetStatus(sid string) string {
	if guest, ok := m.GetServer(sid); ok {
		if guest.IsRunning() && guest.Monitor != nil && guest.IsMaster() {
			mirrorStatus := guest.MirrorJobStatus()
			if mirrorStatus.InProcess() {
				return GUEST_BLOCK_STREAM
			} else if mirrorStatus.IsFailed() {
				timeutils2.AddTimeout(1*time.Second,
					func() { guest.SyncMirrorJobFailed("Block job missing") })
				return GUEST_BLOCK_STREAM_FAIL
			}
		}
		if guest.IsRunning() {
			if guest.Monitor != nil && guest.BlockJobsCount() > 0 {
				return GUEST_BLOCK_STREAM
			}
			return GUEST_RUNNING
		} else if guest.IsSuspend() {
			return GUEST_SUSPEND
		} else {
			return GUSET_STOPPED
		}
	} else {
		return GUEST_NOT_FOUND
	}
}

func (m *SGuestManager) Delete(sid string) (*SKVMGuestInstance, error) {
	if guest, ok := m.GetServer(sid); ok {
		m.Servers.Delete(sid)
		// 这里应该不需要append到deleted servers
		// 据观察 deleted servers 目的是为了给ofp_delegate使用，ofp已经不用了
		return guest, nil
	} else if guest, ok := m.GetUnknownServer(sid); ok {
		m.UnknownServers.Delete(sid)
		return guest, nil
	} else {
		return nil, httperrors.NewNotFoundError("Not found")
	}
}

func (m *SGuestManager) GuestStart(ctx context.Context, sid string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if guest, ok := m.GetServer(sid); ok {
		if desc, err := body.Get("desc"); err == nil {
			guest.SaveDesc(desc)
		}
		if guest.IsStopped() {
			var data *jsonutils.JSONDict
			params, err := body.Get("params")
			if err != nil {
				data = params.(*jsonutils.JSONDict)
			}
			guest.StartGuest(ctx, data)
			res := jsonutils.NewDict()
			res.Set("vnc_port", jsonutils.NewInt(0))
			return res, nil
		} else {
			vncPort := guest.GetVncPort()
			if vncPort > 0 {
				res := jsonutils.NewDict()
				res.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
				res.Set("is_running", jsonutils.JSONTrue)
				return res, nil
			} else {
				return nil, httperrors.NewBadRequestError("Seems started, but no VNC info")
			}
		}
	} else {
		return nil, httperrors.NewNotFoundError("Not found")
	}
}

func (m *SGuestManager) GuestStop(ctx context.Context, sid string, timeout int64) error {
	if guest, ok := m.GetServer(sid); ok {
		hostutils.DelayTaskWithoutReqctx(ctx, guest.ExecStopTask, timeout)
		return nil
	} else {
		return httperrors.NewNotFoundError("Guest %s not found", sid)
	}
}

func (m *SGuestManager) GuestSync(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	syncParams, ok := params.(*SBaseParms)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(syncParams.Sid)
	if syncParams.Body.Contains("desc") {
		desc, _ := syncParams.Body.Get("desc")
		fwOnly := jsonutils.QueryBoolean(syncParams.Body, "fw_only", false)
		return guest.SyncConfig(ctx, desc, fwOnly)
	}
	return nil, nil
}

func (m *SGuestManager) GuestSuspend(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sid, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, ok := m.GetServer(sid)
	guest.ExecSuspendTask(ctx)
	return nil, nil
}

func (m *SGuestManager) GuestIoThrottle(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	guestIoThrottle, ok := params.(*SGuestIoThrottle)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(guestIoThrottle.Sid)
	if guest.IsRunning() {
		return nil, guest.BlockIoThrottle(ctx, guestIoThrottle.BPS, guestIoThrottle.IOPS)
	}
	return nil, httperrors.NewInvalidStatusError("Guest not running")
}

func (m *SGuestManager) SrcPrepareMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SSrcPrepareMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(migParams.Sid)
	disksPrepare, err := guest.PrepareMigrate(migParams.LiveMigrate)
	if err != nil {
		return nil, err
	}
	if disksPrepare.Length() > 0 {
		ret := jsonutils.NewDict()
		ret.Set("disks_back", disksPrepare)
		return ret, nil
	}
	return nil, nil
}

func (m *SGuestManager) DestPrepareMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SDestPrepareMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, _ := m.GetServer(migParams.Sid)
	if err := guest.CreateFromDesc(migParams.Desc); err != nil {
		return nil, err
	}

	if len(migParams.TargetStorageId) > 0 {
		iStorage := storageman.GetManager().GetStorage(migParams.TargetStorageId)
		if iStorage == nil {
			return nil, fmt.Errorf("Target storage %s not found", migParams.TargetStorageId)
		}

		err := iStorage.DestinationPrepareMigrate(
			ctx, migParams.LiveMigrate, migParams.DisksUri, migParams.SnapshotsUri,
			migParams.Desc, migParams.DisksBackingFile, migParams.SrcSnapshots, migParams.RebaseDisks)
		if err != nil {
			return nil, fmt.Errorf("dest prepare migrate failed %s", err)
		}

		if err = guest.SaveDesc(migParams.Desc); err != nil {
			log.Errorln(err)
			return nil, err
		}
	}

	if migParams.LiveMigrate {
		startParams := jsonutils.NewDict()
		startParams.Set("qemu_version", jsonutils.NewString(migParams.QemuVersion))
		startParams.Set("need_migrate", jsonutils.JSONTrue)
		hostutils.DelayTaskWithoutReqctx(ctx, guest.asyncScriptStart, startParams)
	}

	return nil, nil
}

func (m *SGuestManager) LiveMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SLiveMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, _ := m.GetServer(migParams.Sid)
	task := NewGuestLiveMigrateTask(ctx, guest, migParams)
	task.Start()
	return nil, nil
}

func (m *SGuestManager) CanMigrate(sid string) bool {
	m.ServersLock.Lock()
	defer m.ServersLock.Unlock()

	if _, ok := m.GetServer(sid); ok {
		log.Infof("Guest %s exists", sid)
		return false
	}

	guest := NewKVMGuestInstance(sid, m)
	m.Servers.Store(sid, guest)
	return true
}

func (m *SGuestManager) GetFreePortByBase(basePort int) int {
	var port = 1
	for {
		if netutils2.IsTcpPortUsed("0.0.0.0", basePort+port) {
			port += 1
		} else {
			return basePort + port
		}
	}
}

func (m *SGuestManager) GetFreeVncPort() int {
	vncPorts := make(map[int]struct{}, 0)
	m.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		inUsePort := guest.GetVncPort()
		if inUsePort > 0 {
			vncPorts[inUsePort] = struct{}{}
		}
		return true
	})
	var port = LAST_USED_PORT + 1
	for {
		if _, ok := vncPorts[port]; !ok &&
			!netutils2.IsTcpPortUsed("0.0.0.0", VNC_PORT_BASE+port) &&
			!netutils2.IsTcpPortUsed("127.0.0.1", MONITOR_PORT_BASE+port) {
			break
		} else {
			port += 1
		}
	}
	LAST_USED_PORT = port
	if LAST_USED_PORT > 5000 {
		LAST_USED_PORT = 0
	}
	return port
}

func (m *SGuestManager) ReloadDiskSnapshot(
	ctx context.Context, params interface{},
) (jsonutils.JSONObject, error) {
	reloadParams, ok := params.(*SReloadDisk)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(reloadParams.Sid)
	return guest.ExecReloadDiskTask(ctx, reloadParams.Disk)
}

func (m *SGuestManager) DoSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotParams, ok := params.(*SDiskSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(snapshotParams.Sid)
	return guest.ExecDiskSnapshotTask(ctx, snapshotParams.Disk, snapshotParams.SnapshotId)
}

func (m *SGuestManager) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	delParams, ok := params.(*SDeleteDiskSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if len(delParams.ConvertSnapshot) > 0 {
		guest, _ := m.GetServer(delParams.Sid)
		return guest.ExecDeleteSnapshotTask(ctx, delParams.Disk, delParams.DeleteSnapshot,
			delParams.ConvertSnapshot, delParams.PendingDelete)
	} else {
		res := jsonutils.NewDict()
		res.Set("deleted", jsonutils.JSONTrue)
		return res, delParams.Disk.DeleteSnapshot(delParams.DeleteSnapshot, "", false)
	}
}

func (m *SGuestManager) Resume(ctx context.Context, sid string, isLiveMigrate bool) (jsonutils.JSONObject, error) {
	guest, _ := m.GetServer(sid)
	resumeTask := NewGuestResumeTask(ctx, guest)
	if isLiveMigrate {
		guest.StartPresendArp()
	}
	resumeTask.Start()
	return nil, nil
}

func (m *SGuestManager) OnlineResizeDisk(ctx context.Context, sid string, diskId string, sizeMb int64) (jsonutils.JSONObject, error) {
	guest, ok := m.GetServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("guest %s not found", sid)
	}
	if guest.IsRunning() {
		guest.onlineResizeDisk(ctx, diskId, sizeMb)
		return nil, nil
	} else {
		return nil, httperrors.NewInvalidStatusError("guest is not runnign")
	}
}

// func (m *SGuestManager) StartNbdServer(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
// 	sid, ok := params.(string)
// 	if !ok {
// 		return nil, hostutils.ParamsError
// 	}
// 	guest := guestManger.Servers[sid]
// 	port := m.GetFreePortByBase(BUILT_IN_NBD_SERVER_PORT_BASE)

// }

func (m *SGuestManager) StartDriveMirror(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	mirrorParams, ok := params.(*SDriverMirror)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(mirrorParams.Sid)
	if err := guest.SaveDesc(mirrorParams.Desc); err != nil {
		return nil, err
	}
	task := NewDriveMirrorTask(ctx, guest, mirrorParams.NbdServerUri, "top", nil)
	task.Start()
	return nil, nil
}

func (m *SGuestManager) CancelBlockJobs(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sid, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("STACK: %v \n %s", r, debug.Stack())
			hostutils.TaskFailed(ctx, fmt.Sprintf("recover: %v", r))
		}
	}()
	guest, _ := m.GetServer(sid)
	NewCancelBlockJobsTask(ctx, guest).Start()
	return nil, nil
}

func (m *SGuestManager) HotplugCpuMem(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	hotplugParams, ok := params.(*SGuestHotplugCpuMem)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(hotplugParams.Sid)
	NewGuestHotplugCpuMemTask(ctx, guest, int(hotplugParams.AddCpuCount), int(hotplugParams.AddMemSize)).Start()
	return nil, nil
}

func (m *SGuestManager) ExitGuestCleanup() {
	m.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		guest.ExitCleanup(false)
		return true
	})

	cgrouputils.CgroupCleanAll()
}

func (m *SGuestManager) GetHost() hostutils.IHost {
	return m.host
}

func (m *SGuestManager) RequestVerifyDirtyServer(s *SKVMGuestInstance) {
	hostId, _ := s.Desc.GetString("host_id")
	var body = jsonutils.NewDict()
	body.Set("guest_id", jsonutils.NewString(s.Id))
	body.Set("host_id", jsonutils.NewString(hostId))
	ret, err := modules.Servers.PerformClassAction(
		hostutils.GetComputeSession(context.Background()), "dirty-server-verify", body)
	if err != nil {
		log.Errorf("Dirty server request start error: %s", err)
	} else if jsonutils.QueryBoolean(ret, "guest_unknown_need_clean", false) {
		m.Delete(s.Id)
		s.CleanGuest(context.Background(), true)
	}
}

var guestManger *SGuestManager

func Stop() {
	guestManger.ExitGuestCleanup()
}

func Init(host hostutils.IHost, serversPath string) {
	if guestManger == nil {
		guestManger = NewGuestManager(host, serversPath)
	}
}

func GetGuestManager() *SGuestManager {
	return guestManger
}
