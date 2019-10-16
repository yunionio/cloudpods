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
	"context"
	"fmt"
	"net"
	"net/http"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	"yunion.io/x/onecloud/pkg/cloudcommon/agent"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	AGENT_TYPE_BAREMETAL = "baremetal"
)

var (
	baremetalAgent *SBaremetalAgent
)

//
// BaremetalAgent has two types of address
// - AccessAddress/Address: this is the address controller to accesss the agent
// - ListenAddress: this is the address baremetal to access the agent
//
type SBaremetalAgent struct {
	agent.SBaseAgent

	PXEServer *pxe.Server
	Manager   *SBaremetalManager
}

func newBaremetalAgent() (*SBaremetalAgent, error) {
	agent := &SBaremetalAgent{}
	err := agent.Init(agent, o.Options.ListenInterface, o.Options.CachePath)
	if err != nil {
		return nil, err
	}
	// set guest fs NetDevPrefix
	fsdriver.NetDevPrefix = "en"
	return agent, nil
}

func (agent *SBaremetalAgent) GetAgentType() string {
	return AGENT_TYPE_BAREMETAL
}

func (agent *SBaremetalAgent) GetPort() int {
	return o.Options.Port
}

func (agent *SBaremetalAgent) GetEnableSsl() bool {
	return o.Options.EnableSsl
}

func (agent *SBaremetalAgent) GetZoneName() string {
	return o.Options.Zone
}

func (agent *SBaremetalAgent) GetAdminSession() *mcclient.ClientSession {
	return auth.GetAdminSession(context.TODO(), o.Options.Region, "v2")
}

func (agent *SBaremetalAgent) GetListenIP() (net.IP, error) {
	return agent.FindListenIP(o.Options.ListenAddress)
}

func (agent *SBaremetalAgent) GetDHCPServerListenIP() (net.IP, error) {
	ips := agent.GetListenIPs()

	// baremetal dhcp server can't bind address 0.0.0.0:67, conflict with host agent
	// but can bind specific ip address, because socket set reuseaddr option
	if o.Options.ListenAddress == "" || o.Options.ListenAddress == "0.0.0.0" {
		return ips[0], nil
	}
	for _, ip := range ips {
		if ip.String() == o.Options.ListenAddress {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("Not found ListenAddress %s on %s", o.Options.ListenAddress, o.Options.ListenInterface)
}

func (agent *SBaremetalAgent) GetAccessIP() (net.IP, error) {
	if o.Options.AccessAddress != "" && o.Options.AccessAddress != "0.0.0.0" {
		return net.ParseIP(o.Options.AccessAddress), nil
	}
	if o.Options.Address != "" && o.Options.Address != "0.0.0.0" {
		return net.ParseIP(o.Options.Address), nil
	}
	return agent.FindAccessIP(o.Options.AccessAddress)
}

func (agent *SBaremetalAgent) GetDHCPServerIP() (net.IP, error) {
	listenIP := o.Options.ListenAddress
	if len(listenIP) == 0 || listenIP == "0.0.0.0" {
		return agent.GetAccessIP()
	}
	return agent.GetListenIP()
}

func (agent *SBaremetalAgent) StartService() error {
	manager, err := NewBaremetalManager(agent)
	if err != nil {
		return fmt.Errorf("New baremetal manager error: %v", err)
	}

	err = manager.loadConfigs()
	if err != nil {
		return fmt.Errorf("Baremetal manager load config error: %v", err)
	}

	agent.Manager = manager

	agent.startPXEServices(manager)
	agent.startFileServer()

	agent.DoOnline(agent.GetAdminSession())
	return nil
}

func (agent *SBaremetalAgent) StopService() error {
	if agent.Manager != nil {
		agent.Manager.Stop()
	}
	return nil
}

func (agent *SBaremetalAgent) GetManager() *SBaremetalManager {
	return agent.Manager
}

func (agent *SBaremetalAgent) TuneSystem() error {
	agent.disableUDPOffloading()
	return nil
}

func (agent *SBaremetalAgent) disableUDPOffloading() {
	log.Infof("Disable UDP offloading")
	offTx := procutils.NewCommand("ethtool", "--offload", o.Options.ListenInterface, "tx", "off")
	offTx.Run()
	offGso := procutils.NewCommand("ethtool", "-K", o.Options.ListenInterface, "gso", "off")
	offGso.Run()
}

func (agent *SBaremetalAgent) startPXEServices(manager *SBaremetalManager) {
	dhcpListenIp, err := agent.GetDHCPServerListenIP()
	if err != nil {
		log.Fatalf("Get dhcp listen ip address error: %v", err)
	}
	agent.PXEServer = &pxe.Server{
		TFTPRootDir:      o.Options.TftpRoot,
		Address:          dhcpListenIp.String(),
		BaremetalManager: manager,
		ListenIface:      agent.ListenInterface.Name,
	}
	go func() {
		err := agent.PXEServer.Serve()
		if err != nil {
			log.Fatalf("Start PXE server error: %v", err)
		}
	}()
}

func (agent *SBaremetalAgent) startFileServer() {
	dhcpListenIp, err := agent.GetDHCPServerListenIP()
	if err != nil {
		log.Fatalf("Get dhcp listen ip address error: %v", err)
	}
	fs := http.FileServer(httputils.Dir(o.Options.TftpRoot))
	http.Handle("/tftp/", http.StripPrefix("/tftp/", fs))
	cacheFs := http.FileServer(httputils.Dir(o.Options.CachePath))
	http.Handle("/images/", http.StripPrefix("/images/", cacheFs))
	isoFs := http.FileServer(httputils.Dir(o.Options.BootIsoPath))
	http.Handle("/bootiso/", http.StripPrefix("/bootiso/", isoFs))
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("%s:%d", dhcpListenIp, o.Options.Port+1000), nil); err != nil {
			panic(fmt.Sprintf("start http file server: %v", err))
		}
	}()
}

func Start(app *appsrv.Application) error {
	var err error
	if baremetalAgent != nil {
		log.Warningf("Global baremetalAgent already start")
		return nil
	}
	baremetalAgent, err = newBaremetalAgent()
	if err != nil {
		return err
	}
	err = baremetalAgent.Start()
	if err != nil {
		return err
	}
	baremetalAgent.AddImageCacheHandler("", app)
	return nil
}

func Stop() error {
	if baremetalAgent != nil {
		log.Infof("baremetalAgent stop ...")
		tmpAgent := baremetalAgent
		baremetalAgent = nil
		tmpAgent.Stop()
	}
	return nil
}

func GetBaremetalAgent() *SBaremetalAgent {
	return baremetalAgent
}

func GetBaremetalManager() *SBaremetalManager {
	return GetBaremetalAgent().GetManager()
}
