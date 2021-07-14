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

package azure

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancer struct {
	region    *SRegion
	eip       cloudprovider.ICloudEIP
	lbbgs     []cloudprovider.ICloudLoadbalancerBackendGroup
	listeners []cloudprovider.ICloudLoadbalancerListener

	Name       string                  `json:"name"`
	ID         string                  `json:"id"`
	Etag       string                  `json:"etag"`
	Type       string                  `json:"type"`
	Tags       map[string]string       `json:"tags"`
	Location   string                  `json:"location"`
	Properties SLoadbalancerProperties `json:"properties"`
	Sku        Sku                     `json:"sku"`
}

func (self *SLoadbalancer) GetId() string {
	return self.ID
}

func (self *SLoadbalancer) GetName() string {
	return self.Name
}

func (self *SLoadbalancer) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancer) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Deleting":
		return api.LB_STATUS_DELETING
	case "Failed":
		return api.LB_STATUS_START_FAILED
	case "Updating":
		return api.LB_SYNC_CONF
	}

	switch self.Properties.OperationalState {
	case "Running":
		return api.LB_STATUS_ENABLED
	case "Stopped":
		return api.LB_STATUS_DISABLED
	case "Starting", "Stopping":
		return api.LB_SYNC_CONF
	default:
		if self.Properties.ProvisioningState == "Succeeded" {
			return api.LB_STATUS_ENABLED
		}

		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancer) Refresh() error {
	lb, err := self.region.GetILoadBalancerById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerById.Refresh")
	}

	err = jsonutils.Update(self, lb)
	if err != nil {
		return errors.Wrap(err, "jsonutils.Update")
	}

	self.eip = nil
	self.lbbgs = nil
	self.listeners = nil
	return nil
}

func (self *SLoadbalancer) IsEmulated() bool {
	return false
}

func (self *SLoadbalancer) GetSysTags() map[string]string {
	data := map[string]string{}
	data["loadbalance_type"] = self.Type
	data["properties"] = jsonutils.Marshal(self.Properties).String()
	return data
}

func (self *SLoadbalancer) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadbalancer) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SLoadbalancer) getAddresses() [][]string {
	ret := [][]string{[]string{}, []string{}}
	for _, fip := range self.Properties.FrontendIPConfigurations {
		eip := fip.Properties.PublicIPAddress
		if eip != nil && len(eip.Properties.IPAddress) > 0 {
			ret[0] = append(ret[0], fip.Properties.PublicIPAddress.Properties.IPAddress)
			continue
		}

		if len(fip.Properties.PrivateIPAddress) > 0 {
			ret[1] = append(ret[1], fip.Properties.PrivateIPAddress)
			continue
		}
	}

	return ret
}

func (self *SLoadbalancer) GetAddress() string {
	ips := self.getAddresses()
	if len(ips[0]) > 0 {
		return ips[0][0]
	} else if len(ips[1]) > 0 {
		return ips[1][0]
	} else {
		return ""
	}
}

func (self *SLoadbalancer) GetAddressType() string {
	ips := self.getAddresses()
	if len(ips[0]) > 0 {
		return api.LB_ADDR_TYPE_INTERNET
	} else if len(ips[1]) > 0 {
		return api.LB_ADDR_TYPE_INTRANET
	} else {
		return api.LB_ADDR_TYPE_INTRANET
	}
}

func (self *SLoadbalancer) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SLoadbalancer) getNetworkIds() []string {
	ret := []string{}
	for _, fip := range self.Properties.FrontendIPConfigurations {
		subnet := fip.Properties.Subnet
		if len(subnet.ID) > 0 {
			ret = append(ret, subnet.ID)
		}
	}

	return ret
}

func (self *SLoadbalancer) getLbbgNetworkIds() []string {
	ret := []string{}
	for _, b := range self.Properties.BackendAddressPools {
		bips := b.Properties.BackendIPConfigurations
		if len(bips) > 0 {
			for _, ip := range bips {
				if strings.Contains(ip.ID, "Microsoft.Network/networkInterfaces") {
					nic, _ := self.region.GetNetworkInterface(strings.Split(ip.ID, "/ipConfigurations")[0])
					if nic != nil && len(nic.Properties.IPConfigurations) > 0 {
						ipc := nic.Properties.IPConfigurations[0]
						if len(ipc.Properties.Subnet.ID) > 0 {
							return []string{ipc.Properties.Subnet.ID}
						}
					}
				}
			}
		}
	}

	return ret
}

func (self *SLoadbalancer) GetNetworkIds() []string {
	return self.getNetworkIds()
}

func (self *SLoadbalancer) GetVpcId() string {
	if self.Type == "Microsoft.Network/loadBalancers" {
		subnets := self.GetNetworkIds()
		if len(subnets) == 0 {
			subnets = self.getLbbgNetworkIds()
		}

		if len(subnets) > 0 {
			network, err := self.region.GetNetwork(subnets[0])
			if network != nil {
				return strings.Split(network.GetId(), "/subnets")[0]
			}

			log.Errorf("GetNetwork %s: %s", subnets[0], err)
		}
	} else {
		gips := self.Properties.GatewayIPConfigurations
		for i := range gips {
			netId := gips[i].Properties.Subnet.ID
			if len(netId) > 0 {
				network, err := self.region.GetNetwork(netId)
				if network != nil {
					return strings.Split(network.GetId(), "/subnets")[0]
				}
				log.Errorf("GetNetwork %s: %s", netId, err)
			}
		}
	}

	return ""
}

func (self *SLoadbalancer) GetZoneId() string {
	ips := self.Properties.FrontendIPConfigurations
	if len(ips) > 0 && len(ips[0].Zones) > 0 {
		return ips[0].Zones[0]
	}

	return ""
}

func (self *SLoadbalancer) GetZone1Id() string {
	ips := self.Properties.FrontendIPConfigurations
	if len(ips) > 0 && len(ips[0].Zones) > 1 {
		return ips[0].Zones[1]
	}

	return ""
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	if len(self.Properties.Sku.Name) > 0 {
		return self.Properties.Sku.Name
	}

	if len(self.Sku.Name) > 0 {
		return self.Sku.Name
	}
	return ""
}

func (self *SLoadbalancer) GetChargeType() string {
	return api.LB_CHARGE_TYPE_POSTPAID
}

func (self *SLoadbalancer) GetEgressMbps() int {
	return 0
}

func (self *SLoadbalancer) getEipIds() []string {
	ret := []string{}
	for _, fip := range self.Properties.FrontendIPConfigurations {
		eip := fip.Properties.PublicIPAddress
		if eip != nil && len(eip.ID) > 0 {
			ret = append(ret, eip.ID)
			continue
		}
	}

	return ret
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if self.eip != nil {
		return self.eip, nil
	}

	eips := self.getEipIds()
	if len(eips) > 0 {
		eip, err := self.region.GetIEipById(eips[0])
		self.eip = eip
		return eip, err
	}

	return nil, nil
}

func (self *SLoadbalancer) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancer) Start() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Start")
}

func (self *SLoadbalancer) Stop() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Stop")
}

/*
应用型LB： urlPathMaps（defaultBackendAddressPool+defaultBackendHttpSettings+requestRoutingRules+httpListeners）= Onecloud监听器
4层LB: loadBalancingRules（前端） = Onecloud监听器
*/
func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	if self.listeners != nil {
		return self.listeners, nil
	}

	var err error
	if self.Type == "Microsoft.Network/applicationGateways" {
		self.listeners, err = self.getAppLBListeners()
	} else {
		self.listeners, err = self.getNetworkLBListeners()
	}

	return self.listeners, err
}

func (self *SLoadbalancer) getFrontendIPConfiguration(id string) *FrontendIPConfiguration {
	fips := self.Properties.FrontendIPConfigurations
	for i := range fips {
		if fips[i].ID == id {
			return &fips[i]
		}
	}
	log.Debugf("getFrontendIPConfiguration %s not found", id)
	return nil
}

func (self *SLoadbalancer) getHttpListener(id string) *HTTPListener {
	ss := self.Properties.HTTPListeners
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}

		for j := range ss[i].Properties.RequestRoutingRules {
			if ss[i].Properties.RequestRoutingRules[j].ID == id {
				return &ss[i]
			}
		}
	}
	log.Debugf("getHttpListener %s not found", id)
	return nil
}

func (self *SLoadbalancer) getBackendAddressPool(id string) *BackendAddressPool {
	ss := self.Properties.BackendAddressPools
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getBackendAddressPool %s not found", id)
	return nil
}

func (self *SLoadbalancer) getBackendHTTPSettingsCollection(id string) *BackendHTTPSettingsCollection {
	ss := self.Properties.BackendHTTPSettingsCollection
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getBackendHTTPSettingsCollection %s not found", id)
	return nil
}

func (self *SLoadbalancer) getRedirectConfiguration(id string) *RedirectConfiguration {
	ss := self.Properties.RedirectConfigurations
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getRedirectConfiguration %s not found", id)
	return nil
}

func (self *SLoadbalancer) getProbe(id string) *Probe {
	ss := self.Properties.Probes
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getProbe %s not found", id)
	return nil
}

func (self *SLoadbalancer) getFrontendPort(id string) *FrontendPort {
	ss := self.Properties.FrontendPorts
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getFrontendPort %s not found", id)
	return nil
}

func (self *SLoadbalancer) getRequestRoutingRule(id string) *RequestRoutingRule {
	ss := self.Properties.RequestRoutingRules
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getRequestRoutingRule %s not found", id)
	return nil
}

func (self *SLoadbalancer) getURLPathMap(id string) *URLPathMap {
	ss := self.Properties.URLPathMaps
	for i := range ss {
		if ss[i].ID == id {
			return &ss[i]
		}
	}
	log.Debugf("getURLPathMap %s not found", id)
	return nil
}

/*
应用型LB： urlPathMaps（defaultBackendAddressPool+defaultBackendHttpSettings+requestRoutingRules+httpListeners）= Onecloud监听器
*/
func (self *SLoadbalancer) getAppLBListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	lbls := []cloudprovider.ICloudLoadbalancerListener{}
	for i := range self.Properties.RequestRoutingRules {
		r := self.Properties.RequestRoutingRules[i]
		uid := strings.Replace(r.ID, "requestRoutingRules", "urlPathMaps", 1)
		u := self.getURLPathMap(uid)
		lbl := self.getAppLBListener(&r, u)
		if lbl != nil {
			lbls = append(lbls, lbl)
		}
	}

	return lbls, nil
}

func (self *SLoadbalancer) getAppLBListener(r *RequestRoutingRule, u *URLPathMap) *SLoadBalancerListener {
	var redirect *RedirectConfiguration
	var listener *HTTPListener
	var fp *FrontendIPConfiguration
	var fpp *FrontendPort
	// frontendPort, frontendIPConfiguration, sslCertificate, requestRoutingRules
	listener = self.getHttpListener(r.Properties.HTTPListener.ID)
	// frontendPorts
	fp = self.getFrontendIPConfiguration(listener.Properties.FrontendIPConfiguration.ID)
	fpp = self.getFrontendPort(listener.Properties.FrontendPort.ID)
	// 配置有异常？？
	if fpp == nil || listener == nil {
		return nil
	}

	lbbgId := r.Properties.BackendAddressPool.ID
	lbbgSettingId := r.Properties.BackendHTTPSettings.ID
	provisioningState := "Succeeded"
	rules := make([]PathRule, 0)
	var backendGroup *BackendAddressPool
	var backendSetting *BackendHTTPSettingsCollection
	var backendPort int
	var healthcheck *Probe
	if u != nil {
		provisioningState = u.Properties.ProvisioningState
		rules = u.Properties.PathRules
		if u.Properties.DefaultRedirectConfiguration != nil {
			redirect = self.getRedirectConfiguration(u.Properties.DefaultRedirectConfiguration.ID)
		}

		if len(lbbgId) == 0 && u.Properties.DefaultBackendAddressPool != nil {
			lbbgId = u.Properties.DefaultBackendAddressPool.ID
		}

		if len(lbbgSettingId) == 0 && u.Properties.DefaultBackendHTTPSettings != nil {
			lbbgSettingId = u.Properties.DefaultBackendHTTPSettings.ID
		}
	}

	if len(lbbgId) > 0 {
		backendGroup = self.getBackendAddressPool(lbbgId)
	}

	if len(lbbgSettingId) > 0 {
		backendSetting = self.getBackendHTTPSettingsCollection(lbbgSettingId)
		backendPort = backendSetting.Properties.Port
	}

	if backendSetting != nil && backendSetting.Properties.Probe != nil {
		healthcheck = self.getProbe(backendSetting.Properties.Probe.ID)
	}

	return &SLoadBalancerListener{
		lb:                self,
		fp:                fp,
		listener:          listener,
		backendSetting:    backendSetting,
		backendGroup:      backendGroup,
		redirect:          redirect,
		healthcheck:       healthcheck,
		Name:              r.Name,
		ID:                r.ID,
		ProvisioningState: provisioningState,
		IPVersion:         "",
		Protocol:          listener.Properties.Protocol,
		LoadDistribution:  "",
		FrontendPort:      fpp.Properties.Port,
		BackendPort:       backendPort,
		ClientIdleTimeout: 0,
		EnableFloatingIP:  false,
		EnableTcpReset:    false,
		rules:             rules,
	}
}

func (self *SLoadbalancer) getNetworkLBListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	lbls := []cloudprovider.ICloudLoadbalancerListener{}
	for i := range self.Properties.LoadBalancingRules {
		u := self.Properties.LoadBalancingRules[i]
		lbl := self.getNetworkLBListener(&u)
		lbls = append(lbls, &lbl)
	}

	return lbls, nil
}

func (self *SLoadbalancer) getNetworkLBListener(u *LoadBalancingRule) SLoadBalancerListener {
	fp := self.getFrontendIPConfiguration(u.Properties.FrontendIPConfiguration.ID)
	backendGroup := self.getBackendAddressPool(u.Properties.BackendAddressPool.ID)
	healthcheck := self.getProbe(u.Properties.Probe.ID)

	return SLoadBalancerListener{
		lb:                self,
		fp:                fp,
		backendGroup:      backendGroup,
		healthcheck:       healthcheck,
		Name:              u.Name,
		ID:                u.ID,
		ProvisioningState: u.Properties.ProvisioningState,
		IPVersion:         "",
		Protocol:          u.Properties.Protocol,
		LoadDistribution:  u.Properties.LoadDistribution,
		FrontendPort:      u.Properties.FrontendPort,
		BackendPort:       u.Properties.BackendPort,
		ClientIdleTimeout: u.Properties.IdleTimeoutInMinutes * 60,
		EnableFloatingIP:  u.Properties.EnableFloatingIP,
		EnableTcpReset:    u.Properties.EnableTCPReset,
	}
}

// 应用型LB：  HTTP 设置 + 后端池 = onecloud 后端服务器组
// 4层LB: loadBalancingRules(backendPort)+ 后端池 = onecloud 后端服务器组
func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	if self.lbbgs != nil {
		return self.lbbgs, nil
	}

	lbls, err := self.GetILoadBalancerListeners()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadBalancerListeners")
	}

	bgIds := []string{}
	ilbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := range lbls {
		lbl := lbls[i].(*SLoadBalancerListener)
		if lbl.backendGroup == nil {
			continue
		}

		bg := SLoadbalancerBackendGroup{
			lb:           self,
			Pool:         *lbl.backendGroup,
			DefaultPort:  lbl.BackendPort,
			HttpSettings: lbl.backendSetting,
			BackendIps:   lbl.backendGroup.Properties.BackendIPConfigurations,
		}

		if !utils.IsInStringArray(bg.GetId(), bgIds) {
			bgIds = append(bgIds, bg.GetId())
			ilbbgs = append(ilbbgs, &bg)
		}

		// sync rules lbbg
		if utils.IsInStringArray(lbl.GetListenerType(), []string{api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS}) {
			rules, err := lbl.GetILoadbalancerListenerRules()
			if err != nil {
				return nil, errors.Wrap(err, "GetILoadbalancerListenerRules")
			}

			for j := range rules {
				rule := rules[j].(*SLoadbalancerListenerRule)
				if rule.lbbg != nil && !utils.IsInStringArray(rule.lbbg.GetId(), bgIds) {
					bgIds = append(bgIds, bg.GetId())
					ilbbgs = append(ilbbgs, &bg)
				}
			}
		}
	}

	self.lbbgs = ilbbgs
	return ilbbgs, nil
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	lbbgs, err := self.GetILoadBalancerBackendGroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadBalancerBackendGroups")
	}

	for i := range lbbgs {
		if lbbgs[i].GetId() == groupId {
			return lbbgs[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetILoadBalancerBackendGroupById")
}

func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerBackendGroup")
}

func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerListener")
}

func (self *SLoadbalancer) getNetworkLBListenerById(id string) (cloudprovider.ICloudLoadbalancerListener, error) {
	for i := range self.Properties.LoadBalancingRules {
		u := self.Properties.LoadBalancingRules[i]
		if u.ID == id {
			lbl := self.getNetworkLBListener(&u)
			return &lbl, nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "getNetworkLBListenerById")
}

func (self *SLoadbalancer) getAppLBListenerById(id string) (cloudprovider.ICloudLoadbalancerListener, error) {
	for i := range self.Properties.RequestRoutingRules {
		r := self.Properties.RequestRoutingRules[i]
		if r.ID == id {
			u := self.getURLPathMap(strings.Replace(r.ID, "requestRoutingRules", "urlPathMaps", 1))
			return self.getAppLBListener(&r, u), nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "getAppLBListenerById")
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	if self.Type == "Microsoft.Network/applicationGateways" {
		return self.getAppLBListenerById(listenerId)
	} else {
		return self.getNetworkLBListenerById(listenerId)
	}
}

func (self *SLoadbalancer) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	icerts := []cloudprovider.ICloudLoadbalancerCertificate{}
	ssl := self.Properties.SSLCertificates
	for i := range ssl {
		s := ssl[i]
		cert := SLoadbalancerCert{
			lb:        self,
			Name:      s.Name,
			ID:        s.ID,
			PublicKey: s.Properties.PublicCertData,
		}

		icerts = append(icerts, &cert)
	}

	return icerts, nil
}

func (self *SLoadbalancer) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	ssl := self.Properties.SSLCertificates
	for i := range ssl {
		s := ssl[i]
		if s.ID == certId {
			return &SLoadbalancerCert{
				lb:        self,
				Name:      s.Name,
				ID:        s.ID,
				PublicKey: s.Properties.PublicCertData,
			}, nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetILoadBalancerCertificateById")
}
