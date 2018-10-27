package baremetal

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/workqueue"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	"yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/procutils"
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
		return instance, nil
	}
	bm, err := newBaremetalInstance(m, desc)
	if err != nil {
		return nil, err
	}
	m.baremetals.Add(bm)
	return bm, nil
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

type SBaremetalInstance struct {
	manager *SBaremetalManager
	desc    jsonutils.JSONObject
}

func newBaremetalInstance(man *SBaremetalManager, desc jsonutils.JSONObject) (*SBaremetalInstance, error) {
	bm := &SBaremetalInstance{
		manager: man,
		desc:    desc,
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

func (b *SBaremetalInstance) GetDir() string {
	return filepath.Join(b.manager.configPath, b.GetId())
}

func (b *SBaremetalInstance) GetDescFilePath() string {
	return filepath.Join(b.GetDir(), "desc")
}

func (b *SBaremetalInstance) GetServerDescFilePath() string {
	return filepath.Join(b.GetDir(), "server")
}

func (b *SBaremetalInstance) GetSshConfigFilePath() string {
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
	b.desc = desc
	return ioutil.WriteFile(b.GetDescFilePath(), []byte(desc.String()), 0644)
}

func (b *SBaremetalInstance) SyncStatusBackground() {

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

func (b *SBaremetalInstance) getAdminNic() *types.Nic {
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

func (b *SBaremetalInstance) GetDHCPConfig(cliMac net.HardwareAddr) (*pxe.ResponseConfig, error) {
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

func (b *SBaremetalInstance) GetPXEDHCPConfig(arch uint16) (*pxe.ResponseConfig, error) {
	return b.getDHCPConfig(b.getAdminNic(), "", true, arch)
}

func (b *SBaremetalInstance) getDHCPConfig(
	nic *types.Nic,
	hostName string,
	isPxe bool,
	arch uint16,
) (*pxe.ResponseConfig, error) {
	if hostName == "" {
		hostName = b.GetName()
	}
	listenIP, err := b.manager.Agent.GetListenIP()
	if err != nil {
		return nil, err
	}
	return GetNicDHCPConfig(nic, listenIP.String(), hostName, isPxe, arch)
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

// TODO: imple
func (b *SBaremetalInstance) InitAdminNetif(
	cliMac net.HardwareAddr,
	netConf *types.NetworkConfig,
	nicType string,
) error {
	// start prepare task
	// sync status to PREPARE
	nic := b.getNicByMac(cliMac)
	if nic == nil || nic.WireId == "" {
		// Attach wire
	}
	return nil
}

func (b *SBaremetalInstance) RegisterNetif(
	cliMac net.HardwareAddr,
	netConf *types.NetworkConfig,
) error {
	nic := b.getNicByMac(cliMac)
	if nic == nil || nic.WireId == "" || nic.WireId != netConf.WireId {
		return b.attachWire(cliMac, netConf.WireId, nic.Type)
	}
	return nil
}

func (b *SBaremetalInstance) attachWire(mac net.HardwareAddr, wireId string, nicType string) error {
	session := b.manager.GetClientSession()
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(mac.String()), "mac")
	if nicType != "" {
		params.Add(jsonutils.NewString(nicType), "nic_type")
	}
	params.Add(jsonutils.NewString(wireId), "wire")
	params.Add(jsonutils.JSONTrue, "link_up")
	_, err := modules.Hosts.PerformAction(session, b.GetId(), "add-netif", params)
	return err
}

func (b *SBaremetalInstance) enableWire(mac net.HardwareAddr, ipAddr string, nicType string) error {
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
	_, err := modules.Hosts.PerformAction(session, b.GetId(), "enable-netif", params)
	return err
}
