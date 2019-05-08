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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

const (
	VNC_PORT_BASE = 5900

	GUEST_RUNNING      = "running"
	GUEST_BLOCK_STREAM = "block_stream"
	GUEST_SUSPEND      = "suspend"
	GUSET_STOPPED      = "stopped"
	GUEST_NOT_FOUND    = "notfound"
)

type SGuestManager struct {
	host             hostutils.IHost
	ServersPath      string
	Servers          map[string]*SKVMGuestInstance
	CandidateServers map[string]*SKVMGuestInstance
	ServersLock      *sync.Mutex

	GuestStartWorker *appsrv.SWorkerManager

	isLoaded bool
}

func NewGuestManager(host hostutils.IHost, serversPath string) *SGuestManager {
	manager := &SGuestManager{}
	manager.host = host
	manager.ServersPath = serversPath
	manager.Servers = make(map[string]*SKVMGuestInstance, 0)
	manager.CandidateServers = make(map[string]*SKVMGuestInstance, 0)
	manager.ServersLock = &sync.Mutex{}
	manager.GuestStartWorker = appsrv.NewWorkerManager("GuestStart", 1, appsrv.DEFAULT_BACKLOG, false)
	manager.StartCpusetBalancer()
	manager.LoadExistingGuests()
	return manager
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
		var unknownServerrs = make([]*SKVMGuestInstance, 0)
		for _, server := range m.CandidateServers {
			go server.RequestVerifyDirtyServer()
			log.Errorf("Server %s not found on this host", server.GetName())
			unknownServerrs = append(unknownServerrs, server)
		}
		for _, server := range unknownServerrs {
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
	for _, guest := range m.Servers {
		guest.CleanupCpuset()
	}
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
	if _, ok := guestManger.Servers[sid]; !ok {
		return false
	} else {
		return true
	}
}

func (m *SGuestManager) GetGuestById(sid string) *SKVMGuestInstance {
	guest, _ := guestManger.Servers[sid]
	return guest
}

func (m *SGuestManager) LoadExistingGuests() {
	files, err := ioutil.ReadDir(m.ServersPath)
	if err != nil {
		log.Errorf("List servers path %s error %s", m.ServersPath, err)
	}
	for _, f := range files {
		if _, ok := m.Servers[f.Name()]; !ok && m.IsGuestDir(f) {
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

//isDeleted先不加，目测只是在ofp中用到了
func (m *SGuestManager) GetGuestNicDesc(mac, ip, port, bridge string, isCandidate bool) (jsonutils.JSONObject, jsonutils.JSONObject) {
	servers := m.Servers
	if isCandidate {
		servers = m.CandidateServers
	}
	for _, guest := range servers {
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
	if _, ok := m.Servers[sid]; ok {
		return httperrors.NewBadRequestError("Guest %s exists", sid)
	}
	guest := NewKVMGuestInstance(sid, m)
	m.Servers[sid] = guest
	return guest.PrepareDir()
}

func (m *SGuestManager) PrepareDeploy(sid string) error {
	m.ServersLock.Lock()
	defer m.ServersLock.Unlock()
	if guest, ok := m.Servers[sid]; !ok {
		return httperrors.NewBadRequestError("Guest %s not exists", sid)
	} else {
		if guest.IsRunning() || guest.IsSuspend() {
			return httperrors.NewBadRequestError("Cannot deploy on running/suspend guest")
		}
	}
	return nil
}

func (m *SGuestManager) Monitor(sid, cmd string, callback func(string)) error {
	if guest, ok := m.Servers[sid]; ok {
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

// Delay process
func (m *SGuestManager) GuestDeploy(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	deployParams, ok := params.(*SGuestDeploy)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, ok := m.Servers[deployParams.Sid]
	if ok {
		desc, _ := deployParams.Body.Get("desc")
		if desc != nil {
			guest.SaveDesc(desc)
		}
		if jsonutils.QueryBoolean(deployParams.Body, "k8s_pod", false) {
			return nil, nil
		}
		publicKey := sshkeys.GetKeys(deployParams.Body)
		deploys, _ := deployParams.Body.GetArray("deploys")
		password, _ := deployParams.Body.GetString("password")
		resetPassword := jsonutils.QueryBoolean(deployParams.Body, "reset_password", false)
		if resetPassword && len(password) == 0 {
			password = seclib.RandomPassword(12)
		}

		guestInfo, err := guest.DeployFs(guestfs.NewDeployInfo(
			publicKey, deploys, password, deployParams.IsInit, false,
			options.HostOptions.LinuxDefaultRootUser, options.HostOptions.WindowsDefaultAdminUser))
		if err != nil {
			log.Errorf("Deploy guest fs error: %s", err)
			return nil, err
		} else {
			return guestInfo, nil
		}
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
	if status == GUEST_RUNNING && m.Servers[sid].Monitor == nil && !m.Servers[sid].IsStopping() {
		m.Servers[sid].StartMonitor(context.Background())
	}
	return status
}

func (m *SGuestManager) GetStatus(sid string) string {
	if guest, ok := m.Servers[sid]; ok {
		if guest.Monitor != nil && guest.IsMaster() && !guest.IsMirrorJobSucc() {
			return GUEST_BLOCK_STREAM
		}
		if guest.IsRunning() {
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
	if guest, ok := m.Servers[sid]; ok {
		delete(m.Servers, sid)
		// 这里应该不需要append到deleted servers
		// 据观察 deleted servers 目的是为了给ofp_delegate使用，ofp已经不用了
		return guest, nil
	} else {
		return nil, httperrors.NewNotFoundError("Not found")
	}
}

func (m *SGuestManager) GuestStart(ctx context.Context, sid string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if guest, ok := m.Servers[sid]; ok {
		if desc, err := body.Get("desc"); err == nil {
			guest.SaveDesc(desc)
		}
		if guest.IsStopped() {
			params, _ := body.Get("params")
			guest.StartGuest(ctx, params)
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
	if guest, ok := m.Servers[sid]; ok {
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
	guest := m.Servers[syncParams.Sid]
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
	guest := m.Servers[sid]
	guest.ExecSuspendTask(ctx)
	return nil, nil
}

func (m *SGuestManager) SrcPrepareMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SSrcPrepareMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest := m.Servers[migParams.Sid]
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

	guest := m.Servers[migParams.Sid]
	if err := guest.CreateFromDesc(migParams.Desc); err != nil {
		return nil, err
	}

	if len(migParams.TargetStorageId) > 0 {
		iStorage := storageman.GetManager().GetStorage(migParams.TargetStorageId)
		if iStorage == nil {
			return nil, fmt.Errorf("Target storage %s not found", migParams.TargetStorageId)
		}

		// 可能可以不用?
		// guest.CreateFromUrl(ctx, migParams.ServerUrl, migParams.Desc)

		disks, _ := migParams.Desc.GetArray("disks")
		for i, diskinfo := range disks {
			var (
				diskId, _    = diskinfo.GetString("disk_id")
				snapshots, _ = migParams.SrcSnapshots.GetArray(diskId)
				disk         = iStorage.CreateDisk(diskId)
			)

			if disk == nil {
				return nil, fmt.Errorf(
					"Storage %s create disk %s failed", iStorage.GetId(), diskId)
			}

			// prepare disk snapshot dir
			if len(snapshots) > 0 && !fileutils2.Exists(disk.GetSnapshotDir()) {
				_, err := procutils.NewCommand("mkdir", "-p", disk.GetSnapshotDir()).Run()
				if err != nil {
					return nil, err
				}
			}

			// create snapshots form remote url
			diskStorageId, _ := diskinfo.GetString("storage_id")
			for _, snapshotId := range snapshots {
				snapId, _ := snapshotId.GetString()

				snapshotUrl := fmt.Sprintf("%s/%s/%s/%s",
					migParams.SnapshotsUri, diskStorageId, diskId, snapId)
				snapshotPath := path.Join(disk.GetSnapshotDir(), snapId)
				log.Infof("Disk %s snapshot %s url: %s", diskId, snapId, snapshotUrl)
				iStorage.CreateSnapshotFormUrl(ctx, snapshotUrl, diskId, snapshotPath)
			}

			if migParams.LiveMigrate {
				// create local disk
				backingFile, _ := migParams.DisksBackingFile.GetString(diskId)
				size, _ := diskinfo.Int("size")
				_, err := disk.CreateRaw(ctx, int(size), "qcow2", "", false, "", backingFile)
				if err != nil {
					log.Errorln(err)
					return nil, err
				}
			} else {
				// download disk form remote url
				diskUrl := fmt.Sprintf("%s/%s/%s", migParams.DisksUri, diskStorageId, diskId)
				if err := disk.CreateFromUrl(ctx, diskUrl); err != nil {
					log.Errorln(err)
					return nil, err
				}
			}
			diskDesc, _ := disks[i].(*jsonutils.JSONDict)
			diskDesc.Set("path", jsonutils.NewString(disk.GetPath()))
		}

		// 可能可以不要
		if err := guest.SaveDesc(migParams.Desc); err != nil {
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

	guest := m.Servers[migParams.Sid]
	task := NewGuestLiveMigrateTask(ctx, guest, migParams)
	task.Start()
	return nil, nil
}

func (m *SGuestManager) CanMigrate(sid string) bool {
	m.ServersLock.Lock()
	defer m.ServersLock.Unlock()

	if _, ok := m.Servers[sid]; ok {
		log.Infof("Guest %s exists", sid)
		return false
	}

	guest := NewKVMGuestInstance(sid, m)
	m.Servers[sid] = guest
	return true
}

func (m *SGuestManager) GetFreePortByBase(basePort int) int {
	var port = 1
	for {
		if netutils2.IsTcpPortUsed("0.0.0.0", basePort+port) ||
			netutils2.IsTcpPortUsed("127.0.0.1", basePort+port) {
			port += 1
		} else {
			return basePort + port
		}
	}
}

func (m *SGuestManager) GetFreeVncPort() int {
	vncPorts := make(map[int]struct{}, 0)
	for _, guest := range m.Servers {
		inUsePort := guest.GetVncPort()
		if inUsePort > 0 {
			vncPorts[inUsePort] = struct{}{}
		}
	}
	var port = 1
	for {
		if _, ok := vncPorts[port]; !ok && !netutils2.IsTcpPortUsed("0.0.0.0", VNC_PORT_BASE+port) &&
			!netutils2.IsTcpPortUsed("127.0.0.1", MONITOR_PORT_BASE+port) {
			break
		} else {
			port += 1
		}
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
	guest := guestManger.Servers[reloadParams.Sid]
	return guest.ExecReloadDiskTask(ctx, reloadParams.Disk)
}

func (m *SGuestManager) DoSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotParams, ok := params.(*SDiskSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest := guestManger.Servers[snapshotParams.Sid]
	return guest.ExecDiskSnapshotTask(ctx, snapshotParams.Disk, snapshotParams.SnapshotId)
}

func (m *SGuestManager) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	delParams, ok := params.(*SDeleteDiskSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if len(delParams.ConvertSnapshot) > 0 {
		guest := guestManger.Servers[delParams.Sid]
		return guest.ExecDeleteSnapshotTask(ctx, delParams.Disk, delParams.DeleteSnapshot,
			delParams.ConvertSnapshot, delParams.PendingDelete)
	} else {
		res := jsonutils.NewDict()
		res.Set("deleted", jsonutils.JSONTrue)
		return res, delParams.Disk.DeleteSnapshot(delParams.DeleteSnapshot, "", false)
	}
}

func (m *SGuestManager) Resume(ctx context.Context, sid string, isLiveMigrate bool) (jsonutils.JSONObject, error) {
	guest := guestManger.Servers[sid]
	resumeTask := NewGuestResumeTask(ctx, guest)
	if isLiveMigrate {
		guest.StartPresendArp()
	}
	resumeTask.Start()
	return nil, nil
}

func (m *SGuestManager) OnlineResizeDisk(ctx context.Context, sid string, diskId string, sizeMb int64) (jsonutils.JSONObject, error) {
	guest, ok := guestManger.Servers[sid]
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
	guest := guestManger.Servers[mirrorParams.Sid]
	if err := guest.SaveDesc(mirrorParams.Desc); err != nil {
		return nil, err
	}
	task := NewDriveMirrorTask(ctx, guest, mirrorParams.NbdServerUri, "top", nil)
	task.Start()
	return nil, nil
}

func (m *SGuestManager) HotplugCpuMem(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	hotplugParams, ok := params.(*SGuestHotplugCpuMem)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest := guestManger.Servers[hotplugParams.Sid]
	NewGuestHotplugCpuMemTask(ctx, guest, int(hotplugParams.AddCpuCount), int(hotplugParams.AddMemSize)).Start()
	return nil, nil
}

func (m *SGuestManager) ExitGuestCleanup() {
	for _, guest := range m.Servers {
		guest.ExitCleanup(false)
	}

	cgrouputils.CgroupCleanAll()
}

func (m *SGuestManager) GetHost() hostutils.IHost {
	return m.host
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
