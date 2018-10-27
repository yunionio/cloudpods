package baremetal

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	BaremetalAgent *SBaremetalAgent
)

type SZone struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type SBaremetalAgent struct {
	PXEServer       *pxe.Server
	ListenInterface *net.Interface
	AgentId         string
	AgentName       string
	Zone            *SZone
}

func newBaremetalAgent() (*SBaremetalAgent, error) {
	iface, err := net.InterfaceByName(o.Options.ListenInterface)
	if err != nil {
		return nil, err
	}
	ips, err := getIfaceIPs(iface)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("Interface %s ip address not found", o.Options.ListenInterface)
	}

	agent := &SBaremetalAgent{
		ListenInterface: iface,
	}
	return agent, nil
}

func GetAdminSession() *mcclient.ClientSession {
	return auth.GetAdminSession(o.Options.Region, "v2")
}

func (agent *SBaremetalAgent) GetListenIP() (net.IP, error) {
	ips, err := getIfaceIPs(agent.ListenInterface)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("Interface %s ip address not found", agent.ListenInterface.Name)
	}
	return ips[0], nil
}

func getIfaceIPs(iface *net.Interface) ([]net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0)
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	return ips, nil
}

func (agent *SBaremetalAgent) startRegister() {
	if agent.AgentId != "" {
		return
	}

	var delayRetryTime time.Duration = 30 * time.Second

	for {
		err := agent.register()
		if err != nil {
			log.Errorf("Register error: %v, retry after %s...", err, delayRetryTime)
			time.Sleep(delayRetryTime)
			continue
		}
		break
	}
	return
}

func (agent *SBaremetalAgent) register() error {
	session := GetAdminSession()
	var err error
	err = agent.fetchZone(session)
	if err != nil {
		return err
	}
	err = agent.createOrUpdateBaremetalAgent(session)
	if err != nil {
		return err
	}
	log.Infof("Baremetal %s:%s register success, do offline", agent.AgentName, agent.AgentId)
	err = agent.doOffline(session)
	if err != nil {
		return err
	}

	agent.tuneSystem()

	manager, err := NewBaremetalManager(agent)
	if err != nil {
		return fmt.Errorf("New baremetal manager error: %v", err)
	}

	err = manager.loadConfigs()
	if err != nil {
		return fmt.Errorf("Baremetal manager load config error: %v", err)
	}

	agent.startServices(manager)
	return nil
}

func (agent *SBaremetalAgent) getZoneByIP(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	listenIP, err := agent.GetListenIP()
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(listenIP.String()), "ip")
	networks, err := modules.Networks.List(session, params)
	if err != nil {
		return nil, err
	}
	if len(networks.Data) == 0 {
		return nil, fmt.Errorf("Not found networks by agent listen ip: %s", listenIP)
	}
	wireId, err := networks.Data[0].GetString("wire_id")
	if err != nil {
		return nil, err
	}
	wire, err := modules.Wires.Get(session, wireId, nil)
	if err != nil {
		return nil, err
	}
	zoneId, err := wire.GetString("zone_id")
	if err != nil {
		return nil, err
	}

	zone, err := modules.Zones.Get(session, zoneId, nil)
	if err != nil {
		return nil, err
	}
	return zone, nil
}

func (agent *SBaremetalAgent) fetchZone(session *mcclient.ClientSession) error {
	zoneName := o.Options.Zone
	var zoneInfoObj jsonutils.JSONObject
	var err error
	if zoneName != "" {
		zoneInfoObj, err = modules.Zones.Get(session, zoneName, nil)
	} else {
		zoneInfoObj, err = agent.getZoneByIP(session)
	}
	if err != nil {
		return err
	}
	zone := SZone{}
	err = zoneInfoObj.Unmarshal(&zone)
	if err != nil {
		return err
	}
	agent.Zone = &zone
	return nil
}

func (agent *SBaremetalAgent) createOrUpdateBaremetalAgent(session *mcclient.ClientSession) error {
	params := jsonutils.NewDict()
	listenIP, err := agent.GetListenIP()
	if err != nil {
		return err
	}
	params.Add(jsonutils.NewString(listenIP.String()), "access_ip")
	ret, err := modules.Baremetalagents.List(session, params)
	if err != nil {
		return err
	}
	var (
		cloudObj  jsonutils.JSONObject
		agentId   string
		agentName string
	)
	// create or update BaremetalAgent
	if len(ret.Data) == 0 {
		cloudObj, err = agent.createBaremetalAgent(session)
		if err != nil {
			return err
		}
	} else {
		cloudBmAgent := ret.Data[0]
		accessIP, _ := cloudBmAgent.GetString("access_ip")
		managerUri, _ := cloudBmAgent.GetString("manager_uri")
		zoneId, _ := cloudBmAgent.GetString("zone_id")
		agentId, _ := cloudBmAgent.GetString("id")
		if listenIP.String() != accessIP ||
			agent.GetManagerUri() != managerUri ||
			zoneId != agent.Zone.Id {
			cloudObj, err = agent.updateBaremetalAgent(session, agentId)
			if err != nil {
				return err
			}
		} else {
			cloudObj = cloudBmAgent
		}
	}

	agentId, err = cloudObj.GetString("id")
	if err != nil {
		return err
	}
	agentName, err = cloudObj.GetString("name")
	if err != nil {
		return err
	}

	agent.AgentId = agentId
	agent.AgentName = agentName
	return nil
}

func (agent *SBaremetalAgent) GetManagerUri() string {
	listenIP, _ := agent.GetListenIP()
	proto := "http"
	if o.Options.EnableSsl {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s:%d", proto, listenIP, o.Options.Port)
}

func (agent *SBaremetalAgent) getCreateUpdateInfo() (jsonutils.JSONObject, error) {
	listenIP, err := agent.GetListenIP()
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	if agent.AgentId == "" {
		params.Add(jsonutils.NewString(fmt.Sprintf("baremetal_%s", listenIP)), "name")
	}
	params.Add(jsonutils.NewString(listenIP.String()), "access_ip")
	params.Add(jsonutils.NewString(agent.GetManagerUri()), "manager_uri")
	params.Add(jsonutils.NewString(agent.Zone.Id), "zone_id")
	return params, nil
}

func (agent *SBaremetalAgent) createBaremetalAgent(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	params, err := agent.getCreateUpdateInfo()
	if err != nil {
		return nil, err
	}
	return modules.Baremetalagents.Create(session, params)
}

func (agent *SBaremetalAgent) updateBaremetalAgent(session *mcclient.ClientSession, id string) (jsonutils.JSONObject, error) {
	params, err := agent.getCreateUpdateInfo()
	if err != nil {
		return nil, err
	}
	return modules.Baremetalagents.Update(session, id, params)
}

func (agent *SBaremetalAgent) doOffline(session *mcclient.ClientSession) error {
	_, err := modules.Baremetalagents.PerformAction(session, agent.AgentId, "offline", nil)
	return err
}

func (agent *SBaremetalAgent) doOnline(session *mcclient.ClientSession) error {
	_, err := modules.Baremetalagents.PerformAction(session, agent.AgentId, "online", nil)
	return err
}

func (agent *SBaremetalAgent) tuneSystem() {
	agent.disableUDPOffloading()
}

func (agent *SBaremetalAgent) disableUDPOffloading() {
	log.Infof("Disable UDP offloading")
	offTx := procutils.NewCommand("ethtool", "--offload", o.Options.ListenInterface, "tx", "off")
	offTx.Run()
	offGso := procutils.NewCommand("ethtool", "-K", o.Options.ListenInterface, "gso", "off")
	offGso.Run()
}

func (agent *SBaremetalAgent) startServices(manager *SBaremetalManager) {
	listenIP, err := agent.GetListenIP()
	if err != nil {
		log.Fatalf("Get listen ip address error: %v", err)
	}
	agent.PXEServer = &pxe.Server{
		TFTPRootDir:      o.Options.TftpRoot,
		Address:          listenIP.String(),
		BaremetalManager: manager,
	}
	go func() {
		err := agent.PXEServer.Serve()
		if err != nil {
			log.Fatalf("Start PXE server error: %v", err)
		}
	}()
}

func Start() error {
	var err error
	if BaremetalAgent != nil {
		log.Warningf("Global BaremetalAgent already start")
		return nil
	}
	BaremetalAgent, err = newBaremetalAgent()
	if err != nil {
		return err
	}
	BaremetalAgent.startRegister()

	return nil
}
