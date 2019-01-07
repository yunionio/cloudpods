package guestman

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

const VNC_PORT_BASE = 5900

type SGuestManager struct {
	ServersPath      string
	Servers          map[string]*SKVMGuestInstance
	CandidateServers map[string]*SKVMGuestInstance
	ServersLock      *sync.Mutex

	isLoaded bool
}

func NewGuestManager(serversPath string) *SGuestManager {
	manager := &SGuestManager{}
	manager.ServersPath = serversPath
	manager.Servers = make(map[string]*SKVMGuestInstance, 0)
	manager.CandidateServers = make(map[string]*SKVMGuestInstance, 0)
	manager.ServersLock = &sync.Mutex{}
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
	params := url.Values{
		"limit":          {"0"},
		"admin":          {"True"},
		"system":         {"True"},
		"pending_delete": {fmt.Sprintf("%s", pendingDelete)},
	}
	// TODO get host id
	params.Set("filter.0", fmt.Sprintf("host_id.equals(%s)", "get host id "))
	if len(m.CandidateServers) > 0 {
		keys := make([]string, len(m.CandidateServers))
		var index = 0
		for k := range m.CandidateServers {
			keys[index] = k
			index++
		}
		params.Set("filter.1", strings.Join(keys, ","))
	}
	res, err := modules.Servers.List(hostutils.GetComputeSession(context.Background()), id, params)
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
			log.Errorf("Server %s not found on this host", server.GetName())
			unknownServerrs = append(unknownServerrs, server)
		}
		for _, server := range unknownServerrs {
			m.RemoveCandidateServer(server)
		}
	}
}

func (m *SGuestManager) RemoveCandidateServer(server *SKVMGuestInstance) {
	if _, ok := m.CandidateServers[server.GetId()]; ok {
		delete(m.CandidateServers, server.GetId())
		if len(m.CandidateServers) == 0 {
			m.OnLoadExistingGuestsComplete()
		}
	}
}

func (m *SGuestManager) OnLoadExistingGuestsComplete() {
	// TODO
}

func (m *SGuestManager) StartCpusetBalancer() {
	// TODO
}

func (m *SGuestManager) IsGuestDir(f os.FileInfo) bool {
	if !regutils.MatchUUID(f.Name()) {
		return false
	}
	if !f.Mode().IsDir() {
		return false
	}
	descFile := path.Join(m.ServersPath, f.Name(), "desc")
	if _, err := os.Stat(descFile); os.IsNotExist(err) {
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
			guest.Monitor.SimpleCommand(cmd, callback)
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

		guestInfo, err := guest.DeployFs(&guestfs.SDeployInfo{
			publicKey, deploys, password, deployParams.IsInit})
		if err != nil {
			log.Errorf("Deploy guest fs error: %s", err)
			return nil, err
		} else {
			return guestInfo, nil
		}
	} else {
		return nil, fmt.Errorf("Guest %s not found", sid)
	}
}

// delay cpuset balance
func (m *SGuestManager) CpusetBalance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	// TODO
}

func (m *SGuestManager) Status(sid string) string {
	if guest, ok := m.Servers[sid]; ok {
		// TODO
		// if guest.IsMaster() && !guest.IsMirrorJobSucc() {
		// 	return "block_stream"
		// }
		if guest.IsRunning() {
			return "running"
		} else if guest.IsSuspend() {
			return "suspend"
		} else {
			return "stopped"
		}
	} else {
		return "notfound"
	}
}

func (m *SGuestManager) Delete(sid string) (*SKVMGuestInstance, error) {
	if guest, ok := m.Servers[sid]; ok {
		delete(m.Servers, sid)
		// 这里应该不需要append到deleted servers, 据观察 deleted servers没有用到
		return guest, nil
	} else {
		return nil, httperrors.NewNotFoundError("Not found")
	}
}

func (m *SGuestManager) GuestStart(ctx context.Context, sid string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if guest, ok := m.Servers[sid]; ok {
		if desc, err := body.Get("desc"); err != nil {
			guest.SaveDesc(desc)
		}
		if guest.IsStopped() {
			params, _ := body.Get("params")
			if err := guest.StartGuest(ctx, params); err != nil {
				return nil, httperrors.NewBadRequestError("Failed to start server")
			} else {
				return jsonutils.NewDict(jsonutils.JSONPair{"vnc_port", jsonutils.NewInt(0)}), nil
			}
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
	if guest, ok := m.Servers[sid]; !ok {
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
		// TODO :SyncConfig
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

func (m *SGuestManager) DoSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	// TODO
	return nil, nil
}

func (m *SGuestManager) GetFreePortByBase(basePort int64) int64 {
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

func (m *SGuestManager) GetFreeVncPort() int64 {
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

var guestManger *SGuestManager

func Stop() {
	// guestManger.ExitGuestCleanup()
}

func Init(serversPath string) {
	if guestManger == nil {
		guestManger = NewGuestManager(serversPath)
	}
}

func GetGuestManager() *SGuestManager {
	return guestManger
}
