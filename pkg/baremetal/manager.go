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
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	"yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	"yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
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
		if instance.getNicByMac(mac) != nil {
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
	desc      jsonutils.JSONObject
	descLock  *sync.Mutex
	taskQueue *tasks.TaskQueue
}

func newBaremetalInstance(man *SBaremetalManager, desc jsonutils.JSONObject) (*SBaremetalInstance, error) {
	bm := &SBaremetalInstance{
		manager:   man,
		desc:      desc,
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

func (b *SBaremetalInstance) SaveDesc(desc jsonutils.JSONObject) error {
	b.descLock.Lock()
	defer b.descLock.Unlock()
	b.desc = desc
	return ioutil.WriteFile(b.GetDescFilePath(), []byte(desc.String()), 0644)
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
	b.SyncSSHConfig(emptyConfig)
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

func (b *SBaremetalInstance) SyncStatus(status string) {
	log.Infof("sync baremetal %s status %s", b.GetName(), status)
	// TODO
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

func (b *SBaremetalInstance) getNicByMac(mac net.HardwareAddr) *types.Nic {
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

func (b *SBaremetalInstance) GetDHCPConfig(cliMac net.HardwareAddr) (*dhcp.ResponseConfig, error) {
	/*
		if self.get_server() is not None and (self.get_task() is None or not self.get_task().__pxe_boot__)
		nic = self.get_server().get_nic_by_mac(mac)
		hostname = self.get_server().get_name()
		else:
		nic = self.get_nic_by_mac(mac)
		hostname = None
	*/
	nic := b.getNicByMac(cliMac)
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
		//tasks.ExecuteTask(task, nil)
		log.Infof("Set task equal")
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
			[]string{status.INIT, status.PREPARE, status.PREPARE_FAIL, status.UNKNOWN}) &&
		b.GetTask() == nil && b.GetServer() == nil {
		b.SetTask(tasks.NewBaremetalServerPrepareTask(b))
		b.SyncStatus(status.PREPARE)
	}

	nic := b.getNicByMac(cliMac)
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
	nic := b.getNicByMac(cliMac)
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
	return modules.Hosts.PerformAction(session, b.GetId(), "enable-netif", params)
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

func (b *SBaremetalInstance) DoPowerShutdown(soft bool) {
	log.Infof("DoPowerShutdown")
}
