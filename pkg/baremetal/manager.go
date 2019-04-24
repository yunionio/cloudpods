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

package baremetal

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/workqueue"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	baremetalstatus "yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	baremetaltypes "yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/baremetal/utils/detect_storages"
	"yunion.io/x/onecloud/pkg/baremetal/utils/disktool"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	raiddrivers "yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/sshpart"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/dhcp"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalManager struct {
	Agent      *SBaremetalAgent
	configPath string
	baremetals *sBaremetalMap
}

func NewBaremetalManager(agent *SBaremetalAgent) (*SBaremetalManager, error) {
	bmPaths := o.Options.BaremetalsPath
	err := os.MkdirAll(bmPaths, 0755)
	if err != nil {
		return nil, err
	}
	return &SBaremetalManager{
		Agent:      agent,
		configPath: bmPaths,
		baremetals: newBaremetalMap(),
	}, nil
}

func (m *SBaremetalManager) killAllIPMITool() {
	procutils.NewCommand("killall", "-9", "ipmitool").Run()
}

func (m *SBaremetalManager) GetClientSession() *mcclient.ClientSession {
	return m.Agent.GetAdminSession()
}

func (m *SBaremetalManager) GetZoneId() string {
	return m.Agent.Zone.Id
}

func (m *SBaremetalManager) loadConfigs() error {
	m.killAllIPMITool()
	files, err := ioutil.ReadDir(m.configPath)
	if err != nil {
		return err
	}
	bmIds := make([]string, 0)
	for _, file := range files {
		if file.IsDir() && regutils.MatchUUID(file.Name()) {
			bmIds = append(bmIds, file.Name())
		}
	}

	session := m.GetClientSession()
	errsChannel := make(chan error, len(bmIds))
	initBaremetal := func(i int) {
		bmId := bmIds[i]
		err := m.initBaremetal(session, bmId)
		if err != nil {
			errsChannel <- err
			return
		}
	}
	workqueue.Parallelize(4, len(bmIds), initBaremetal)
	errs := make([]error, 0)
	if len(errsChannel) > 0 {
		length := len(errsChannel)
		for ; length > 0; length-- {
			errs = append(errs, <-errsChannel)
		}
	}
	return errors.NewAggregate(errs)
}

func (m *SBaremetalManager) initBaremetal(session *mcclient.ClientSession, bmId string) error {
	desc, err := m.updateBaremetal(session, bmId)
	if err != nil {
		return err
	}
	bmInstance, err := m.AddBaremetal(desc)
	if err != nil {
		return err
	}
	bmObj := bmInstance.(*SBaremetalInstance)
	if !sets.NewString(INIT, PREPARE, UNKNOWN).Has(bmObj.GetStatus()) {
		bmObj.SyncStatusBackground()
	}
	return nil
}

func (m *SBaremetalManager) CleanBaremetal(bmId string) {
	bm := m.baremetals.Pop(bmId)
	if bm != nil {
		bm.Stop()
	}
	path := bm.GetDir()
	procutils.NewCommand("rm", "-fr", path).Run()
}

func (m *SBaremetalManager) updateBaremetal(session *mcclient.ClientSession, bmId string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "is_baremetal")
	obj, err := modules.Hosts.Put(session, bmId, params)
	if err != nil {
		return nil, err
	}
	log.Infof("Baremetal %s update success", bmId)
	return obj, nil
}

func (m *SBaremetalManager) AddBaremetal(desc jsonutils.JSONObject) (pxe.IBaremetalInstance, error) {
	id, err := desc.GetString("id")
	if err != nil {
		return nil, fmt.Errorf("Not found baremetal id in desc %s", desc)
	}
	if instance, ok := m.baremetals.Get(id); ok {
		return instance, instance.SaveDesc(desc)
	}
	bm, err := newBaremetalInstance(m, desc)
	if err != nil {
		return nil, err
	}
	m.baremetals.Add(bm)
	return bm, nil
}

func (m *SBaremetalManager) GetBaremetals() []*SBaremetalInstance {
	objs := make([]*SBaremetalInstance, 0)
	getter := func(key, val interface{}) bool {
		objs = append(objs, val.(*SBaremetalInstance))
		return true
	}
	m.baremetals.Range(getter)
	return objs
}

func (m *SBaremetalManager) GetBaremetalById(bmId string) *SBaremetalInstance {
	obj, _ := m.baremetals.Get(bmId)
	return obj
}

func (m *SBaremetalManager) GetBaremetalByMac(mac net.HardwareAddr) pxe.IBaremetalInstance {
	var obj *SBaremetalInstance
	getter := func(key, val interface{}) bool {
		instance := val.(*SBaremetalInstance)
		if instance.GetNicByMac(mac) != nil {
			obj = instance
			// stop the iteration
			return false
		}
		// not found, continue iteration
		return true
	}
	m.baremetals.Range(getter)
	return obj
}

func (m *SBaremetalManager) Stop() {
	for _, bm := range m.GetBaremetals() {
		bm.Stop()
	}
}

type sBaremetalMap struct {
	*sync.Map
}

func newBaremetalMap() *sBaremetalMap {
	return &sBaremetalMap{
		Map: new(sync.Map),
	}
}

func (m *sBaremetalMap) Add(bm *SBaremetalInstance) {
	m.Store(bm.GetId(), bm)
}

func (m *sBaremetalMap) Get(id string) (*SBaremetalInstance, bool) {
	obj, ok := m.Load(id)
	if !ok {
		return nil, false
	}
	return obj.(*SBaremetalInstance), true
}

func (m *sBaremetalMap) Delete(id string) {
	m.Map.Delete(id)
}

func (m *sBaremetalMap) Pop(id string) *SBaremetalInstance {
	obj, exist := m.Get(id)
	if exist {
		m.Delete(id)
	}
	return obj
}

type SBaremetalInstance struct {
	manager    *SBaremetalManager
	desc       *jsonutils.JSONDict
	descLock   *sync.Mutex
	taskQueue  *tasks.TaskQueue
	server     baremetaltypes.IBaremetalServer
	serverLock *sync.Mutex
}

func newBaremetalInstance(man *SBaremetalManager, desc jsonutils.JSONObject) (*SBaremetalInstance, error) {
	bm := &SBaremetalInstance{
		manager:    man,
		desc:       desc.(*jsonutils.JSONDict),
		descLock:   new(sync.Mutex),
		taskQueue:  tasks.NewTaskQueue(),
		serverLock: new(sync.Mutex),
	}
	err := os.MkdirAll(bm.GetDir(), 0755)
	if err != nil {
		return nil, err
	}
	err = bm.SaveDesc(desc)
	if err != nil {
		return nil, err
	}
	bm.loadServer()
	return bm, nil
}

func (b *SBaremetalInstance) GetClientSession() *mcclient.ClientSession {
	return b.manager.GetClientSession()
}

func (b *SBaremetalInstance) GetId() string {
	id, err := b.desc.GetString("id")
	if err != nil {
		log.Fatalf("Get id from desc error: %v", err)
	}
	return id
}

func (b *SBaremetalInstance) GetName() string {
	id, err := b.desc.GetString("name")
	if err != nil {
		log.Fatalf("Get name from desc error: %v", err)
	}
	return id
}

func (b *SBaremetalInstance) Stop() {
	// TODO:
}

func (b *SBaremetalInstance) GetDir() string {
	return filepath.Join(b.manager.configPath, b.GetId())
}

func (b *SBaremetalInstance) GetDescFilePath() string {
	return filepath.Join(b.GetDir(), "desc")
}

func (b *SBaremetalInstance) GetServerDescFilePath() string {
	return filepath.Join(b.GetDir(), "server")
}

func (b *SBaremetalInstance) GetSSHConfigFilePath() string {
	return filepath.Join(b.GetDir(), "ssh")
}

func (b *SBaremetalInstance) GetStatus() string {
	status, err := b.desc.GetString("status")
	if err != nil {
		log.Fatalf("Get status from desc error: %v", err)
	}
	return status
}

func (b *SBaremetalInstance) AutoSaveDesc() error {
	return b.SaveDesc(nil)
}

func (b *SBaremetalInstance) SaveDesc(desc jsonutils.JSONObject) error {
	b.descLock.Lock()
	defer b.descLock.Unlock()
	if desc != nil {
		b.desc = desc.(*jsonutils.JSONDict)
	}
	return ioutil.WriteFile(b.GetDescFilePath(), []byte(b.desc.String()), 0644)
}

func (b *SBaremetalInstance) loadServer() {
	b.serverLock.Lock()
	defer b.serverLock.Unlock()
	if !b.desc.Contains("server_id") {
		return
	}
	descPath := b.GetServerDescFilePath()
	desc, err := ioutil.ReadFile(descPath)
	if err != nil {
		log.Errorf("Failed to read server desc %s: %v", descPath, err)
		return
	}
	descObj, err := jsonutils.Parse(desc)
	if err != nil {
		log.Errorf("Failed to parse server json string: %v", err)
		return
	}
	srv, err := newBaremetalServer(b, descObj.(*jsonutils.JSONDict))
	if err != nil {
		log.Errorf("New server error: %v", err)
		return
	}
	if bmSrvId, _ := b.desc.GetString("server_id"); srv.GetId() != bmSrvId {
		log.Errorf("Server id %q not equal baremetal %q server id %q", srv.GetId(), b.GetName(), bmSrvId)
		return
	}
	b.server = srv
}

func (b *SBaremetalInstance) SaveSSHConfig(remoteAddr string, key string) error {
	var err error
	key, err = utils.EncryptAESBase64(b.GetId(), key)
	if err != nil {
		return err
	}
	sshConf := types.SSHConfig{
		Username: "root",
		Password: key,
		RemoteIP: remoteAddr,
	}
	conf := jsonutils.Marshal(sshConf)
	err = ioutil.WriteFile(b.GetSSHConfigFilePath(), []byte(conf.String()), 0644)
	if err != nil {
		return err
	}
	b.SyncSSHConfig(sshConf)
	return err
}

func (b *SBaremetalInstance) GetSSHConfig() (*types.SSHConfig, error) {
	path := b.GetSSHConfigFilePath()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	conf := types.SSHConfig{}
	obj, err := jsonutils.Parse(content)
	if err != nil {
		return nil, err
	}
	err = obj.Unmarshal(&conf)
	if err != nil {
		return nil, err
	}
	conf.Password, err = utils.DescryptAESBase64(b.GetId(), conf.Password)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

func (b *SBaremetalInstance) TestSSHConfig() bool {
	conf, err := b.GetSSHConfig()
	if err != nil {
		return false
	}
	if conf == nil {
		return false
	}
	sshCli, err := ssh.NewClient(conf.RemoteIP, 22, "root", conf.Password, "")
	if err != nil {
		return false
	}
	ret, err := sshCli.Run("whoami")
	if err != nil {
		return false
	}
	if strings.Contains(strings.Join(ret, ""), "root") {
		return true
	}
	return false
}

func (b *SBaremetalInstance) ClearSSHConfig() {
	path := b.GetSSHConfigFilePath()
	err := os.Remove(path)
	if err != nil {
		log.V(2).Warningf("Clear ssh config %s error: %v", path, err)
	}
	emptyConfig := types.SSHConfig{
		Username: "None",
		Password: "None",
		RemoteIP: "None",
	}
	err = b.SyncSSHConfig(emptyConfig)
	if err != nil {
		log.Errorf("Sync emtpy SSH config error: %v", err)
	}
}

func (b *SBaremetalInstance) SyncSSHConfig(conf types.SSHConfig) error {
	session := b.manager.GetClientSession()
	var err error
	// encrypt twice
	conf.Password, err = utils.EncryptAESBase64(b.GetId(), conf.Password)
	if err != nil {
		return err
	}
	info := &api.HostLoginInfo{
		Username: conf.Username,
		Password: conf.Password,
		Ip:       conf.RemoteIP,
	}
	data := info.JSON(info)
	_, err = modules.Hosts.SetMetadata(session, b.GetId(), data)
	return err
}

func (b *SBaremetalInstance) SyncStatusBackground() {
	go func() {
		b.AutoSyncAllStatus()
	}()
}

func PowerStatusToBaremetalStatus(status string) string {
	switch status {
	case types.POWER_STATUS_ON:
		return baremetalstatus.RUNNING
	case types.POWER_STATUS_OFF:
		return baremetalstatus.READY
	}
	return baremetalstatus.UNKNOWN
}

func PowerStatusToServerStatus(bm *SBaremetalInstance, status string) string {
	switch status {
	case types.POWER_STATUS_ON:
		if conf, _ := bm.GetSSHConfig(); conf == nil {
			return baremetalstatus.SERVER_RUNNING
		} else {
			return baremetalstatus.SERVER_ADMIN
		}
	case types.POWER_STATUS_OFF:
		return baremetalstatus.SERVER_READY
	}
	return baremetalstatus.UNKNOWN
}

func (b *SBaremetalInstance) AutoSyncStatus() {
	b.SyncStatus("", "")
}

func (b *SBaremetalInstance) SyncStatus(status string, reason string) {
	if status == "" {
		powerStatus, err := b.GetPowerStatus()
		if err != nil {
			log.Errorf("Get power status error: %v", err)
		}
		status = PowerStatusToBaremetalStatus(powerStatus)
	}
	b.desc.Set("status", jsonutils.NewString(status))
	b.AutoSaveDesc()
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(status), "status")
	if reason != "" {
		params.Add(jsonutils.NewString(reason), "reason")
	}
	_, err := modules.Hosts.PerformAction(b.GetClientSession(), b.GetId(), "status", params)
	if err != nil {
		log.Errorf("Update baremetal %s status %s error: %v", b.GetId(), status, err)
		return
	}
	log.Infof("Update baremetal %s to status %s", b.GetId(), status)
}

func (b *SBaremetalInstance) AutoSyncAllStatus() {
	b.SyncAllStatus("")
}

func (b *SBaremetalInstance) DelayedSyncStatus(_ jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	b.AutoSyncAllStatus()
	return nil, nil
}

func (b *SBaremetalInstance) SyncAllStatus(status string) {
	var err error
	if status == "" {
		status, err = b.GetPowerStatus()
		if err != nil {
			log.Errorf("Get power status error: %v", err)
		}
	}
	b.SyncStatus(PowerStatusToBaremetalStatus(status), "")
	b.SyncServerStatus(PowerStatusToServerStatus(b, status))
}

func (b *SBaremetalInstance) SyncServerStatus(status string) {
	if b.GetServerId() == "" {
		return
	}
	if status == "" {
		powerStatus, err := b.GetPowerStatus()
		if err != nil {
			log.Errorf("Get power status error: %v", err)
		}
		status = PowerStatusToServerStatus(b, powerStatus)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(status), "status")
	_, err := modules.Servers.PerformAction(b.GetClientSession(), b.GetServerId(), "status", params)
	if err != nil {
		log.Errorf("Update server %s status %s error: %v", b.GetServerName(), status, err)
		return
	}
	log.Infof("Update server %s to status %s", b.GetServerName(), status)
}

func (b *SBaremetalInstance) getNics() []types.SNic {
	nics := []types.SNic{}
	err := b.desc.Unmarshal(&nics, "nic_info")
	if err != nil {
		log.Errorf("Unmarshal desc to get nics error: %v", err)
		return nil
	}
	return nics
}

func (b *SBaremetalInstance) getNicByType(nicType string) *types.SNic {
	nics := b.getNics()
	if len(nics) == 0 {
		return nil
	}
	for _, nic := range nics {
		tmp := nic
		if tmp.Type == nicType {
			return &tmp
		}
	}
	return nil
}

func (b *SBaremetalInstance) GetNicByMac(mac net.HardwareAddr) *types.SNic {
	nics := b.getNics()
	if len(nics) == 0 {
		return nil
	}
	for _, nic := range nics {
		tmp := nic
		if tmp.Mac == mac.String() {
			return &tmp
		}
	}
	return nil
}

func (b *SBaremetalInstance) GetAdminNic() *types.SNic {
	return b.getNicByType(NIC_TYPE_ADMIN)
}

func (b *SBaremetalInstance) NeedPXEBoot() bool {
	task := b.GetTask()
	taskName := "nil"
	serverId := b.GetServerId()
	if task != nil {
		taskName = task.GetName()
	}
	taskNeedPXEBoot := false
	if task != nil && task.NeedPXEBoot() {
		taskNeedPXEBoot = true
	}
	ret := false
	if taskNeedPXEBoot || (task == nil && len(serverId) == 0 && b.GetHostType() == "baremetal") {
		ret = true
	}
	log.Infof("Check task %s, server %s NeedPXEBoot: %v", taskName, serverId, ret)
	return ret
}

func (b *SBaremetalInstance) GetHostType() string {
	hostType, _ := b.desc.GetString("host_type")
	return hostType
}

func (b *SBaremetalInstance) GetIPMINic(cliMac net.HardwareAddr) *types.SNic {
	nic := b.getNicByType(types.NIC_TYPE_IPMI)
	if nic == nil {
		return nil
	}
	if nic.Mac == cliMac.String() {
		return nic
	}
	return nil
}

func (b *SBaremetalInstance) GetIPMINicIPAddr() string {
	nic := b.getNicByType(types.NIC_TYPE_IPMI)
	if nic == nil {
		return ""
	}
	return nic.IpAddr
}

func (b *SBaremetalInstance) GetDHCPConfig(cliMac net.HardwareAddr) (*dhcp.ResponseConfig, error) {
	var nic *types.SNic
	var hostname string
	if b.GetServer() != nil && (b.GetTask() == nil || !b.GetTask().NeedPXEBoot()) {
		nic = b.GetServer().GetNicByMac(cliMac)
		hostname = b.GetServer().GetName()
	} else {
		nic = b.GetNicByMac(cliMac)
	}
	if nic == nil {
		return nil, fmt.Errorf("GetNicDHCPConfig no nic found by mac: %s", cliMac)
	}
	return b.getDHCPConfig(nic, hostname, false, 0)
}

func (b *SBaremetalInstance) GetPXEDHCPConfig(arch uint16) (*dhcp.ResponseConfig, error) {
	return b.getDHCPConfig(b.GetAdminNic(), "", true, arch)
}

func (b *SBaremetalInstance) getDHCPConfig(
	nic *types.SNic,
	hostName string,
	isPxe bool,
	arch uint16,
) (*dhcp.ResponseConfig, error) {
	if hostName == "" {
		hostName = b.GetName()
	}
	serverIP, err := b.manager.Agent.GetDHCPServerIP()
	if err != nil {
		return nil, err
	}
	return GetNicDHCPConfig(nic, serverIP.String(), hostName, isPxe, arch)
}

func (b *SBaremetalInstance) GetNotifyUrl() string {
	return fmt.Sprintf("%s/baremetals/%s/notify", b.manager.Agent.GetManagerUri(), b.GetId())
}

func (b *SBaremetalInstance) GetTFTPResponse() string {
	resp := `default start
serial 1 115200

label start
    menu label ^Start
    menu default
`

	if b.NeedPXEBoot() {
		resp += "    kernel kernel\n"
		resp += fmt.Sprintf("    append initrd=initramfs token=%s url=%s",
			auth.GetTokenString(), b.GetNotifyUrl())
	} else {
		resp += "    COM32 chain.c32\n"
		resp += "    APPEND hd0 0\n"
	}
	log.Infof("%s", resp)
	return resp
}

func (b *SBaremetalInstance) GetTaskQueue() *tasks.TaskQueue {
	return b.taskQueue
}

func (b *SBaremetalInstance) GetTask() tasks.ITask {
	return b.taskQueue.GetTask()
}

func (b *SBaremetalInstance) SetTask(task tasks.ITask) {
	b.taskQueue.AppendTask(task)
	if reflect.DeepEqual(task, b.taskQueue.GetTask()) {
		log.Infof("Set task equal, ExecuteTask %s", task.GetName())
		tasks.ExecuteTask(task, nil)
	}
}

func (b *SBaremetalInstance) InitAdminNetif(
	cliMac net.HardwareAddr,
	netConf *types.SNetworkConfig,
	nicType string,
	netType string,
) error {
	// start prepare task
	// sync status to PREPARE
	if nicType == types.NIC_TYPE_ADMIN &&
		utils.IsInStringArray(b.GetStatus(),
			[]string{baremetalstatus.INIT,
				baremetalstatus.PREPARE,
				baremetalstatus.PREPARE_FAIL,
				baremetalstatus.UNKNOWN}) &&
		b.GetTask() == nil && b.GetServer() == nil {
		b.SetTask(tasks.NewBaremetalServerPrepareTask(b))
		b.SyncStatus(baremetalstatus.PREPARE, "")
	}

	nic := b.GetNicByMac(cliMac)
	if nic == nil || nic.WireId == "" {
		_, err := b.attachWire(cliMac, netConf.WireId, nicType)
		if err != nil {
			return err
		}
		return b.postAttachWire(cliMac, nicType, netType)
	} else if nic.IpAddr == "" {
		return b.postAttachWire(cliMac, nicType, netType)
	}
	return nil
}

func (b *SBaremetalInstance) RegisterNetif(
	cliMac net.HardwareAddr,
	netConf *types.SNetworkConfig,
) error {
	nic := b.GetNicByMac(cliMac)
	if nic == nil || nic.WireId == "" || nic.WireId != netConf.WireId {
		desc, err := b.attachWire(cliMac, netConf.WireId, nic.Type)
		if err != nil {
			return err
		}
		return b.SaveDesc(desc)
	}
	return nil
}

func (b *SBaremetalInstance) attachWire(mac net.HardwareAddr, wireId string, nicType string) (jsonutils.JSONObject, error) {
	session := b.manager.GetClientSession()
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(mac.String()), "mac")
	if nicType != "" {
		params.Add(jsonutils.NewString(nicType), "nic_type")
	}
	params.Add(jsonutils.NewString(wireId), "wire")
	params.Add(jsonutils.NewInt(-1), "index")
	params.Add(jsonutils.JSONTrue, "link_up")
	return modules.Hosts.PerformAction(session, b.GetId(), "add-netif", params)
}

func (b *SBaremetalInstance) postAttachWire(mac net.HardwareAddr, nicType string, netType string) error {
	ipAddr := ""
	if nicType == types.NIC_TYPE_IPMI {
		oldIPMIConf := b.GetRawIPMIConfig()
		if oldIPMIConf != nil && oldIPMIConf.IpAddr != "" {
			ipAddr = oldIPMIConf.IpAddr
		}
	}
	desc, err := b.enableWire(mac, ipAddr, nicType, netType)
	if err != nil {
		return err
	}
	return b.SaveDesc(desc)
}

func (b *SBaremetalInstance) enableWire(mac net.HardwareAddr, ipAddr string, nicType string, netType string) (jsonutils.JSONObject, error) {
	session := b.manager.GetClientSession()
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(mac.String()), "mac")
	if nicType != "" {
		params.Add(jsonutils.NewString(nicType), "nic_type")
	}
	if ipAddr != "" {
		params.Add(jsonutils.NewString(ipAddr), "ip_addr")
	}
	if nicType == types.NIC_TYPE_IPMI {
		params.Add(jsonutils.NewString("stepup"), "alloc_dir") // alloc bottom up
	}
	if len(netType) > 0 {
		params.Add(jsonutils.NewString(netType), "net_type")
	}
	log.Errorf("enable net if params: %s", params.String())
	return modules.Hosts.PerformAction(session, b.GetId(), "enable-netif", params)
}

func (b *SBaremetalInstance) GetIPMIConfig() *types.SIPMIInfo {
	conf := b.GetRawIPMIConfig()
	if conf == nil || conf.Password == "" {
		return nil
	}
	if conf.Username == "" {
		sysInfo := types.SIPMISystemInfo{}
		err := b.desc.Unmarshal(&sysInfo, "sys_info")
		if err != nil {
			log.Errorf("Unmarshal get sys_info error: %v", err)
		}
		conf.Username = profiles.GetRootName(&sysInfo)
	}
	if conf.IpAddr == "" {
		nicIPAddr := b.GetIPMINicIPAddr()
		if nicIPAddr != "" {
			conf.IpAddr = nicIPAddr
		}
	}
	conf.Password = utils.Unquote(conf.Password) // XXX: remove quotes!!!
	if conf.IpAddr == "" {
		return nil
	}
	return conf
}

func (b *SBaremetalInstance) GetRawIPMIConfig() *types.SIPMIInfo {
	ipmiInfo := types.SIPMIInfo{}
	err := b.desc.Unmarshal(&ipmiInfo, "ipmi_info")
	if err != nil {
		log.Errorf("Unmarshal IPMIInfo error: %v", err)
		return nil
	}
	if ipmiInfo.Password != "" {
		ipmiInfo.Password, err = utils.DescryptAESBase64(b.GetId(), ipmiInfo.Password)
		if err != nil {
			log.Errorf("DescryptAESBase64 IPMI password error: %v", err)
			return nil
		}
	}
	return &ipmiInfo
}

func (b *SBaremetalInstance) GetServer() baremetaltypes.IBaremetalServer {
	b.serverLock.Lock()
	defer b.serverLock.Unlock()
	if !b.desc.Contains("server_id") && b.server != nil {
		log.Warningf("baremetal %s server_id not present, remove server %q", b.GetName(), b.server.GetName())
		b.RemoveServer()
		return nil
	}
	return b.server
}

func (b *SBaremetalInstance) GetServerId() string {
	srv := b.GetServer()
	if srv == nil {
		return ""
	}
	return srv.GetId()
}

func (b *SBaremetalInstance) GetServerName() string {
	srv := b.GetServer()
	if srv == nil {
		return ""
	}
	return srv.GetName()
}

func (b *SBaremetalInstance) RemoveServer() {
	b.serverLock.Lock()
	defer b.serverLock.Unlock()
	b.removeServer()
}

func (b *SBaremetalInstance) removeServer() {
	if b.server != nil {
		b.server.RemoveDesc()
		b.server = nil
	}
}

func (b *SBaremetalInstance) SetExistingIPMIIPAddr(ipAddr string) {
	info, _ := b.desc.Get("ipmi_info")
	if info == nil {
		info = jsonutils.NewDict()
	}
	oIPAddr, _ := info.GetString("ip_addr")
	if oIPAddr == "" {
		info.(*jsonutils.JSONDict).Add(jsonutils.NewString(ipAddr), "ip_addr")
	}
	b.desc.Set("ipmi_info", info)
}

func (b *SBaremetalInstance) GetIPMITool() *ipmitool.LanPlusIPMI {
	conf := b.GetIPMIConfig()
	if conf == nil {
		return nil
	}
	return ipmitool.NewLanPlusIPMI(conf.IpAddr, conf.Username, conf.Password)
}

func (b *SBaremetalInstance) GetIPMILanChannel() int {
	conf := b.GetIPMIConfig()
	if conf == nil {
		return 0
	}
	return conf.LanChannel
}

func (b *SBaremetalInstance) DoPXEBoot() error {
	log.Infof("Do PXE Boot ........., wait")
	b.ClearSSHConfig()
	ipmiCli := b.GetIPMITool()
	if ipmiCli != nil {
		return ipmitool.DoRebootToPXE(ipmiCli)
	}
	return fmt.Errorf("Baremetal %s ipmitool is nil", b.GetId())
}

/*
func (b *SBaremetalInstance) DoDiskBoot() error {
	log.Infof("Do DISK Boot ........., wait")
	b.ClearSSHConfig()
	ipmiCli := b.GetIPMITool()
	if ipmiCli != nil {
		return ipmitool.DoRebootToDisk(ipmiCli)
	}
	return fmt.Errorf("Baremetal %s ipmitool is nil", b.GetId())
}
*/

func (b *SBaremetalInstance) GetPowerStatus() (string, error) {
	ipmiCli := b.GetIPMITool()
	if ipmiCli == nil {
		return "", fmt.Errorf("Baremetal %s ipmitool is nil", b.GetId())
	}
	return ipmitool.GetChassisPowerStatus(ipmiCli)
}

func (b *SBaremetalInstance) DoPowerShutdown(soft bool) error {
	b.ClearSSHConfig()
	ipmiCli := b.GetIPMITool()
	if ipmiCli != nil {
		if soft {
			return ipmitool.DoSoftShutdown(ipmiCli)
		}
		return ipmitool.DoHardShutdown(ipmiCli)
	}
	return fmt.Errorf("Baremetal %s ipmitool is nil", b.GetId())
}

func (b *SBaremetalInstance) GetStorageDriver() string {
	driver, _ := b.desc.GetString("storage_driver")
	return driver
}

func (b *SBaremetalInstance) GetZoneId() string {
	return b.manager.GetZoneId()
}

func (b *SBaremetalInstance) DelayedRemove(_ jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	b.remove()
	return nil, nil
}

func (b *SBaremetalInstance) remove() {
	b.manager.CleanBaremetal(b.GetId())
	b.manager = nil
	b.desc = nil
}

func (b *SBaremetalInstance) StartNewTask(factory tasks.TaskFactory, taskId string, data jsonutils.JSONObject) {
	go func() {
		task, err := factory(b, taskId, data)
		if err != nil {
			tasks.SetTaskFail(task, err)
			return
		}
		b.SetTask(task)
	}()
}

func (b *SBaremetalInstance) StartBaremetalMaintenanceTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(data, "force_reboot", false) {
		b.ClearSSHConfig()
	}
	if jsonutils.QueryBoolean(data, "guest_running", false) {
		data.(*jsonutils.JSONDict).Set("soft_reboot", jsonutils.JSONTrue)
	}
	b.StartNewTask(tasks.NewBaremetalMaintenanceTask, taskId, data)
}

func (b *SBaremetalInstance) StartBaremetalUnmaintenanceTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	b.StartNewTask(tasks.NewBaremetalUnmaintenanceTask, taskId, data)
}

func (b *SBaremetalInstance) StartBaremetalReprepareTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	b.StartNewTask(tasks.NewBaremetalReprepareTask, taskId, data)
}

func (b *SBaremetalInstance) StartBaremetalResetBMCTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) error {
	b.StartNewTask(tasks.NewBaremetalResetBMCTask, taskId, data)
	return nil
}

func (b *SBaremetalInstance) DelayedServerReset(_ jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := b.DoPXEBoot()
	return nil, err
}

func (b *SBaremetalInstance) StartServerCreateTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) error {
	b.serverLock.Lock()
	defer b.serverLock.Unlock()
	if b.server != nil {
		return fmt.Errorf("Baremetal %s already have server %s", b.GetName(), b.server.GetName())
	}
	descData, err := data.Get("desc")
	if err != nil {
		return fmt.Errorf("Create data not found server desc: %v", err)
	}
	server, err := newBaremetalServer(b, descData.(*jsonutils.JSONDict))
	if err != nil {
		return fmt.Errorf("New server error: %v", err)
	}
	b.server = server
	b.desc.Set("server_id", jsonutils.NewString(b.server.GetId()))
	if err := b.AutoSaveDesc(); err != nil {
		return err
	}
	b.StartNewTask(tasks.NewBaremetalServerCreateTask, taskId, data)
	return nil
}

func (b *SBaremetalInstance) StartServerDeployTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) error {
	desc, err := data.Get("desc")
	if err != nil {
		return fmt.Errorf("Not found desc in data")
	}
	if err := b.GetServer().SaveDesc(desc); err != nil {
		return fmt.Errorf("Save server desc: %v", err)
	}
	b.StartNewTask(tasks.NewBaremetalServerDeployTask, taskId, data)
	return nil
}

func (b *SBaremetalInstance) StartServerRebuildTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) error {
	desc, err := data.Get("desc")
	if err != nil {
		return fmt.Errorf("Not found desc in data")
	}
	if err := b.GetServer().SaveDesc(desc); err != nil {
		return fmt.Errorf("Save server desc: %v", err)
	}
	b.StartNewTask(tasks.NewBaremetalServerRebuildTask, taskId, data)
	return nil
}

func (b *SBaremetalInstance) StartServerStartTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) error {
	b.StartNewTask(tasks.NewBaremetalServerStartTask, taskId, data)
	return nil
}

func (b *SBaremetalInstance) StartServerStopTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) error {
	b.StartNewTask(tasks.NewBaremetalServerStopTask, taskId, data)
	return nil
}

func (b *SBaremetalInstance) StartServerDestroyTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	b.StartNewTask(tasks.NewBaremetalServerDestroyTask, taskId, data)
}

func (b *SBaremetalInstance) DelayedSyncIPMIInfo(data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ipmiCli := b.GetIPMITool()
	lanChannel := b.GetIPMILanChannel()
	sysInfo, err := ipmitool.GetSysInfo(ipmiCli)
	if err != nil {
		return nil, err
	}
	if lanChannel <= 0 {
		lanChannel = ipmitool.GetDefaultLanChannel(sysInfo)
	}
	retObj := make(map[string]string)
	if ipAddr, _ := data.GetString("ip_addr"); ipAddr != "" {
		err = ipmitool.SetLanStaticIP(ipmiCli, lanChannel, ipAddr)
		if err != nil {
			return nil, err
		}
		// TODO: netutils.wait_ip_alive(ipAddr, 120)
		retObj["ipmi_ip_addr"] = ipAddr
	}
	if passwd, _ := data.GetString("password"); passwd != "" {
		err = ipmitool.SetLanPasswd(ipmiCli, ipmitool.GetRootId(sysInfo), passwd)
		if err != nil {
			return nil, err
		}
		retObj["ipmi_password"] = passwd
	}
	return jsonutils.Marshal(retObj), nil
}

func (b *SBaremetalInstance) DelayedSyncDesc(data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := b.SaveDesc(data)
	return nil, err
}

func (b *SBaremetalInstance) DelayedServerStatus(data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ps, err := b.GetPowerStatus()
	if err != nil {
		return nil, err
	}
	status := PowerStatusToServerStatus(b, ps)
	resp := jsonutils.NewDict()
	resp.Add(jsonutils.NewString(status), "status")
	return resp, err
}

type SBaremetalServer struct {
	baremetal *SBaremetalInstance
	desc      *jsonutils.JSONDict
}

func newBaremetalServer(baremetal *SBaremetalInstance, desc *jsonutils.JSONDict) (*SBaremetalServer, error) {
	server := &SBaremetalServer{
		baremetal: baremetal,
		desc:      desc,
	}
	err := server.SaveDesc(desc)
	return server, err
}

func (server *SBaremetalServer) GetId() string {
	id, err := server.desc.GetString("uuid")
	if err != nil {
		log.Fatalf("Get id from desc error: %v", err)
	}
	return id
}

func (server *SBaremetalServer) GetName() string {
	id, err := server.desc.GetString("name")
	if err != nil {
		log.Fatalf("Get name from desc error: %v", err)
	}
	return id
}

func (server *SBaremetalServer) SaveDesc(desc jsonutils.JSONObject) error {
	if desc != nil {
		server.desc = desc.(*jsonutils.JSONDict)
	}
	return ioutil.WriteFile(server.baremetal.GetServerDescFilePath(), []byte(server.desc.String()), 0644)
}

func (s *SBaremetalServer) RemoveDesc() {
	os.Remove(s.baremetal.GetServerDescFilePath())
	s.desc = nil
	s.baremetal = nil
}

func (s *SBaremetalServer) GetRootTemplateId() string {
	rootDisk, err := s.desc.GetAt(0, "disks")
	if err != nil {
		log.Errorf("Can't found root disk")
		return ""
	}
	id, _ := rootDisk.GetString("template_id")
	return id
}

func (s *SBaremetalServer) GetDiskConfig() ([]*api.BaremetalDiskConfig, error) {
	layouts := make([]baremetal.Layout, 0)
	err := s.desc.Unmarshal(&layouts, "disk_config")
	if err != nil {
		return nil, err
	}
	return baremetal.GetLayoutRaidConfig(layouts), nil
}

func (s *SBaremetalServer) DoDiskConfig(term *ssh.Client) error {
	raid, nonRaid, pcie, err := detect_storages.DetectStorageInfo(term, true)
	if err != nil {
		return err
	}
	storages := make([]*baremetal.BaremetalStorage, 0)
	storages = append(storages, raid...)
	storages = append(storages, nonRaid...)
	storages = append(storages, pcie...)
	confs, err := s.GetDiskConfig()
	if err != nil {
		return err
	}
	layouts, err := baremetal.CalculateLayout(confs, storages)
	if err != nil {
		return fmt.Errorf("CalculateLayout: %v", err)
	}
	diskConfs := baremetal.GroupLayoutResultsByDriverAdapter(layouts)
	for _, dConf := range diskConfs {
		driver := dConf.Driver
		raidDrv := raiddrivers.GetDriver(driver, term)
		if raidDrv != nil {
			if err := raidDrv.ParsePhyDevs(); err != nil {
				return fmt.Errorf("RaidDriver %s parse physical devices: %v", raidDrv.GetName(), err)
			}
			raidDrv.CleanRaid()
		}
	}

	for _, dConf := range diskConfs {
		driver := dConf.Driver
		adapter := dConf.Adapter
		raidDrv := raiddrivers.GetDriver(driver, term)
		if raidDrv != nil {
			if err := raidDrv.ParsePhyDevs(); err != nil {
				return fmt.Errorf("RaidDriver %s parse physical devices: %v", raidDrv.GetName(), err)
			}
			if err := raiddrivers.BuildRaid(raidDrv, dConf.Configs, adapter); err != nil {
				return fmt.Errorf("Build %s raid failed: %v", raidDrv.GetName(), err)
			}
			time.Sleep(10 * time.Second) // wait 10 seconds for raid status OK
		}
	}

	tool := disktool.NewSSHPartitionTool(term)
	tool.FetchDiskConfs(baremetal.GetDiskConfigurations(layouts))
	err = tool.RetrieveDiskInfo()
	if err != nil {
		return err
	}
	maxTries := 60
	for tried := 0; !tool.IsAllDisksReady() && tried < maxTries; tried++ {
		time.Sleep(5 * time.Second)
		tool.RetrieveDiskInfo()
	}

	if !tool.IsAllDisksReady() {
		return fmt.Errorf("Raid disks are not ready???")
	}

	return nil
}

func (s *SBaremetalServer) DoDiskUnconfig(term *ssh.Client) error {
	// tear down raid
	driver := s.baremetal.GetStorageDriver()
	raidDrv := raiddrivers.GetDriver(driver, term)
	if raidDrv != nil {
		if err := raidDrv.ParsePhyDevs(); err != nil {
			return err
		}
		raidDrv.CleanRaid()
	}
	return nil
}

func (s *SBaremetalServer) DoEraseDisk(term *ssh.Client) error {
	cmd := "/lib/mos/partdestroy.sh"
	_, err := term.Run(cmd)
	return err
}

func (s *SBaremetalServer) doCreateRoot(term *ssh.Client, devName string) error {
	session := s.baremetal.GetClientSession()
	token := session.GetToken().GetTokenString()
	url, err := session.GetServiceURL("image", "internalURL")
	if err != nil {
		return err
	}
	imageId := s.GetRootTemplateId()
	cmd := fmt.Sprintf("/lib/mos/rootcreate.sh %s %s %s %s", token, url, imageId, devName)
	log.Infof("rootcreate cmd: %q", cmd)
	if _, err := term.Run(cmd); err != nil {
		return fmt.Errorf("Root create fail: %v", err)
	}
	return nil
}

func (s *SBaremetalServer) DoPartitionDisk(term *ssh.Client) ([]*disktool.Partition, error) {
	raid, nonRaid, pcie, err := detect_storages.DetectStorageInfo(term, false)
	if err != nil {
		return nil, err
	}
	storages := make([]*baremetal.BaremetalStorage, 0)
	storages = append(storages, raid...)
	storages = append(storages, nonRaid...)
	storages = append(storages, pcie...)
	confs, err := s.GetDiskConfig()
	if err != nil {
		return nil, err
	}
	layouts, err := baremetal.CalculateLayout(confs, storages)
	if err != nil {
		return nil, err
	}

	tool := disktool.NewSSHPartitionTool(term)
	tool.FetchDiskConfs(baremetal.GetDiskConfigurations(layouts))
	err = tool.RetrieveDiskInfo()
	if err != nil {
		return nil, err
	}

	disks, _ := s.desc.GetArray("disks")
	if len(disks) == 0 {
		return nil, fmt.Errorf("Empty disks in desc")
	}

	rootDisk := disks[0]
	rootSize, _ := rootDisk.Int("size")
	err = s.doCreateRoot(term, tool.GetRootDisk().GetDevName())
	if err != nil {
		return nil, fmt.Errorf("Failed to create root: %v", err)
	}

	tool.RetrievePartitionInfo()
	parts := tool.GetPartitions()
	if len(parts) == 0 {
		return nil, fmt.Errorf("Root disk create failed, no partitions")
	}
	log.Infof("Resize root to %d MB", rootSize)
	if err := tool.ResizePartition(0, rootSize); err != nil {
		return nil, fmt.Errorf("Fail to resize root to %d, err: %v", rootSize, err)
	}
	if len(disks) > 1 {
		for _, disk := range disks[1:] {
			sz, err := disk.Int("size")
			if err != nil {
				sz = -1
			}
			fs, _ := disk.GetString("fs")
			uuid, _ := disk.GetString("disk_id")
			driver, _ := disk.GetString("driver")
			log.Infof("Create partition %d %s", sz, fs)
			if err := tool.CreatePartition(-1, sz, fs, true, driver, uuid); err != nil {
				return nil, fmt.Errorf("Fail to create disk %s: %v", disk.String(), err)
			}
		}
	}
	log.Infof("Finish create partitions")

	return tool.GetPartitions(), nil
}

func (s *SBaremetalServer) DoRebuildRootDisk(term *ssh.Client) ([]*disktool.Partition, error) {
	raid, nonRaid, pcie, err := detect_storages.DetectStorageInfo(term, false)
	if err != nil {
		return nil, err
	}
	storages := make([]*baremetal.BaremetalStorage, 0)
	storages = append(storages, raid...)
	storages = append(storages, nonRaid...)
	storages = append(storages, pcie...)
	confs, err := s.GetDiskConfig()
	if err != nil {
		return nil, err
	}
	layouts, err := baremetal.CalculateLayout(confs, storages)
	if err != nil {
		return nil, err
	}

	tool := disktool.NewSSHPartitionTool(term)
	tool.FetchDiskConfs(baremetal.GetDiskConfigurations(layouts))
	err = tool.RetrieveDiskInfo()
	if err != nil {
		return nil, err
	}

	disks, _ := s.desc.GetArray("disks")
	if len(disks) == 0 {
		return nil, fmt.Errorf("Empty disks in desc")
	}

	rootDisk := disks[0]
	rootSize, _ := rootDisk.Int("size")
	err = s.doCreateRoot(term, tool.GetRootDisk().GetDevName())
	if err != nil {
		return nil, fmt.Errorf("Failed to create root: %v", err)
	}

	tool.RetrievePartitionInfo()

	log.Infof("Resize root to %d MB", rootSize)
	if err := tool.ResizePartition(0, rootSize); err != nil {
		return nil, fmt.Errorf("Fail to resize root to %d, err: %v", rootSize, err)
	}
	if len(disks) > 1 {
		for _, disk := range disks[1:] {
			sz, err := disk.Int("size")
			if err != nil {
				sz = -1
			}
			fs, _ := disk.GetString("fs")
			uuid, _ := disk.GetString("disk_id")
			driver, _ := disk.GetString("driver")
			log.Infof("Create partition %d %s", sz, fs)
			if err := tool.CreatePartition(0, sz, fs, false, driver, uuid); err != nil {
				log.Errorf("Rebuild root create (%s, %d, %s, %s) partition error: %v", uuid, sz, fs, driver, err)
				break
			}
		}
	}
	log.Infof("Finish create partitions")

	return tool.GetPartitions(), nil
}

func (s *SBaremetalServer) SyncPartitionSize(term *ssh.Client, parts []*disktool.Partition) ([]jsonutils.JSONObject, error) {
	disks, _ := s.desc.GetArray("disks")
	rootPartsCnt := len(parts) - len(disks) + 1
	rootParts := parts[0:rootPartsCnt]
	dataParts := parts[rootPartsCnt:]
	idx := 0
	size := (rootParts[len(rootParts)-1].GetEnd() + 1) * 512 / 1024 / 1024
	disks[idx].(*jsonutils.JSONDict).Set("size", jsonutils.NewInt(int64(size)))
	idx += 1
	for _, p := range dataParts {
		sizeMB, err := p.GetSizeMB()
		if err != nil {
			return nil, err
		}
		disks[idx].(*jsonutils.JSONDict).Set("size", jsonutils.NewInt(int64(sizeMB)))
		disks[idx].(*jsonutils.JSONDict).Set("dev", jsonutils.NewString(p.GetDev()))
		idx++
	}
	return disks, nil
}

func (s *SBaremetalServer) DoDeploy(term *ssh.Client, data jsonutils.JSONObject, isInit bool) (jsonutils.JSONObject, error) {
	publicKey := sshkeys.GetKeys(data)
	deploys, _ := data.GetArray("deploys")
	password, _ := data.GetString("password")
	resetPassword := jsonutils.QueryBoolean(data, "reset_password", false)
	if resetPassword && len(password) == 0 {
		password = seclib.RandomPassword(12)
	}
	deployInfo := guestfs.NewDeployInfo(publicKey, deploys, password, isInit, true, o.Options.LinuxDefaultRootUser, o.Options.WindowsDefaultAdminUser)
	return s.deployFs(term, deployInfo)
}

func (s *SBaremetalServer) deployFs(term *ssh.Client, deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error) {
	raid, nonRaid, pcie, err := detect_storages.DetectStorageInfo(term, false)
	if err != nil {
		return nil, err
	}
	storages := make([]*baremetal.BaremetalStorage, 0)
	storages = append(storages, raid...)
	storages = append(storages, nonRaid...)
	storages = append(storages, pcie...)
	confs, err := s.GetDiskConfig()
	if err != nil {
		return nil, err
	}
	layouts, err := baremetal.CalculateLayout(confs, storages)
	if err != nil {
		return nil, err
	}
	rootDev, rootfs, err := sshpart.MountSSHRootfs(term, layouts)
	if err != nil {
		return nil, fmt.Errorf("Find rootfs error: %s", err)
	}
	defer rootDev.Umount()
	if strings.ToLower(rootfs.GetOs()) == "windows" {
		return nil, fmt.Errorf("Unsupported OS: %s", rootfs.GetOs())
	}
	return guestfs.DeployGuestFs(rootfs, s.desc, deployInfo)
}

func (s *SBaremetalServer) GetNics() []types.SServerNic {
	nics := []types.SServerNic{}
	err := s.desc.Unmarshal(&nics, "nics")
	if err != nil {
		log.Errorf("Unmarshal desc to get server nics error: %v", err)
		return nil
	}
	return nics
}

func (s *SBaremetalServer) GetNicByMac(mac net.HardwareAddr) *types.SNic {
	for _, n := range s.GetNics() {
		if n.GetMac().String() == mac.String() {
			nic := n.ToNic()
			return &nic
		}
	}
	return nil
}
