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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/workqueue"
	"yunion.io/x/pkg/utils"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	baremetalstatus "yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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
	return GetAdminSession()
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
	manager   *SBaremetalManager
	desc      *jsonutils.JSONDict
	descLock  *sync.Mutex
	taskQueue *tasks.TaskQueue
}

func newBaremetalInstance(man *SBaremetalManager, desc jsonutils.JSONObject) (*SBaremetalInstance, error) {
	bm := &SBaremetalInstance{
		manager:   man,
		desc:      desc.(*jsonutils.JSONDict),
		descLock:  new(sync.Mutex),
		taskQueue: tasks.NewTaskQueue(),
	}
	err := os.MkdirAll(bm.GetDir(), 0755)
	if err != nil {
		return nil, err
	}
	err = bm.SaveDesc(desc)
	if err != nil {
		return nil, err
	}
	// TODO: load server and init task queue
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
		log.Errorf("Clear ssh config %s error: %v", path, err)
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
	data := jsonutils.Marshal(conf)
	_, err = modules.Hosts.SetMetadata(session, b.GetId(), data)
	return err
}

func (b *SBaremetalInstance) SyncStatusBackground() {

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
	// b.SyncServerStatus(PowerStatusToServerStatus(status))
}

func (b *SBaremetalInstance) getNicInfo() *types.NicInfo {
	nicInfo := types.NicInfo{}
	err := b.desc.Unmarshal(&nicInfo)
	if err != nil {
		log.Errorf("Unmarshal desc to get nic info error: %v", err)
		return nil
	}
	return &nicInfo
}

func (b *SBaremetalInstance) getNicByType(nicType string) *types.Nic {
	nicInfo := b.getNicInfo()
	if nicInfo == nil {
		return nil
	}
	for _, nic := range nicInfo.Nics {
		tmp := nic
		if tmp.Type == nicType {
			return &tmp
		}
	}
	return nil
}

func (b *SBaremetalInstance) GetNicByMac(mac net.HardwareAddr) *types.Nic {
	nicInfo := b.getNicInfo()
	if nicInfo == nil {
		return nil
	}
	for _, nic := range nicInfo.Nics {
		tmp := nic
		if tmp.Mac == mac.String() {
			return &tmp
		}
	}
	return nil
}

func (b *SBaremetalInstance) GetAdminNic() *types.Nic {
	return b.getNicByType(NIC_TYPE_ADMIN)
}

func (b *SBaremetalInstance) NeedPXEBoot() bool {
	// TODO:
	return true
}

func (b *SBaremetalInstance) GetIPMINic(cliMac net.HardwareAddr) *types.Nic {
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
	/*
		if self.get_server() is not None and (self.get_task() is None or not self.get_task().__pxe_boot__)
		nic = self.get_server().get_nic_by_mac(mac)
		hostname = self.get_server().get_name()
		else:
		nic = self.get_nic_by_mac(mac)
		hostname = None
	*/
	nic := b.GetNicByMac(cliMac)
	if nic == nil {
		return nil, fmt.Errorf("GetNicDHCPConfig no nic found")
	}
	return b.getDHCPConfig(nic, "", false, 0)
}

func (b *SBaremetalInstance) GetPXEDHCPConfig(arch uint16) (*dhcp.ResponseConfig, error) {
	return b.getDHCPConfig(b.GetAdminNic(), "", true, arch)
}

func (b *SBaremetalInstance) getDHCPConfig(
	nic *types.Nic,
	hostName string,
	isPxe bool,
	arch uint16,
) (*dhcp.ResponseConfig, error) {
	if hostName == "" {
		hostName = b.GetName()
	}
	accessIP, err := b.manager.Agent.GetAccessIP()
	if err != nil {
		return nil, err
	}
	return GetNicDHCPConfig(nic, accessIP.String(), hostName, isPxe, arch)
}

func (b *SBaremetalInstance) GetNotifyUrl() string {
	return fmt.Sprintf("%s/baremetals/%s/notify", b.manager.Agent.GetManagerUri(), b.GetId())
}

func (b *SBaremetalInstance) GetTFTPResponse() string {
	return fmt.Sprintf(
		`default start
serial 1 115200
label start
    menu label ^Start
    menu default
    kernel kernel
    append initrd=initramfs token=%s url=%s`,
		auth.GetTokenString(), b.GetNotifyUrl())
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
		log.Infof("Set task equal")
		tasks.ExecuteTask(task, nil)
	}
}

// TODO: impl
func (b *SBaremetalInstance) InitAdminNetif(
	cliMac net.HardwareAddr,
	netConf *types.NetworkConfig,
	nicType string,
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
		return b.postAttachWire(cliMac, nicType)
	} else if nic.IpAddr == "" {
		return b.postAttachWire(cliMac, nicType)
	}
	return nil
}

func (b *SBaremetalInstance) RegisterNetif(
	cliMac net.HardwareAddr,
	netConf *types.NetworkConfig,
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
	params.Add(jsonutils.JSONTrue, "link_up")
	return modules.Hosts.PerformAction(session, b.GetId(), "add-netif", params)
}

func (b *SBaremetalInstance) postAttachWire(mac net.HardwareAddr, nicType string) error {
	ipAddr := ""
	if nicType == types.NIC_TYPE_IPMI {
		oldIPMIConf := b.GetRawIPMIConfig()
		if oldIPMIConf != nil && oldIPMIConf.IpAddr != "" {
			ipAddr = oldIPMIConf.IpAddr
		}
	}
	desc, err := b.enableWire(mac, ipAddr, nicType)
	if err != nil {
		return err
	}
	return b.SaveDesc(desc)
}

func (b *SBaremetalInstance) enableWire(mac net.HardwareAddr, ipAddr string, nicType string) (jsonutils.JSONObject, error) {
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
	log.Errorf("enable net if params: %s", params.String())
	return modules.Hosts.PerformAction(session, b.GetId(), "enable-netif", params)
}

func (b *SBaremetalInstance) GetIPMIConfig() *types.IPMIInfo {
	conf := b.GetRawIPMIConfig()
	if conf == nil || conf.Password == "" {
		return nil
	}
	if conf.Username == "" {
		sysInfo := types.IPMISystemInfo{}
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

func (b *SBaremetalInstance) GetRawIPMIConfig() *types.IPMIInfo {
	ipmiInfo := types.IPMIInfo{}
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

func (b *SBaremetalInstance) GetServer() interface{} {
	return nil
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

func (b *SBaremetalInstance) DoDiskBoot() error {
	log.Infof("Do DISK Boot ........., wait")
	b.ClearSSHConfig()
	ipmiCli := b.GetIPMITool()
	if ipmiCli != nil {
		return ipmitool.DoRebootToDisk(ipmiCli)
	}
	return fmt.Errorf("Baremetal %s ipmitool is nil", b.GetId())
}

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

func (b *SBaremetalInstance) StartBaremetalMaintenanceTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(data, "force_reboot", false) {
		b.ClearSSHConfig()
	}
	if jsonutils.QueryBoolean(data, "guest_running", false) {
		data.(*jsonutils.JSONDict).Set("soft_reboot", jsonutils.JSONTrue)
	}
	task := tasks.NewBaremetalMaintenanceTask(b, taskId, data)
	b.SetTask(task)
}

func (b *SBaremetalInstance) StartBaremetalUnmaintenanceTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	task := tasks.NewBaremetalUnmaintenanceTask(b, taskId, data)
	b.SetTask(task)
}

func (b *SBaremetalInstance) StartBaremetalReprepareTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	task := tasks.NewBaremetalReprepareTask(b, taskId, data)
	b.SetTask(task)
}

func (b *SBaremetalInstance) StartBaremetalResetBMCTask(userCred mcclient.TokenCredential, taskId string, data jsonutils.JSONObject) {
	task := tasks.NewBaremetalResetBMCTask(b, taskId, data)
	b.SetTask(task)
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
