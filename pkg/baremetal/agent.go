package baremetal

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

	return &SBaremetalAgent{
		PXEServer: &pxe.Server{
			TFTPRootDir: o.Options.TftpRoot,
			Address:     ips[0].String(),
		},
		ListenInterface: iface,
	}, nil
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

	var delayRetryTime time.Duration = 5 * time.Second

	for {
		err := agent.register()
		if err != nil {
			log.Errorf("Register error: %v", err)
			time.Sleep(delayRetryTime)
			continue
		}
		break
	}
	return
}

func (agent *SBaremetalAgent) register() error {
	var err error
	err = agent.fetchZone()
	if err != nil {
		return err
	}
	err = agent.fetchBaremetalAgent()
	if err != nil {
		return err
	}
}

func (agent *SBaremetalAgent) fetchZone() error {
	zoneName := o.Options.Zone
	var zoneInfoObj jsonutils.JSONObject
	var err error
	if zoneName != "" {
		zoneInfoObj, err = modules.Zones.Get(GetAdminSession(), zoneName, nil)
		if err != nil {
			return err
		}
	} else {

	}
	zone := SZone{}
	err = zoneInfoObj.Unmarshal(&zone)
	if err != nil {
		return err
	}
	agent.Zone = &zone
	return nil
}

func (agent *SBaremetalAgent) fetchBaremetalAgent() error {
	params := jsonutils.NewDict()
	listenIP, err := agent.GetListenIP()
	if err != nil {
		return err
	}
	params.Add(jsonutils.NewString(listenIP.String()), "access_ip")
	ret, err := modules.Baremetalagents.List(GetAdminSession(), params)
	if err != nil {
		return err
	}
	if len(ret.Data) == 0 {
		agent.
	}
	return nil
}

func (agent *SBaremetalAgent) startServices() error {
	return agent.PXEServer.Serve()
}

func Start() error {
	if BaremetalAgent != nil {
		log.Warningf("Global BaremetalAgent already start")
		return nil
	}
	BaremetalAgent, err := newBaremetalAgent()
	if err != nil {
		return err
	}
	BaremetalAgent.startRegister()
	err = BaremetalAgent.startServices()
	if err != nil {
		return err
	}

	return nil
}
