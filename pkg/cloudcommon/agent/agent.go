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

package agent

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"

	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type IAgent interface {
	GetAgentType() string
	GetAccessIP() (net.IP, error)
	GetListenIP() (net.IP, error)
	GetPort() int
	GetEnableSsl() bool
	GetZoneName() string
	GetAdminSession() *mcclient.ClientSession
	TuneSystem() error
	StartService() error
	StopService() error
}

type SZoneInfo struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type SBaseAgent struct {
	object.SObject

	ListenInterface *net.Interface
	ListenIPs       []net.IP
	AgentId         string
	AgentName       string
	Zone            *SZoneInfo

	CachePath    string
	CacheManager *storageman.SLocalImageCacheManager

	stop bool
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

func (agent *SBaseAgent) IAgent() IAgent {
	return agent.GetVirtualObject().(IAgent)
}

func (agent *SBaseAgent) Init(iagent IAgent, ifname string, cachePath string) error {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return err
	}
	var ips []net.IP
	MAX := 60
	wait := 0
	for wait < MAX {
		ips, err = getIfaceIPs(iface)
		if err != nil {
			return err
		}
		if len(ips) == 0 {
			time.Sleep(2 * time.Second)
			wait += 2
		} else {
			break
		}
	}
	if len(ips) == 0 {
		return fmt.Errorf("Interface %s ip address not found", ifname)
	}
	log.Debugf("Interface %s ip address: %v", iface.Name, ips)
	agent.SetVirtualObject(iagent)
	agent.ListenInterface = iface
	agent.ListenIPs = ips
	agent.CachePath = cachePath
	return nil
}

func (agent *SBaseAgent) GetListenIPs() []net.IP {
	return agent.ListenIPs
}

func (agent *SBaseAgent) FindListenIP(listenAddr string) (net.IP, error) {
	ips := agent.GetListenIPs()
	if listenAddr == "" {
		return ips[0], nil
	}
	if listenAddr == "0.0.0.0" {
		return net.ParseIP(listenAddr), nil
	}
	for _, ip := range ips {
		if ip.String() == listenAddr {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("Not found Address %s on Interface %#v", listenAddr, agent.ListenInterface)
}

func (agent *SBaseAgent) FindAccessIP(accessAddr string) (net.IP, error) {
	if accessAddr == "0.0.0.0" {
		return nil, fmt.Errorf("Access address must be specific, should not be 0.0.0.0")
	}
	return agent.FindListenIP(accessAddr)
}

func (agent *SBaseAgent) startRegister() error {
	// if agent.AgentId != "" {
	// 	return
	// }

	var delayRetryTime = 30 * time.Second
	var lastTry time.Time

	for !agent.stop {
		if time.Now().Sub(lastTry) >= delayRetryTime {
			session := agent.IAgent().GetAdminSession()
			err := agent.register(session)
			if err == nil {
				log.Infof("Register success!")
				return nil
			}
			log.Errorf("Register error: %v, retry after %s...", err, delayRetryTime)
			lastTry = time.Now()
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("Error Stop")
}

func (agent *SBaseAgent) register(session *mcclient.ClientSession) error {
	var err error
	err = agent.fetchZone(session)
	if err != nil {
		return err
	}
	err = agent.createOrUpdateBaremetalAgent(session)
	if err != nil {
		return err
	}
	log.Infof("%s %s:%s register success, do offline", agent.IAgent().GetAgentType(), agent.AgentName, agent.AgentId)
	err = agent.doOffline(session)
	if err != nil {
		return err
	}
	return nil
}

func (agent *SBaseAgent) fetchZone(session *mcclient.ClientSession) error {
	zoneName := agent.IAgent().GetZoneName()
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
	zone := SZoneInfo{}
	err = zoneInfoObj.Unmarshal(&zone)
	if err != nil {
		return err
	}
	agent.Zone = &zone
	return nil
}

func (agent *SBaseAgent) getZoneByIP(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	listenIP, err := agent.IAgent().GetListenIP()
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(listenIP.String()), "ip")
	params.Add(jsonutils.JSONTrue, "is_on_premise")
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

func (agent *SBaseAgent) createOrUpdateBaremetalAgent(session *mcclient.ClientSession) error {
	params := jsonutils.NewDict()
	naccessIP, err := agent.IAgent().GetAccessIP()
	if err != nil {
		return err
	}
	params.Add(jsonutils.NewString(naccessIP.String()), "access_ip")
	params.Add(jsonutils.NewString(agent.IAgent().GetAgentType()), "agent_type")
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
		agentId, _ := cloudBmAgent.GetString("id")
		cloudObj, err = agent.updateBaremetalAgent(session, agentId, "")
		if err != nil {
			return err
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

	storageCacheId, _ := cloudObj.GetString("storagecache_id")
	if len(storageCacheId) == 0 {
		storageCacheId, err = agent.createStorageCache(session)
		if err != nil {
			return err
		}
		_, err = agent.updateBaremetalAgent(session, agentId, storageCacheId)
		if err != nil {
			return err
		}
	} else {
		err = agent.updateStorageCache(session, storageCacheId)
		if err != nil {
			return err
		}
	}
	agent.CacheManager = storageman.NewLocalImageCacheManager(agent.IAgent(), agent.CachePath, storageCacheId)

	return nil
}

func (agent *SBaseAgent) GetManagerUri() string {
	accessIP, _ := agent.IAgent().GetAccessIP()
	proto := "http"
	if agent.IAgent().GetEnableSsl() {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s:%d", proto, accessIP, agent.IAgent().GetPort())
}

func (agent *SBaseAgent) GetListenUri() string {
	listenIP, _ := agent.IAgent().GetListenIP()
	proto := "http"
	if agent.IAgent().GetEnableSsl() {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s:%d", proto, listenIP, agent.IAgent().GetPort())
}

func (agent *SBaseAgent) getCreateUpdateInfo() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if agent.AgentId == "" {
		agentName, err := agent.getName()
		if err != nil {
			return nil, errors.Wrap(err, "agent.getName")
		}
		params.Add(jsonutils.NewString(agentName), "name")
	}
	accessIP, err := agent.IAgent().GetAccessIP()
	if err != nil {
		return nil, errors.Wrap(err, "agent.IAgent().GetAccessIP()")
	}
	params.Add(jsonutils.NewString(accessIP.String()), "access_ip")
	params.Add(jsonutils.NewString(agent.GetManagerUri()), "manager_uri")
	params.Add(jsonutils.NewString(agent.Zone.Id), "zone_id")
	params.Add(jsonutils.NewString(agent.IAgent().GetAgentType()), "agent_type")
	params.Add(jsonutils.NewString(version.GetShortString()), "version")

	return params, nil
}

func (agent *SBaseAgent) getName() (string, error) {
	accessIP, err := agent.IAgent().GetAccessIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", agent.IAgent().GetAgentType(), accessIP), nil
}

func (agent *SBaseAgent) createStorageCache(session *mcclient.ClientSession) (string, error) {
	body := jsonutils.NewDict()
	agentName, err := agent.getName()
	if err != nil {
		return "", errors.Wrap(err, "agent.getName")
	}
	body.Set("name", jsonutils.NewString("imagecache-"+agentName))
	body.Set("path", jsonutils.NewString(agent.CachePath))
	body.Set("external_id", jsonutils.NewString(agent.AgentId))
	sc, err := modules.Storagecaches.Create(session, body)
	if err != nil {
		return "", errors.Wrap(err, "modules.Storagecaches.Create")
	}
	storageCacheId, err := sc.GetString("id")
	if err != nil {
		return "", errors.Wrap(err, "sc.GetString id")
	}
	return storageCacheId, nil
}

func (agent *SBaseAgent) updateStorageCache(session *mcclient.ClientSession, storageCacheId string) error {
	body := jsonutils.NewDict()
	body.Set("path", jsonutils.NewString(agent.CachePath))
	body.Set("external_id", jsonutils.NewString(agent.AgentId))
	_, err := modules.Storagecaches.Update(session, storageCacheId, body)
	if err != nil {
		return errors.Wrap(err, "modules.Storagecaches.Update")
	}
	return nil
}

func (agent *SBaseAgent) createBaremetalAgent(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	params, err := agent.getCreateUpdateInfo()
	if err != nil {
		return nil, err
	}
	return modules.Baremetalagents.Create(session, params)
}

func (agent *SBaseAgent) updateBaremetalAgent(session *mcclient.ClientSession, id string, storageCacheId string) (jsonutils.JSONObject, error) {
	var params jsonutils.JSONObject
	var err error
	if len(storageCacheId) > 0 {
		params = jsonutils.NewDict()
		params.(*jsonutils.JSONDict).Set("storagecache_id", jsonutils.NewString(storageCacheId))
	} else {
		params, err = agent.getCreateUpdateInfo()
		if err != nil {
			return nil, err
		}
	}
	return modules.Baremetalagents.Update(session, id, params)
}

func (agent *SBaseAgent) doOffline(session *mcclient.ClientSession) error {
	_, err := modules.Baremetalagents.PerformAction(session, agent.AgentId, "offline", nil)
	return err
}

func (agent *SBaseAgent) DoOnline(session *mcclient.ClientSession) error {
	_, err := modules.Baremetalagents.PerformAction(session, agent.AgentId, "online", nil)
	return err
}

func (agent *SBaseAgent) Start() error {
	err := agent.startRegister()
	if err != nil {
		return err
	}
	agent.IAgent().TuneSystem()
	return agent.IAgent().StartService()
}

func (agent *SBaseAgent) Stop() {
	agent.stop = true
	agent.IAgent().StopService()
}
