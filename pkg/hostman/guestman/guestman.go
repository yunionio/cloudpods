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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/guestman/arch"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	fwd "yunion.io/x/onecloud/pkg/hostman/guestman/forwarder"
	fwdpb "yunion.io/x/onecloud/pkg/hostman/guestman/forwarder/api"
	"yunion.io/x/onecloud/pkg/hostman/guestman/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/pod"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

var (
	LAST_USED_PORT            = 0
	LAST_USED_NBD_SERVER_PORT = 0
	LAST_USED_MIGRATE_PORT    = 0
	NbdWorker                 = appsrv.NewWorkerManager("nbd_worker", 1, appsrv.DEFAULT_BACKLOG, false)
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
	CandidateServers map[string]GuestRuntimeInstance
	UnknownServers   *sync.Map
	ServersLock      *sync.Mutex
	portsInUse       *sync.Map

	// guests nics traffics lock
	TrafficLock *sync.Mutex

	GuestStartWorker *appsrv.SWorkerManager

	isLoaded bool

	// dirty servers chan
	dirtyServers     []GuestRuntimeInstance
	dirtyServersChan chan struct{}

	qemuMachineCpuMax map[string]uint
	qemuMaxMem        int

	cpuSet     *CpuSetCounter
	pythonPath string
}

func NewGuestManager(host hostutils.IHost, serversPath string) *SGuestManager {
	manager := &SGuestManager{}
	manager.host = host
	manager.ServersPath = serversPath
	manager.Servers = new(sync.Map)
	manager.portsInUse = new(sync.Map)
	manager.CandidateServers = make(map[string]GuestRuntimeInstance, 0)
	manager.UnknownServers = new(sync.Map)
	manager.ServersLock = &sync.Mutex{}
	manager.TrafficLock = &sync.Mutex{}
	manager.GuestStartWorker = appsrv.NewWorkerManager("GuestStart", 1, appsrv.DEFAULT_BACKLOG, false)
	manager.cpuSet = NewGuestCpuSetCounter(host.GetHostTopology(), host.GetReservedCpusInfo())
	// manager.StartCpusetBalancer()
	manager.LoadExistingGuests()
	manager.host.StartDHCPServer()
	manager.dirtyServersChan = make(chan struct{})
	manager.dirtyServers = make([]GuestRuntimeInstance, 0)
	manager.qemuMachineCpuMax = make(map[string]uint, 0)
	procutils.NewCommand("mkdir", "-p", manager.QemuLogDir()).Run()
	return manager
}

func (m *SGuestManager) InitQemuMaxCpus(machineCaps []monitor.MachineInfo, kvmMaxCpus uint) {
	m.qemuMachineCpuMax[compute.VM_MACHINE_TYPE_PC] = arch.X86_MAX_CPUS
	m.qemuMachineCpuMax[compute.VM_MACHINE_TYPE_Q35] = arch.X86_MAX_CPUS
	m.qemuMachineCpuMax[compute.VM_MACHINE_TYPE_ARM_VIRT] = arch.ARM_MAX_CPUS
	if len(machineCaps) == 0 {
		return
	}
	minFunc := func(a, b uint) uint {
		if a < b {
			return a
		}
		return b
	}
	log.Infof("KVM max cpus count: %d", kvmMaxCpus)
	supportedMachineType := []string{"pc", "q35", "virt"}
	for _, machine := range supportedMachineType {
		for i := 0; i < len(machineCaps); i++ {
			if (machineCaps[i].Alias != nil && *machineCaps[i].Alias == machine) ||
				machineCaps[i].Name == machine {
				cpuMax := uint(machineCaps[i].CPUMax)
				if kvmMaxCpus > 0 {
					cpuMax = minFunc(cpuMax, kvmMaxCpus)
				}
				if utils.IsInStringArray(machine, []string{"pc", "q35"}) {
					// Note: if max cpux exceed 255, machine requires Extended Interrupt Mode enabled.
					// You can add an IOMMU using: -device intel-iommu,intremap=on,eim=on
					// Set x86 machine max cpu 240 for now.
					m.qemuMachineCpuMax[machine] = minFunc(cpuMax, m.qemuMachineCpuMax[machine])
				} else {
					m.qemuMachineCpuMax[machine] = cpuMax
				}
				log.Infof("Machine type %s max cpus: %d", machine, m.qemuMachineCpuMax[machine])
			}
		}
	}

}

func (m *SGuestManager) InitQemuMaxMems(maxMems uint) {
	if maxMems > arch.X86_MAX_MEM_MB {
		arch.X86_MAX_MEM_MB = maxMems
	}
}

func (m *SGuestManager) InitPythonPath() error {
	defer func() {
		log.Infof("Python path %s", m.pythonPath)
	}()
	out, err := procutils.NewRemoteCommandAsFarAsPossible("which", "python").Output()
	if err == nil {
		m.pythonPath = string(bytes.TrimSpace(out))
		return nil
	}
	log.Infof("No python found %s: %s", out, err)

	out, err = procutils.NewRemoteCommandAsFarAsPossible("which", "python3").Output()
	if err == nil {
		m.pythonPath = string(bytes.TrimSpace(out))
		return nil
	}
	log.Infof("No python3 found %s: %s", out, err)

	out, err = procutils.NewRemoteCommandAsFarAsPossible("which", "python2").Output()
	if err == nil {
		m.pythonPath = string(bytes.TrimSpace(out))
		return nil
	}
	log.Infof("No python2 found %s: %s", out, err)
	return errors.Errorf("No python/python2/python3 found in PATH")
}

func (m *SGuestManager) GetCRI() pod.CRI {
	return m.host.GetCRI()
}

func (m *SGuestManager) getPythonPath() string {
	return m.pythonPath
}

func (m *SGuestManager) QemuLogDir() string {
	return path.Join(m.ServersPath, "logs")
}

func (m *SGuestManager) GetServer(sid string) (GuestRuntimeInstance, bool) {
	s, ok := m.Servers.Load(sid)
	if ok {
		return s.(GuestRuntimeInstance), ok
	} else {
		return nil, ok
	}
}

// 临时解决方案，后面应该统一 SKVMInstance 和 SPodInstance 使用 GuestRuntimeInstance 接口
func (m *SGuestManager) GetKVMServer(sid string) (*SKVMGuestInstance, bool) {
	s, ok := m.GetServer(sid)
	if !ok {
		return nil, false
	}
	return s.(*SKVMGuestInstance), true
}

func (m *SGuestManager) GetUnknownServer(sid string) (GuestRuntimeInstance, bool) {
	s, ok := m.UnknownServers.Load(sid)
	if ok {
		return s.(GuestRuntimeInstance), ok
	} else {
		return nil, ok
	}
}

func (m *SGuestManager) SaveServer(sid string, s GuestRuntimeInstance) {
	m.Servers.Store(sid, s)
}

func (m *SGuestManager) CleanServer(sid string) {
	m.Servers.Delete(sid)
}

func (m *SGuestManager) Bootstrap() chan struct{} {
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
	return m.dirtyServersChan
}

func (m *SGuestManager) VerifyExistingGuests(pendingDelete bool) {
	params := jsonutils.NewDict()
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("system", jsonutils.JSONTrue)
	params.Set("pending_delete", jsonutils.NewBool(pendingDelete))
	params.Set("get_all_guests_on_host", jsonutils.NewString(m.host.GetHostId()))
	if len(m.CandidateServers) > 0 {
		keys := make([]string, len(m.CandidateServers))
		var index = 0
		for k := range m.CandidateServers {
			keys[index] = k
			index++
		}
		params.Set("filter.0", jsonutils.NewString(fmt.Sprintf("id.in(%s)", strings.Join(keys, ","))))
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
			m.dirtyServers = append(m.dirtyServers, server)
			log.Errorf("Server %s not found on this host", server.GetName())
			m.RemoveCandidateServer(server)
		}
	}
}

func (m *SGuestManager) RemoveCandidateServer(server GuestRuntimeInstance) {
	if _, ok := m.CandidateServers[server.GetInitialId()]; ok {
		delete(m.CandidateServers, server.GetInitialId())
		if len(m.CandidateServers) == 0 {
			m.OnLoadExistingGuestsComplete()
		}
	}
}

func (m *SGuestManager) OnLoadExistingGuestsComplete() {
	log.Infof("Load existing guests complete...")
	err := m.host.PutHostOnline()
	if err != nil {
		log.Fatalf("put host online failed %s", err)
	}

	go m.verifyDirtyServers()

	if !options.HostOptions.EnableCpuBinding {
		m.ClenaupCpuset()
	}
}

func (m *SGuestManager) verifyDirtyServers() {
	select {
	case <-m.dirtyServersChan:
	}
	for i := 0; i < len(m.dirtyServers); i++ {
		go m.RequestVerifyDirtyServer(m.dirtyServers[i])
	}
	m.dirtyServers = nil
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
	if !options.HostOptions.DisableSetCgroup {
		cgrouputils.RebalanceProcesses(nil)
	}
}

func (m *SGuestManager) CPUSet(ctx context.Context, sid string, req *compute.ServerCPUSetInput) (*compute.ServerCPUSetResp, error) {
	guest, ok := m.GetKVMServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("Not found")
	}
	return guest.CPUSet(ctx, req.CPUS)
}

func (m *SGuestManager) CPUSetRemove(ctx context.Context, sid string) error {
	guest, ok := m.GetKVMServer(sid)
	if !ok {
		return httperrors.NewNotFoundError("Not found")
	}
	return guest.CPUSetRemove(ctx)
}

func (m *SGuestManager) IsGuestDir(f os.FileInfo) bool {
	return hostutils.IsGuestDir(f, m.ServersPath)
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

func (m *SGuestManager) GetServerDescFilePath(sid string) string {
	return path.Join(m.ServersPath, sid, "desc")
}

func (m *SGuestManager) GetServerDesc(sid string) (*desc.SGuestDesc, error) {
	descPath := m.GetServerDescFilePath(sid)
	descStr, err := ioutil.ReadFile(descPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", descPath)
	}
	desc := new(desc.SGuestDesc)
	jsonSrcDesc, err := jsonutils.Parse(descStr)
	if err != nil {
		return nil, errors.Wrapf(err, "json parse: %s", descStr)
	}
	if err := jsonSrcDesc.Unmarshal(desc); err != nil {
		return nil, errors.Wrap(err, "unmarshal desc")
	}
	return desc, nil
}

func (m *SGuestManager) LoadServer(sid string) {
	desc, err := m.GetServerDesc(sid)
	if err != nil {
		log.Errorf("Get server %s desc: %v", sid, err)
		return
	}
	guest := NewGuestRuntimeManager().NewRuntimeInstance(sid, m, desc.Hypervisor)
	if err := guest.LoadDesc(); err != nil {
		log.Errorf("On load server error: %s", err)
		return
	}

	m.CandidateServers[sid] = guest
	if err := guest.PostLoad(m); err != nil {
		log.Errorf("Post load server %s: %v", sid, err)
		return
	}
}

func (m *SGuestManager) ShutdownServers() {
	m.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		log.Infof("Start shutdown server %s", guest.GetName())

		// scriptStop maybe stuck on guest storage offline
		if !guest.forceScriptStop() {
			log.Errorf("shutdown server %s failed", guest.GetName())
		}
		return true
	})
}

func (m *SGuestManager) GetGuestNicDesc(
	mac, ip, port, bridge string, isCandidate bool,
) (*desc.SGuestDesc, *desc.SGuestNetwork) {
	if isCandidate {
		return m.getGuestNicDescInCandidate(mac, ip, port, bridge)
	}
	var nic *desc.SGuestNetwork
	var guestDesc *desc.SGuestDesc
	m.Servers.Range(func(k interface{}, v interface{}) bool {
		guest := v.(GuestRuntimeInstance)
		if guest.IsLoaded() {
			nic = guest.GetNicDescMatch(mac, ip, port, bridge)
			if nic != nil {
				guestDesc = guest.GetDesc()
				return false
			}
		}
		return true
	})
	return guestDesc, nic
}

func (m *SGuestManager) getGuestNicDescInCandidate(
	mac, ip, port, bridge string,
) (*desc.SGuestDesc, *desc.SGuestNetwork) {
	for _, guest := range m.CandidateServers {
		if guest.IsLoaded() {
			nic := guest.GetNicDescMatch(mac, ip, port, bridge)
			if nic != nil {
				return guest.GetDesc(), nic
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
	return PrepareDir(guest)
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

func (m *SGuestManager) Monitor(sid, cmd string, qmp bool, callback func(string)) error {
	if guest, ok := m.GetKVMServer(sid); ok {
		if guest.IsRunning() {
			if guest.Monitor == nil {
				return httperrors.NewBadRequestError("Monitor disconnected??")
			}
			if qmp {
				if err := guest.Monitor.QemuMonitorCommand(cmd, callback); err != nil {
					return errors.Wrap(err, "qemu monitor command")
				}
			} else {
				guest.Monitor.HumanMonitorCommand(cmd, callback)
			}
			return nil
		} else {
			return httperrors.NewBadRequestError("Server stopped??")
		}
	} else {
		return httperrors.NewNotFoundError("Not found KVM server: %s", sid)
	}
}

func (m *SGuestManager) sdnClient() (fwdpb.ForwarderClient, error) {
	sockPath := options.HostOptions.SdnSocketPath
	if strings.HasPrefix(sockPath, "/") {
		sockPath = "unix://" + sockPath
	}
	cli, err := fwd.NewClient(sockPath)
	return cli, err
}

func (m *SGuestManager) OpenForward(ctx context.Context, sid string, req *hostapi.GuestOpenForwardRequest) (*hostapi.GuestOpenForwardResponse, error) {
	guest, ok := m.GetKVMServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("Not found")
	}
	if !guest.IsRunning() {
		return nil, httperrors.NewBadRequestError("Server stopped??")
	}

	nic := guest.GetVpcNIC()
	if nic == nil {
		return nil, httperrors.NewBadRequestError("no vpc nic")
	}

	netId := nic.NetId
	if netId == "" {
		return nil, httperrors.NewBadRequestError("no network id")
	}
	var ip string
	if req.Addr != "" {
		ip = req.Addr
	} else {
		ip := nic.Ip
		if ip == "" {
			return nil, httperrors.NewBadRequestError("no vpc ip")
		}
	}
	pbreq := &fwdpb.OpenRequest{
		NetId:      netId,
		Proto:      req.Proto,
		BindAddr:   m.host.GetMasterIp(),
		RemoteAddr: ip,
		RemotePort: uint32(req.Port),
	}
	cli, err := m.sdnClient()
	if err != nil {
		log.Errorf("new sdn client error: %v", err)
		return nil, httperrors.NewBadGatewayError("lost sdn connection")
	}
	resp, err := cli.Open(ctx, pbreq)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	output := &hostapi.GuestOpenForwardResponse{
		Proto: resp.Proto,
		Addr:  resp.RemoteAddr,
		Port:  int(resp.RemotePort),

		ProxyAddr: resp.BindAddr,
		ProxyPort: int(resp.BindPort),
	}
	return output, nil
}

func (m *SGuestManager) CloseForward(ctx context.Context, sid string, req *hostapi.GuestCloseForwardRequest) (*hostapi.GuestCloseForwardResponse, error) {
	guest, ok := m.GetKVMServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("Not found")
	}

	nic := guest.GetVpcNIC()
	if nic == nil {
		return nil, httperrors.NewBadRequestError("no vpc nic")
	}

	netId := nic.NetId
	if netId == "" {
		return nil, httperrors.NewBadRequestError("no network id")
	}
	pbreq := &fwdpb.CloseRequest{
		NetId:    netId,
		Proto:    req.Proto,
		BindAddr: req.ProxyAddr,
		BindPort: uint32(req.ProxyPort),
	}
	cli, err := m.sdnClient()
	if err != nil {
		log.Errorf("new sdn client error: %v", err)
		return nil, httperrors.NewBadGatewayError("lost sdn connection")
	}
	resp, err := cli.Close(ctx, pbreq)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	output := &hostapi.GuestCloseForwardResponse{
		Proto:     resp.Proto,
		ProxyAddr: resp.BindAddr,
		ProxyPort: int(resp.BindPort),
	}
	return output, nil
}

func (m *SGuestManager) ListForward(ctx context.Context, sid string, req *hostapi.GuestListForwardRequest) (*hostapi.GuestListForwardResponse, error) {
	guest, ok := m.GetKVMServer(sid)
	if !ok {
		return nil, httperrors.NewNotFoundError("Not found")
	}
	if !guest.IsRunning() {
		return nil, httperrors.NewBadRequestError("Server stopped??")
	}

	nic := guest.GetVpcNIC()
	if nic == nil {
		return nil, httperrors.NewBadRequestError("no vpc nic")
	}

	netId := nic.NetId
	if netId == "" {
		return nil, httperrors.NewBadRequestError("no network id")
	}
	pbreq := &fwdpb.ListByRemoteRequest{
		NetId:      netId,
		Proto:      req.Proto,
		RemoteAddr: req.Addr,
		RemotePort: uint32(req.Port),
	}
	cli, err := m.sdnClient()
	if err != nil {
		log.Errorf("new sdn client error: %v", err)
		return nil, httperrors.NewBadGatewayError("lost sdn connection")
	}
	resp, err := cli.ListByRemote(ctx, pbreq)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	var outputForwards []hostapi.GuestOpenForwardResponse
	for i := range resp.Forwards {
		outputForwards = append(outputForwards, hostapi.GuestOpenForwardResponse{
			Proto: resp.Forwards[i].Proto,
			Addr:  resp.Forwards[i].RemoteAddr,
			Port:  int(resp.Forwards[i].RemotePort),

			ProxyAddr: resp.Forwards[i].BindAddr,
			ProxyPort: int(resp.Forwards[i].BindPort),
		})
	}
	output := &hostapi.GuestListForwardResponse{
		Forwards: outputForwards,
	}
	return output, nil
}

func (m *SGuestManager) GuestCreate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	deployParams, ok := params.(*SGuestDeploy)
	if !ok {
		return nil, hostutils.ParamsError
	}

	var guest GuestRuntimeInstance
	e := func() error {
		m.ServersLock.Lock()
		defer m.ServersLock.Unlock()
		if _, ok := m.GetServer(deployParams.Sid); ok {
			return httperrors.NewBadRequestError("Guest %s exists", deployParams.Sid)
		}
		var (
			descInfo   *desc.SGuestDesc = nil
			hypervisor                  = ""
		)
		if deployParams.Body.Contains("desc") {
			descInfo = new(desc.SGuestDesc)
			err := deployParams.Body.Unmarshal(descInfo, "desc")
			if err != nil {
				return httperrors.NewBadRequestError("Guest desc unmarshal failed %s", err)
			}
			hypervisor = descInfo.Hypervisor
		}
		//guest = NewKVMGuestInstance(deployParams.Sid, m)
		factory := NewGuestRuntimeManager()
		guest = factory.NewRuntimeInstance(deployParams.Sid, m, hypervisor)

		if descInfo != nil {
			if err := factory.CreateFromDesc(guest, descInfo); err != nil {
				return errors.Wrap(err, "create from desc")
			}
		}

		m.SaveServer(deployParams.Sid, guest)
		return nil
	}()
	if e != nil {
		return nil, errors.Wrap(e, "prepare guest")
	}
	return m.startDeploy(ctx, deployParams, guest)
}

func (m *SGuestManager) startDeploy(
	ctx context.Context, deployParams *SGuestDeploy, guest GuestRuntimeInstance) (jsonutils.JSONObject, error) {
	publicKey := deployapi.GetKeys(deployParams.Body)
	deployArray := make([]*deployapi.DeployContent, 0)
	if deployParams.Body.Contains("deploys") {
		err := deployParams.Body.Unmarshal(&deployArray, "deploys")
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshal to array of deployapi.DeployContent")
		}
	}
	password, _ := deployParams.Body.GetString("password")
	resetPassword := jsonutils.QueryBoolean(deployParams.Body, "reset_password", false)
	if resetPassword && len(password) == 0 {
		password = seclib.RandomPassword(12)
	}
	enableCloudInit := jsonutils.QueryBoolean(deployParams.Body, "enable_cloud_init", false)
	loginAccount, _ := deployParams.Body.GetString("login_account")
	deployTelegraf := jsonutils.QueryBoolean(deployParams.Body, "deploy_telegraf", false)
	telegrafConfig, _ := deployParams.Body.GetString("telegraf_conf")
	if deployTelegraf && telegrafConfig == "" {
		return nil, errors.Errorf("missing telegraf_conf")
	}

	guestInfo, err := guest.DeployFs(ctx, deployParams.UserCred,
		deployapi.NewDeployInfo(
			publicKey, deployArray,
			password, deployParams.IsInit, false,
			options.HostOptions.LinuxDefaultRootUser, options.HostOptions.WindowsDefaultAdminUser,
			enableCloudInit, loginAccount, deployTelegraf, telegrafConfig,
			guest.GetDesc().UserData,
		),
	)
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
		if deployParams.Body.Contains("desc") {
			var guestDesc = new(desc.SGuestDesc)
			err := deployParams.Body.Unmarshal(guestDesc, "desc")
			if err != nil {
				return nil, httperrors.NewBadRequestError("Failed unmarshal guest desc %s", err)
			}
			if err := SaveDesc(guest, guestDesc); err != nil {
				return nil, errors.Wrap(err, "failed save desc")
			}
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
	status := m.getStatus(sid)
	return status
}

func (m *SGuestManager) GetGuestStatus(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sid := params.(string)
	status := m.getStatus(sid)
	guest, _ := m.GetServer(sid)
	body := jsonutils.NewDict()
	if guest != nil {
		body.Set("power_status", jsonutils.NewString(GetPowerStates(guest)))
	}

	return guest.HandleGuestStatus(ctx, status, body)
}

func (m *SGuestManager) getStatus(sid string) string {
	if guest, ok := m.GetServer(sid); ok {
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

func (m *SGuestManager) Delete(sid string) (GuestRuntimeInstance, error) {
	if guest, ok := m.GetServer(sid); ok {
		m.CleanServer(sid)
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

func (m *SGuestManager) GuestStart(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if guest, ok := m.GetServer(sid); ok {
		guestDesc := new(desc.SGuestDesc)
		if err := body.Unmarshal(guestDesc, "desc"); err == nil {
			if err = SaveDesc(guest, guestDesc); err != nil {
				return nil, errors.Wrap(err, "save desc")
			}
		}
		return guest.HandleGuestStart(ctx, userCred, body)
	} else {
		return nil, httperrors.NewNotFoundError("Not found server %s", sid)
	}
}

func (m *SGuestManager) GuestStop(ctx context.Context, sid string, timeout int64) error {
	if server, ok := m.GetServer(sid); ok {
		if err := server.HandleStop(ctx, timeout); err != nil {
			return errors.Wrap(err, "Do stop")
		}
	} else {
		return httperrors.NewNotFoundError("Guest %s not found", sid)
	}
	return nil
}

func (m *SGuestManager) GuestStartRescue(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	baremetalManagerUri, err := body.GetString("manager_uri")
	if err != nil {
		return nil, httperrors.NewInputParameterError("manager_uri required")
	}
	if guest, ok := m.GetKVMServer(sid); ok {
		guest.ExecStartRescueTask(ctx, baremetalManagerUri)
		return nil, nil
	} else {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
}

func (m *SGuestManager) GuestStopRescue(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if guest, ok := m.GetKVMServer(sid); ok {
		guest.ExecStopRescueTask(ctx, body)
		return nil, nil
	} else {
		return nil, httperrors.NewNotFoundError("Guest %s not found", sid)
	}
}

func (m *SGuestManager) GuestSync(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	syncParams, ok := params.(*SBaseParams)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(syncParams.Sid)
	if syncParams.Body.Contains("desc") {
		guestDesc := new(desc.SGuestDesc)
		if err := syncParams.Body.Unmarshal(guestDesc, "desc"); err != nil {
			return nil, errors.Wrap(err, "unmarshal guest desc")
		}

		fwOnly := jsonutils.QueryBoolean(syncParams.Body, "fw_only", false)
		return guest.SyncConfig(ctx, guestDesc, fwOnly)
	}
	return nil, nil
}

func (m *SGuestManager) GuestSuspend(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sid, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, ok := m.GetKVMServer(sid)
	if !ok {
		return nil, errors.Errorf("Not found KVM server: %s", sid)
	}
	guest.ExecSuspendTask(ctx)
	return nil, nil
}

func (m *SGuestManager) GuestIoThrottle(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	guestIoThrottle, ok := params.(*SGuestIoThrottle)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(guestIoThrottle.Sid)
	for i := range guest.GetDesc().Disks {
		diskId := guest.GetDesc().Disks[i].DiskId
		if bps, ok := guestIoThrottle.Input.Bps[diskId]; ok {
			guest.GetDesc().Disks[i].Bps = bps
		}
		if iops, ok := guestIoThrottle.Input.IOPS[diskId]; ok {
			guest.GetDesc().Disks[i].Iops = iops
		}
	}
	if err := SaveLiveDesc(guest, guest.GetDesc()); err != nil {
		return nil, errors.Wrap(err, "guest save desc")
	}

	if guest.IsRunning() {
		guest.BlockIoThrottle(ctx)
		return nil, nil
	}
	return nil, httperrors.NewInvalidStatusError("Guest not running")
}

func (m *SGuestManager) SrcPrepareMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SSrcPrepareMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(migParams.Sid)
	disksBack, diskSnapsChain, sysDiskHasTemplate, err := guest.PrepareDisksMigrate(migParams.LiveMigrate)
	if err != nil {
		return nil, errors.Wrap(err, "PrepareDisksMigrate")
	}
	ret := jsonutils.NewDict()
	if disksBack.Length() > 0 {
		ret.Set("disks_back", disksBack)
	}
	if diskSnapsChain.Length() > 0 {
		ret.Set("disk_snaps_chain", diskSnapsChain)
	}
	if sysDiskHasTemplate {
		ret.Set("sys_disk_has_template", jsonutils.JSONTrue)
	}

	if migParams.LiveMigrate && migParams.LiveMigrateUseTLS {
		certs, err := guest.PrepareMigrateCerts()
		if err != nil {
			return nil, errors.Wrap(err, "PrepareMigrateCerts")
		}
		ret.Set("migrate_certs", jsonutils.Marshal(certs))
	}
	if migParams.LiveMigrate {
		if guest.GetDesc().Machine == "" {
			guest.GetDesc().Machine = guest.getMachine()
		}
		if err = guest.syncVirtioDiskNumQueues(); err != nil {
			return nil, errors.Wrap(err, "syncVirtioDiskNumQueues")
		}
		ret.Set("src_desc", jsonutils.Marshal(guest.GetDesc()))
	}
	return ret, nil
}

func (m *SGuestManager) DestPrepareMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SDestPrepareMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, _ := m.GetKVMServer(migParams.Sid)
	if err := NewGuestRuntimeManager().CreateFromDesc(guest, migParams.Desc); err != nil {
		return nil, err
	}

	disks := migParams.Desc.Disks
	if len(migParams.TargetStorageIds) > 0 {
		var encInfo *apis.SEncryptInfo
		if guest.isEncrypted() {
			info, err := guest.getEncryptKey(ctx, migParams.UserCred)
			if err != nil {
				return nil, errors.Wrap(err, "getEncryptKey")
			}
			encInfo = &info
		}
		for i := 0; i < len(migParams.TargetStorageIds); i++ {
			iStorage := storageman.GetManager().GetStorage(migParams.TargetStorageIds[i])
			if iStorage == nil {
				return nil, fmt.Errorf("Target storage %s not found", migParams.TargetStorageIds[i])
			}

			err := iStorage.DestinationPrepareMigrate(
				ctx, migParams.LiveMigrate, migParams.DisksUri, migParams.SnapshotsUri,
				migParams.DisksBackingFile, migParams.DiskSnapsChain, migParams.OutChainSnaps,
				migParams.RebaseDisks, disks[i], migParams.Sid, i+1, len(disks), encInfo, migParams.SysDiskHasTemplate,
			)
			if err != nil {
				return nil, fmt.Errorf("dest prepare migrate failed %s", err)
			}
		}
		if err := SaveDesc(guest, migParams.Desc); err != nil {
			log.Errorln(err)
			return nil, err
		}

	}

	body := jsonutils.NewDict()

	if len(migParams.SrcMemorySnapshots) > 0 {
		preparedMs, err := m.destinationPrepareMigrateMemorySnapshots(ctx, migParams.Sid, migParams.MemorySnapshotsUri, migParams.SrcMemorySnapshots)
		if err != nil {
			return nil, errors.Wrap(err, "destination prepare migrate memory snapshots")
		}
		body.Add(jsonutils.Marshal(preparedMs), "dest_prepared_memory_snapshots")
	}

	if migParams.LiveMigrate {
		startParams := jsonutils.NewDict()
		startParams.Set("qemu_version", jsonutils.NewString(migParams.QemuVersion))
		startParams.Set("need_migrate", jsonutils.JSONTrue)
		startParams.Set("live_migrate_use_tls", jsonutils.NewBool(migParams.EnableTLS))
		startParams.Set("src_desc", jsonutils.Marshal(migParams.SrcDesc))
		if len(migParams.MigrateCerts) > 0 {
			if err := guest.WriteMigrateCerts(migParams.MigrateCerts); err != nil {
				return nil, errors.Wrap(err, "write migrate certs")
			}
		}
		var err error
		startParams, err = guest.prepareEncryptKeyForStart(ctx, migParams.UserCred, startParams)
		if err != nil {
			return nil, errors.Wrap(err, "prepareEncryptKeyForStart")
		}
		hostutils.DelayTaskWithoutReqctx(ctx, guest.asyncScriptStart, startParams)
	} else {
		hostutils.UpdateServerProgress(context.Background(), migParams.Sid, 100.0, 0)
	}

	return body, nil
}

func (m *SGuestManager) destinationPrepareMigrateMemorySnapshots(ctx context.Context, serverId string, uri string, ids []string) (map[string]string, error) {
	ret := make(map[string]string, 0)
	for _, id := range ids {
		url := fmt.Sprintf("%s/%s/%s", uri, serverId, id)
		msPath := GetMemorySnapshotPath(serverId, id)
		dir := filepath.Dir(msPath)
		if err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Run(); err != nil {
			return nil, errors.Wrapf(err, "mkdir -p %q", dir)
		}
		remotefile := remotefile.NewRemoteFile(ctx, url, msPath, false, "", -1, nil, "", "")
		if err := remotefile.Fetch(nil); err != nil {
			return nil, errors.Wrapf(err, "fetch memory snapshot file %s", url)
		} else {
			ret[id] = msPath
		}
	}
	return ret, nil
}

func (m *SGuestManager) LiveMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	migParams, ok := params.(*SLiveMigrate)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, _ := m.GetKVMServer(migParams.Sid)
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
	m.SaveServer(sid, guest)
	return true
}

func (m *SGuestManager) checkAndSetPort(port int) bool {
	_, loaded := m.portsInUse.LoadOrStore(port, struct{}{})
	return !loaded
}

func (m *SGuestManager) unsetPort(port int) {
	m.portsInUse.Delete(port)
}

func (m *SGuestManager) GetFreePortByBase(basePort int) int {
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

func (m *SGuestManager) GetLiveMigrateFreePort() int {
	port := m.GetFreePortByBase(LIVE_MIGRATE_PORT_BASE + LAST_USED_MIGRATE_PORT)
	LAST_USED_MIGRATE_PORT = port - LIVE_MIGRATE_PORT_BASE
	if LAST_USED_MIGRATE_PORT > 5000 {
		LAST_USED_MIGRATE_PORT = 0
	}
	return port
}

func (m *SGuestManager) GetNBDServerFreePort() int {
	port := m.GetFreePortByBase(BUILT_IN_NBD_SERVER_PORT_BASE + LAST_USED_NBD_SERVER_PORT)
	LAST_USED_NBD_SERVER_PORT = port - BUILT_IN_NBD_SERVER_PORT_BASE
	if LAST_USED_NBD_SERVER_PORT > 5000 {
		LAST_USED_NBD_SERVER_PORT = 0
	}
	return port
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
		if _, ok := vncPorts[port]; ok ||
			netutils2.IsTcpPortUsed("0.0.0.0", VNC_PORT_BASE+port) ||
			netutils2.IsTcpPortUsed("127.0.0.1", MONITOR_PORT_BASE+port) ||
			netutils2.IsTcpPortUsed("127.0.0.1", QMP_MONITOR_PORT_BASE+port) {
			port += 1
		} else {
			if !m.checkAndSetPort(port) {
				continue
			}
			break
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
	guest, _ := m.GetKVMServer(reloadParams.Sid)
	return guest.ExecReloadDiskTask(ctx, reloadParams.Disk)
}

func (m *SGuestManager) DoSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotParams, ok := params.(*SDiskSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(snapshotParams.Sid)
	return guest.ExecDiskSnapshotTask(ctx, snapshotParams.UserCred, snapshotParams.Disk, snapshotParams.SnapshotId)
}

func (m *SGuestManager) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	delParams, ok := params.(*SDeleteDiskSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if len(delParams.ConvertSnapshot) > 0 {
		guest, _ := m.GetKVMServer(delParams.Sid)
		return guest.ExecDeleteSnapshotTask(ctx, delParams.Disk, delParams.DeleteSnapshot,
			delParams.ConvertSnapshot, delParams.PendingDelete)
	} else {
		res := jsonutils.NewDict()
		res.Set("deleted", jsonutils.JSONTrue)
		return res, delParams.Disk.DeleteSnapshot(delParams.DeleteSnapshot, "", false)
	}
}

func (m *SGuestManager) DoMemorySnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SMemorySnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, _ := m.GetKVMServer(input.Sid)
	return guest.ExecMemorySnapshotTask(ctx, input.GuestMemorySnapshotRequest)
}

func (m *SGuestManager) DoResetMemorySnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SMemorySnapshotReset)
	if !ok {
		return nil, hostutils.ParamsError
	}

	guest, _ := m.GetKVMServer(input.Sid)
	return guest.ExecMemorySnapshotResetTask(ctx, input.GuestMemorySnapshotResetRequest)
}

func (m *SGuestManager) DoDeleteMemorySnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SMemorySnapshotDelete)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if err := procutils.NewRemoteCommandAsFarAsPossible("rm", input.Path).Run(); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "No such file or directory") {
			return nil, err
		}
	}
	log.Infof("Memory snapshot file %q removed", input.Path)
	return nil, nil
}

func (m *SGuestManager) Resume(ctx context.Context, sid string, isLiveMigrate bool, cleanTLS bool) (jsonutils.JSONObject, error) {
	guest, _ := m.GetKVMServer(sid)
	if guest.IsStopping() || guest.IsStopped() {
		return nil, httperrors.NewInvalidStatusError("resume stopped server???")
	}

	var onLiveMigrateCleanup = func(res string) {
		guest.DoResumeTask(ctx, !isLiveMigrate, cleanTLS)
	}
	var onMonitorConnected = func() {
		if isLiveMigrate {
			guest.StartPresendArp()
			guest.Monitor.StopNbdServer(onLiveMigrateCleanup)
		} else {
			onLiveMigrateCleanup("")
		}
	}
	if guest.Monitor == nil {
		guest.StartMonitor(ctx, nil)
		return nil, nil
	} else {
		onMonitorConnected()
	}
	return nil, nil
}

func (m *SGuestManager) OnlineResizeDisk(ctx context.Context, sid string, diskId string, sizeMb int64) (jsonutils.JSONObject, error) {
	guest, ok := m.GetKVMServer(sid)
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
// 	guest := guestManager.Servers[sid]
// 	port := m.GetFreePortByBase(BUILT_IN_NBD_SERVER_PORT_BASE)

// }

func (m *SGuestManager) StartBlockReplication(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	mirrorParams, ok := params.(*SDriverMirror)
	if !ok {
		return nil, hostutils.ParamsError
	}

	nbdOpts := strings.Split(mirrorParams.NbdServerUri, ":")
	if len(nbdOpts) != 3 {
		return nil, fmt.Errorf("Nbd url is not vaild %s", mirrorParams.NbdServerUri)
	}
	guest, _ := m.GetKVMServer(mirrorParams.Sid)
	// TODO: check desc
	if err := SaveDesc(guest, mirrorParams.Desc); err != nil {
		return nil, err
	}
	onSucc := func() {
		if err := guest.updateChildIndex(); err != nil {
			hostutils.TaskFailed(ctx, err.Error())
			return
		}
		hostutils.TaskComplete(ctx, nil)
	}
	task := NewGuestBlockReplicationTask(ctx, guest, nbdOpts[1], nbdOpts[2], "full", onSucc, nil)
	task.Start()
	return nil, nil
}

func (m *SGuestManager) CancelBlockJobs(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sid, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	status := m.getStatus(sid)
	if status == GUSET_STOPPED {
		hostutils.TaskComplete(ctx, nil)
		return nil, nil
	}
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("STACK: %v \n %s", r, debug.Stack())
			hostutils.TaskFailed(ctx, fmt.Sprintf("recover: %v", r))
		}
	}()
	guest, _ := m.GetKVMServer(sid)
	NewCancelBlockJobsTask(ctx, guest).Start()
	return nil, nil
}

func (m *SGuestManager) CancelBlockReplication(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sid, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	status := m.getStatus(sid)
	if status == GUSET_STOPPED {
		hostutils.TaskComplete(ctx, nil)
		return nil, nil
	}
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("STACK: %v \n %s", r, debug.Stack())
			hostutils.TaskFailed(ctx, fmt.Sprintf("recover: %v", r))
		}
	}()
	guest, _ := m.GetKVMServer(sid)
	NewCancelBlockReplicationTask(ctx, guest).Start()
	return nil, nil
}

func (m *SGuestManager) HotplugCpuMem(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	hotplugParams, ok := params.(*SGuestHotplugCpuMem)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(hotplugParams.Sid)
	NewGuestHotplugCpuMemTask(ctx, guest, int(hotplugParams.AddCpuCount), int(hotplugParams.AddMemSize)).Start()
	return nil, nil
}

func (m *SGuestManager) ExitGuestCleanup() {
	m.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*SKVMGuestInstance)
		guest.ExitCleanup(false)
		return true
	})
	if !options.HostOptions.DisableSetCgroup {
		cgrouputils.CgroupCleanAll()
	}
}

type SStorageCloneDisk struct {
	ServerId       string
	SourceStorage  storageman.IStorage
	SourceDisk     storageman.IDisk
	TargetStorage  storageman.IStorage
	TargetDiskId   string
	DiskFormat     string
	TargetDiskDesc *compute.GuestdiskJsonDesc

	// clone progress
	CompletedDiskCount int
	CloneDiskCount     int
}

func (m *SGuestManager) StorageCloneDisk(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input := params.(*SStorageCloneDisk)
	guest, _ := m.GetKVMServer(input.ServerId)
	if guest == nil {
		return nil, httperrors.NewNotFoundError("Not found guest by id %s", input.ServerId)
	}
	guestRunning := guest.IsRunning() || guest.IsSuspend()
	NewGuestStorageCloneDiskTask(ctx, guest, input).Start(guestRunning)
	return nil, nil
}

func (m *SGuestManager) LiveChangeDisk(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input := params.(*SStorageCloneDisk)
	guest, _ := m.GetKVMServer(input.ServerId)
	if guest == nil {
		return nil, httperrors.NewNotFoundError("Not found guest by id %s", input.ServerId)
	}
	if !(guest.IsRunning() || guest.IsSuspend()) {
		return nil, httperrors.NewBadRequestError("Guest %s is not in state running", input.ServerId)
	}
	task, err := NewGuestLiveChangeDiskTask(ctx, guest, input)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Start live change disk task failed: %s", err)
	}
	task.Start()
	return nil, nil
}

func (m *SGuestManager) GetHost() hostutils.IHost {
	return m.host
}

func (m *SGuestManager) RequestVerifyDirtyServer(s GuestRuntimeInstance) {
	hostId := s.GetDesc().HostId
	var body = jsonutils.NewDict()
	body.Set("guest_id", jsonutils.NewString(s.GetInitialId()))
	body.Set("host_id", jsonutils.NewString(hostId))
	ret, err := modules.Servers.PerformClassAction(
		hostutils.GetComputeSession(context.Background()), "dirty-server-verify", body)
	if err != nil {
		log.Errorf("Dirty server request start error: %s", err)
	} else if jsonutils.QueryBoolean(ret, "guest_unknown_need_clean", false) {
		m.Delete(s.GetInitialId())
		s.CleanGuest(context.Background(), true)
	}
}

func (m *SGuestManager) ResetGuestNicTrafficLimit(guestId string, input []compute.ServerNicTrafficLimit) error {
	guest, ok := m.GetKVMServer(guestId)
	if !ok {
		return httperrors.NewNotFoundError("guest %s not found", guestId)
	}

	m.TrafficLock.Lock()
	defer m.TrafficLock.Unlock()
	for i := range input {
		if err := m.resetGuestNicTrafficLimit(guest, input[i]); err != nil {
			return errors.Wrap(err, "reset guest nic traffic limit")
		}
	}

	if err := SaveLiveDesc(guest, guest.Desc); err != nil {
		return errors.Wrap(err, "guest save desc")
	}
	return nil
}

func (m *SGuestManager) resetGuestNicTrafficLimit(guest *SKVMGuestInstance, input compute.ServerNicTrafficLimit) error {
	var nic *desc.SGuestNetwork
	for i := range guest.Desc.Nics {
		if guest.Desc.Nics[i].Mac == input.Mac {
			nic = guest.Desc.Nics[i]
			break
		}
	}
	if nic == nil {
		return httperrors.NewNotFoundError("guest nic %s not found", input.Mac)
	}

	recordPath := guest.NicTrafficRecordPath()
	if fileutils2.Exists(recordPath) {
		record, err := m.GetGuestTrafficRecord(guest.Id)
		if err != nil {
			return errors.Wrap(err, "failed load guest traffic record")
		}
		if nicRecord, ok := record[strconv.Itoa(int(nic.Index))]; ok {
			if nicRecord.TxTraffic >= nic.TxTrafficLimit || nicRecord.RxTraffic >= nic.RxTrafficLimit {
				err = guest.SetNicUp(nic)
				if err != nil {
					return errors.Wrap(err, "set nic up")
				}
			}
		}
		delete(record, strconv.Itoa(int(nic.Index)))
		if err = m.SaveGuestTrafficRecord(guest.Id, record); err != nil {
			return errors.Wrap(err, "failed save guest traffic record")
		}
	}
	if input.RxTrafficLimit != nil {
		nic.RxTrafficLimit = *input.RxTrafficLimit
	}
	if input.TxTrafficLimit != nil {
		nic.TxTrafficLimit = *input.TxTrafficLimit
	}
	return nil
}

func (m *SGuestManager) setNicTrafficLimit(guest *SKVMGuestInstance, input compute.ServerNicTrafficLimit) error {
	var nic *desc.SGuestNetwork
	for i := range guest.Desc.Nics {
		if guest.Desc.Nics[i].Mac == input.Mac {
			nic = guest.Desc.Nics[i]
			break
		}
	}
	if nic == nil {
		return httperrors.NewNotFoundError("guest nic %s not found", input.Mac)
	}

	if input.RxTrafficLimit != nil {
		nic.RxTrafficLimit = *input.RxTrafficLimit
	}
	if input.TxTrafficLimit != nil {
		nic.TxTrafficLimit = *input.TxTrafficLimit
	}
	recordPath := guest.NicTrafficRecordPath()
	if fileutils2.Exists(recordPath) {
		record, err := m.GetGuestTrafficRecord(guest.Id)
		if err != nil {
			return errors.Wrap(err, "failed load guest traffic record")
		}
		if nicRecord, ok := record[strconv.Itoa(int(nic.Index))]; ok {
			if nicRecord.TxTraffic < nic.TxTrafficLimit && nicRecord.RxTraffic < nic.RxTrafficLimit {
				err = guest.SetNicUp(nic)
				if err != nil {
					return errors.Wrap(err, "set nic up")
				}
			}
		}
		return m.SaveGuestTrafficRecord(guest.Id, record)
	}
	return nil
}

func (m *SGuestManager) SetGuestNicTrafficLimit(guestId string, input []compute.ServerNicTrafficLimit) error {
	guest, ok := m.GetKVMServer(guestId)
	if !ok {
		return httperrors.NewNotFoundError("guest %s not found", guestId)
	}

	m.TrafficLock.Lock()
	defer m.TrafficLock.Unlock()

	for i := range input {
		if err := m.setNicTrafficLimit(guest, input[i]); err != nil {
			return errors.Wrap(err, "set nic traffic limit")
		}
	}

	if err := SaveLiveDesc(guest, guest.Desc); err != nil {
		return errors.Wrap(err, "guest save desc")
	}

	return nil
}

func (m *SGuestManager) SaveGuestTrafficRecord(sid string, record map[string]compute.SNicTrafficRecord) error {
	guest, _ := m.GetServer(sid)
	recordPath := guest.NicTrafficRecordPath()
	v, _ := json.Marshal(record)
	return fileutils2.FilePutContents(recordPath, string(v), false)
}

func (m *SGuestManager) GetGuestTrafficRecord(sid string) (map[string]compute.SNicTrafficRecord, error) {
	guest, _ := m.GetServer(sid)
	recordPath := guest.NicTrafficRecordPath()
	if !fileutils2.Exists(recordPath) {
		return nil, nil
	}
	recordStr, err := ioutil.ReadFile(recordPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read traffic record %s", recordPath)
	}
	record := make(map[string]compute.SNicTrafficRecord)
	err = json.Unmarshal(recordStr, &record)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshal traffic record %s", recordPath)
	}
	return record, nil
}

func SyncGuestNicsTraffics(guestNicsTraffics map[string]map[string]compute.SNicTrafficRecord) {
	session := hostutils.GetComputeSession(context.Background())
	hostId := guestManager.host.GetHostId()
	data := jsonutils.Marshal(guestNicsTraffics)
	_, err := modules.Hosts.PerformAction(session, hostId, "sync-guest-nic-traffics", data)
	if err != nil {
		log.Errorf("failed sync-guest-nic-traffics %s", err)
	}
}

var guestManager *SGuestManager

func Stop() {
	guestManager.ExitGuestCleanup()
}

func Init(host hostutils.IHost, serversPath string) {
	if guestManager == nil {
		guestManager = NewGuestManager(host, serversPath)
		types.HealthCheckReactor = guestManager
		types.GuestDescGetter = guestManager
	}
}

func GetGuestManager() *SGuestManager {
	return guestManager
}
